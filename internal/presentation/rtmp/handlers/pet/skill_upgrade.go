package pet

import "mcgame-server/internal/infrastructure/rtmp"

func (h *Handler) PetUpSkill(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return nil, err
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return nil, err
	}
	slot, err := parseIntArg(args[1])
	if err != nil {
		return nil, err
	}
	result, err := h.petService.UpgradePetSkill(ctx.Context, charID, petID, slot)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	it := result.Item
	if it.StackCount == 0 {
		_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
	} else {
		_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", float64(it.ID), float64(it.StackCount), float64(it.CalculateSID()))
	}
	_ = ctx.Connection.SendCallback("onUpdatePet", float64(result.Pet.ID), result.SlotKey, float64(result.SkillID))
	sendPetPropRefresh(ctx, result.Pet)
	return map[string]interface{}{"success": true}, nil
}
