package item

import (
	"go.uber.org/zap"
	"math/rand"
	appitem "mcgame-server/internal/application/item"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
	"strconv"
	"time"
)

func stackUpgradeRate(count, colour int, jewel bool, vipBonus int) int {
	base := 20
	if !jewel && colour == 4 {
		base = 6
	}
	rate := count * base
	if base == 6 && vipBonus > 0 {
		rate += vipBonus
	}
	if rate > 100 {
		rate = 100
	}
	return rate
}

func (h *Handler) runStackUpgrade(ctx *rtmp.RPCContext, args []interface{}, jewel, all bool) (interface{}, error) {
	if len(args) < 3 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	num, ok := parseFlexibleInt64(args[0])
	if !ok || num < 2 || num > 5 {
		return map[string]interface{}{"flag": false}, nil
	}
	itemID, ok := parseFlexibleInt64(args[1])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	temp, ok := args[2].(bool)
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	charID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}
	var source *domainitem.Item
	if temp {
		source, err = h.itemService.GetItemInSlot(ctx.Context, charID, domainitem.SlotTypeTempBag, int(itemID))
	} else {
		source, err = h.itemService.GetItemByID(ctx.Context, charID, itemID)
	}
	if err != nil || source == nil || source.StackCount < int(num) || (source.SlotType != domainitem.SlotTypeBag && source.SlotType != domainitem.SlotTypeTempBag) {
		return map[string]interface{}{"flag": false}, nil
	}
	tpl := h.gameData.GetItem(source.TemplateID)
	if tpl == nil {
		return map[string]interface{}{"flag": false}, nil
	}
	nextID := source.TemplateID
	colour := source.ColorCode
	if colour == 0 {
		colour = int(tpl.Color)
	}
	nextColour := colour + 1
	nextTpl := tpl
	if jewel {
		if int(tpl.Type) != 503 || tpl.NextJewelTid <= 0 {
			return map[string]interface{}{"flag": false}, nil
		}
		nextID = int(tpl.NextJewelTid)
		nextTpl = h.gameData.GetItem(nextID)
		if nextTpl == nil || int(nextTpl.Type) != 503 {
			return map[string]interface{}{"flag": false}, nil
		}
		nextColour = 0
	} else if (int(tpl.Kind) != 5 && int(tpl.Kind) != 6) || colour < 0 || colour >= 5 {
		return map[string]interface{}{"flag": false}, nil
	}
	vipBonus := 0
	if !jewel && colour == 4 && h.charService != nil {
		char, loadErr := h.charService.GetByID(ctx.Context, charID)
		if loadErr != nil {
			return nil, loadErr
		}
		if level := char.CurrentPMLevel(time.Now()); level > 0 {
			if right := h.gameData.GetPmRight(8); right != nil {
				if value, ok := parseFlexibleInt64(right.ToDTO()["value"+strconv.Itoa(level)]); ok && value > 0 && value <= 100 {
					vipBonus = int(value)
				}
			}
		}
	}
	attempts := 1
	if all {
		attempts = source.StackCount / int(num)
	}
	rate := stackUpgradeRate(int(num), colour, jewel, vipBonus)
	produced := 0
	for i := 0; i < attempts; i++ {
		if rand.Intn(100) < rate {
			produced++
		}
	}
	rewards := []appitem.BagReward{}
	if produced > 0 {
		rewards = append(rewards, appitem.BagReward{TemplateID: nextID, ItemType: source.ItemType, Count: produced, IsBound: source.IsBound, ColorCode: nextColour})
	}
	result, err := h.itemService.ExchangeUpgradeItems(ctx.Context, charID, source.ID, attempts*int(num), rewards)
	if err != nil {
		h.logger.Error("Stack upgrade could not commit", zap.Int64("character_id", charID), zap.Int64("source_id", source.ID), zap.Error(err))
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể hoàn tất ghép vật phẩm: kiểm tra chỗ trống túi đồ và túi tạm. Không tính là thất bại may mắn.")
		}
		return map[string]interface{}{"flag": false}, nil
	}
	if ctx.Connection != nil {
		if result.Deleted {
			_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(result.Source.ID), float64(result.Source.CalculateSID()))
		} else {
			h.sendSlotUpdate(ctx, h.itemDTO(result.Source))
		}
		for _, product := range result.Products {
			h.sendSlotUpdate(ctx, h.itemDTO(product))
		}
		if produced > 0 {
			_ = ctx.Connection.SendCallback("onAddItem", map[string]interface{}{"t": 29, "i": nextID, "n": nextTpl.Name, "c": nextColour, "q": nextColour, "tt": int(nextTpl.Kind), "s": ""})
		}
	}
	h.logger.Info("Stack upgrade committed", zap.Int64("character_id", charID), zap.Int64("source_id", source.ID), zap.Bool("jewel", jewel), zap.Int("attempts", attempts), zap.Int("rate", rate), zap.Int("produced", produced), zap.Int("remaining", result.Source.StackCount))
	if produced > 0 {
		h.checkBagCapacityWarning(ctx, charID)
	}
	slotID := source.ID
	if temp {
		slotID = itemID
	}
	return map[string]interface{}{"slotId": float64(slotID), "num": float64(result.Source.StackCount), "flag": produced > 0, "finalNum": float64(produced)}, nil
}
