package cache

import (
	"context"
	"errors"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"testing"
)

type evolutionCommitDelegate struct {
	*mockPetRepo
	err error
}

func (r *evolutionCommitDelegate) CommitPetEvolutionAction(context.Context, *pet.Pet, *character.Character, []*item.Item, []int, int, int64) error {
	return r.err
}

func TestPetEvolutionCacheCommitsWalletPetAndItemsTogether(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, chars, _, _, pets, _ := newTestPlayerCache(t)
		chars.characters[1] = &character.Character{ID: 1, NpPoint: 1000, Gold: 5000}
		data, err := c.LoadPlayer(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		data.SetPet(&pet.Pet{ID: 1, CharacterID: 1, TemplateID: 2281})
		stone := &item.Item{ID: 11, CharacterID: 1, TemplateID: 3911, StackCount: 3, SlotType: item.SlotTypeBag}
		data.SetItem(stone)
		delegate := &evolutionCommitDelegate{mockPetRepo: pets}
		if fail {
			delegate.err = errors.New("commit failed")
		}
		repo := NewCachedPetRepository(c, delegate)
		next := &pet.Pet{ID: 1, CharacterID: 1, TemplateID: 2282}
		char := &character.Character{ID: 1, NpPoint: 950, Gold: 4900, PMProcessData: map[string]interface{}{"localNpPnt": 950}}
		err = repo.CommitPetEvolutionAction(context.Background(), next, char, []*item.Item{stone}, []int{2}, 50, 100)
		if fail {
			if err == nil || data.GetCharacter().Gold != 5000 || data.GetCharacter().NpPoint != 1000 || data.GetPet(1).TemplateID != 2281 || data.GetItem(11).StackCount != 3 {
				t.Fatal("failed transaction mutated cache")
			}
		} else {
			if err != nil || data.GetCharacter().Gold != 4900 || data.GetCharacter().NpPoint != 950 || data.GetPet(1).TemplateID != 2282 || data.GetItem(11).StackCount != 1 {
				t.Fatal("incorrect committed cache")
			}
		}
	}
}
