package item

import (
	"context"
	"testing"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func expansionService(t *testing.T, templateID int, count int) (*Service, *itemServiceTestRepo) {
	t.Helper()
	repo := newItemServiceTestRepo()
	repo.items[11] = &domainitem.Item{ID: 11, CharacterID: 1, TemplateID: templateID, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: count}
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: int64(templateID), Type: 500, UseType: 1})
	svc.SetGameDataManager(manager)
	return svc, repo
}

func TestExpansionBooksConsumeOneAndRefreshOnlyChangedCapacity(t *testing.T) {
	for _, tc := range []struct {
		id   int
		key  string
		want int
	}{{325, "bagSlotNum", 2}, {324, "bankSlotNum", 2}, {326, "petMaxNum", 7}} {
		t.Run(tc.key, func(t *testing.T) {
			svc, repo := expansionService(t, tc.id, 2)
			char := character.NewCharacter([16]byte{}, "Test", 1, 1)
			char.ID = 1
			_, result, err := svc.UseItem(context.Background(), char, 11, nil)
			if err != nil {
				t.Fatal(err)
			}
			updates, ok := result["character_updates"].(map[string]interface{})
			if !ok || len(updates) != 1 || updates[tc.key] != tc.want {
				t.Fatalf("unexpected updates: %v", result)
			}
			if repo.items[11].StackCount != 1 {
				t.Fatal("must consume exactly one book")
			}
			if char.BagSlots != char.BagSlotNum*30 || char.BankSlots != char.BankSlotNum*30 || char.PetSlots != char.PetMaxNum {
				t.Fatal("derived capacity is stale")
			}
		})
	}
}

func TestExpansionLimitKeepsBook(t *testing.T) {
	for _, id := range []int{324, 325, 326} {
		svc, repo := expansionService(t, id, 1)
		char := &character.Character{ID: 1, BagSlotNum: 7, BagSlots: 210, BankSlotNum: 5, BankSlots: 150, PetMaxNum: domainpet.MaxPets, PetSlots: domainpet.MaxPets}
		_, result, err := svc.UseItem(context.Background(), char, 11, nil)
		if err != nil {
			t.Fatal(err)
		}
		if repo.items[11] == nil || repo.items[11].StackCount != 1 {
			t.Fatalf("book %d consumed at cap", id)
		}
		if result["message"] == nil || result["character_updates"] != nil {
			t.Fatalf("bad limit response: %v", result)
		}
	}
}

func TestUnsupportedScrollKeepsBook(t *testing.T) {
	svc, repo := expansionService(t, 99999, 2)
	_, _, err := svc.UseItem(context.Background(), &character.Character{ID: 1}, 11, nil)
	if !pkgerrors.Is(err, pkgerrors.ErrCannotUseItem) || repo.items[11].StackCount != 2 {
		t.Fatalf("unsupported item changed: %v", err)
	}
}

func TestMultiUseRejectsInvalidCountAndExpansionWithoutMutation(t *testing.T) {
	for _, count := range []int{-1, 0, 2} {
		svc, repo := expansionService(t, 325, 2)
		char := &character.Character{ID: 1, BagSlotNum: 1, BagSlots: 30}
		_, err := svc.UseMultiItem(context.Background(), char, 11, count, 1, -1)
		if err == nil || repo.items[11].StackCount != 2 || char.BagSlotNum != 1 {
			t.Fatalf("multi-use mutated state for count %d: %v", count, err)
		}
	}
}
