package item

import (
	"context"
	"errors"
	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

type upgradeFailRepo struct {
	*itemServiceTestRepo
	failCreate bool
	failDelete bool
}

func (r *upgradeFailRepo) Create(ctx context.Context, it *domainitem.Item) error {
	if r.failCreate {
		return errors.New("injected grant failure")
	}
	return r.itemServiceTestRepo.Create(ctx, it)
}
func (r *upgradeFailRepo) Delete(ctx context.Context, id int64) error {
	if r.failDelete && id == 50 {
		return errors.New("injected consume failure")
	}
	return r.itemServiceTestRepo.Delete(ctx, id)
}

func upgradeTestService(t *testing.T, repo domainitem.Repository) *Service {
	t.Helper()
	s := NewService(repo, zap.NewNop())
	m := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, m, models.ItemTemplateTemplate{ID: 2, StackMax: 3})
	s.SetGameDataManager(m)
	return s
}

func TestUpgradeExchangeReturnsEveryChangedStack(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.nextID = 100
	repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 10}
	repo.items[51] = &domainitem.Item{ID: 51, CharacterID: 1, TemplateID: 2, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 1, StackCount: 2}
	s := upgradeTestService(t, repo)
	result, err := s.ExchangeUpgradeItems(context.Background(), 1, 50, 10, []BagReward{{TemplateID: 2, ItemType: domainitem.ItemTypeConsumable, Count: 5}})
	if err != nil || !result.Deleted || result.Source.CalculateSID() != 2101 || len(result.Products) != 3 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if repo.items[50] != nil {
		t.Fatal("source not consumed")
	}
	total := 0
	for _, p := range result.Products {
		total += p.StackCount
	}
	if total != 7 {
		t.Fatal(total)
	}
}

func TestUpgradeExchangeTechnicalErrorsRestoreSource(t *testing.T) {
	for _, consumeFail := range []bool{false, true} {
		repo := &upgradeFailRepo{itemServiceTestRepo: newItemServiceTestRepo(), failCreate: !consumeFail, failDelete: consumeFail}
		repo.nextID = 100
		repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 5}
		s := upgradeTestService(t, repo)
		_, err := s.ExchangeUpgradeItems(context.Background(), 1, 50, 5, []BagReward{{TemplateID: 2, ItemType: domainitem.ItemTypeConsumable, Count: 1}})
		if err == nil || len(repo.items) != 1 || repo.items[50].StackCount != 5 || repo.items[50].SlotType != domainitem.SlotTypeBag || repo.items[50].SlotIndex != 0 {
			t.Fatalf("lost input or retained product: %v %+v", err, repo.items)
		}
	}
}

func TestUpgradeExchangeUsesFreedSourceSlotInFullBag(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.nextID = 1000
	for i := 0; i < domainitem.DefaultBagSlots; i++ {
		id := int64(i + 50)
		repo.items[id] = &domainitem.Item{ID: id, CharacterID: 1, TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: i, StackCount: 5}
	}
	s := upgradeTestService(t, repo)
	result, err := s.ExchangeUpgradeItems(context.Background(), 1, 50, 5, []BagReward{{TemplateID: 2, ItemType: domainitem.ItemTypeConsumable, Count: 1}})
	if err != nil || !result.Deleted || len(result.Products) != 1 || result.Products[0].SlotIndex != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestUpgradeExchangeRandomFailureConsumesOnlyInput(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 8}
	s := upgradeTestService(t, repo)
	result, err := s.ExchangeUpgradeItems(context.Background(), 1, 50, 3, nil)
	if err != nil || result.Deleted || result.Source.StackCount != 5 || len(result.Products) != 0 || repo.items[50].StackCount != 5 {
		t.Fatalf("%+v %v", result, err)
	}
}
