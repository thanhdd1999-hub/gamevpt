package pet

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"
	domainpet "mcgame-server/internal/domain/pet"
)

type normalFusionRepo struct {
	*openSkillPetRepo
	err     error
	commits int
}

func (r *normalFusionRepo) CommitPetFusion(ctx context.Context, p *domainpet.Pet, ids []int64) error {
	if r.err != nil {
		return r.err
	}
	r.commits++
	if err := r.Save(ctx, p); err != nil {
		return err
	}
	for _, id := range ids {
		delete(r.pets, id)
	}
	return nil
}

func newNormalFusionTest(t *testing.T, grow float64) (*Service, *normalFusionRepo, []int64) {
	t.Helper()
	repo := &normalFusionRepo{openSkillPetRepo: &openSkillPetRepo{pets: map[int64]*domainpet.Pet{}}}
	ids := []int64{1, 2, 3, 4, 5}
	for _, id := range ids {
		repo.pets[id] = &domainpet.Pet{ID: id, CharacterID: 77, TemplateID: 19, Level: 20, GrowRate: grow, GrowRateAdd: 0.2, UpgradeNum: 4, Element: 2, Property: map[string]interface{}{"state": 3, "skill1": 5527, "binded": 0}, CreatureData: map[string]interface{}{"qLevel": 2}}
	}
	return NewService(repo, zap.NewNop()), repo, ids
}

func TestNormalFusionSuccessPreservesMainAndConsumesFour(t *testing.T) {
	for _, tc := range []struct {
		grow      float64
		low, high int
	}{{1.1, 111, 130}, {1.3, 131, 160}, {1.6, 161, 200}, {2, 201, 250}} {
		for _, upper := range []bool{false, true} {
			s, repo, ids := newNormalFusionTest(t, tc.grow)
			repo.pets[3].SetBinded(true)
			calls := 0
			result, err := s.fuseFivePets(context.Background(), 77, ids, func(n int) (int, error) {
				calls++
				if calls == 1 {
					return 0, nil
				}
				if upper {
					return n - 1, nil
				}
				return 0, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			want := float64(tc.low) / 100
			if upper {
				want = float64(tc.high) / 100
			}
			if !result.Success || result.Pet.GrowRate != want || !result.Pet.IsBound() {
				t.Fatalf("result=%+v grow=%v", result, result.Pet.GrowRate)
			}
			p := result.Pet
			if p.Level != 20 || p.UpgradeNum != 4 || p.GrowRateAdd != 0.2 || petSkillSlotValue(p, 1) != 5527 {
				t.Fatal("main progression changed")
			}
			if len(repo.pets) != 1 || !reflect.DeepEqual(result.ConsumedIDs, ids[1:]) {
				t.Fatal("wrong pets consumed")
			}
		}
	}
}

func TestNormalFusionRatesAndFailure(t *testing.T) {
	for _, tc := range []struct {
		grow float64
		rate int
	}{{1.1, 100}, {1.3, 80}, {1.6, 50}, {2, 60}} {
		for _, delta := range []int{-1, 0} {
			if tc.rate+delta >= 100 {
				continue
			}
			s, repo, ids := newNormalFusionTest(t, tc.grow)
			repo.pets[3].SetBinded(true)
			before := clonePetSnapshot(repo.pets[1])
			calls := 0
			result, err := s.fuseFivePets(context.Background(), 77, ids, func(n int) (int, error) {
				calls++
				if calls == 1 {
					return tc.rate + delta, nil
				}
				return 0, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Success != (delta < 0) {
				t.Fatalf("grow%v roll%d success%v", tc.grow, tc.rate+delta, result.Success)
			}
			if !result.Success && !reflect.DeepEqual(repo.pets[1], before) {
				t.Fatal("failed fusion changed main")
			}
			if len(repo.pets) != 1 {
				t.Fatal("failure did not consume all four")
			}
		}
	}
}

func TestNormalFusionRejectsInvalidInputsWithoutWrites(t *testing.T) {
	for _, kind := range []string{"duplicate", "missing", "wrong owner", "wrong template", "wrong colour", "wrong element", "battle", "following", "mounting", "equipment", "soul", "maximum", "atomic failure", "no atomic repository", "random failure"} {
		t.Run(kind, func(t *testing.T) {
			s, repo, ids := newNormalFusionTest(t, 1.5)
			switch kind {
			case "duplicate":
				ids[4] = ids[1]
			case "missing":
				ids = ids[:4]
			case "wrong owner":
				repo.pets[2].CharacterID = 8
			case "wrong template":
				repo.pets[2].TemplateID = 20
			case "wrong colour":
				repo.pets[2].GrowRate = 2
			case "wrong element":
				repo.pets[2].Element = 3
			case "battle":
				repo.pets[2].Property["state"] = 1
			case "following":
				repo.pets[2].IsFollowing = true
			case "mounting":
				repo.pets[2].IsMounting = true
			case "equipment":
				repo.pets[2].SetEquipmentItemID(1, 100)
			case "soul":
				repo.pets[2].SoulInfo = map[string]interface{}{"data": map[string]interface{}{"1": 1}}
			case "maximum":
				for _, p := range repo.pets {
					p.GrowRate = 2.5
				}
			case "atomic failure":
				repo.err = errors.New("commit failed")
			case "no atomic repository":
				s.petRepo = repo.openSkillPetRepo
			}
			before := map[int64]*domainpet.Pet{}
			for id, p := range repo.pets {
				before[id] = clonePetSnapshot(p)
			}
			_, err := s.fuseFivePets(context.Background(), 77, ids, func(int) (int, error) {
				if kind == "random failure" {
					return 0, errors.New("rng failed")
				}
				return 0, nil
			})
			if err == nil {
				t.Fatal("expected failure")
			}
			if repo.commits != 0 {
				t.Fatal("unexpected commit")
			}
			if !reflect.DeepEqual(repo.pets, before) {
				t.Fatal("failed request mutated pets")
			}
		})
	}
}
