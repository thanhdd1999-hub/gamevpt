// Open-sourced by BaoLT

package pet

import (
	"errors"

	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func (h *Handler) PetJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) != 5 || h.petService == nil {
		return map[string]interface{}{"flag": false}, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 5)
	for i := range ids {
		ids[i], err = parseInt64Arg(args[i])
		if err != nil {
			return map[string]interface{}{"flag": false}, nil
		}
	}
	result, err := h.petService.FuseFivePets(ctx.Context, charID, ids)
	if err != nil {
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể dung hợp: kiểm tra 5 pet khác nhau, cùng loài/cùng màu; pet phụ phải nghỉ, tháo trang bị và hồn. Không tiêu hao pet.")
		}
		h.logger.Warn("Pet fusion rejected", zap.Int64("character_id", charID), zap.Int64s("pet_ids", ids), zap.Error(err))
		return map[string]interface{}{"flag": false}, nil
	}
	if ctx.Connection != nil {
		for _, id := range result.ConsumedIDs {
			_ = ctx.Connection.SendCallback("onDelPet", float64(id))
		}
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "growRate", result.Pet.GrowRate)
		if result.Pet.IsBound() {
			_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "binded", 1)
		}
		sendPetPropRefresh(ctx, result.Pet)
	}
	h.logger.Info("Five pet fusion committed", zap.Int64("character_id", charID), zap.Int64s("consumed_pet_ids", result.ConsumedIDs), zap.Bool("success", result.Success), zap.Float64("growth", result.Pet.GrowRate))
	return map[string]interface{}{"flag": result.Success, "pid": float64(result.Pet.ID), "grow": result.Pet.GrowRate}, nil
}

func (h *Handler) SeniorPetJoin(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) != 1 || h.petService == nil {
		return map[string]interface{}{"flag": false}, nil
	}
	ids := parseInt64SliceArg(args[0])
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.petService.FuseSeniorPets(ctx.Context, charID, ids)
	if err != nil {
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể dung hợp cao cấp: kiểm tra pet phụ cùng loài/cùng màu, trạng thái nghỉ, tháo trang bị/hồn và đủ vật phẩm dung hợp 3014. Không tiêu hao khi sai điều kiện.")
		}
		h.logger.Warn("Senior fusion rejected", zap.Int64("character_id", charID), zap.Error(err))
		return map[string]interface{}{"flag": false}, nil
	}
	if ctx.Connection != nil {
		for _, id := range result.ConsumedIDs {
			_ = ctx.Connection.SendCallback("onDelPet", float64(id))
		}
		for _, it := range result.ConsumedItems {
			if it.StackCount <= 0 {
				_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
			} else {
				dto := it.ToDTO()
				_ = ctx.Connection.SendCallback("onCreateItemInstance", dto)
				_ = ctx.Connection.SendCallback("onAddCharactorSlot", dto)
			}
		}
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "growRate", result.Pet.GrowRate)
		if result.Pet.IsBound() {
			_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "binded", 1)
		}
		sendPetPropRefresh(ctx, result.Pet)
	}
	h.logger.Info("Senior fusion committed", zap.Int64("character_id", charID), zap.Bool("success", result.Success), zap.Int64s("consumed_pet_ids", result.ConsumedIDs))
	return map[string]interface{}{"flag": result.Success, "pid": float64(result.Pet.ID), "grow": result.Pet.GrowRate}, nil
}

func (h *Handler) PetXd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petItemAction(ctx, args, 1)
}

func (h *Handler) PetPzXsd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petItemAction(ctx, args, 4)
}

func (h *Handler) PetXsd(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petItemAction(ctx, args, 2)
}

func (h *Handler) PetBook(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}
	skillBookItemID, err := parseInt64Arg(args[1])
	if err != nil {
		return map[string]interface{}{"f": apppet.PetSkillBookFailureInput}, nil
	}

	result, err := h.petService.LearnSkillFromBook(ctx.Context, characterID, petID, skillBookItemID)
	if err != nil {
		var bookErr *apppet.PetSkillBookError
		if errors.As(err, &bookErr) {
			response := map[string]interface{}{
				"f": bookErr.Failure,
			}
			if bookErr.Failure == apppet.PetSkillBookFailureLearn {
				response["rf"] = false
			}
			if bookErr.Remaining >= 0 {
				response["n"] = bookErr.Remaining
			}
			return response, nil
		}
		return rtmp.ErrorToResponse(err), nil
	}

	_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), result.SlotKey, float64(result.SkillID))
	if it:=result.ConsumedItem;it!=nil{
		if it.StackCount<=0{_ = ctx.Connection.SendCallback("onDelCharactorSlot",float64(it.ID),float64(it.CalculateSID()))}else{
			_ = ctx.Connection.SendCallback("onCreateItemInstance",it.ToDTO())
			_ = ctx.Connection.SendCallback("onAddCharactorSlot",it.ToDTO())
			_ = ctx.Connection.SendCallback("onUpChaSlotStackNum",float64(it.ID),float64(it.StackCount),float64(it.CalculateSID()))
		}
	}
	sendPetPropRefresh(ctx, result.Pet)

	h.logger.Info("Pet learned skill from book",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID),
		zap.Int64("skill_book_item_id", skillBookItemID),
		zap.Int("skill_id", result.SkillID),
		zap.String("slot", result.SlotKey))

	return map[string]interface{}{
		"f":  1,
		"rf": true,
		"n":  result.RemainingStack,
	}, nil
}
