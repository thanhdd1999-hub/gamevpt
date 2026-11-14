package cache

import (
	"context"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedPetRepository) CommitPetEvolutionAction(ctx context.Context, p *pet.Pet, char *character.Character, items []*item.Item, counts []int, npCost int, goldCost int64) error {
	if p == nil || char == nil || p.CharacterID != char.ID || len(items) != len(counts) || npCost < 0 || goldCost < 0 || char.Gold < 0 || char.NpPoint < 0 {
		return pkgerrors.ErrInvalidInput
	}
	delegate, ok := r.delegate.(interface {
		CommitPetEvolutionAction(context.Context, *pet.Pet, *character.Character, []*item.Item, []int, int, int64) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(char.ID)
	if data == nil {
		return pkgerrors.ErrSystemError
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	if data.Character == nil || data.Character.Gold != char.Gold+goldCost || data.Character.NpPoint != char.NpPoint+npCost || data.Pets[p.ID] == nil {
		return pkgerrors.ErrInvalidInput
	}
	for i, it := range items {
		current := data.Items[it.ID]
		if current == nil || current.CharacterID != char.ID || current.StackCount != it.StackCount || current.TemplateID != 3911 || counts[i] < 1 || counts[i] > it.StackCount {
			return pkgerrors.ErrInvalidInput
		}
	}
	if err := delegate.CommitPetEvolutionAction(ctx, p, char, items, counts, npCost, goldCost); err != nil {
		return err
	}
	data.Character = char
	data.dirtyChar = true
	data.Pets[p.ID] = p
	delete(data.dirtyPets, p.ID)
	for i, it := range items {
		remaining := *it
		remaining.StackCount -= counts[i]
		if remaining.StackCount == 0 {
			delete(data.Items, it.ID)
			delete(data.dirtyItems, it.ID)
			if r.cache.warmLoader != nil {
				_ = r.cache.warmLoader.UpdateItem(ctx, char.ID, 0, it.SlotType, it.SlotIndex)
			}
		} else {
			data.Items[it.ID] = &remaining
			delete(data.dirtyItems, it.ID)
		}
	}
	r.upsertColdPet(ctx, p)
	return nil
}
