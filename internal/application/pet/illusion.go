// Illusion follows the client two-phase proposal/apply flow and published chance/amount ranges.
// Uniform integer percentage selection within 10–15% / 25–30% is a local distribution choice.
package pet

import (
	"context"
	"math"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type AtomicPetIllusionRepository interface {
	CommitPetIllusionAttempt(context.Context, *domainpet.Pet, int64, *domainitem.Item, int) error
}

type PetIllusionResult struct {
	Pet      *domainpet.Pet
	Item     *domainitem.Item
	DonorID  int64
	CostItem int
	Success  bool
	IsHint   bool
	AptType  int
	Field    string
	Value    int
}

func (s *Service) PreviewPetIllusion(ctx context.Context, charID, mainID, donorID, luckID int64, luckCount, aptType int) (*PetIllusionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	return s.previewPetIllusion(ctx, charID, mainID, donorID, luckID, luckCount, aptType, petItemRandom)
}

func (s *Service) previewPetIllusion(ctx context.Context, charID, mainID, donorID, luckID int64, luckCount, aptType int, draw func(int) (int, error)) (*PetIllusionResult, error) {
	atomic, ok := s.petRepo.(AtomicPetIllusionRepository)
	if !ok || s.gameDataManager == nil {
		return nil, pkgerrors.ErrSystemError
	}
	if mainID <= 0 || donorID <= 0 || mainID == donorID || aptType < 1 || aptType > 5 || luckCount < 0 || luckCount > 5 {
		return nil, pkgerrors.ErrInvalidInput
	}
	main, err := s.illusionPet(ctx, charID, mainID)
	if err != nil {
		return nil, err
	}
	donor, err := s.illusionPet(ctx, charID, donorID)
	if err != nil {
		return nil, err
	}
	if !main.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}
	for slot := 1; slot <= domainpet.TotalPetEquipmentSlots; slot++ {
		if donor.EquipmentItemID(slot) > 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
	}
	if normalFusionHasSoul(donor.SoulInfo) {
		return nil, pkgerrors.ErrInvalidInput
	}
	mainTemplate := s.gameDataManager.GetCreature(main.TemplateID)
	donorTemplate := s.gameDataManager.GetCreature(donor.TemplateID)
	if mainTemplate == nil || donorTemplate == nil {
		return nil, pkgerrors.ErrNotFound
	}
	chance := illusionSingleRate(main.GrowRate, int(mainTemplate.ClassIds)) + illusionSingleRate(donor.GrowRate, int(donorTemplate.ClassIds))
	if int(mainTemplate.ClassID) == int(donorTemplate.ClassID) {
		chance += 20
	} else {
		chance /= 2
	}
	if luckID <= 0 {
		luckCount = 0
	}
	for luckCount > 0 && chance+float64(20*luckCount) >= 120 {
		luckCount--
	}
	var luck *domainitem.Item
	if luckCount > 0 {
		if s.itemRepo == nil {
			return nil, pkgerrors.ErrSystemError
		}
		luck, err = s.itemRepo.FindByID(ctx, luckID)
		if err != nil {
			return nil, err
		}
		if luck == nil || luck.CharacterID != charID || luck.StackCount < luckCount || (luck.SlotType != domainitem.SlotTypeBag && luck.SlotType != domainitem.SlotTypePetItemBag) {
			return nil, pkgerrors.ErrInvalidInput
		}
		tpl := s.gameDataManager.GetItem(luck.TemplateID)
		if tpl == nil || int(tpl.Type) != 507 || int(tpl.PropType) != 7 || int(tpl.Color) < domainpet.ColorByGrowRate(donor.GrowRate) {
			return nil, pkgerrors.ErrInvalidInput
		}
	}
	chance = math.Min(100, chance+float64(20*luckCount))
	roll, err := draw(1000)
	if err != nil {
		return nil, err
	}
	field := illusionField(aptType)
	next := clonePetSnapshot(main)
	delete(next.Property, "petIllusionPending")
	result := &PetIllusionResult{Pet: next, DonorID: donorID, CostItem: luckCount, AptType: aptType, Field: field}
	if float64(roll) < chance*10 {
		percent := 10
		if domainpet.ColorByGrowRate(donor.GrowRate) == 4 {
			percent = 25
		}
		offset, err := draw(6)
		if err != nil {
			return nil, err
		}
		base := illusionBaseAptitude(donor, aptType)
		if base <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		result.Value = int(math.Round(float64(base) * float64(percent+offset) / 100))
		next.Property["petIllusionPending"] = map[string]interface{}{"aptType": aptType, "value": result.Value, "base": illusionBaseAptitude(main, aptType), "previous": illusionValue(main, field)}
		result.IsHint = true
	}
	if err := atomic.CommitPetIllusionAttempt(ctx, next, donorID, luck, luckCount); err != nil {
		return nil, err
	}
	if luck != nil {
		copy := *luck
		copy.StackCount -= luckCount
		result.Item = &copy
	}
	return result, nil
}

func (s *Service) ConfirmPetIllusion(ctx context.Context, charID, petID int64) (*PetIllusionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	p, err := s.illusionPet(ctx, charID, petID)
	if err != nil {
		return nil, err
	}
	if !p.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}
	pending, ok := p.Property["petIllusionPending"].(map[string]interface{})
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	apt, ok := parsePetSkillSlotNumber(pending["aptType"])
	if !ok || apt < 1 || apt > 5 {
		return nil, pkgerrors.ErrInvalidInput
	}
	value, ok := parsePetSkillSlotNumber(pending["value"])
	if !ok || value < 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	base, ok := parsePetSkillSlotNumber(pending["base"])
	if !ok || base != illusionBaseAptitude(p, apt) {
		return nil, pkgerrors.ErrInvalidInput
	}
	previous, ok := parsePetSkillSlotNumber(pending["previous"])
	if !ok || previous != illusionValue(p, illusionField(apt)) {
		return nil, pkgerrors.ErrInvalidInput
	}
	next := clonePetSnapshot(p)
	next.Property[illusionField(apt)] = value
	delete(next.Property, "petIllusionPending")
	next.RecalculateStats()
	if err := s.petRepo.Save(ctx, next); err != nil {
		return nil, err
	}
	return &PetIllusionResult{Pet: next, Success: true, AptType: apt, Field: illusionField(apt), Value: value}, nil
}

func (s *Service) illusionPet(ctx context.Context, charID, id int64) (*domainpet.Pet, error) {
	p, err := s.petRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CharacterID != charID {
		return nil, pkgerrors.ErrNotFound
	}
	next := clonePetSnapshot(p)
	s.enrichPet(ctx, next)
	if next.ClientState() != int(domainpet.PetStateRest) || next.IsFollowing || next.IsMounting || math.IsNaN(next.GrowRate) || math.IsInf(next.GrowRate, 0) || domainpet.ColorByGrowRate(next.GrowRate) < 3 {
		return nil, pkgerrors.ErrInvalidInput
	}
	return next, nil
}
func illusionField(apt int) string {
	return []string{"", "aptStrengthIllusion", "aptAgilityIllusion", "aptStaminaIllusion", "aptIntelligenceIllusion", "aptEnergyIllusion"}[apt]
}
func illusionValue(p *domainpet.Pet, field string) int {
	value, _ := parsePetSkillSlotNumber(p.Property[field])
	return value
}
func illusionBaseAptitude(p *domainpet.Pet, apt int) int {
	base := []int{0, p.AptStrength, p.AptAgility, p.AptStamina, p.AptIntelligence, p.AptEnergy}[apt]
	field := []string{"", "aptStrengthEvolution", "aptAgilityEvolution", "aptStaminaEvolution", "aptIntelligenceEvolution", "aptEnergyEvolution"}[apt]
	bonus, _ := parsePetSkillSlotNumber(p.Property[field])
	return base + bonus
}
func illusionSingleRate(grow float64, class int) float64 {
	if domainpet.ColorByGrowRate(grow) == 4 {
		if class == 10 {
			return 30
		}
		return 15
	}
	if class == 10 {
		return 15
	}
	return 10
}
