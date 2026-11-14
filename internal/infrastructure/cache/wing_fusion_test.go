package cache

import (
	"context"
	"errors"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"testing"
)

type wingFusionDelegate struct {
	*mockItemRepo
	err   error
	calls int
}

func (r *wingFusionDelegate) CommitWingFusion(ctx context.Context, main *item.Item, ids []int64) error {
	r.calls++
	return r.err
}

func TestWingFusionCacheCommitAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, chars, items, _, _, _ := newTestPlayerCache(t)
		chars.characters[1] = &character.Character{ID: 1}
		data, err := c.LoadPlayer(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []int64{1, 2, 3, 4, 5} {
			data.SetItem(&item.Item{ID: id, CharacterID: 1, ItemType: item.ItemTypeEquipment, SlotType: item.SlotTypeBag, SlotIndex: int(id), StackCount: 1, ColorCode: 2})
		}
		delegate := &wingFusionDelegate{mockItemRepo: items}
		if fail {
			delegate.err = errors.New("commit failed")
		}
		repo := NewCachedItemRepository(c, delegate)
		main := *data.GetItem(1)
		main.ColorCode = 3
		err = repo.CommitWingFusion(context.Background(), &main, []int64{2, 3, 4, 5})
		if fail {
			if err == nil || len(data.GetItems()) != 5 || data.GetItem(1).ColorCode != 2 {
				t.Fatal("failed commit mutated cache")
			}
		} else {
			if err != nil || len(data.GetItems()) != 1 || data.GetItem(1).ColorCode != 3 {
				t.Fatal("wrong committed cache")
			}
		}
	}
}
