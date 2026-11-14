package shop

import (
	"strconv"
	"time"
)

func (s *shopService) GetLimitTimeCatalog() map[string]interface{} {
	result := map[string]interface{}{}
	if s.gameDataRec == nil {
		return result
	}
	now := time.Now()
	for i, id := range []int{9003012, 9003755} {
		slot := s.gameDataRec.GetShopSlot(id)
		if slot == nil {
			continue
		}
		result[strconv.Itoa(i+1)] = map[string]interface{}{"id": id, "tid": 29, "itemId": int(slot.ItemID), "gold": slot.Gold, "money": 0, "point": 0, "q": slot.Quality, "bind": 0, "num": -1, "remain": -1, "limitNu": -1, "flag": 0, "startTime": now.Add(-24 * time.Hour).Unix(), "endTime": now.AddDate(10, 0, 0).Unix()}
	}
	return result
}
