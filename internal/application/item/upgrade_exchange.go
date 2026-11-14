package item

import (
	"context"
	"errors"
	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
	"sync"
)

var upgradeExchangeMu sync.Mutex

type UpgradeExchangeResult struct {
	Source   *domainitem.Item
	Deleted  bool
	Products []*domainitem.Item
}

func (s *Service) ExchangeUpgradeItems(ctx context.Context, charID, sourceID int64, count int, rewards []BagReward) (*UpgradeExchangeResult, error) {
	upgradeExchangeMu.Lock()
	defer upgradeExchangeMu.Unlock()
	source, err := s.GetItemByID(ctx, charID, sourceID)
	if err != nil {
		return nil, err
	}
	if count <= 0 || source.StackCount < count || (source.SlotType != domainitem.SlotTypeBag && source.SlotType != domainitem.SlotTypeTempBag) {
		return nil, pkgerrors.ErrInvalidInput
	}
	original := *source
	moved := false
	if count == source.StackCount && source.SlotType == domainitem.SlotTypeBag && len(rewards) > 0 {
		slot, err := s.itemRepo.FindFirstEmptySlot(ctx, charID, domainitem.SlotTypeTempBag, domainitem.DefaultTempBagSlots)
		if err != nil {
			return nil, err
		}
		if err := s.itemRepo.MoveItem(ctx, sourceID, domainitem.SlotTypeTempBag, slot); err != nil {
			return nil, err
		}
		moved = true
	}
	var consumed *domainitem.Item
	var deleted bool
	products, err := s.GrantBagRewards(ctx, charID, rewards, func() error {
		var consumeErr error
		consumed, deleted, consumeErr = s.ConsumeItemStack(ctx, charID, sourceID, count)
		return consumeErr
	})
	if err != nil {
		if moved {
			err = errors.Join(err, s.itemRepo.MoveItem(ctx, sourceID, original.SlotType, original.SlotIndex))
		}
		return nil, err
	}
	consumed.SlotType = original.SlotType
	consumed.SlotIndex = original.SlotIndex
	productUpdates := make([]*domainitem.Item, 0, len(products))
	for _, product := range products {
		if product.ID != sourceID {
			productUpdates = append(productUpdates, product)
		}
	}
	return &UpgradeExchangeResult{Source: consumed, Deleted: deleted, Products: productUpdates}, nil
}
