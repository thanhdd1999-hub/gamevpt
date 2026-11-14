package item

import (
	"context"
	"errors"
	"testing"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type failingRewardRepo struct {
	*itemServiceTestRepo
	creates int
}

func (r *failingRewardRepo) Create(ctx context.Context, it *domainitem.Item) error {
	r.creates++
	if r.creates == 2 {
		return errors.New("injected create failure")
	}
	return r.itemServiceTestRepo.Create(ctx, it)
}

func TestRewardBatchRollsBackPartialCreation(t *testing.T) {
	repo := &failingRewardRepo{itemServiceTestRepo: newItemServiceTestRepo()}
	svc := NewService(repo, zap.NewNop())
	committed := false
	_, err := svc.GrantBagRewards(context.Background(), 1, []BagReward{{TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, Count: 2}}, func() error { committed = true; return nil })
	if err == nil || committed || len(repo.items) != 0 {
		t.Fatalf("partial purchase not rolled back: error=%v committed=%v items=%v", err, committed, repo.items)
	}
}

func TestRewardBatchRollsBackWhenCurrencyCommitFails(t *testing.T) {
	repo := newItemServiceTestRepo()
	svc := NewService(repo, zap.NewNop())
	_, err := svc.GrantBagRewards(context.Background(), 1, []BagReward{{TemplateID: 1, ItemType: domainitem.ItemTypeConsumable, Count: 1}}, func() error { return errors.New("currency save failed") })
	if err == nil || len(repo.items) != 0 {
		t.Fatalf("reward remained after failed payment: %v %v", err, repo.items)
	}
}

func TestRewardBatchIncludesEveryChangedStack(t *testing.T) {
	repo := newItemServiceTestRepo()
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 10, StackMax: 3, Type: 550, UseType: 1})
	svc.SetGameDataManager(manager)
	repo.items[20] = &domainitem.Item{ID: 20, CharacterID: 1, TemplateID: 10, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 2}
	changed, err := svc.GrantBagRewards(context.Background(), 1, []BagReward{{TemplateID: 10, ItemType: domainitem.ItemTypeConsumable, Count: 5}}, nil)
	if err != nil || len(changed) != 3 {
		t.Fatalf("missing stack updates: %v %v", changed, err)
	}
	total := 0
	for _, it := range changed {
		total += it.StackCount
		dto := svc.BuildClientItemDTO(it)
		if dto["sid"] == nil || dto["type"] != 28 || dto["stackNum"] != it.StackCount {
			t.Fatalf("invalid Flash slot: %v", dto)
		}
	}
	if total != 7 {
		t.Fatalf("stack total=%d want7", total)
	}
}

func TestWingGiftCannotMatchPetWithSameName(t *testing.T) {
	h := &giftBoxEffectHandler{}
	tpl := &models.ItemTemplateTemplate{ID: 3376, Type: 550, UseType: 1, Name: "Túi Quà Cánh Xanh Dương", Description: "Trong túi có 3 loại cánh Thiên Sứ, Bích Không, Ác Ma màu xanh dương. Có thể chọn lấy 1 trong 3 loại cánh này"}
	if h.CanHandle(tpl) {
		t.Fatal("wing gift must not use creature-name selection")
	}
}

func TestUnsupportedConsumablesKeepTheirStack(t *testing.T) {
	for _, kind := range []int{999, 505, 516} {
		repo := newItemServiceTestRepo()
		svc := NewService(repo, zap.NewNop())
		manager := gamedata.NewManager(nil, zap.NewNop())
		loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 9999, Type: float64(kind), UseType: 1})
		svc.SetGameDataManager(manager)
		repo.items[1] = &domainitem.Item{ID: 1, CharacterID: 1, TemplateID: 9999, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 2}
		_, _, err := svc.UseItem(context.Background(), &character.Character{ID: 1}, 1, nil)
		if err == nil || repo.items[1].StackCount != 2 {
			t.Fatalf("unsupported type %d was consumed: %v", kind, err)
		}
	}
}
