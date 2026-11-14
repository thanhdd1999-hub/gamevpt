package pet

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type illusionRepo struct {
	*petItemActionTestRepo
	attempts int
}

func (r *illusionRepo) CommitPetIllusionAttempt(ctx context.Context, p *domainpet.Pet, donor int64, luck *domainitem.Item, count int) error {
	if r.err != nil {
		return r.err
	}
	r.attempts++
	if err := r.Save(ctx, p); err != nil {
		return err
	}
	delete(r.pets, donor)
	if luck != nil {
		r.items.items[luck.ID].StackCount -= count
	}
	return nil
}
func newIllusionTest(t *testing.T) (*Service, *illusionRepo) {
	t.Helper()
	s, base := newPetItemActionTest(t, 1)
	base.pets[1].GrowRate = 1.8
	base.pets[1].Property["aptStrengthIllusion"] = 50
	donor := clonePetSnapshot(base.pets[1])
	donor.ID = 2
	donor.AptStrength = 1000
	donor.Property["binded"] = 0
	donor.Property["aptStrengthIllusion"] = 9999
	base.pets[2] = donor
	repo := &illusionRepo{petItemActionTestRepo: base}
	s.petRepo = repo
	return s, repo
}

func TestPetIllusionPreviewThenConfirm(t *testing.T) {
	s, repo := newIllusionTest(t)
	result, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 0, 0, 1, func(int) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsHint || result.Success || result.Value != 100 || repo.pets[2] != nil || illusionValue(repo.pets[1], "aptStrengthIllusion") != 50 {
		t.Fatal("preview applied prematurely or wrong donor aptitude")
	}
	applied, err := s.ConfirmPetIllusion(context.Background(), 77, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Success || illusionValue(repo.pets[1], "aptStrengthIllusion") != 100 || repo.pets[1].AptStrengthEx != 4 {
		t.Fatal("confirmation changed allocated points")
	}
	if _, err := s.ConfirmPetIllusion(context.Background(), 77, 1); err == nil {
		t.Fatal("proposal replay succeeded")
	}
}

func TestPetIllusionFailureConsumesDonorPreservesIllusion(t *testing.T) {
	s, repo := newIllusionTest(t)
	result, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 0, 0, 1, func(int) (int, error) { return 999, nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.IsHint || result.Success || repo.pets[2] != nil || illusionValue(repo.pets[1], "aptStrengthIllusion") != 50 {
		t.Fatal("failure did not preserve previous illusion")
	}
	if _, err := s.ConfirmPetIllusion(context.Background(), 77, 1); err == nil {
		t.Fatal("failed roll created proposal")
	}
}

func TestPetIllusionChanceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		grow   float64
		class  int
		family bool
		chance int
	}{{1.8, 10, true, 50}, {1.8, 3, true, 40}, {2.3, 10, true, 80}, {2.3, 3, true, 50}, {1.8, 10, false, 15}, {2.3, 3, false, 15}} {
		for _, delta := range []int{-1, 0} {
			s, repo := newIllusionTest(t)
			repo.pets[1].GrowRate = tc.grow
			repo.pets[2].GrowRate = tc.grow
			family := 1
			if !tc.family {
				family = 2
			}
			repo.pets[2].TemplateID = 20
			rows := []json.RawMessage{json.RawMessage(`{"id":19,"class_ids":` + map[int]string{10: "10", 3: "3"}[tc.class] + `,"class_id":1}`), json.RawMessage(`{"id":20,"class_ids":` + map[int]string{10: "10", 3: "3"}[tc.class] + `,"class_id":` + map[int]string{1: "1", 2: "2"}[family] + `}`)}
			if err := s.gameDataManager.GetCache().LoadTable(models.TableCreature, rows); err != nil {
				t.Fatal(err)
			}
			calls := 0
			result, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 0, 0, 1, func(int) (int, error) {
				calls++
				if calls == 1 {
					return tc.chance*10 + delta, nil
				}
				return 0, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsHint != (delta < 0) {
				t.Fatalf("case%+v delta%d hint%v", tc, delta, result.IsHint)
			}
		}
	}
}

func TestPetIllusionRejectsWithoutConsumption(t *testing.T) {
	for _, kind := range []string{"same", "owner", "unbound", "white", "battle", "equipment", "soul", "invalid aptitude", "commit", "rng", "missing luck", "wrong luck", "confirm save failure"} {
		t.Run(kind, func(t *testing.T) {
			s, repo := newIllusionTest(t)
			donor, apt, luck, count := int64(2), 1, int64(0), 0
			switch kind {
			case "same":
				donor = 1
			case "owner":
				repo.pets[2].CharacterID = 8
			case "unbound":
				repo.pets[1].SetBinded(false)
			case "white":
				repo.pets[2].GrowRate = 0.8
			case "battle":
				repo.pets[2].SetClientState(1)
			case "equipment":
				repo.pets[2].SetEquipmentItemID(1, 9)
			case "soul":
				repo.pets[2].SoulInfo = map[string]interface{}{"data": map[string]interface{}{"1": 1}}
			case "invalid aptitude":
				apt = 6
			case "commit":
				repo.err = errors.New("commit failed")
			case "missing luck":
				luck = 9
				count = 1
			case "wrong luck":
				luck = 201
				count = 1
			}
			if kind == "confirm save failure" {
				if _, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 0, 0, 1, func(int) (int, error) { return 0, nil }); err != nil {
					t.Fatal(err)
				}
				repo.saveErr = errors.New("save failed")
				before := clonePetSnapshot(repo.pets[1])
				if _, err := s.ConfirmPetIllusion(context.Background(), 77, 1); err == nil || !reflect.DeepEqual(repo.pets[1], before) {
					t.Fatal("failed confirmation changed pending")
				}
				return
			}
			before := clonePetSnapshot(repo.pets[1])
			_, err := s.previewPetIllusion(context.Background(), 77, 1, donor, luck, count, apt, func(int) (int, error) {
				if kind == "rng" {
					return 0, errors.New("rng failed")
				}
				return 0, nil
			})
			if err == nil || repo.attempts != 0 || repo.pets[2] == nil || !reflect.DeepEqual(repo.pets[1], before) {
				t.Fatal("invalid attempt changed pets")
			}
		})
	}
}

func TestPetIllusionLuckConsumesExactRequestedQuantity(t *testing.T) {
	s, repo := newIllusionTest(t)
	repo.items.items[201].TemplateID = 2325
	repo.items.items[201].StackCount = 5
	if err := s.gameDataManager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":2325,"type":507,"prop_type":7,"color":3}`)}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	result, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 201, 2, 1, func(int) (int, error) {
		calls++
		if calls == 1 {
			return 899, nil
		}
		return 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsHint || result.CostItem != 2 || result.Item.StackCount != 3 || repo.items.items[201].StackCount != 3 {
		t.Fatal("wrong luck consumption or chance")
	}
}

func TestPetIllusionOrangeDonorRequiresOrangeLuck(t *testing.T) {
	s, repo := newIllusionTest(t)
	repo.pets[2].GrowRate = 2.1
	repo.items.items[201].TemplateID = 2325
	if err := s.gameDataManager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":2325,"type":507,"prop_type":7,"color":3}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 201, 1, 1, func(int) (int, error) { return 0, nil }); err == nil || repo.attempts != 0 {
		t.Fatal("purple luck accepted for orange donor")
	}
}

func TestPetIllusionRejectsStaleProposal(t *testing.T) {
	s, repo := newIllusionTest(t)
	if _, err := s.previewPetIllusion(context.Background(), 77, 1, 2, 0, 0, 1, func(int) (int, error) { return 0, nil }); err != nil {
		t.Fatal(err)
	}
	repo.pets[1].AptStrength++
	if _, err := s.ConfirmPetIllusion(context.Background(), 77, 1); err == nil {
		t.Fatal("accepted stale aptitude proposal")
	}
}
