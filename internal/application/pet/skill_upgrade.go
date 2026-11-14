package pet

import (
	"context"
	"errors"
	"sync"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

var petSkillUpgradeMu sync.Mutex

type PetSkillUpgradeResult struct {
	Pet     *domainpet.Pet
	SlotKey string
	SkillID int
	Item    *domainitem.Item
}

func (s *Service) UpgradePetSkill(ctx context.Context, charID, petID int64, slot int) (*PetSkillUpgradeResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()

	petSkillUpgradeMu.Lock()
	defer petSkillUpgradeMu.Unlock()
	if s.petRepo == nil || s.itemRepo == nil || s.gameDataManager == nil {
		return nil, pkgerrors.ErrSystemError
	}
	if slot < 1 || slot > domainpet.TotalPetSkillSlots {
		return nil, pkgerrors.ErrInvalidInput
	}
	p, err := s.getPetAndValidate(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	current := s.gameDataManager.GetSkill(petSkillSlotValue(p, slot))
	if current == nil || current.CodeName == "" || (current.Level != 1 && current.Level != 2) {
		return nil, pkgerrors.ErrInvalidInput
	}
	nextID := 0
	for _, candidate := range s.gameDataManager.GetAllSkills() {
		if candidate.CodeName == current.CodeName && candidate.Level == current.Level+1 {
			if nextID != 0 {
				return nil, pkgerrors.ErrInvalidInput
			}
			nextID = candidate.GetID()
		}
	}
	if nextID == 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	quality := s.petSkillQualityLevel(p)
	if quality < 1 || quality > 3 {
		return nil, pkgerrors.ErrInvalidInput
	}
	count := 1
	if current.Level == 2 {
		count = 3
	}
	var material *domainitem.Item
	for _, bag := range []domainitem.SlotType{domainitem.SlotTypeBag, domainitem.SlotTypePetItemBag} {
		items, err := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, bag)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if it != nil && it.CharacterID == charID && it.TemplateID == 1364+quality && it.StackCount >= count {
				material = it
				break
			}
		}
		if material != nil {
			break
		}
	}
	if material == nil {
		return nil, pkgerrors.ErrInsufficientItems
	}
	original := clonePetSnapshot(p)
	next := clonePetSnapshot(p)
	setPetSkillSlotValue(next, slot, nextID)
	s.syncPetSkillBonuses(next)
	next.RecalculateStats()
	if err := s.petRepo.Save(ctx, next); err != nil {
		return nil, err
	}
	remaining := material.StackCount - count
	if remaining == 0 {
		err = s.itemRepo.Delete(ctx, material.ID)
	} else {
		err = s.itemRepo.UpdateStack(ctx, material.ID, remaining)
	}
	if err != nil {
		return nil, errors.Join(err, s.petRepo.Save(ctx, original))
	}
	copy := *material
	copy.StackCount = remaining
	return &PetSkillUpgradeResult{Pet: next, SlotKey: petSkillSlotKey(slot), SkillID: nextID, Item: &copy}, nil
}
