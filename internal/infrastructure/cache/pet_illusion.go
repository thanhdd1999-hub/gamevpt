package cache

import (
	"context"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedPetRepository) CommitPetIllusionAttempt(ctx context.Context, p *pet.Pet, sub int64, it *item.Item, count int) error {
	delegate, ok := r.delegate.(interface {
		CommitPetIllusionAttempt(context.Context, *pet.Pet, int64, *item.Item, int) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(p.CharacterID)
	if data == nil {
		if err := delegate.CommitPetIllusionAttempt(ctx, p, sub, it, count); err != nil {
			return err
		}
		if r.cache.warmLoader != nil {
			_ = r.cache.warmLoader.DeleteFromRedis(ctx, p.CharacterID)
		}
		r.deleteColdPet(ctx, p.CharacterID, sub)
		r.upsertColdPet(ctx, p)
		return nil
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	currentPet := data.Pets[p.ID]
	donor := data.Pets[sub]
	if currentPet == nil || donor == nil || sub == p.ID || donor.CharacterID != p.CharacterID {
		return pkgerrors.ErrInvalidInput
	}
	if count > 0 && it == nil {
		return pkgerrors.ErrInvalidInput
	}
	if count > 0 {
		current := data.Items[it.ID]
		if current == nil || current.StackCount != it.StackCount || current.TemplateID != it.TemplateID || count > it.StackCount {
			return pkgerrors.ErrInvalidInput
		}
	}
	if err := delegate.CommitPetIllusionAttempt(ctx, p, sub, it, count); err != nil {
		return err
	}
	data.Pets[p.ID] = p
	delete(data.dirtyPets, p.ID)
	delete(data.Pets, sub)
	delete(data.dirtyPets, sub)
	r.deleteColdPet(ctx, p.CharacterID, sub)
	if count > 0 && it != nil {
		remaining := *it
		remaining.StackCount -= count
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
