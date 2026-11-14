package cache

import (
	"context"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedItemRepository) CommitWingFusion(ctx context.Context, main *item.Item, ids []int64) error {
	if main == nil || main.ID <= 0 || main.CharacterID <= 0 || len(ids) < 4 || len(ids) > 16 {
		return pkgerrors.ErrInvalidInput
	}
	seen := map[int64]bool{main.ID: true}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return pkgerrors.ErrInvalidInput
		}
		seen[id] = true
	}
	delegate, ok := r.delegate.(interface {
		CommitWingFusion(context.Context, *item.Item, []int64) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(main.CharacterID)
	if data == nil {
		if err := delegate.CommitWingFusion(ctx, main, ids); err != nil {
			return err
		}
		if r.cache.warmLoader != nil {
			_ = r.cache.warmLoader.DeleteFromRedis(ctx, main.CharacterID)
		}
		return nil
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	for id := range seen {
		it := data.Items[id]
		if it == nil || it.CharacterID != main.CharacterID || it.ItemType != item.ItemTypeEquipment || it.SlotType != item.SlotTypeBag || it.StackCount != 1 {
			return pkgerrors.ErrInvalidInput
		}
	}
	if err := delegate.CommitWingFusion(ctx, main, ids); err != nil {
		return err
	}
	if r.cache.warmLoader != nil {
		for _, id := range ids {
			it := data.Items[id]
			_ = r.cache.warmLoader.UpdateItem(ctx, main.CharacterID, 0, it.SlotType, it.SlotIndex)
		}
	}
	for _, id := range ids {
		delete(data.Items, id)
		delete(data.dirtyItems, id)
	}
	data.Items[main.ID] = main
	delete(data.dirtyItems, main.ID)
	if r.cache.warmLoader != nil {
		_ = r.cache.warmLoader.UpdateItem(ctx, main.CharacterID, main.ID, main.SlotType, main.SlotIndex)
	}
	return nil
}
