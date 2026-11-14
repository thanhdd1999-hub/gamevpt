// Checkpoint failure and round-boundary transitions follow the client checkpoints; server-only rounding is explicit here.
package pet

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type PetEvolutionResult struct {
	Pet        *domainpet.Pet
	Character  *domainchar.Character
	Items      []*domainitem.Item
	Success    bool
	Protected  bool
	Forced     bool
	PMExpDelta int64
}

type atomicPetEvolutionRepository interface {
	CommitPetEvolutionAction(context.Context, *domainpet.Pet, *domainchar.Character, []*domainitem.Item, []int, int, int64) error
}

func (s *Service) BeginPetEvolution(ctx context.Context, charID, petID int64) (*PetEvolutionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	atomic, ok := s.petRepo.(atomicPetEvolutionRepository)
	if !ok || s.gameDataManager == nil || s.charRepo == nil {
		return nil, pkgerrors.ErrSystemError
	}
	raw, err := s.petRepo.FindByID(ctx, petID)
	if err != nil {
		return nil, err
	}
	if raw == nil || raw.CharacterID != charID || !raw.IsBound() {
		return nil, pkgerrors.ErrInvalidInput
	}
	p := clonePetSnapshot(raw)
	if p.Property == nil {
		p.Property = map[string]interface{}{}
	}
	if active, _ := p.EvolutionInfo()["envo"].(bool); active {
		return nil, pkgerrors.ErrInvalidInput
	}
	tpl := s.gameDataManager.GetCreature(p.TemplateID)
	if tpl == nil || int(tpl.ClassIds) != 10 {
		return nil, pkgerrors.ErrInvalidInput
	}
	var handbookID, target int
	for _, row := range s.gameDataManager.GetAllCreatureHandbooks() {
		if int(row.RelateID) == p.TemplateID && int(row.EvolutionID) > 0 {
			if target != 0 {
				return nil, pkgerrors.ErrInvalidInput
			}
			handbookID = int(row.ID)
			target = int(row.EvolutionID)
		}
	}
	targetTpl := s.gameDataManager.GetCreature(target)
	if targetTpl == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	if _, ok := featherStepValues(target, 0, 0); !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	provider, ok := s.featureBonusProvider.(interface {
		BuildCharacterLoginState(context.Context, int64) (map[string]interface{}, error)
	})
	if !ok {
		return nil, pkgerrors.ErrSystemError
	}
	state, err := provider.BuildCharacterLoginState(ctx, charID)
	if err != nil {
		return nil, err
	}
	activePets, _ := state["activePetObject"].(map[string]interface{})
	entry, _ := activePets[strconv.Itoa(handbookID)].(map[string]interface{})
	if !evolutionActive(entry["actived"]) {
		return nil, pkgerrors.ErrInvalidInput
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, err
	}
	if char == nil {
		return nil, pkgerrors.ErrNotFound
	}
	nextChar := cloneEvolutionCharacter(char)
	base := [5]int{int(math.Round(float64(p.AptStrength) * 1.2)), int(math.Round(float64(p.AptStamina) * 1.2)), int(math.Round(float64(p.AptAgility) * 1.2)), int(math.Round(float64(p.AptIntelligence) * 1.2)), int(math.Round(float64(p.AptEnergy) * 1.2))}
	p.TemplateID = target
	if p.Name == "" || p.Name == tpl.Name {
		p.Name = targetTpl.Name
	}
	p.EvolutionLv++
	p.Property["envoInfo"] = map[string]interface{}{"envo": true, "round": 0, "stage": 0}
	p.Property["evolutionBaseApt"] = []int{base[0], base[1], base[2], base[3], base[4]}
	applyFeatherAptitudes(p, base, [5]int{})
	p = s.enrichPetPreservingVitals(ctx, p)
	if err := atomic.CommitPetEvolutionAction(ctx, p, nextChar, nil, nil, 0, 0); err != nil {
		return nil, err
	}
	return &PetEvolutionResult{Pet: p, Character: nextChar, Success: true}, nil
}

func (s *Service) AdvancePetFeather(ctx context.Context, charID, petID int64, protect, forced bool) (*PetEvolutionResult, error) {
	return s.advancePetFeather(ctx, charID, petID, protect, forced, petItemRandom)
}

func (s *Service) advancePetFeather(ctx context.Context, charID, petID int64, protect, forced bool, draw func(int) (int, error)) (*PetEvolutionResult, error) {
	petMutationMu.Lock()
	defer petMutationMu.Unlock()
	atomic, ok := s.petRepo.(atomicPetEvolutionRepository)
	if !ok || s.charRepo == nil {
		return nil, pkgerrors.ErrSystemError
	}
	raw, err := s.petRepo.FindByID(ctx, petID)
	if err != nil {
		return nil, err
	}
	if raw == nil || raw.CharacterID != charID {
		return nil, pkgerrors.ErrInvalidInput
	}
	p := clonePetSnapshot(raw)
	info := p.EvolutionInfo()
	active, _ := info["envo"].(bool)
	round, stage := evolutionNumber(info["round"]), evolutionNumber(info["stage"])
	if !active || round < 0 || round > 2 || stage < 0 || stage > 9 || (round == 2 && stage == 9) {
		return nil, pkgerrors.ErrInvalidInput
	}
	if stage == 9 {
		round++
		stage = 0
	}
	if round > 2 {
		return nil, pkgerrors.ErrInvalidInput
	}
	currentSteps, ok := featherStepValues(p.TemplateID, round, stage)
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	base := evolutionBaseApt(p, currentSteps)
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, err
	}
	if char == nil {
		return nil, pkgerrors.ErrNotFound
	}
	nextChar := cloneEvolutionCharacter(char)
	npCost := 0
	var goldCost int64
	var items []*domainitem.Item
	var counts []int
	if forced {
		goldCost = featherGoldCosts[round][stage]
	} else {
		npCost = 5 * featherPointCosts[round][stage]
		if protect && stage%3 != 0 {
			items, counts, err = s.featherProtectItems(ctx, charID, featherProtectionCosts[round][stage])
			if err != nil {
				return nil, err
			}
		}
	}
	if nextChar.NpPoint < npCost || nextChar.Gold < goldCost {
		return nil, pkgerrors.ErrInsufficientItems
	}
	nextChar.NpPoint -= npCost
	nextChar.Gold -= goldCost
	nextChar.PMProcessData["localNpPnt"] = nextChar.NpPoint
	pmDelta := nextChar.AwardSystemGoldSpend(goldCost)
	success := forced
	if !forced {
		roll, err := draw(100)
		if err != nil {
			return nil, err
		}
		if roll < 0 || roll >= 100 {
			return nil, pkgerrors.ErrInvalidInput
		}
		success = roll < featherSuccessRates[round][stage]
	}
	protected := len(items) > 0
	if forced {
		stage = (stage/3 + 1) * 3
	} else if success {
		stage++
	} else if !protected {
		stage = (stage / 3) * 3
	}
	if stage == 9 && round < 2 {
		round++
		stage = 0
	}
	steps, ok := featherStepValues(p.TemplateID, round, stage)
	if !ok {
		return nil, pkgerrors.ErrInvalidInput
	}
	p.Property["envoInfo"] = map[string]interface{}{"envo": true, "round": round, "stage": stage}
	p.Property["evolutionBaseApt"] = []int{base[0], base[1], base[2], base[3], base[4]}
	applyFeatherAptitudes(p, base, steps)
	p = s.enrichPetPreservingVitals(ctx, p)
	if err := atomic.CommitPetEvolutionAction(ctx, p, nextChar, items, counts, npCost, goldCost); err != nil {
		return nil, err
	}
	after := make([]*domainitem.Item, 0, len(items))
	for i, it := range items {
		copy := *it
		copy.StackCount -= counts[i]
		after = append(after, &copy)
	}
	return &PetEvolutionResult{Pet: p, Character: nextChar, Items: after, Success: success, Protected: protected, Forced: forced, PMExpDelta: pmDelta}, nil
}

func cloneEvolutionCharacter(char *domainchar.Character) *domainchar.Character {
	copy := *char
	copy.PMProcessData = make(map[string]interface{}, len(char.PMProcessData)+1)
	for k, v := range char.PMProcessData {
		copy.PMProcessData[k] = v
	}
	return &copy
}

func (s *Service) featherProtectItems(ctx context.Context, charID int64, count int) ([]*domainitem.Item, []int, error) {
	if count < 1 || s.itemRepo == nil {
		return nil, nil, pkgerrors.ErrInvalidInput
	}
	items, err := s.itemRepo.FindByCharacterID(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	var selected []*domainitem.Item
	var counts []int
	for _, it := range items {
		if it.CharacterID != charID || it.TemplateID != 3911 || it.StackCount < 1 || (it.SlotType != domainitem.SlotTypeBag && it.SlotType != domainitem.SlotTypePetItemBag) {
			continue
		}
		take := it.StackCount
		if take > count {
			take = count
		}
		copy := *it
		selected = append(selected, &copy)
		counts = append(counts, take)
		count -= take
		if count == 0 {
			return selected, counts, nil
		}
	}
	return nil, nil, pkgerrors.ErrInsufficientItems
}

func evolutionActive(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return evolutionNumber(v) > 0
}
func evolutionNumber(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}
func evolutionBaseApt(p *domainpet.Pet, steps [5]int) [5]int {
	current := [5]int{p.AptStrength, p.AptStamina, p.AptAgility, p.AptIntelligence, p.AptEnergy}
	if values, ok := p.Property["evolutionBaseApt"].([]interface{}); ok && len(values) == 5 {
		for i, v := range values {
			current[i] = evolutionNumber(v)
		}
		return current
	}
	if values, ok := p.Property["evolutionBaseApt"].([]int); ok && len(values) == 5 {
		copy(current[:], values)
		return current
	}
	for i := range current {
		current[i] -= steps[i]
	}
	return current
}
func applyFeatherAptitudes(p *domainpet.Pet, base, steps [5]int) {
	p.AptStrength = base[0] + steps[0]
	p.AptStamina = base[1] + steps[1]
	p.AptAgility = base[2] + steps[2]
	p.AptIntelligence = base[3] + steps[3]
	p.AptEnergy = base[4] + steps[4]
	p.UpdatedAt = time.Now()
}
