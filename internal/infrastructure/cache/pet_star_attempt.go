package cache

import (
	"context"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedPetRepository) CommitPetStarAttempt(ctx context.Context, p *pet.Pet, items []*item.Item) error {
	delegate, ok := r.delegate.(interface {
		CommitPetStarAttempt(context.Context, *pet.Pet, []*item.Item) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(p.CharacterID)
	if data == nil {
		if err := delegate.CommitPetStarAttempt(ctx, p, items); err != nil {
			return err
		}
		if r.cache.warmLoader != nil {
			_ = r.cache.warmLoader.DeleteFromRedis(ctx, p.CharacterID)
		}
		r.upsertColdPet(ctx, p)
		return nil
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	currentPet := data.Pets[p.ID]
	if currentPet == nil || currentPet.CharacterID != p.CharacterID {
		return pkgerrors.ErrInvalidInput
	}
	for _, it := range items {
		current := data.Items[it.ID]
		if current == nil || current.CharacterID != p.CharacterID || current.StackCount != it.StackCount || current.TemplateID != it.TemplateID {
			return pkgerrors.ErrInvalidInput
		}
	}
	if err := delegate.CommitPetStarAttempt(ctx, p, items); err != nil {
		return err
	}
	data.Pets[p.ID] = p
	delete(data.dirtyPets, p.ID)
	for _, it := range items {
		remaining := *it
		remaining.StackCount--
		if remaining.StackCount == 0 {
			delete(data.Items, it.ID)
			delete(data.dirtyItems, it.ID)
			if r.cache.warmLoader != nil {
				_ = r.cache.warmLoader.UpdateItem(ctx, p.CharacterID, 0, it.SlotType, it.SlotIndex)
			}
		} else {
			data.Items[it.ID] = &remaining
			delete(data.dirtyItems, it.ID)
		}
	}
	r.upsertColdPet(ctx, p)
	return nil
}
