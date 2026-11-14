package item

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

func TestBloodBagPotions(t *testing.T) {
	for _, tc := range []struct{ id, index, amount int }{
		{2846, 1, 50000}, {2847, 0, 50000}, {2037, 1, 1000000}, {2038, 0, 1000000},
	} {
		char := &character.Character{CurrentHP: 100, MaxHP: 100, CurrentMP: 100, MaxMP: 100}
		result, err := (&potionEffectHandler{}).Apply(context.Background(), nil, char, &domainitem.Item{}, &models.ItemTemplateTemplate{ID: int64(tc.id)})
		if err != nil || !result.ConsumeItem || char.BloodBag()[tc.index] != tc.amount {
			t.Fatalf("template %d failed: %v %#v", tc.id, err, result)
		}
		if result.CharacterUpdates["ep"] != char.BloodBagString() {
			t.Fatal("missing reserve update")
		}
		values := char.BloodBag()
		values[tc.index] = character.BloodBagCapacity
		char.SetBloodBag(values)
		result, _ = applyBloodBag(char, &models.ItemTemplateTemplate{ID: int64(tc.id)})
		if result.ConsumeItem {
			t.Fatal("full reserve must retain item")
		}
	}
}

func TestBulkRecoveryConsumesOnlyEffectiveDoses(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[11] = &domainitem.Item{ID: 11, CharacterID: 1, TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 5}
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 1, Type: 501, UseType: 3})
	svc.SetGameDataManager(manager)
	char := &character.Character{ID: 1, CurrentHP: 50, MaxHP: 100}
	result, err := svc.UseMultiItem(context.Background(), char, 11, 5, 1, -1)
	if err != nil || result["consumed"] != 1 || repo.items[11].StackCount != 4 || char.CurrentHP != 100 {
		t.Fatalf("bad bulk result %v %v, hp %d", result, err, char.CurrentHP)
	}
}

func TestRecoveryConsumableNoEffectRetainsItem(t *testing.T) {
	char := &character.Character{CurrentHP: 100, MaxHP: 100, CurrentMP: 100, MaxMP: 100}
	for _, h := range []ItemEffectHandler{&potionEffectHandler{}, &foodEffectHandler{}} {
		for _, tpl := range []*models.ItemTemplateTemplate{{ID: 99999}, {ID: 1, I1: 300}} {
			result, err := h.Apply(context.Background(), nil, char, &domainitem.Item{}, tpl)
			if err != nil || result.ConsumeItem {
				t.Fatal("no-effect item must be retained")
			}
		}
	}
}
