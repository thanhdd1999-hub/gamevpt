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
	pkgerrors "mcgame-server/pkg/errors"
)

func TestUpgradePetSkill(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		skill, quantity, owner int
		saveFail, consumeFail  bool
		want                   int
		wantErr                error
	}{
		{name: "level one consumes one", skill: 5527, quantity: 5, owner: 1, want: 5528},
		{name: "level two consumes three", skill: 5528, quantity: 3, owner: 1, want: 5529},
		{name: "maximum level", skill: 5529, quantity: 5, owner: 1, wantErr: pkgerrors.ErrInvalidInput},
		{name: "missing material", skill: 5527, owner: 1, wantErr: pkgerrors.ErrInsufficientItems},
		{name: "insufficient material", skill: 5528, quantity: 2, owner: 1, wantErr: pkgerrors.ErrInsufficientItems},
		{name: "wrong owner", skill: 5527, quantity: 5, owner: 2, wantErr: pkgerrors.ErrNotFound},
		{name: "save failure preserves items", skill: 5527, quantity: 5, owner: 1, saveFail: true},
		{name: "consume failure restores pet", skill: 5527, quantity: 5, owner: 1, consumeFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &openSkillPetRepo{pets: map[int64]*domainpet.Pet{101: {ID: 101, CharacterID: int64(tc.owner), TemplateID: 1001, Level: 1, Property: map[string]interface{}{"skill1": tc.skill}}}}
			items := &bookSkillItemRepo{items: map[int64]*domainitem.Item{201: {ID: 201, CharacterID: 1, TemplateID: 1367, SlotType: domainitem.SlotTypeBag, StackCount: tc.quantity}}}
			service := NewService(repo, zap.NewNop())
			service.SetItemRepository(items)
			manager := newPetSkillBookGameData(t)
			if err := manager.GetCache().LoadTable(models.TableSkill, []json.RawMessage{
				json.RawMessage(`{"id":5527,"code_name":"SKILL543901","level":1}`),
				json.RawMessage(`{"id":5528,"code_name":"SKILL543901","level":2}`),
				json.RawMessage(`{"id":5529,"code_name":"SKILL543901","level":3}`),
			}); err != nil {
				t.Fatal(err)
			}
			service.SetGameDataManager(manager)
			injected := errors.New("injected failure")
			if tc.saveFail {
				repo.saveErr = injected
			}
			if tc.consumeFail {
				items.updateStackErr = injected
			}
			result, err := service.UpgradePetSkill(context.Background(), 1, 101, 1)
			if tc.saveFail || tc.consumeFail {
				if !errors.Is(err, injected) {
					t.Fatalf("error=%v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v want %v", err, tc.wantErr)
			}
			if err != nil {
				if items.items[201].StackCount != tc.quantity {
					t.Fatal("material changed on failure")
				}
				if petSkillSlotValue(repo.pets[101], 1) != tc.skill {
					t.Fatal("pet changed on failure")
				}
				return
			}
			if result.SkillID != tc.want || petSkillSlotValue(repo.pets[101], 1) != tc.want {
				t.Fatalf("wrong next skill: %+v", result)
			}
			count := 1
			if tc.skill == 5528 {
				count = 3
			}
			if result.Item.StackCount != tc.quantity-count {
				t.Fatal("wrong material consumption")
			}
			if result.Item.StackCount == 0 {
				if _, ok := items.items[201]; ok {
					t.Fatal("empty material stack not deleted")
				}
			}
		})
	}
}
