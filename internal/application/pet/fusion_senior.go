// Senior fusion uses the Flash base rate tables; inactive birthday event overrides are excluded.
// Uniform hundredth growth within the next colour is a local rule, not recovered server code.
package pet

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"sort"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type AtomicSeniorPetFusionRepository interface {
	CommitSeniorPetFusion(context.Context, *domainpet.Pet, []int64, int) error
}

func (s *Service) FuseSeniorPets(ctx context.Context, charID int64, ids []int64) (*PetFusionResult, error) {
	return s.fuseSeniorPets(ctx, charID, ids, func(n int) (int, error) {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, err
		}
		return int(v.Int64()), nil
	})
}

func (s *Service) fuseSeniorPets(ctx context.Context, charID int64, ids []int64, draw func(int) (int, error)) (*PetFusionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()

	atomic, ok := s.petRepo.(AtomicSeniorPetFusionRepository)
	if !ok || s.itemRepo == nil {
		return nil, pkgerrors.ErrSystemError
	}
	if charID <= 0 || len(ids) < 5 || len(ids) > 17 {
		return nil, pkgerrors.ErrInvalidInput
	}
	seen := map[int64]bool{}
	var main, original *domainpet.Pet
	bound := false
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
		p := clonePetSnapshot(stored)
		s.enrichPet(ctx, p)
		if p.ClientState() != int(domainpet.PetStateRest) || p.IsFollowing || p.IsMounting || math.IsNaN(p.GrowRate) || math.IsInf(p.GrowRate, 0) || p.GrowRate <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if main == nil {
			main = p
			original = clonePetSnapshot(stored)
		} else {
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
	}
	colour := domainpet.ColorByGrowRate(main.GrowRate)
	count := len(ids) - 1
	var rates []int
	cost := 1
	switch colour {
	case 2:
		rates = []int{50, 53, 58, 65, 74, 85, 100}
	case 3:
		rates = []int{60, 60, 60, 60, 60, 60, 60, 60, 66, 73, 81, 90, 100}
		cost = 2 * count
	default:
		return nil, pkgerrors.ErrInvalidInput
	}
	if count-4 >= len(rates) {
		return nil, pkgerrors.ErrInvalidInput
	}
	total := 0
	var materialBefore []*domainitem.Item
	for _, bag := range []domainitem.SlotType{domainitem.SlotTypeBag, domainitem.SlotTypePetItemBag} {
		items, err := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, bag)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if it != nil && it.CharacterID == charID && it.TemplateID == 3014 && it.StackCount > 0 {
				copy := *it
				materialBefore = append(materialBefore, &copy)
				total += it.StackCount
			}
		}
	}
	if total < cost {
		return nil, pkgerrors.ErrInsufficientItems
	}
	roll, err := draw(100)
	if err != nil {
		return nil, err
	}
	success := roll < rates[count-4]
	next := clonePetSnapshot(original)
	if success {
		low, high := 161, 200
		if colour == 3 {
			low, high = 201, 250
		}
		value, err := draw(high - low + 1)
		if err != nil {
			return nil, err
		}
		s.enrichPet(ctx, next)
		applyPetGrowRate(next, float64(low+value)/100)
		next.SetBinded(bound)
	}
	consumed := append([]int64(nil), ids[1:]...)
	if err := atomic.CommitSeniorPetFusion(ctx, next, consumed, cost); err != nil {
		return nil, err
	}
	sort.Slice(materialBefore, func(i, j int) bool { return materialBefore[i].ID < materialBefore[j].ID })
	remainingCost := cost
	var changed []*domainitem.Item
	for _, before := range materialBefore {
		if remainingCost <= 0 {
			break
		}
		used := min(remainingCost, before.StackCount)
		remainingCost -= used
		after, err := s.itemRepo.FindByID(ctx, before.ID)
		copy := *before
		copy.StackCount = before.StackCount - used
		if err == nil && after != nil && after.StackCount != before.StackCount {
			copy = *after
		}
		changed = append(changed, &copy)
	}
	return &PetFusionResult{Pet: next, Success: success, ConsumedIDs: consumed, ConsumedItems: changed}, nil
}
