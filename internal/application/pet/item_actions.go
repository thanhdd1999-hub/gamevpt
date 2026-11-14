// Aptitude rerolls use documented 80–120% bounds; uniform integer draws are a local distribution choice.
package pet

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"strings"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type AtomicPetItemActionRepository interface {
	CommitPetItemAction(context.Context, *domainpet.Pet, *domainitem.Item) error
}

type PetItemActionResult struct {
	Pet     *domainpet.Pet
	Item    *domainitem.Item
	Field   string
	Before  float64
	After   float64
	Premium bool
}

func (s *Service) ResetPetPoints(ctx context.Context, charID, petID, itemID int64) (*PetItemActionResult, error) {
	return s.applyPetItemAction(ctx, charID, petID, itemID, 1, petItemRandom)
}
func (s *Service) RerollPetAptitude(ctx context.Context, charID, petID, itemID int64) (*PetItemActionResult, error) {
	return s.applyPetItemAction(ctx, charID, petID, itemID, 2, petItemRandom)
}
func (s *Service) RerollPetGrowth(ctx context.Context, charID, petID, itemID int64) (*PetItemActionResult, error) {
	return s.applyPetItemAction(ctx, charID, petID, itemID, 4, petItemRandom)
}
func petItemRandom(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

func (s *Service) applyPetItemAction(ctx context.Context, charID, petID, itemID int64, kind int, draw func(int) (int, error)) (*PetItemActionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()

	atomic, ok := s.petRepo.(AtomicPetItemActionRepository)
	if !ok || s.itemRepo == nil || s.gameDataManager == nil {
		return nil, pkgerrors.ErrSystemError
	}
	stored, err := s.petRepo.FindByID(ctx, petID)
	if err != nil {
		return nil, err
	}
	if stored == nil || stored.CharacterID != charID {
		return nil, pkgerrors.ErrNotFound
	}
	next := clonePetSnapshot(stored)
	s.enrichPet(ctx, next)
	if !next.IsBound() || next.ClientState() != int(domainpet.PetStateRest) || next.IsFollowing || next.IsMounting {
		return nil, pkgerrors.ErrInvalidInput
	}
	it, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if it == nil || it.CharacterID != charID || it.StackCount <= 0 || (it.SlotType != domainitem.SlotTypeBag && it.SlotType != domainitem.SlotTypePetItemBag) {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetItem(it.TemplateID)
	if tpl == nil || int(tpl.Type) != 507 || int(tpl.PropType) != kind {
		return nil, pkgerrors.ErrInvalidInput
	}
	creature := s.gameDataManager.GetCreature(next.TemplateID)
	if creature == nil {
		return nil, pkgerrors.ErrNotFound
	}
	result := &PetItemActionResult{Pet: next}
	switch kind {
	case 1:
		if int(creature.ClassIds) != 10 {
			return nil, pkgerrors.ErrInvalidInput
		}
		allocated := next.AptStrengthEx + next.AptAgilityEx + next.AptStaminaEx + next.AptIntelligenceEx + next.AptEnergyEx
		if allocated <= 0 || next.AptStrengthEx < 0 || next.AptAgilityEx < 0 || next.AptStaminaEx < 0 || next.AptIntelligenceEx < 0 || next.AptEnergyEx < 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		next.AptStrengthEx = 0
		next.AptAgilityEx = 0
		next.AptStaminaEx = 0
		next.AptIntelligenceEx = 0
		next.AptEnergyEx = 0
		next.AttrPoints += allocated
		next.DistributedAttrPoints = 0
		levelBonus := max(0, next.Level-1)
		next.Property["attStrength"] = int(creature.AttStrength) + levelBonus
		next.Property["attAgility"] = int(creature.AttAgility) + levelBonus
		next.Property["attStamina"] = int(creature.AttStamina) + levelBonus
		next.Property["attIntelligence"] = int(creature.AttIntelligence) + levelBonus
		next.Property["attEnergy"] = int(creature.AttEnergy) + levelBonus
	case 2:
		field, base, current := petRerollAptitude(next, creature, int(tpl.ProplNum))
		if field == "" || base <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		low, high := int(math.Round(base*0.8)), int(math.Round(base*1.2))
		premium := strings.Contains(strings.ToLower(tpl.Name), "cao cấp")
		if premium && current >= low {
			low = current + 1
		}
		if low > high {
			return nil, pkgerrors.ErrInvalidInput
		}
		roll, err := draw(high - low + 1)
		if err != nil {
			return nil, err
		}
		after := low + roll
		switch int(tpl.ProplNum) {
		case 1:
			next.AptStrength = after
		case 2:
			next.AptAgility = after
		case 3:
			next.AptStamina = after
		case 4:
			next.AptIntelligence = after
		case 5:
			next.AptEnergy = after
		}
		result.Field = field
		result.Before = float64(current)
		result.After = float64(after)
		result.Premium = premium
	case 4:
		if math.IsNaN(next.GrowRate) || math.IsInf(next.GrowRate, 0) || next.GrowRate <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		colour := domainpet.ColorByGrowRate(next.GrowRate)
		if colour < 1 || colour > 4 {
			return nil, pkgerrors.ErrInvalidInput
		}
		low := []int{100, 111, 131, 161, 201}[colour]
		high := []int{110, 130, 160, 200, 250}[colour]
		roll, err := draw(high - low + 1)
		if err != nil {
			return nil, err
		}
		result.Field = "growRate"
		result.Before = next.GrowRate
		result.After = float64(low+roll) / 100
		applyPetGrowRate(next, result.After)
	default:
		return nil, pkgerrors.ErrInvalidInput
	}
	next.RecalculateStats()
	snapshot := *it
	if err := atomic.CommitPetItemAction(ctx, next, &snapshot); err != nil {
		return nil, err
	}
	remaining := snapshot
	remaining.StackCount--
	result.Item = &remaining
	return result, nil
}

func petRerollAptitude(p *domainpet.Pet, t *models.CreatureTemplate, kind int) (string, float64, int) {
	switch kind {
	case 1:
		return "aptStrength", t.AptStrength, p.AptStrength
	case 2:
		return "aptAgility", t.AptAgility, p.AptAgility
	case 3:
		return "aptStamina", t.AptStamina, p.AptStamina
	case 4:
		return "aptIntelligence", t.AptIntelligence, p.AptIntelligence
	case 5:
		return "aptEnergy", t.AptEnergy, p.AptEnergy
	}
	return "", 0, 0
}
