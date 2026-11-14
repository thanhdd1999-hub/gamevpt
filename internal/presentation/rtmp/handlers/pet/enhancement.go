// Open-sourced by BaoLT

package pet

import (
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"

	"go.uber.org/zap"
)

func (h *Handler) PetEat(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 6); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	mainPetID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	costPetID, err := parseInt64Arg(args[1])
	if err != nil {
		return nil, err
	}
	useLuckItem := parseBoolArg(args[2])
	luckNum, err := parseIntArg(args[3])
	if err != nil {
		luckNum = 0
	}
	aptValue, err := parseIntArg(args[4])
	if err != nil {
		return nil, err
	}
	aptType := domainpet.AptitudeType(aptValue)
	passHash, ok := args[5].(string)
	if !ok || passHash == "" {
		h.logger.Warn("PetEat: missing secondary password", zap.Int64("character_id", characterID))
		return map[string]interface{}{"success": false}, nil
	}

	if err := rtmputils.VerifySecondaryPassword(ctx.Context, h.accountRepo, ctx.AccountID, passHash); err != nil {
		h.logger.Warn("PetEat: secondary password verification failed", zap.Int64("character_id", characterID), zap.Error(err))
		if notifyErr := rtmputils.NotifySecondaryPasswordFailure(ctx.Connection); notifyErr != nil {
			h.logger.Error("PetEat: failed to notify client about secondary password failure", zap.Error(notifyErr))
		}
		return map[string]interface{}{"success": false}, nil
	}

	p, err := h.petService.PetEat(ctx.Context, characterID, mainPetID, costPetID, useLuckItem, luckNum, aptType)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet eat completed",
		zap.Int64("character_id", characterID),
		zap.Int64("main_pet_id", mainPetID),
		zap.Int64("cost_pet_id", costPetID))

	if err := rtmputils.CacheSecondaryPassword(ctx.Connection, passHash); err != nil {
		h.logger.Error("PetEat: failed to sync client password", zap.Error(err))
	}

	return map[string]interface{}{
		"success": true,
		"pet":     p.ToDTO(),
	}, nil
}

func (h *Handler) EnsurePetEat(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return map[string]interface{}{"success": true}, nil
}

func (h *Handler) PetStar(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return map[string]interface{}{"flag": false}, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	stoneID, err := parseInt64Arg(args[1])
	if err != nil {
		return nil, err
	}
	var protectID int64
	if len(args) > 2 && args[2] != nil {
		protectID, err = parseInt64Arg(args[2])
		if err != nil {
			return nil, err
		}
	}
	result, err := h.petService.AttemptPetStar(ctx.Context, charID, petID, stoneID, protectID)
	if err != nil {
		h.logger.Warn("Pet star rejected", zap.Int64("character_id", charID), zap.Int64("pet_id", petID), zap.Error(err))
		if ctx.Connection != nil {
			_ = ctx.Connection.SendCallback("onSystemMidMsgOrNote", "Không thể thăng sao: kiểm tra nội đơn, trạng thái khóa và giới hạn 12 sao. Không tiêu hao vật phẩm.")
		}
		response := map[string]interface{}{"flag": false, "petId": float64(petID), "starSlotId": float64(stoneID)}
		if p, e := h.petService.GetPetDetail(ctx.Context, charID, petID); e == nil && p != nil {
			response["starNum"] = p.UpgradeNum
			response["growRateAdd"] = p.GrowRateAdd
		}
		if h.petService != nil {
			if it, e := h.petService.GetOwnedPetItem(ctx.Context, charID, stoneID); e == nil && it != nil {
				response["num"] = it.StackCount
			}
			if protectID > 0 {
				if it, e := h.petService.GetOwnedPetItem(ctx.Context, charID, protectID); e == nil && it != nil {
					response["starSafeNum"] = it.StackCount
				}
			}
		}
		return response, nil
	}
	if ctx.Connection != nil {
		for _, it := range []*domainitem.Item{result.Stone, result.Protection} {
			if it == nil {
				continue
			}
			if it.StackCount == 0 {
				_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
			} else {
				dto := it.ToDTO()
				if h.petService != nil {
					dto = it.ToDTO()
				}
				_ = ctx.Connection.SendCallback("onCreateItemInstance", dto)
				_ = ctx.Connection.SendCallback("onAddCharactorSlot", dto)
			}
		}
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "upgradeNum", result.Pet.UpgradeNum)
		_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), "growRateAdd", result.Pet.GrowRateAdd)
		sendPetPropRefresh(ctx, result.Pet)
	}
	safeNum := 0
	if result.Protection != nil {
		safeNum = result.Protection.StackCount
	}
	h.logger.Info("Pet star attempt committed", zap.Int64("character_id", charID), zap.Int64("pet_id", petID), zap.Bool("success", result.Success), zap.Int("stars", result.Pet.UpgradeNum))
	return map[string]interface{}{"flag": result.Success, "petId": float64(result.Pet.ID), "starNum": result.Pet.UpgradeNum, "growRateAdd": result.Pet.GrowRateAdd, "starSlotId": float64(result.Stone.ID), "num": result.Stone.StackCount, "starSafeNum": safeNum}, nil
}
func (h *Handler) ContractPet(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 1); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	templateID, err := parseIntArg(args[0])
	if err != nil {
		return nil, err
	}

	p, err := h.petService.ContractPet(ctx.Context, characterID, templateID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet contracted",
		zap.Int64("character_id", characterID),
		zap.Int("template_id", templateID),
		zap.Int64("pet_id", p.ID))

	return map[string]interface{}{
		"success": true,
		"pet":     p.ToDTO(),
	}, nil
}

func (h *Handler) PetStarClear(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 1); err != nil {
		return nil, err
	}

	characterID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}

	p, err := h.petService.ClearStars(ctx.Context, characterID, petID)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}

	h.logger.Info("Pet stars cleared",
		zap.Int64("character_id", characterID),
		zap.Int64("pet_id", petID))

	return map[string]interface{}{
		"success": true,
		"pet":     p.ToDTO(),
	}, nil
}

func (h *Handler) ToPetFeather(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petEvolutionAction(ctx, args, 0)
}
func (h *Handler) ToPetFeatherNextStep(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petEvolutionAction(ctx, args, 1)
}
func (h *Handler) ToPetEnvolution(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.petEvolutionAction(ctx, args, 2)
}
