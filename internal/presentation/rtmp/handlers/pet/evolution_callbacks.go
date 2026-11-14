package pet

import (
	"go.uber.org/zap"
	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
	"strconv"
	"time"
)

func (h *Handler) petEvolutionAction(ctx *rtmp.RPCContext, args []interface{}, mode int) (interface{}, error) {
	if len(args) < 1 {
		return map[string]interface{}{"num": 0}, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	var result *apppet.PetEvolutionResult
	if mode == 2 {
		result, err = h.petService.BeginPetEvolution(ctx.Context, charID, petID)
	} else {
		protect := false
		if mode == 0 && len(args) > 1 {
			protect = parseBoolArg(args[1])
		}
		result, err = h.petService.AdvancePetFeather(ctx.Context, charID, petID, protect, mode == 1)
	}
	if err != nil {
		h.logger.Warn("Pet evolution action rejected", zap.Int64("character_id", charID), zap.Int64("pet_id", petID), zap.Int("mode", mode), zap.Error(err))
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể thực hiện: tiến hóa cần pet khóa, sổ tay đã kích hoạt và dạng tiến hóa hợp lệ; nâng lông vũ cần đủ điểm npPnt/vàng và bảo hộ nếu chọn. Không tiêu hao tài nguyên.")
		}
		response := map[string]interface{}{"num": 0, "pid": float64(petID), "success": false}
		if p, e := h.petService.GetPetDetail(ctx.Context, charID, petID); e == nil && p != nil {
			response["tid"] = p.TemplateID
			response["envo"] = p.EvolutionInfo()
		}
		return response, nil
	}
	p := result.Pet
	info := p.EvolutionInfo()
	if ctx.Connection != nil {
		for _, it := range result.Items {
			if it.StackCount == 0 {
				_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
			} else {
				dto := it.ToDTO()
				_ = ctx.Connection.SendCallback("onCreateItemInstance", dto)
				_ = ctx.Connection.SendCallback("onAddCharactorSlot", dto)
			}
		}
		_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{"gold": result.Character.Gold, "npPnt": result.Character.NpPoint})
		rtmputils.SendPMGoldSpendUpdate(ctx, result.PMExpDelta, result.Character.PMExp, result.Character.CurrentPMLevel(time.Now()))
		for field, value := range map[string]interface{}{"tid": p.TemplateID, "envoInfo": info, "aptStrength": strconv.Itoa(p.AptStrength), "aptAgility": strconv.Itoa(p.AptAgility), "aptStamina": strconv.Itoa(p.AptStamina), "aptIntelligence": strconv.Itoa(p.AptIntelligence), "aptEnergy": strconv.Itoa(p.AptEnergy)} {
			_ = ctx.Connection.SendCallback("onUpdatePet", float64(p.ID), field, value)
		}
		sendPetPropRefresh(ctx, p)
	}
	h.logger.Info("Pet evolution action committed", zap.Int64("character_id", charID), zap.Int64("pet_id", petID), zap.Int("mode", mode), zap.Bool("success", result.Success), zap.Any("progress", info))
	if mode == 2 {
		return map[string]interface{}{"pid": float64(p.ID), "tid": p.TemplateID, "envo": info}, nil
	}
	return map[string]interface{}{"num": 1, "pid": float64(p.ID), "success": result.Success, "round": info["round"], "stage": info["stage"], "protect": result.Protected}, nil
}
