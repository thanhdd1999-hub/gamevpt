package shop

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type vipFailureCharacterRepo struct {
	*vipShopCharacterRepo
	failUpdate bool
}

func (r *vipFailureCharacterRepo) FindByID(ctx context.Context, id int64) (*character.Character, error) {
	char, ok := r.items[id]
	if !ok {
		return nil, pkgerrors.ErrCharacterNotFound
	}
	return char, nil
}

func (r *vipFailureCharacterRepo) Update(ctx context.Context, char *character.Character) error {
	if r.failUpdate {
		return errors.New("wallet persistence unavailable")
	}
	return r.vipShopCharacterRepo.Update(ctx, char)
}

type vipFailurePurchaseRepo struct {
	*fakeVIPShopRepo
	failRecord   bool
	rejectRecord bool
}

func (r *vipFailurePurchaseRepo) RecordPurchase(ctx context.Context, characterID, epoch int64, slotID, amount int32) (bool, error) {
	if r.failRecord {
		return false, errors.New("purchase persistence unavailable")
	}
	if r.rejectRecord {
		return false, nil
	}
	return r.fakeVIPShopRepo.RecordPurchase(ctx, characterID, epoch, slotID, amount)
}

func newVIPPurchaseFailureService(t *testing.T, gold int64) (*shopService, *vipFailureCharacterRepo, *vipFailurePurchaseRepo, *purchaseSyncItemRepo, int) {
	t.Helper()
	char := &character.Character{
		ID: 1, Name: "vip-purchase", Gold: gold, BagSlotNum: 1,
		VIPType: 1, VIPExpiresAt: vipExpiryFromNow(24 * time.Hour),
	}
	charRepo := &vipFailureCharacterRepo{vipShopCharacterRepo: newVIPShopCharacterRepo(char)}
	service := newVIPShopTestServiceWithRepo(t, charRepo.vipShopCharacterRepo).(*shopService)
	service.SetCharacterRepository(charRepo)
	purchaseRepo := &vipFailurePurchaseRepo{fakeVIPShopRepo: newFakeVIPShopRepo()}
	service.SetVIPShopRepository(purchaseRepo)
	items := newPurchaseSyncItemRepo()
	itemService := appitem.NewService(items, zap.NewNop())
	itemService.SetGameDataManager(service.gameDataRec)
	itemService.SetCharacterRepository(charRepo)
	service.SetItemService(itemService)
	records := make([]json.RawMessage, 0, 7)
	for i := 0; i < 7; i++ {
		data, err := json.Marshal(models.ItemTemplateTemplate{
			ID: int64(6001 + i), Name: "VIP reward", Kind: 5, Type: 500, StackMax: 99,
		})
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, data)
	}
	if err := service.gameDataRec.GetCache().LoadTable(models.TableItemTemplate, records); err != nil {
		t.Fatal(err)
	}
	config, err := service.GetVIPShopConfig(context.Background(), 1)
	if err != nil || len(config.VIPShop) == 0 {
		t.Fatalf("VIP config unavailable: %v", err)
	}
	return service, charRepo, purchaseRepo, items, slotID(config.VIPShop[0]["id"])
}

func assertVIPPurchaseUnchanged(t *testing.T, charRepo *vipFailureCharacterRepo, repo *vipFailurePurchaseRepo, gold int64) {
	t.Helper()
	if got := charRepo.items[1].Gold; got != gold {
		t.Fatalf("gold = %d, want unchanged %d", got, gold)
	}
	for _, epochs := range repo.purchases {
		for _, slots := range epochs {
			if len(slots) != 0 {
				t.Fatal("failed purchase consumed rotation slot")
			}
		}
	}
}

func TestVIPPurchaseInsufficientGoldDoesNotConsumeSlot(t *testing.T) {
	s, chars, repo, items, id := newVIPPurchaseFailureService(t, 0)
	if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); !errors.Is(err, ErrVIPShopInsufficientGold) {
		t.Fatalf("error = %v, want insufficient gold", err)
	}
	assertVIPPurchaseUnchanged(t, chars, repo, 0)
	if len(items.items) != 0 {
		t.Fatal("failed purchase granted item")
	}
	chars.items[1].Gold = 1000
	if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); err != nil {
		t.Fatalf("funded retry should succeed: %v", err)
	}
}

func TestVIPPurchaseFullBagDoesNotConsumeSlotOrGold(t *testing.T) {
	s, chars, repo, items, id := newVIPPurchaseFailureService(t, 1000)
	for i := 0; i < 30; i++ {
		it := domainitem.NewItem(1, 9000+i, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, i)
		if err := items.Create(context.Background(), it); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); !errors.Is(err, pkgerrors.ErrInventoryFull) {
		t.Fatalf("error = %v, want bag full", err)
	}
	assertVIPPurchaseUnchanged(t, chars, repo, 1000)
	if len(items.items) != 30 {
		t.Fatal("failed purchase changed inventory")
	}
	delete(items.items, 1)
	if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); err != nil {
		t.Fatalf("retry with room should succeed: %v", err)
	}
}

func TestVIPPurchaseCommitFailuresRestoreInventoryAndGold(t *testing.T) {
	for _, mode := range []string{"wallet", "record", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			s, chars, repo, items, id := newVIPPurchaseFailureService(t, 1000)
			chars.failUpdate = mode == "wallet"
			repo.failRecord = mode == "record"
			repo.rejectRecord = mode == "duplicate"
			if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); err == nil {
				t.Fatal("expected commit failure")
			}
			assertVIPPurchaseUnchanged(t, chars, repo, 1000)
			if len(items.items) != 0 {
				t.Fatal("failed commit left reward in inventory")
			}
			chars.failUpdate, repo.failRecord, repo.rejectRecord = false, false, false
			if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); err != nil {
				t.Fatalf("retry after recovery should succeed: %v", err)
			}
		})
	}
}

func TestVIPPurchaseRejectsInvalidQuantityAndPrice(t *testing.T) {
	for _, amount := range []int{-1, 0, 2} {
		s, chars, repo, items, id := newVIPPurchaseFailureService(t, 1000)
		if _, err := s.BuyVIPShopItem(context.Background(), 1, id, amount); err == nil {
			t.Fatalf("accepted quantity %d", amount)
		}
		assertVIPPurchaseUnchanged(t, chars, repo, 1000)
		if len(items.items) != 0 {
			t.Fatal("invalid quantity granted reward")
		}
	}
	for _, price := range []float64{-1, 0, 0.5, math.NaN(), math.Inf(1), float64(math.MaxInt64)} {
		s, chars, repo, items, id := newVIPPurchaseFailureService(t, 1000)
		s.gameDataRec.GetShopSlot(id).Gold = price
		if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); err == nil {
			t.Fatalf("accepted price %v", price)
		}
		assertVIPPurchaseUnchanged(t, chars, repo, 1000)
		if len(items.items) != 0 {
			t.Fatal("invalid price granted reward")
		}
	}
}

func TestVIPPurchaseSuccessChargesOnceAndSyncsStack(t *testing.T) {
	s, chars, _, items, id := newVIPPurchaseFailureService(t, 1000)
	slot := s.gameDataRec.GetShopSlot(id)
	it := domainitem.NewItem(1, int(slot.ItemID), domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, 0)
	it.StackCount, it.ColorCode = 5, int(math.Ceil(slot.Quality/5))
	if err := items.Create(context.Background(), it); err != nil {
		t.Fatal(err)
	}
	result, err := s.BuyVIPShopItem(context.Background(), 1, id, 1)
	if err != nil {
		t.Fatal(err)
	}
	if chars.items[1].Gold != 1000-result.Purchase.Cost || len(items.items) != 1 || items.items[it.ID].StackCount != 6 {
		t.Fatal("purchase did not debit once and update existing stack")
	}
	if len(result.Purchase.AddedItemDTOs) != 1 || dtoInt(t, result.Purchase.AddedItemDTOs[0], "stackNum") != 6 {
		t.Fatal("purchase callback omitted updated total stack")
	}
	goldAfter := chars.items[1].Gold
	if _, err := s.BuyVIPShopItem(context.Background(), 1, id, 1); !errors.Is(err, ErrVIPShopSlotAlreadyPurchased) {
		t.Fatalf("duplicate error = %v", err)
	}
	if chars.items[1].Gold != goldAfter || items.items[it.ID].StackCount != 6 {
		t.Fatal("duplicate purchase changed wallet or inventory")
	}
}
