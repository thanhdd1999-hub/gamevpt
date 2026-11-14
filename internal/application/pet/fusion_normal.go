// Normal fusion follows the Flash five-pet contract and published Go colour/rate rules.
// The original within-colour distribution is unpublished; use uniform hundredth steps.
package pet

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"time"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type AtomicPetFusionRepository interface {
	CommitPetFusion(context.Context, *domainpet.Pet, []int64) error
}

type PetFusionResult struct {
	Pet           *domainpet.Pet
	Success       bool
	ConsumedIDs   []int64
	ConsumedItems []*domainitem.Item
}

func (s *Service) FuseFivePets(ctx context.Context, charID int64, ids []int64) (*PetFusionResult, error) {
	return s.fuseFivePets(ctx, charID, ids, func(n int) (int, error) {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, err
		}
		return int(v.Int64()), nil
	})
}

func (s *Service) fuseFivePets(ctx context.Context, charID int64, ids []int64, draw func(int) (int, error)) (*PetFusionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()

	atomic, ok := s.petRepo.(AtomicPetFusionRepository)
	if !ok {
		return nil, pkgerrors.ErrSystemError
	}
	if charID <= 0 || len(ids) != 5 {
		return nil, pkgerrors.ErrInvalidInput
	}
	seen := map[int64]bool{}
	pets := make([]*domainpet.Pet, 0, 5)
	bound := false
	var originalMain *domainpet.Pet
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return nil, pkgerrors.ErrInvalidInput
		}
		seen[id] = true
		stored, err := s.petRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if stored == nil || stored.CharacterID != charID {
			return nil, pkgerrors.ErrNotFound
		}
		if len(pets) == 0 {
			originalMain = clonePetSnapshot(stored)
		}
		p := clonePetSnapshot(stored)
		s.enrichPet(ctx, p)
		if p.ClientState() != int(domainpet.PetStateRest) || p.IsFollowing || p.IsMounting || math.IsNaN(p.GrowRate) || math.IsInf(p.GrowRate, 0) || p.GrowRate <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if len(pets) > 0 {
			main := pets[0]
			if p.TemplateID != main.TemplateID || p.Element != main.Element || domainpet.ColorByGrowRate(p.GrowRate) != domainpet.ColorByGrowRate(main.GrowRate) {
				return nil, pkgerrors.ErrInvalidInput
			}
			for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
				if p.EquipmentItemID(slot) > 0 {
					return nil, pkgerrors.ErrInvalidInput
				}
			}
			if normalFusionHasSoul(p.SoulInfo) {
				return nil, pkgerrors.ErrInvalidInput
			}
		}
		bound = bound || p.IsBound()
		pets = append(pets, p)
	}
	main := pets[0]
	colour := domainpet.ColorByGrowRate(main.GrowRate)
	if colour >= 4 {
		return nil, pkgerrors.ErrInvalidInput
	}
	rate := []int{100, 80, 50, 60}[colour]
	if colour == 3 && s.charRepo != nil && s.gameDataManager != nil {
		char, err := s.charRepo.FindByID(ctx, charID)
		if err != nil {
			return nil, err
		}
		if level := char.CurrentPMLevel(time.Now()); level > 0 {
			if right := s.gameDataManager.GetPmRight(6); right != nil {
				bonus, _ := right.ToDTO()["value"+strconv.Itoa(level)].(float64)
				if !math.IsNaN(bonus) && !math.IsInf(bonus, 0) && bonus > 0 {
					rate += int(bonus)
				}
			}
		}
	}
	if rate > 100 {
		rate = 100
	}
	roll, err := draw(100)
	if err != nil {
		return nil, err
	}
	next := clonePetSnapshot(originalMain)
	success := roll < rate
	if success {
		s.enrichPet(ctx, next)
		low := []int{111, 131, 161, 201}[colour]
		high := []int{130, 160, 200, 250}[colour]
		value, err := draw(high - low + 1)
		if err != nil {
			return nil, err
		}
		applyPetGrowRate(next, float64(low+value)/100)
		next.SetBinded(bound)
	}
	consumed := append([]int64(nil), ids[1:]...)
	if err := atomic.CommitPetFusion(ctx, next, consumed); err != nil {
		return nil, err
	}
	return &PetFusionResult{Pet: next, Success: success, ConsumedIDs: consumed}, nil
}

func normalFusionHasSoul(info map[string]interface{}) bool {
	data, ok := info["data"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, v := range data {
		if v != nil && !reflect.ValueOf(v).IsZero() {
			return true
		}
	}
	return false
}
