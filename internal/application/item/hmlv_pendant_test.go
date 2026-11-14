package item

import (
	"context"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"strings"
	"testing"
)

func TestHMLVPendantWithoutPetTargetExplainsCtrlUseAndRetainsItem(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 6859, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 2}
	svc, _ := hmlvGiftTestService(t, repo)
	used, result, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 1, Level: 50}, 50, &UseContext{TargetType: 1, TargetID: -1})
	if err != nil || used.StackCount != 2 || repo.items[50].StackCount != 2 {
		t.Fatalf("%v %v %v", used, result, err)
	}
	if msg, _ := result["message"].(string); !strings.Contains(msg, "CTRL") {
		t.Fatalf("missing use instructions %v", result)
	}
	if _, err := svc.applyHMLVPendant(context.Background(), &UseContext{TargetType: 2, TargetID: 5}, &domainchar.Character{ID: 1}, repo.items[50]); err == nil {
		t.Fatal("unsupported pet-service interface must reject")
	}
}
