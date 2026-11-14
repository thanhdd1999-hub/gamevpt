package pet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type evolutionFeatureProvider struct{ active bool }

func (p evolutionFeatureProvider) BuildCharacterLoginState(context.Context, int64) (map[string]interface{}, error) {
	return map[string]interface{}{"activePetObject": map[string]interface{}{"90": map[string]interface{}{"actived": p.active}}}, nil
}
func (evolutionFeatureProvider) AggregatePetBonuses(context.Context, int64) (domainpet.EquipmentStatBonuses, domainpet.SkillStatBonuses) {
	return domainpet.NewEquipmentStatBonuses(), domainpet.NewSkillStatBonuses()
}

type evolutionActionRepo struct {
	*openSkillPetRepo
	chars   *openSkillCharacterRepo
	items   *bookSkillItemRepo
	err     error
	commits int
}

func (r *evolutionActionRepo) CommitPetEvolutionAction(ctx context.Context, p *domainpet.Pet, c *character.Character, items []*domainitem.Item, counts []int, np int, gold int64) error {
	if r.err != nil {
		return r.err
	}
	r.commits++
	if err := r.Save(ctx, p); err != nil {
		return err
	}
	if err := r.chars.Update(ctx, c); err != nil {
		return err
	}
	for i, it := range items {
		next := *it
		next.StackCount -= counts[i]
		if next.StackCount == 0 {
			delete(r.items.items, it.ID)
		} else {
			r.items.items[it.ID] = &next
		}
	}
	return nil
}

func evolutionActionService(t *testing.T, round, stage int, active bool) (*Service, *evolutionActionRepo) {
	t.Helper()
	chars := &openSkillCharacterRepo{characters: map[int64]*character.Character{1: {ID: 1, NpPoint: 10000, Gold: 50000, PMProcessData: map[string]interface{}{"other": "preserved"}}}}
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{11: {ID: 11, CharacterID: 1, TemplateID: 3911, SlotType: domainitem.SlotTypeBag, StackCount: 30}}}
	base := [5]int{4080, 6360, 5760, 2580, 2712}
	p := &domainpet.Pet{ID: 1, CharacterID: 1, TemplateID: 2282, Level: 20, GrowRate: 1.5, Property: map[string]interface{}{"binded": 1, "state": 3, "skill1": 5527, "envoInfo": map[string]interface{}{"envo": true, "round": round, "stage": stage}, "evolutionBaseApt": []int{base[0], base[1], base[2], base[3], base[4]}}}
	steps, _ := featherStepValues(2282, round, stage)
	applyFeatherAptitudes(p, base, steps)
	r := &evolutionActionRepo{openSkillPetRepo: &openSkillPetRepo{pets: map[int64]*domainpet.Pet{1: p}}, chars: chars, items: items}
	s := NewService(r, zap.NewNop())
	s.SetItemRepository(items)
	s.SetCharacterRepository(chars)
	s.SetFeatureBonusProvider(evolutionFeatureProvider{active})
	m := newPetSkillBookGameData(t)
	if err := m.GetCache().LoadTable(models.TableCreature, []json.RawMessage{json.RawMessage(`{"id":2281,"class_ids":10,"name":"Huyền Minh Linh Vũ","apt_strength":3400,"apt_stamina":5300,"apt_agility":4800,"apt_intelligence":2150,"apt_energy":2260}`), json.RawMessage(`{"id":2282,"class_ids":10,"name":"Huyền Minh Tiến Hóa","apt_strength":3400,"apt_stamina":5300,"apt_agility":4800,"apt_intelligence":2150,"apt_energy":2260}`)}); err != nil {
		t.Fatal(err)
	}
	if err := m.GetCache().LoadTable(models.TableCreatureHandbook, []json.RawMessage{json.RawMessage(`{"id":90,"relate_id":2281,"evolution_id":2282}`)}); err != nil {
		t.Fatal(err)
	}
	s.SetGameDataManager(m)
	return s, r
}

func TestBeginPetEvolutionRequiresActivationAndUsesExactTarget(t *testing.T) {
	for _, active := range []bool{false, true} {
		s, r := evolutionActionService(t, 0, 0, active)
		p := r.pets[1]
		p.TemplateID = 2281
		p.Property["envoInfo"] = map[string]interface{}{"envo": false}
		p.AptStrength = 3400
		p.AptStamina = 5300
		p.AptAgility = 4800
		p.AptIntelligence = 2150
		p.AptEnergy = 2260
		result, err := s.BeginPetEvolution(context.Background(), 1, 1)
		if !active {
			if err == nil || r.pets[1].TemplateID != 2281 || r.commits != 0 {
				t.Fatal("inactive handbook must retain pet")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if result.Pet.TemplateID != 2282 || result.Pet.AptStrength != 4080 || result.Pet.AptStamina != 6360 || result.Pet.AptAgility != 5760 || result.Pet.GrowRate != 1.5 || petSkillSlotValue(result.Pet, 1) != 5527 {
			t.Fatal("incorrect evolution transformation")
		}
	}
}

func TestFeatherProgressCostProtectionAndCheckpoints(t *testing.T) {
	for _, tc := range []struct {
		name                                             string
		round, stage, roll                               int
		protect, forced                                  bool
		wantRound, wantStage, wantNP, wantGold, wantItem int
		strength                                         int
	}{
		{"first success", 0, 0, 0, false, false, 0, 1, 9950, 50000, 30, 4102},
		{"failed reset", 0, 2, 99, false, false, 0, 0, 9950, 50000, 30, 4080},
		{"protected failure", 0, 2, 99, true, false, 0, 2, 9950, 50000, 28, 4102},
		{"checkpoint failure", 0, 3, 99, false, false, 0, 3, 9900, 50000, 30, 4153},
		{"forced checkpoint", 0, 4, 99, false, true, 0, 6, 10000, 48200, 30, 4258},
		{"round transition", 0, 8, 0, false, false, 1, 0, 9850, 50000, 30, 4467},
		{"final stage", 2, 8, 0, false, false, 2, 9, 9700, 50000, 30, 6232},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r := evolutionActionService(t, tc.round, tc.stage, true)
			result, err := s.advancePetFeather(context.Background(), 1, 1, tc.protect, tc.forced, func(int) (int, error) { return tc.roll, nil })
			if err != nil {
				t.Fatal(err)
			}
			info := result.Pet.EvolutionInfo()
			if evolutionNumber(info["round"]) != tc.wantRound || evolutionNumber(info["stage"]) != tc.wantStage || result.Character.NpPoint != tc.wantNP || result.Character.Gold != int64(tc.wantGold) || r.items.items[11].StackCount != tc.wantItem || result.Pet.AptStrength != tc.strength {
				t.Fatalf("bad feather result progress %v wallet %d/%d items %d strength%d", info, result.Character.NpPoint, result.Character.Gold, r.items.items[11].StackCount, result.Pet.AptStrength)
			}
			if result.Character.PMProcessData["other"] != "preserved" || result.Character.PMProcessData["localNpPnt"] != tc.wantNP {
				t.Fatal("wallet state not persisted correctly")
			}
		})
	}
}

func TestFeatherTechnicalFailuresAndInsufficientFundsDoNotMutate(t *testing.T) {
	for _, tc := range []string{"np", "gold", "protect", "commit", "maximum"} {
		t.Run(tc, func(t *testing.T) {
			s, r := evolutionActionService(t, 0, 2, true)
			forced := false
			protect := false
			switch tc {
			case "np":
				r.chars.characters[1].NpPoint = 0
			case "gold":
				forced = true
				r.chars.characters[1].Gold = 0
			case "protect":
				protect = true
				r.items.items[11].StackCount = 1
			case "commit":
				r.err = errors.New("commit failure")
			case "maximum":
				r.pets[1].Property["envoInfo"] = map[string]interface{}{"envo": true, "round": 2, "stage": 9}
			}
			np, gold, count := r.chars.characters[1].NpPoint, r.chars.characters[1].Gold, r.items.items[11].StackCount
			_, err := s.advancePetFeather(context.Background(), 1, 1, protect, forced, func(int) (int, error) { return 0, nil })
			if err == nil || r.chars.characters[1].NpPoint != np || r.chars.characters[1].Gold != gold || r.items.items[11].StackCount != count || r.commits != 0 {
				t.Fatal("failed feather mutated resources")
			}
		})
	}
}
