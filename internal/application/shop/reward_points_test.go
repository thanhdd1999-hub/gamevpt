package shop

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

func TestRewardPointPurchasesUseExPointAndPreserveHonor(t *testing.T) {
	for _, cart := range []bool{false, true} {
		svc, chars, _ := safePurchaseFixture(t)
		chars.items[1].ExPoint = 1000
		chars.items[1].Honor = 10
		raw, _ := json.Marshal(models.ShopSlotTemplate{ID: 9903, Type: 29, ItemID: 158, Point: 68})
		if err := svc.(*shopService).gameDataRec.GetCache().LoadTable(models.TableShopSlot, []json.RawMessage{raw}); err != nil {
			t.Fatal(err)
		}
		if cart {
			res, err := svc.BuySystemItemMulti(context.Background(), 1, []PurchaseRequest{{ShopSlotID: 9903, Amount: 2}})
			if err != nil {
				t.Fatal(err)
			}
			if res.Balances["exPoint"] != 864 {
				t.Fatal(res.Balances)
			}
		} else {
			res, err := svc.BuySystemItem(context.Background(), 1, 9903, 2)
			if err != nil {
				t.Fatal(err)
			}
			if res.Balances["exPoint"] != 864 {
				t.Fatal(res.Balances)
			}
		}
		if chars.items[1].ExPoint != 864 || chars.items[1].Honor != 10 {
			t.Fatal("wrong wallet charged")
		}
	}
}

func TestHonorCannotPayRewardPointPrice(t *testing.T) {
	svc, chars, items := safePurchaseFixture(t)
	chars.items[1].ExPoint = 67
	chars.items[1].Honor = 10000
	raw, _ := json.Marshal(models.ShopSlotTemplate{ID: 9903, Type: 29, ItemID: 158, Point: 68})
	if err := svc.(*shopService).gameDataRec.GetCache().LoadTable(models.TableShopSlot, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BuySystemItem(context.Background(), 1, 9903, 1); err == nil {
		t.Fatal("honor paid point cost")
	}
	if chars.items[1].ExPoint != 67 || chars.items[1].Honor != 10000 || len(items.items) != 0 {
		t.Fatal("rejected purchase changed state")
	}
}
