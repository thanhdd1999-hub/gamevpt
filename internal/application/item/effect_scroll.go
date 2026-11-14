// Open-sourced by BaoLT

package item

import (
	"context"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type scrollEffectHandler struct{}

func (h *scrollEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.Type) == 500
}

func (h *scrollEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	result := NewEffectResult()
	result.CharacterUpdates = map[string]interface{}{}
	switch tpl.ID {
	case 325:
		if char.BagSlotNum >= character.MaxBagTabs {
			result.ConsumeItem = false
			result.Message = "Túi đồ đã mở tối đa 7 trang; không tiêu hao sách."
			return result, nil
		}
		char.SetBagSlotNum(char.BagSlotNum + 1)
		result.CharacterUpdates["bagSlotNum"] = char.BagSlotNum
	case 324:
		if char.BankSlotNum >= character.MaxBankTabs {
			result.ConsumeItem = false
			result.Message = "Kho đã mở tối đa 5 trang; không tiêu hao sách."
			return result, nil
		}
		char.SetBankSlotNum(char.BankSlotNum + 1)
		result.CharacterUpdates["bankSlotNum"] = char.BankSlotNum
	case 326:
		if char.PetMaxNum >= domainpet.MaxPets {
			result.ConsumeItem = false
			result.Message = "Túi pet đã đạt giới hạn mở rộng; không tiêu hao sách."
			return result, nil
		}
		char.PetMaxNum++
		char.PetSlots = char.PetMaxNum
		result.CharacterUpdates["petMaxNum"] = char.PetMaxNum
	default:
		return nil, pkgerrors.ErrCannotUseItem
	}
	return result, nil
}
