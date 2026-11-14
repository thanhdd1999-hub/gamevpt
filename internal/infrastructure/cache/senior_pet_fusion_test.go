package cache

import (
	"context"
	"errors"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"testing"
)

type seniorFusionDelegate struct {
	*mockPetRepo
	err    error
	counts []int
}

func (r *seniorFusionDelegate) CommitSeniorPetFusionWithItems(ctx context.Context, p *pet.Pet, ids []int64, cost int, items []*item.Item) error {
	for _, it := range items {
		r.counts = append(r.counts, it.StackCount)
	}
	return r.err
}

func TestSeniorFusionCacheAtomicDebitAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, chars, _, _, pets, _ := newTestPlayerCache(t)
		chars.characters[1] = &character.Character{ID: 1}
		data, err := c.LoadPlayer(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		main := &pet.Pet{ID: 1, CharacterID: 1, GrowRate: 1.5}
		data.SetPet(main)
		for _, id := range []int64{2, 3, 4, 5} {
			data.SetPet(&pet.Pet{ID: id, CharacterID: 1})
		}
		data.SetItem(&item.Item{ID: 11, CharacterID: 1, TemplateID: 3014, SlotType: item.SlotTypeBag, StackCount: 3})
		data.SetItem(&item.Item{ID: 12, CharacterID: 1, TemplateID: 3014, SlotType: item.SlotTypeBag, StackCount: 9})
		delegate := &seniorFusionDelegate{mockPetRepo: pets}
		if fail {
			delegate.err = errors.New("database commit failed")
		}
		repo := NewCachedPetRepository(c, delegate)
		next := *main
		next.GrowRate = 1.8
		err = repo.CommitSeniorPetFusion(context.Background(), &next, []int64{2, 3, 4, 5}, 8)
		if fail {
			if err == nil || len(data.GetPets()) != 5 || data.GetItem(11).StackCount != 3 || data.GetItem(12).StackCount != 9 || data.GetPet(1).GrowRate != 1.5 {
				t.Fatal("failed commit mutated cache")
			}
		} else {
			if err != nil || len(data.GetPets()) != 1 || data.GetItem(11) != nil || data.GetItem(12).StackCount != 4 || data.GetPet(1).GrowRate != 1.8 {
				t.Fatal("incorrect committed cache")
			}
		}
		if len(delegate.counts) != 2 || delegate.counts[0] != 3 || delegate.counts[1] != 9 {
			t.Fatal("cached counts not used")
		}
	}
}
