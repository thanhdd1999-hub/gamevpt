package item

import (
	"go.uber.org/zap"
	"mcgame-server/internal/infrastructure/rtmp"
	"strconv"
	"time"
)

func (h *Handler) WingJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runWingFusion(ctx, args, false)
}
func (h *Handler) WingSeniorJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runWingFusion(ctx, args, true)
}

func (h *Handler) runWingFusion(ctx *rtmp.RPCContext, args []interface{}, senior bool) (interface{}, error) {
	failure := map[string]interface{}{"f": false}
	if len(args) != 2 {
		return failure, nil
	}
	charID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return failure, nil
	}
	main, ok := parseFlexibleInt64(args[0])
	if !ok {
		return failure, nil
	}
	ids := []int64{main}
	if senior {
		switch values := args[1].(type) {
		case []interface{}:
			for _, raw := range values {
				id, ok := parseFlexibleInt64(raw)
				if !ok {
					return failure, nil
				}
				ids = append(ids, id)
			}
		default:
			return failure, nil
		}
	} else {
		values, ok := asWingMap(args[1])
		if !ok {
			return failure, nil
		}
		for _, key := range []string{"w1", "w2", "w3", "w4"} {
			id, ok := parseFlexibleInt64(values[key])
			if !ok {
				return failure, nil
			}
			ids = append(ids, id)
		}
	}
	bonus := 0
	if !senior && h.charService != nil && h.gameData != nil {
		char, e := h.charService.GetByID(ctx.Context, charID)
		if e != nil {
			return failure, nil
		}
		if level := char.CurrentPMLevel(time.Now()); level > 0 {
			if right := h.gameData.GetPmRight(9); right != nil {
				value, _ := right.ToDTO()["value"+strconv.Itoa(level)].(float64)
				bonus = int(value)
			}
		}
	}
	result, err := h.itemService.FuseWings(ctx.Context, charID, ids, senior, bonus)
	if err != nil {
		h.logger.Warn("Wing fusion rejected", zap.Int64("character_id", charID), zap.Error(err))
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể dung hợp cánh: kiểm tra đủ cánh phụ cùng màu, tháo cánh về túi; dung hợp cao cấp cần cùng loại. Không tiêu hao khi sai điều kiện.")
		}
		return failure, nil
	}
	if ctx.Connection != nil {
		for _, it := range result.Consumed {
			_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
		}
		h.sendSlotUpdate(ctx, h.itemDTO(result.Main))
	}
	h.logger.Info("Wing fusion committed", zap.Int64("character_id", charID), zap.Bool("success", result.Success), zap.Int("consumed", len(result.Consumed)))
	return map[string]interface{}{"f": result.Success, "i": float64(result.Main.ID), "n": h.itemDTO(result.Main), "e": float64(main), "sid": float64(result.Main.CalculateSID())}, nil
}

func asWingMap(raw interface{}) (map[string]interface{}, bool) {
	if value, ok := raw.(map[string]interface{}); ok {
		return value, true
	}
	return normalizeToMap(raw)
}
