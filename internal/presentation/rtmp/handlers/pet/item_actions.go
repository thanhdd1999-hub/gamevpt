package pet

import (
	apppet "mcgame-server/internal/application/pet"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) petItemAction(ctx *rtmp.RPCContext, args []interface{}, kind int) (interface{}, error) {
	if err := requireArgs(args, 2); err != nil {
		return map[string]interface{}{"f": false}, nil
	}
	charID, err := currentCharacterID(ctx)
	if err != nil {
		return nil, err
	}
	petID, err := parseInt64Arg(args[0])
	if err != nil {
		return map[string]interface{}{"f": false}, nil
	}
	itemID, err := parseInt64Arg(args[1])
	if err != nil {
		return map[string]interface{}{"f": false}, nil
	}
	var result *apppet.PetItemActionResult
	switch kind {
	case 1:
		result, err = h.petService.ResetPetPoints(ctx.Context, charID, petID, itemID)
	case 2:
		result, err = h.petService.RerollPetAptitude(ctx.Context, charID, petID, itemID)
	case 4:
		result, err = h.petService.RerollPetGrowth(ctx.Context, charID, petID, itemID)
	}
	if err != nil {
		_ = ctx.Connection.SendCallback("onSystemSay", "Không thể sử dụng vật phẩm này cho pet đã chọn.")
		return map[string]interface{}{"f": false, "p": float64(petID), "i": float64(itemID)}, nil
	}
	it := result.Item
	if it.StackCount == 0 {
		_ = ctx.Connection.SendCallback("onDelCharactorSlot", float64(it.ID), float64(it.CalculateSID()))
	} else {
		_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", float64(it.ID), float64(it.StackCount), float64(it.CalculateSID()))
	}
	p := result.Pet
	if kind == 1 {
		dto := p.ToDetailDTO()
		for _, key := range []string{"attStrength", "attAgility", "attStamina", "attIntelligence", "attEnergy", "attLastPoint"} {
			if value, ok := p.Property[key]; ok {
				dto[key] = value
			}
			if value, ok := dto[key]; ok {
				_ = ctx.Connection.SendCallback("onUpdatePet", float64(p.ID), key, value)
			}
		}
		sendPetPropRefresh(ctx, p)
		return map[string]interface{}{"f": true, "pet": dto}, nil
	}
	_ = ctx.Connection.SendCallback("onUpdatePet", float64(p.ID), result.Field, result.After)
	sendPetPropRefresh(ctx, p)
	return map[string]interface{}{"f": true, "p": float64(p.ID), "t": result.Field, "num": result.After, "i": float64(it.ID), "n": it.StackCount, "before": result.Before, "after": result.After, "type": result.Field, "aptHigh": result.Premium}, nil
}
