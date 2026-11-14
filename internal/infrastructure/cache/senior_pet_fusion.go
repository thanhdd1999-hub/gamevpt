package cache

import (
	"context"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
	"sort"
)

func seniorFusionItems(items map[int64]*item.Item, charID int64, cost int) ([]*item.Item, error) {
	if cost <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	candidates := make([]*item.Item, 0)
	for _, it := range items {
		if it != nil && it.CharacterID == charID && it.TemplateID == 3014 && it.StackCount > 0 && (it.SlotType == item.SlotTypeBag || it.SlotType == item.SlotTypePetItemBag) {
			copy := *it
			candidates = append(candidates, &copy)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	needed := cost
	for i, it := range candidates {
		needed -= it.StackCount
		if needed <= 0 {
			return candidates[:i+1], nil
		}
	}
	return nil, pkgerrors.ErrInsufficientItems
}

func (r *CachedPetRepository) CommitSeniorPetFusion(ctx context.Context, main *pet.Pet, ids []int64, cost int) error {
	data := r.cache.GetPlayer(main.CharacterID)
	if data == nil {
		delegate, ok := r.delegate.(interface {
			CommitSeniorPetFusion(context.Context, *pet.Pet, []int64, int) error
		})
		if !ok {
			return pkgerrors.ErrSystemError
		}
		if err := delegate.CommitSeniorPetFusion(ctx, main, ids, cost); err != nil {
			return err
		}
		if r.cache.warmLoader != nil {
			_ = r.cache.warmLoader.DeleteFromRedis(ctx, main.CharacterID)
		}
		r.upsertColdPet(ctx, main)
		for _, id := range ids {
			r.deleteColdPet(ctx, main.CharacterID, id)
		}
		return nil
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	selected, err := seniorFusionItems(data.Items, main.CharacterID, cost)
	if err != nil {
		return err
	}
	delegate, ok := r.delegate.(interface {
		CommitSeniorPetFusionWithItems(context.Context, *pet.Pet, []int64, int, []*item.Item) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	if err := delegate.CommitSeniorPetFusionWithItems(ctx, main, ids, cost, selected); err != nil {
		return err
	}
	data.Pets[main.ID] = main
	delete(data.dirtyPets, main.ID)
	for _, id := range ids {
		delete(data.Pets, id)
		delete(data.dirtyPets, id)
	}
	needed := cost
	for _, it := range selected {
		take := it.StackCount
		if take > needed {
			take = needed
		}
		it.StackCount -= take
		needed -= take
		if it.StackCount == 0 {
			delete(data.Items, it.ID)
			delete(data.dirtyItems, it.ID)
			if r.cache.warmLoader != nil {
				_ = r.cache.warmLoader.UpdateItem(ctx, main.CharacterID, 0, it.SlotType, it.SlotIndex)
			}
		} else {
			data.Items[it.ID] = it
		}
	}
	r.upsertColdPet(ctx, main)
	for _, id := range ids {
		r.deleteColdPet(ctx, main.CharacterID, id)
	}
	return nil
}
