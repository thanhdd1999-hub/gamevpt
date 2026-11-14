package cache

import (
	"context"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedPetRepository) CommitPetFusion(ctx context.Context, main *pet.Pet, ids []int64) error {
	delegate, ok := r.delegate.(interface {
		CommitPetFusion(context.Context, *pet.Pet, []int64) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(main.CharacterID)
	if data != nil {
		data.mu.Lock()
		defer data.mu.Unlock()
	}
	if err := delegate.CommitPetFusion(ctx, main, ids); err != nil {
		return err
	}
	if data != nil {
		data.Pets[main.ID] = main
		delete(data.dirtyPets, main.ID)
		for _, id := range ids {
			delete(data.Pets, id)
			delete(data.dirtyPets, id)
		}
	}
	r.upsertColdPet(ctx, main)
	for _, id := range ids {
		r.deleteColdPet(ctx, main.CharacterID, id)
	}
	return nil
}
