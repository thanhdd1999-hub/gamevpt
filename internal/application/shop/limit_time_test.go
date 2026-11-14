package shop

import (
	"encoding/json"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

func TestLimitTabCatalogHasBothBookBagsAndClientFields(t *testing.T) {
	svc, _, _ := safePurchaseFixture(t)
	var rows []json.RawMessage
	for _, slot := range []models.ShopSlotTemplate{{ID: 9003012, ItemID: 3012, Type: 29, Gold: 3000}, {ID: 9003755, ItemID: 3755, Type: 29, Gold: 3000}} {
		raw, _ := json.Marshal(slot)
		rows = append(rows, raw)
	}
	s := svc.(*shopService)
	if err := s.gameDataRec.GetCache().LoadTable(models.TableShopSlot, rows); err != nil {
		t.Fatal(err)
	}
	list := s.GetLimitTimeCatalog()
	if len(list) != 2 {
		t.Fatal("Limit tab empty")
	}
	for _, entry := range list {
		row := entry.(map[string]interface{})
		if row["tid"] != 29 || row["gold"] != float64(3000) || row["itemId"] == nil || row["remain"] != -1 {
			t.Fatal(row)
		}
		if row["startTime"].(int64) > 100000000000 {
			t.Fatal("Flash expects Unix seconds")
		}
	}
}
