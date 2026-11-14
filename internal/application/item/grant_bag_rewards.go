package item

import (
	"context"
	"errors"

	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

type BagReward = bagRewardSpec

func (s *Service) GrantBagRewards(ctx context.Context, charID int64, rewards []BagReward, commit func() error) ([]*domainitem.Item, error) {
	for _, reward := range rewards {
		if reward.TemplateID <= 0 || reward.Count <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
	}
	if err := s.ensureBagCapacityForRewards(ctx, charID, rewards); err != nil {
		return nil, err
	}
	before, err := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, domainitem.SlotTypeBag)
	if err != nil {
		return nil, err
	}
	snapshot := make(map[int64]domainitem.Item, len(before))
	for _, it := range before {
		snapshot[it.ID] = *it
	}
	rollback := func(cause error) ([]*domainitem.Item, error) {
		current, loadErr := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, domainitem.SlotTypeBag)
		if loadErr != nil {
			return nil, errors.Join(cause, loadErr)
		}
		for _, it := range current {
			if _, existed := snapshot[it.ID]; !existed {
				cause = errors.Join(cause, s.itemRepo.Delete(ctx, it.ID))
			}
		}
		for _, original := range snapshot {
			copy := original
			cause = errors.Join(cause, s.itemRepo.Update(ctx, &copy))
		}
		return nil, cause
	}
	for _, reward := range rewards {
		if _, err := s.AddItemWithBindAndColor(ctx, charID, reward.TemplateID, reward.ItemType, reward.Count, reward.IsBound, reward.ColorCode); err != nil {
			return rollback(err)
		}
	}
	after, err := s.itemRepo.FindByCharacterAndSlotType(ctx, charID, domainitem.SlotTypeBag)
	if err != nil {
		return rollback(err)
	}
	if commit != nil {
		if err := commit(); err != nil {
			return rollback(err)
		}
	}
	changed := make([]*domainitem.Item, 0)
	for _, it := range after {
		old, existed := snapshot[it.ID]
		if !existed || old.StackCount != it.StackCount {
			changed = append(changed, it)
		}
	}
	return changed, nil
}
