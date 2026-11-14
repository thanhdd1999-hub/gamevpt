package pet

import (
	"strings"

	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
)

func (h *Handler) PetEatClient(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	failure := map[string]interface{}{"succ": false, "msg": "Không thể ảo hóa các pet đã chọn."}
	if len(args) != 6 {
		return failure, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	main, err := parseInt64Arg(args[0])
	if err != nil {
		return failure, nil
	}
	donor, err := parseInt64Arg(args[1])
	if err != nil {
		return failure, nil
	}
	var luckID int64
	if args[2] != nil && args[2] != false {
		luckID, err = parseInt64Arg(args[2])
		if err != nil {
			return failure, nil
		}
	}
	count, err := parseIntArg(args[3])
	if err != nil {
		return failure, nil
	}
	apt, err := parseIntArg(args[4])
	if err != nil {
		return failure, nil
	}
	pass, _ := args[5].(string)
	if err := rtmputils.VerifySecondaryPassword(ctx.Context, h.accountRepo, ctx.AccountID, pass); err != nil {
		_ = rtmputils.NotifySecondaryPasswordFailure(ctx.Connection)
		return map[string]interface{}{"succ": false, "msg": "Mật khẩu cấp hai không chính xác."}, nil
	}
	result, err := h.petService.PreviewPetIllusion(ctx.Context, charID, main, donor, luckID, count, apt)
	if err != nil {
		return failure, nil
	}
	_ = rtmputils.CacheSecondaryPassword(ctx.Connection, pass)
	_ = ctx.Connection.SendCallback("onDelPet", float64(result.DonorID))
	if it := result.Item; it != nil {
		if it.StackCount == 0 {
			_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
		} else {
			_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", float64(it.ID), float64(it.StackCount), float64(it.CalculateSID()))
		}
	}
	return illusionResponse(result), nil
}

func (h *Handler) EnsurePetEatClient(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	failure := map[string]interface{}{"succ": false, "msg": "Không có kết quả ảo hóa hợp lệ để áp dụng."}
	if len(args) != 1 {
		return failure, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseInt64Arg(args[0])
	if err != nil {
		return failure, nil
	}
	result, err := h.petService.ConfirmPetIllusion(ctx.Context, charID, id)
	if err != nil {
		return failure, nil
	}
	_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), strings.TrimSuffix(result.Field, "Illusion")+"Ex", float64(result.Value))
	sendPetPropRefresh(ctx, result.Pet)
	return illusionResponse(result), nil
}

func illusionResponse(result *apppet.PetIllusionResult) map[string]interface{} {
	msg := "Ảo hóa thất bại."
	if result.IsHint {
		msg = "Ảo hóa thành công, hãy chọn áp dụng kết quả."
	} else if result.Success {
		msg = "Đã áp dụng kết quả ảo hóa."
	}
	return map[string]interface{}{"msg": msg, "costItem": result.CostItem, "costPet": result.DonorID > 0, "succ": result.Success, "isHint": result.IsHint, "petId": float64(result.Pet.ID), "aptType": result.AptType, "propName": strings.TrimSuffix(result.Field, "Illusion") + "Ex", "value": result.Value}
}
