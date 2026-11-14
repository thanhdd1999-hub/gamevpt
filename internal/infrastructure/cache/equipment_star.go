package cache

import (
	"context"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *CachedItemRepository) CommitEquipmentStarAttempt(ctx context.Context, equipment, stone *item.Item, count int) error {
	delegate, ok := r.delegate.(interface {
		CommitEquipmentStarAttempt(context.Context, *item.Item, *item.Item, int) error
	})
	if !ok {
		return pkgerrors.ErrSystemError
	}
	data := r.cache.GetPlayer(equipment.CharacterID)
	if data == nil {
		return delegate.CommitEquipmentStarAttempt(ctx, equipment, stone, count)
	}
	data.mu.Lock()
	defer data.mu.Unlock()
	current := data.Items[stone.ID]
	if current == nil || current.StackCount != stone.StackCount || current.CharacterID != equipment.CharacterID || count <= 0 || count > current.StackCount {
		return pkgerrors.ErrInvalidInput
	}
	if err := delegate.CommitEquipmentStarAttempt(ctx, equipment, stone, count); err != nil {
		return err
	}
	data.Items[equipment.ID] = equipment
	delete(data.dirtyItems, equipment.ID)
	remaining := *stone
	remaining.StackCount -= count
	if remaining.StackCount == 0 {
		delete(data.Items, stone.ID)
		delete(data.dirtyItems, stone.ID)
		if r.cache.warmLoader != nil {
			_ = r.cache.warmLoader.UpdateItem(ctx, equipment.CharacterID, 0, stone.SlotType, stone.SlotIndex)
		}
	} else {
		data.Items[stone.ID] = &remaining
		delete(data.dirtyItems, stone.ID)
	}
	return nil
}
