package pet

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
)

type seniorFusionRepo struct {
	*normalFusionRepo
	cost int
}

func (r *seniorFusionRepo) CommitSeniorPetFusion(ctx context.Context, p *domainpet.Pet, ids []int64, cost int) error {
	if r.err != nil {
		return r.err
	}
	r.cost = cost
	return r.CommitPetFusion(ctx, p, ids)
}
func newSeniorFusionTest(t *testing.T, grow float64, count int) (*Service, *seniorFusionRepo, []int64) {
	t.Helper()
	_, base, _ := newNormalFusionTest(t, grow)
	base.pets = map[int64]*domainpet.Pet{}
	ids := make([]int64, count+1)
	for i := range ids {
		ids[i] = int64(i + 1)
		base.pets[ids[i]] = &domainpet.Pet{ID: ids[i], CharacterID: 77, TemplateID: 19, GrowRate: grow, Level: 20, UpgradeNum: 4, GrowRateAdd: 0.2, Property: map[string]interface{}{"state": 3, "skill1": 5527}}
	}
	repo := &seniorFusionRepo{normalFusionRepo: base}
	s := NewService(repo, zap.NewNop())
	s.SetItemRepository(&bookSkillItemRepo{items: map[int64]*domainitem.Item{100: {ID: 100, CharacterID: 77, TemplateID: 3014, SlotType: domainitem.SlotTypeBag, StackCount: 100}}})
	return s, repo, ids
}

func TestSeniorFusionExactRateTableAndCosts(t *testing.T) {
	for _, tc := range []struct {
		grow  float64
		rates []int
	}{{1.5, []int{50, 53, 58, 65, 74, 85, 100}}, {1.8, []int{60, 60, 60, 60, 60, 60, 60, 60, 66, 73, 81, 90, 100}}} {
		for i, rate := range tc.rates {
			for _, delta := range []int{-1, 0} {
				if rate+delta >= 100 {
					continue
				}
				s, repo, ids := newSeniorFusionTest(t, tc.grow, i+4)
				repo.pets[2].SetBinded(true)
				before := clonePetSnapshot(repo.pets[1])
				calls := 0
				result, err := s.fuseSeniorPets(context.Background(), 77, ids, func(n int) (int, error) {
					calls++
					if calls == 1 {
						return rate + delta, nil
					}
					return n - 1, nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if result.Success != (delta < 0) {
					t.Fatalf("grow%v count%d roll%d success%v", tc.grow, i+4, rate+delta, result.Success)
				}
				wantCost := 1
				if tc.grow == 1.8 {
					wantCost = 2 * (i + 4)
				}
				if repo.cost != wantCost {
					t.Fatalf("cost%d want%d", repo.cost, wantCost)
				}
				if len(result.ConsumedItems) != 1 || result.ConsumedItems[0].ID != 100 || result.ConsumedItems[0].StackCount != 100-wantCost {
					t.Fatal("wrong material callbacks")
				}
				if len(repo.pets) != 1 || len(result.ConsumedIDs) != i+4 {
					t.Fatal("wrong consumed count")
				}
				if result.Success {
					wantGrow := 2.0
					if tc.grow == 1.8 {
						wantGrow = 2.5
					}
					if result.Pet.GrowRate != wantGrow || !result.Pet.IsBound() || result.Pet.UpgradeNum != 4 {
						t.Fatal("incorrect success result")
					}
				} else if !reflect.DeepEqual(repo.pets[1], before) {
					t.Fatal("failure changed main")
				}
			}
		}
	}
}

func TestSeniorFusionRejectsBeforeConsumption(t *testing.T) {
	for _, kind := range []string{"too few", "too many", "duplicate", "wrong owner", "wrong colour", "wrong template", "not resting", "equipped", "soul", "wrong tier", "missing materials", "atomic failure", "no atomic repository", "rng failure"} {
		t.Run(kind, func(t *testing.T) {
			s, repo, ids := newSeniorFusionTest(t, 1.5, 4)
			switch kind {
			case "too few":
				ids = ids[:4]
			case "too many":
				s, repo, ids = newSeniorFusionTest(t, 1.5, 11)
			case "duplicate":
				ids[4] = ids[1]
			case "wrong owner":
				repo.pets[2].CharacterID = 8
			case "wrong colour":
				repo.pets[2].GrowRate = 1.8
			case "wrong template":
				repo.pets[2].TemplateID = 20
			case "not resting":
				repo.pets[2].Property["state"] = 1
			case "equipped":
				repo.pets[2].SetEquipmentItemID(1, 1000)
			case "soul":
				repo.pets[2].SoulInfo = map[string]interface{}{"data": map[string]interface{}{"1": 1}}
			case "wrong tier":
				for _, p := range repo.pets {
					p.GrowRate = 1.1
				}
			case "missing materials":
				s.SetItemRepository(&bookSkillItemRepo{items: map[int64]*domainitem.Item{}})
			case "atomic failure":
				repo.err = errors.New("commit failed")
			case "no atomic repository":
				s.petRepo = repo.openSkillPetRepo
			}
			before := map[int64]*domainpet.Pet{}
			for id, p := range repo.pets {
				before[id] = clonePetSnapshot(p)
			}
			_, err := s.fuseSeniorPets(context.Background(), 77, ids, func(int) (int, error) {
				if kind == "rng failure" {
					return 0, errors.New("rng failed")
				}
				return 0, nil
			})
			if err == nil {
				t.Fatal("expected rejection")
			}
			if repo.commits != 0 || !reflect.DeepEqual(repo.pets, before) {
				t.Fatal("invalid fusion mutated pets")
			}
		})
	}
}
