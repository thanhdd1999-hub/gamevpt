package pet

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

var petStarSuccessRates = []int{100, 100, 100, 90, 80, 70, 60, 50, 30, 30, 30, 30, 20}

type PetStarAttemptResult struct {
	Pet                *domainpet.Pet
	Success            bool
	Stone              *domainitem.Item
	Protection         *domainitem.Item
	ProtectionConsumed bool
}

func (s *Service) AttemptPetStar(ctx context.Context, charID, petID, stoneID, protectID int64) (*PetStarAttemptResult, error) {
	return s.attemptPetStar(ctx, charID, petID, stoneID, protectID, func() (int, error) {
		v, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			return 0, err
		}
		return int(v.Int64()), nil
	})
}

func (s *Service) attemptPetStar(ctx context.Context, charID, petID, stoneID, protectID int64, draw func() (int, error)) (*PetStarAttemptResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	atomic, ok := s.petRepo.(interface {
		CommitPetStarAttempt(context.Context, *domainpet.Pet, []*domainitem.Item) error
	})
	if !ok || s.itemRepo == nil || s.gameDataManager == nil {
		return nil, pkgerrors.ErrSystemError
	}
	p, err := s.petRepo.FindByID(ctx, petID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CharacterID != charID || p.UpgradeNum < 0 || p.UpgradeNum >= 12 {
		return nil, pkgerrors.ErrInvalidInput
	}
	stone, err := s.petStarMaterial(ctx, charID, stoneID, 3)
	if err != nil {
		return nil, err
	}
	if stone.IsBound && !p.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(stone.TemplateID)
	if tpl.ProplNum <= 0 || tpl.ProplNum > 100 || math.IsNaN(tpl.ProplNum) || math.IsInf(tpl.ProplNum, 0) {
		return nil, pkgerrors.ErrInvalidInput
	}
	var protection *domainitem.Item
	if protectID > 0 {
		if protectID == stoneID {
			return nil, pkgerrors.ErrInvalidInput
		}
		protection, err = s.petStarMaterial(ctx, charID, protectID, 6)
		if err != nil {
			return nil, err
		}
	}
	roll, err := draw()
	if err != nil {
		return nil, err
	}
	if roll < 0 || roll >= 100 {
		return nil, pkgerrors.ErrInvalidInput
	}
	next := clonePetSnapshot(p)
	success := roll < petStarSuccessRates[p.UpgradeNum+1]
	useProtection := protection != nil && p.UpgradeNum >= 8
	if success {
		next.UpgradeNum++
		if next.UpgradeNum <= 8 {
			next.GrowRateAdd = math.Round((next.GrowRateAdd+tpl.ProplNum/100)*10000) / 10000
		}
	} else if p.UpgradeNum >= 8 && !useProtection {
		next.UpgradeNum = 0
		next.GrowRateAdd = 0
	}
	consumed := []*domainitem.Item{stone}
	if useProtection {
		consumed = append(consumed, protection)
	}
	if success || next.UpgradeNum != p.UpgradeNum {
		next = s.enrichPetPreservingVitals(ctx, next)
	}
	if err := atomic.CommitPetStarAttempt(ctx, next, consumed); err != nil {
		return nil, err
	}
	remaining := *stone
	remaining.StackCount--
	result := &PetStarAttemptResult{Pet: next, Success: success, Stone: &remaining, ProtectionConsumed: useProtection}
	if protection != nil {
		copy := *protection
		if useProtection {
			copy.StackCount--
		}
		result.Protection = &copy
	}
	return result, nil
}

func (s *Service) petStarMaterial(ctx context.Context, charID, id int64, prop int) (*domainitem.Item, error) {
	it, err := s.itemRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if it == nil || it.CharacterID != charID || it.StackCount < 1 || (it.SlotType != domainitem.SlotTypeBag && it.SlotType != domainitem.SlotTypePetItemBag) {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.Type) != 507 || int(tpl.PropType) != prop {
		return nil, pkgerrors.ErrInvalidInput
	}
	copy := *it
	return &copy, nil
}

func (s *Service) GetOwnedPetItem(ctx context.Context, charID, itemID int64) (*domainitem.Item, error) {
	if s.itemRepo == nil {
		return nil, pkgerrors.ErrSystemError
	}
	it, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if it == nil || it.CharacterID != charID {
		return nil, pkgerrors.ErrNotFound
	}
	return it, nil
}
