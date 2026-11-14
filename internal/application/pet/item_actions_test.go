package pet

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

type petItemActionTestRepo struct {
	*openSkillPetRepo
	items   *bookSkillItemRepo
	err     error
	commits int
}

func (r *petItemActionTestRepo) CommitPetItemAction(ctx context.Context, p *domainpet.Pet, it *domainitem.Item) error {
	if r.err != nil {
		return r.err
	}
	r.commits++
	if err := r.Save(ctx, p); err != nil {
		return err
	}
	if it.StackCount == 1 {
		delete(r.items.items, it.ID)
	} else {
		r.items.items[it.ID].StackCount--
	}
	return nil
}
func newPetItemActionTest(t *testing.T, kind int) (*Service, *petItemActionTestRepo) {
	t.Helper()
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{201: {ID: 201, CharacterID: 77, TemplateID: 1526, SlotType: domainitem.SlotTypeBag, StackCount: 2}}}
	repo := &petItemActionTestRepo{openSkillPetRepo: &openSkillPetRepo{pets: map[int64]*domainpet.Pet{1: {ID: 1, CharacterID: 77, TemplateID: 19, Level: 20, GrowRate: 1.5, GrowRateAdd: 0.3, UpgradeNum: 5, AptStrength: 100, AptAgility: 100, AptStamina: 100, AptIntelligence: 100, AptEnergy: 100, AptStrengthEx: 4, AptAgilityEx: 6, AttrPoints: 5, DistributedAttrPoints: 10, Property: map[string]interface{}{"state": 3, "binded": 1, "skill1": 5527}}}}, items: items}
	manager := gamedata.NewManager(nil, zap.NewNop())
	for table, rows := range map[string][]json.RawMessage{
		models.TableCreature:     {json.RawMessage(`{"id":19,"class_ids":10,"apt_strength":100,"apt_agility":100,"apt_stamina":100,"apt_intelligence":100,"apt_energy":100,"att_strength":10,"att_agility":10,"att_stamina":10,"att_intelligence":10,"att_energy":10}`)},
		models.TableItemTemplate: {json.RawMessage(`{"id":1526,"type":507,"prop_type":` + []string{"0", "1", "2", "3", "4"}[kind] + `,"propl_num":1,"name":"Test"}`)},
	} {
		if err := manager.GetCache().LoadTable(table, rows); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(repo, zap.NewNop())
	s.SetItemRepository(items)
	s.SetGameDataManager(manager)
	return s, repo
}

func TestPetItemActionsCommitAndPreserveProgress(t *testing.T) {
	for _, kind := range []int{1, 2, 4} {
		s, repo := newPetItemActionTest(t, kind)
		result, err := s.applyPetItemAction(context.Background(), 77, 1, 201, kind, func(n int) (int, error) { return n - 1, nil })
		if err != nil {
			t.Fatal(err)
		}
		p := result.Pet
		if repo.commits != 1 || repo.items.items[201].StackCount != 1 || result.Item.StackCount != 1 {
			t.Fatal("wrong item consumption")
		}
		if p.Level != 20 || p.UpgradeNum != 5 || p.GrowRateAdd != 0.3 || petSkillSlotValue(p, 1) != 5527 {
			t.Fatal("progress changed")
		}
		switch kind {
		case 1:
			if p.AptStrengthEx != 0 || p.AptAgilityEx != 0 || p.AttrPoints != 15 || p.DistributedAttrPoints != 0 {
				t.Fatal("points not refunded")
			}
		case 2:
			if p.AptStrength != 120 || p.AptAgility != 100 || result.Field != "aptStrength" {
				t.Fatal("wrong aptitude reroll")
			}
		case 4:
			if p.GrowRate != 1.6 || result.Field != "growRate" {
				t.Fatal("wrong growth reroll")
			}
		}
	}
}

func TestPetItemActionsRejectWithoutItemLoss(t *testing.T) {
	for _, kind := range []string{"owner", "item owner", "unbound", "not rest", "wrong type", "wrong prop", "bank", "missing material", "atomic", "no atomic", "rng", "white growth"} {
		t.Run(kind, func(t *testing.T) {
			s, repo := newPetItemActionTest(t, 4)
			switch kind {
			case "owner":
				repo.pets[1].CharacterID = 8
			case "item owner":
				repo.items.items[201].CharacterID = 8
			case "unbound":
				repo.pets[1].SetBinded(false)
			case "not rest":
				repo.pets[1].SetClientState(1)
			case "wrong type":
				if err := s.gameDataManager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":1526,"type":550,"prop_type":4}`)}); err != nil {
					t.Fatal(err)
				}
			case "wrong prop":
				if err := s.gameDataManager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":1526,"type":507,"prop_type":2}`)}); err != nil {
					t.Fatal(err)
				}
			case "bank":
				repo.items.items[201].SlotType = domainitem.SlotTypeBank
			case "missing material":
				repo.items.items[201].StackCount = 0
			case "atomic":
				repo.err = errors.New("commit failed")
			case "no atomic":
				s.petRepo = repo.openSkillPetRepo
			case "white growth":
				repo.pets[1].GrowRate = 0.8
			}
			before := clonePetSnapshot(repo.pets[1])
			quantity := repo.items.items[201].StackCount
			_, err := s.applyPetItemAction(context.Background(), 77, 1, 201, 4, func(int) (int, error) {
				if kind == "rng" {
					return 0, errors.New("rng failed")
				}
				return 0, nil
			})
			if err == nil || repo.commits != 0 || repo.items.items[201].StackCount != quantity || !reflect.DeepEqual(repo.pets[1], before) {
				t.Fatal("failure changed inventory or pet")
			}
		})
	}
}

func TestAdvancedPetAptitudeRerollNeverDecreases(t *testing.T) {
	for _, current := range []int{80, 100, 119, 120} {
		s, repo := newPetItemActionTest(t, 2)
		repo.pets[1].AptStrength = current
		if err := s.gameDataManager.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{json.RawMessage(`{"id":1526,"type":507,"prop_type":2,"propl_num":1,"name":"Tẩy Tủy Đan Sức Mạnh Cao Cấp"}`)}); err != nil {
			t.Fatal(err)
		}
		result, err := s.applyPetItemAction(context.Background(), 77, 1, 201, 2, func(int) (int, error) { return 0, nil })
		if current == 120 {
			if err == nil || repo.items.items[201].StackCount != 2 {
				t.Fatal("max aptitude consumed item")
			}
			continue
		}
		if err != nil || result.After <= float64(current) || !result.Premium {
			t.Fatalf("result%+v err%v", result, err)
		}
	}
}
