package shop

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

type cachedPurchaseCharacterRepo struct {
	*vipShopCharacterRepo
	failUpdate bool
}

func (r *cachedPurchaseCharacterRepo) FindByID(ctx context.Context, id int64) (*character.Character, error) {
	return r.items[id], nil
}
func (r *cachedPurchaseCharacterRepo) Update(ctx context.Context, char *character.Character) error {
	if r.failUpdate {
		return errors.New("injected wallet failure")
	}
	r.items[char.ID] = char
	return nil
}

func safePurchaseFixture(t *testing.T) (ShopService, *cachedPurchaseCharacterRepo, *purchaseSyncItemRepo) {
	t.Helper()
	manager := newPurchaseSyncTestManager(t, models.ItemTemplateTemplate{ID: 158, Name: "Potion", Price: 1, StackMax: 3})
	slots := []models.ShopSlotTemplate{{ID: 9901, Type: 29, ItemID: 158, Gold: 10, Gt: 1}, {ID: 9902, Type: 29, ItemID: 158, Gold: 5}}
	var rows []json.RawMessage
	for _, slot := range slots {
		raw, err := json.Marshal(slot)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, raw)
	}
	if err := manager.GetCache().LoadTable(models.TableShopSlot, rows); err != nil {
		t.Fatal(err)
	}
	chars := &cachedPurchaseCharacterRepo{vipShopCharacterRepo: newVIPShopCharacterRepo(&character.Character{ID: 1, Gold: 100, GoldBind: 100, MoneyBind: 100, BagSlotNum: 1, SelectedGoldType: 3})}
	items := newPurchaseSyncItemRepo()
	itemSvc := appitem.NewService(items, zap.NewNop())
	itemSvc.SetGameDataManager(manager)
	itemSvc.SetCharacterRepository(chars)
	svc := NewService()
	svc.SetCharacterRepository(chars)
	svc.SetItemService(itemSvc)
	svc.SetGameDataManager(manager)
	return svc, chars, items
}

func TestInvalidPurchaseQuantityNeverChangesCachedWallet(t *testing.T) {
	for _, count := range []int{-1, 0, 1000001} {
		svc, chars, items := safePurchaseFixture(t)
		if _, err := svc.BuyItem(context.Background(), 1, 9901, count); err == nil {
			t.Fatal("NPC shop accepted invalid count")
		}
		if _, err := svc.BuySystemItem(context.Background(), 1, 9901, count); err == nil {
			t.Fatal("system shop accepted invalid count")
		}
		if chars.items[1].Gold != 100 || len(items.items) != 0 {
			t.Fatal("invalid purchase mutated state")
		}
	}
}

func TestFullBagKeepsCachedWalletUnchanged(t *testing.T) {
	svc, chars, items := safePurchaseFixture(t)
	for i := 0; i < 30; i++ {
		it := domainitem.NewItem(1, 999, domainitem.ItemTypeConsumable, domainitem.SlotTypeBag, i)
		if err := items.Create(context.Background(), it); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.BuyItem(context.Background(), 1, 9901, 1); err == nil {
		t.Fatal("full bag purchase succeeded")
	}
	if chars.items[1].Gold != 100 || len(items.items) != 30 {
		t.Fatal("failed purchase charged or granted")
	}
}

func TestWalletSaveFailureRollsBackSingleAndCartRewards(t *testing.T) {
	for _, cart := range []bool{false, true} {
		svc, chars, items := safePurchaseFixture(t)
		chars.failUpdate = true
		var err error
		if cart {
			_, err = svc.BuySystemItemMulti(context.Background(), 1, []PurchaseRequest{{ShopSlotID: 9901, Amount: 2}, {ShopSlotID: 9902, Amount: 2}})
		} else {
			_, err = svc.BuySystemItem(context.Background(), 1, 9901, 2)
		}
		if err == nil || len(items.items) != 0 || chars.items[1].Gold != 100 || chars.items[1].GoldBind != 100 {
			t.Fatalf("failed payment left changes: %v %v", err, items.items)
		}
	}
}

func TestCartHonorsGoldOnlyAndSynchronizesBothBalances(t *testing.T) {
	svc, chars, _ := safePurchaseFixture(t)
	res, err := svc.BuySystemItemMulti(context.Background(), 1, []PurchaseRequest{{ShopSlotID: 9901, Amount: 2}, {ShopSlotID: 9902, Amount: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if chars.items[1].Gold != 80 || chars.items[1].GoldBind != 90 {
		t.Fatalf("wrong currency charged: %v", res.Balances)
	}
	if res.Balances["gold"] != int64(80) || res.Balances["goldBind"] != int64(90) || len(res.AddedItemDTOs) != 2 {
		t.Fatalf("incomplete client state: %v", res)
	}
}

func TestNPCSpecialCurrenciesReturnBothRealBalances(t *testing.T) {
	svc, chars, _ := safePurchaseFixture(t)
	impl := svc.(*shopService)
	chars.items[1].DogMedal = 100
	chars.items[1].AchillesMedal = 50
	slot := models.ShopSlotTemplate{ID: 9903, Type: 29, ItemID: 158, PType1: float64(character.CurrencyDogMedal), PNum1: 2, PType2: float64(character.CurrencyAchillesMedal), PNum2: 3}
	raw, err := json.Marshal(slot)
	if err != nil {
		t.Fatal(err)
	}
	if err := impl.gameDataRec.GetCache().LoadTable(models.TableShopSlot, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.BuyItem(context.Background(), 1, 9903, 2)
	if err != nil {
		t.Fatal(err)
	}
	if chars.items[1].DogMedal != 96 || chars.items[1].AchillesMedal != 44 || res.NewMoney != 96 || res.Balances["dogM"] != 96 || res.Balances["cbM"] != 44 {
		t.Fatalf("special currency display is stale: %v", res.Balances)
	}
}

func TestGoldPurchaseCreditsVIPOnlyAfterSuccessfulCommit(t *testing.T) {
	for _, bound := range []bool{false, true} {
		svc, chars, _ := safePurchaseFixture(t)
		future := time.Now().Add(time.Hour)
		chars.items[1].VIPType = 3
		chars.items[1].VIPExpiresAt = &future
		chars.items[1].PMExp = 47000
		slot := 9901
		want := int64(20)
		if bound {
			slot = 9902
			want = 0
		}
		res, err := svc.BuySystemItem(context.Background(), 1, slot, 2)
		if err != nil {
			t.Fatal(err)
		}
		if res.PMExpDelta != want || chars.items[1].PMExp != 47000+want {
			t.Fatalf("wrong VIP reward: %+v", res)
		}
	}
	service, chars, items := safePurchaseFixture(t)
	future := time.Now().Add(time.Hour)
	chars.items[1].VIPType = 3
	chars.items[1].VIPExpiresAt = &future
	chars.items[1].PMExp = 47000
	chars.failUpdate = true
	if _, err := service.BuySystemItem(context.Background(), 1, 9901, 2); err == nil {
		t.Fatal("expected save failure")
	}
	if chars.items[1].PMExp != 47000 || len(items.items) != 0 {
		t.Fatal("failed purchase granted VIP EXP")
	}
}
