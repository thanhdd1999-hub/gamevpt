package pet

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

func hmlvPendantTestService(t *testing.T) (*Service, *contractTestPetRepo, *bookSkillItemRepo) {
	t.Helper()
	pets := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{5: {ID: 5, CharacterID: 1, TemplateID: 2281, Level: 10, GrowRate: 1.5, Property: map[string]interface{}{"skill1": 7002, "skill2": 7011, "skill3": 0, "state": 1}}}}
	items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{9: {ID: 9, CharacterID: 1, TemplateID: 6859, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 2}}}
	manager := gamedata.NewManager(nil, zap.NewNop())
	skills := []json.RawMessage{}
	for _, skill := range []models.SkillTemplate{{ID: 7011, Name: "Tứ Phương Khiếp Đảm", CodeName: "SKILL702043"}, {ID: 7012, Name: "Tứ Phương Khiếp Đảm", CodeName: "SKILL702043"}, {ID: 7014, Name: "Thợ Săn Sát Thủ", CodeName: "SKILL702044"}} {
		raw, err := json.Marshal(skill)
		if err != nil {
			t.Fatal(err)
		}
		skills = append(skills, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableSkill, skills); err != nil {
		t.Fatal(err)
	}
	svc := NewService(pets, zap.NewNop())
	svc.SetItemRepository(items)
	svc.SetGameDataManager(manager)
	return svc, pets, items
}

func TestHMLVPendantLearnsMissingSkillAndConsumesOne(t *testing.T) {
	svc, pets, items := hmlvPendantTestService(t)
	p, it, msg, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9)
	if err != nil || p == nil || it.StackCount != 1 || items.items[9].StackCount != 1 || petSkillSlotValue(p, 3) != 7014 || msg == "" {
		t.Fatalf("%v %v %v %v", p, it, msg, err)
	}
	if petSkillSlotValue(pets.pets[5], 1) != 7002 || petSkillSlotValue(pets.pets[5], 2) != 7011 {
		t.Fatal("existing skills overwritten")
	}
	p, it, msg, err = svc.UseHMLVPendant(context.Background(), 1, 5, 9)
	if err != nil || p != nil || it.StackCount != 1 || items.items[9].StackCount != 1 || msg == "" {
		t.Fatal("already learned skills consumed pendant")
	}
}

func TestHMLVPendantRejectsWrongPetAndFullSlots(t *testing.T) {
	for _, wrongPet := range []bool{false, true} {
		svc, pets, items := hmlvPendantTestService(t)
		if wrongPet {
			pets.pets[5].TemplateID = 504
		} else {
			for i := 1; i <= domainpet.TotalPetSkillSlots; i++ {
				setPetSkillSlotValue(pets.pets[5], i, 9000+i)
			}
		}
		p, it, msg, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9)
		if err != nil || p != nil || it.StackCount != 2 || items.items[9].StackCount != 2 || msg == "" {
			t.Fatal("invalid pet/full slots must retain pendant")
		}
	}
}

func TestHMLVPendantRecognizesUpgradedSkillFamily(t *testing.T) {
	svc, pets, items := hmlvPendantTestService(t)
	setPetSkillSlotValue(pets.pets[5], 2, 7012)
	setPetSkillSlotValue(pets.pets[5], 3, 7014)
	p, _, _, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9)
	if err != nil || p != nil || items.items[9].StackCount != 2 {
		t.Fatal("upgraded family duplicated")
	}
}

func TestHMLVPendantConsumeFailureRollsBackPet(t *testing.T) {
	svc, pets, items := hmlvPendantTestService(t)
	items.items[9].StackCount = 1
	items.deleteErr = errors.New("injected consumption failure")
	_, _, _, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9)
	if err == nil || petSkillSlotValue(pets.pets[5], 3) != 0 || items.items[9].StackCount != 1 {
		t.Fatal("failed consume left learned skill")
	}
}

func TestHMLVPendantOwnershipChecks(t *testing.T) {
	svc, pets, items := hmlvPendantTestService(t)
	items.items[9].CharacterID = 2
	if _, _, _, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9); err == nil {
		t.Fatal("foreign pendant accepted")
	}
	items.items[9].CharacterID = 1
	pets.pets[5].CharacterID = 2
	if _, _, _, err := svc.UseHMLVPendant(context.Background(), 1, 5, 9); err == nil {
		t.Fatal("foreign pet accepted")
	}
}
