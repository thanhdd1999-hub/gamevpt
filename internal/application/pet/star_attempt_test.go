package pet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type starAttemptRepo struct {
	*openSkillPetRepo
	items   *bookSkillItemRepo
	err     error
	commits int
}

func (r *starAttemptRepo) CommitPetStarAttempt(ctx context.Context, p *domainpet.Pet, items []*domainitem.Item) error {
	if r.err != nil {
		return r.err
	}
	r.commits++
	if err := r.Save(ctx, p); err != nil {
		return err
	}
	for _, it := range items {
		next := *it
		next.StackCount--
		if next.StackCount == 0 {
			delete(r.items.items, it.ID)
		} else {
			r.items.items[it.ID] = &next
		}
	}
	return nil
}

func starAttemptService(t *testing.T, stars int) (*Service, *starAttemptRepo) {
	t.Helper()
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{
		11: {ID: 11, CharacterID: 1, TemplateID: 274, SlotType: domainitem.SlotTypeBag, StackCount: 3},
		12: {ID: 12, CharacterID: 1, TemplateID: 2276, SlotType: domainitem.SlotTypeBag, StackCount: 3},
	}}
	repo := &starAttemptRepo{openSkillPetRepo: &openSkillPetRepo{pets: map[int64]*domainpet.Pet{1: {ID: 1, CharacterID: 1, TemplateID: 19, Level: 20, UpgradeNum: stars, GrowRate: 1.5, GrowRateAdd: .4, Property: map[string]interface{}{"state": 3, "skill1": 5527, "binded": 1}}}}, items: items}
	s := NewService(repo, zap.NewNop())
	s.SetItemRepository(items)
	m := newPetSkillBookGameData(t)
	if err := m.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":274,"type":507,"prop_type":3,"propl_num":5}`), json.RawMessage(`{"id":2276,"type":507,"prop_type":6,"propl_num":1}`)}); err != nil {
		t.Fatal(err)
	}
	s.SetGameDataManager(m)
	return s, repo
}

func TestPetStarAttemptRules(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stars, roll int
		protect     bool
		wantStar    int
		wantAdd     float64
		wantSafe    int
		wantSuccess bool
	}{
		{"low success", 0, 0, false, 1, .45, 3, true},
		{"low failure", 2, 90, false, 2, .4, 3, false},
		{"low protection retained", 2, 99, true, 2, .4, 3, false},
		{"eight loses stars", 8, 99, false, 0, 0, 3, false},
		{"eight protected", 8, 99, true, 8, .4, 2, false},
		{"nine bonus client only", 8, 0, true, 9, .4, 2, true},
		{"twelve maximum", 11, 19, false, 12, .4, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r := starAttemptService(t, tc.stars)
			var protect int64
			if tc.protect {
				protect = 12
			}
			result, err := s.attemptPetStar(context.Background(), 1, 1, 11, protect, func() (int, error) { return tc.roll, nil })
			if err != nil {
				t.Fatal(err)
			}
			if result.Success != tc.wantSuccess || result.Pet.UpgradeNum != tc.wantStar || result.Pet.GrowRateAdd != tc.wantAdd || r.items.items[11].StackCount != 2 || r.items.items[12].StackCount != tc.wantSafe {
				t.Fatalf("bad result %+v", result)
			}
			if petSkillSlotValue(result.Pet, 1) != 5527 {
				t.Fatal("skill changed")
			}
		})
	}
}

func TestPetStarAttemptInvalidAndCommitFailureKeepInputs(t *testing.T) {
	for _, tc := range []string{"maximum", "owner", "stone type", "empty stone", "bound stone", "commit error", "rng error"} {
		t.Run(tc, func(t *testing.T) {
			s, r := starAttemptService(t, 2)
			switch tc {
			case "maximum":
				r.pets[1].UpgradeNum = 12
			case "owner":
				r.pets[1].CharacterID = 2
			case "stone type":
				r.items.items[11].TemplateID = 2276
			case "empty stone":
				r.items.items[11].StackCount = 0
			case "bound stone":
				r.items.items[11].IsBound = true
				r.pets[1].SetBinded(false)
			case "commit error":
				r.err = errors.New("commit failed")
			}
			stars, quantity := r.pets[1].UpgradeNum, r.items.items[11].StackCount
			_, err := s.attemptPetStar(context.Background(), 1, 1, 11, 0, func() (int, error) {
				if tc == "rng error" {
					return 0, errors.New("rng failed")
				}
				return 0, nil
			})
			if err == nil || r.pets[1].UpgradeNum != stars || r.items.items[11].StackCount != quantity {
				t.Fatal("invalid attempt mutated inputs")
			}
		})
	}
}
