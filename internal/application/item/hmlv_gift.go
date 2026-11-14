// Local server rule selected by the owner: every HMLV gift grants one Vạn Linh Bội.
// Inventory and pet compensation handles returned errors; it is not a crash-atomic DB transaction.
package item

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
	"sync"
)

var hmlvGiftMu sync.Mutex

type hmlvGiftPetService interface {
	ContractPetsWithQuality(context.Context, int64, []int, int) ([]*domainpet.Pet, error)
	Save(context.Context, *domainpet.Pet) error
	DeletePet(context.Context, int64, int64) error
}

func (s *Service) applyHMLVGift(ctx context.Context, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	hmlvGiftMu.Lock()
	defer hmlvGiftMu.Unlock()
	petsService, ok := s.petService.(hmlvGiftPetService)
	if !ok || s.gameDataRec == nil || s.gameDataRec.GetItem(6859) == nil {
		return nil, pkgerrors.ErrSystemError
	}
	if it == nil || it.CharacterID != char.ID || it.StackCount <= 0 || (it.SlotType != domainitem.SlotTypeBag && it.SlotType != domainitem.SlotTypeTempBag) {
		return nil, pkgerrors.ErrInvalidInput
	}
	templateIDs, err := s.resolvePetBagTemplateIDs(tpl)
	if err != nil || len(templateIDs) != 1 {
		return nil, pkgerrors.ErrCannotUseItem
	}
	original := *it
	reserved := false
	if it.StackCount == 1 && it.SlotType == domainitem.SlotTypeBag {
		slot, err := s.itemRepo.FindFirstEmptySlot(ctx, char.ID, domainitem.SlotTypeTempBag, domainitem.DefaultTempBagSlots)
		if err != nil {
			return nil, err
		}
		if err := s.itemRepo.MoveItem(ctx, it.ID, domainitem.SlotTypeTempBag, slot); err != nil {
			return nil, err
		}
		reserved = true
	}
	var pets []*domainpet.Pet
	rollbackPets := func(cause error) error {
		for _, pet := range pets {
			if pet != nil && pet.ID > 0 {
				cause = errors.Join(cause, petsService.DeletePet(ctx, char.ID, pet.ID))
			}
		}
		return cause
	}
	var consumed *domainitem.Item
	updated, err := s.GrantBagRewards(ctx, char.ID, []BagReward{{TemplateID: 6859, ItemType: domainitem.ItemTypeConsumable, Count: 1, IsBound: original.IsBound}}, func() error {
		var createErr error
		pets, createErr = petsService.ContractPetsWithQuality(ctx, char.ID, templateIDs, 15)
		if createErr != nil {
			return rollbackPets(createErr)
		}
		if len(pets) != 1 || pets[0] == nil {
			return rollbackPets(pkgerrors.ErrSystemError)
		}
		pets[0].SetBinded(original.IsBound)
		if err := petsService.Save(ctx, pets[0]); err != nil {
			return rollbackPets(err)
		}
		var consumeErr error
		consumed, _, consumeErr = s.ConsumeItemStack(ctx, char.ID, original.ID, 1)
		if consumeErr != nil {
			return rollbackPets(consumeErr)
		}
		return nil
	})
	if err != nil {
		if reserved {
			err = errors.Join(err, s.itemRepo.MoveItem(ctx, original.ID, original.SlotType, original.SlotIndex))
		}
		if s.logger != nil {
			s.logger.Error("HMLV gift could not commit", zap.Int64("character_id", char.ID), zap.Int64("source_id", original.ID), zap.Error(err))
		}
		return nil, err
	}
	it.StackCount = consumed.StackCount
	it.SlotType = original.SlotType
	it.SlotIndex = original.SlotIndex
	result := NewEffectResult()
	result.ConsumeItem = false
	result.Pets = pets
	result.Message = "Nhận Huyền Minh Linh Vũ phẩm chất 1.5 và 1 Vạn Linh Bội."
	for _, item := range updated {
		if item.TemplateID == 6859 {
			result.GrantedItems = append(result.GrantedItems, &GrantedItemResult{TemplateID: 6859, Count: 1, IsBound: item.IsBound, Item: item})
		}
	}
	return result, nil
}
