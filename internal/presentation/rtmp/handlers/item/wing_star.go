package item

import (
	"math/rand"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/infrastructure/rtmp"
	"strconv"
)

func wingStarFailureLevel(current int) int {
	if current < 0 || current >= 10 {
		return 0
	}
	return []int{0, 0, 1, 2, 3, 4, 4, 5, 0, 0}[current]
}

func (h *Handler) AddWingStar(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	failure := map[string]interface{}{"flag": false}
	if len(args) != 4 {
		return failure, nil
	}
	count, ok := parseFlexibleInt64(args[0])
	if !ok || count < 1 || count > 5 {
		return failure, nil
	}
	equipID, ok := parseFlexibleInt64(args[1])
	if !ok {
		return failure, nil
	}
	stoneID, ok := parseFlexibleInt64(args[2])
	if !ok {
		return failure, nil
	}
	target, ok := parseFlexibleInt64(args[3])
	if !ok || target < 0 || target > 10 {
		return failure, nil
	}
	charID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return failure, nil
	}
	equip, err := h.itemService.GetItemByID(ctx.Context, charID, equipID)
	if err != nil {
		return failure, nil
	}
	tpl := h.itemService.GetEquipmentTemplate(equip.TemplateID)
	if tpl == nil || int(tpl.Kind) != 13 || int(tpl.Position) != 22 || equip.ItemType != domainitem.ItemTypeEquipment || equip.StarLevel < 0 || equip.StarLevel >= 10 {
		return failure, nil
	}
	stone, err := h.itemService.GetItemByID(ctx.Context, charID, stoneID)
	if err != nil || stone.TemplateID != 2906 {
		return failure, nil
	}
	last := false
	attempts := 0
	for equip.StarLevel < 10 && stone.StackCount >= int(count) {
		if target > 0 && equip.StarLevel >= int(target) {
			break
		}
		next := equip.StarLevel
		last = rand.Intn(100) < starCalcSuccessRate(next, int(count))
		if last {
			next++
		} else {
			next = wingStarFailureLevel(next)
		}
		updated, remaining, deleted, e := h.itemService.CommitStarAttempt(ctx.Context, charID, equipID, stoneID, int(count), next)
		if e != nil {
			last = false
			if ctx.Connection != nil {
				_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể lưu kết quả thăng tinh cánh. Hãy kiểm tra lại nguyên liệu.")
			}
			break
		}
		equip = updated
		stone = remaining
		attempts++
		if deleted || target == 0 || attempts >= 10000 {
			break
		}
	}
	if ctx.Connection != nil && attempts > 0 {
		h.sendSlotUpdate(ctx, h.itemDTO(equip))
		if stone.StackCount == 0 {
			_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(stone.ID), float64(stone.CalculateSID()))
		} else {
			h.sendSlotUpdate(ctx, h.itemDTO(stone))
		}
	}
	return map[string]interface{}{"equSlotId": float64(equipID), "itemSlotId": float64(stoneID), "num": float64(stone.StackCount), "upgradeNum": equip.StarLevel, "flag": last && attempts > 0}, nil
}
