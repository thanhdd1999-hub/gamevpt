package item

import (
	"context"
	"errors"
	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
	"sync"
)

var starAttemptMu sync.Mutex

func (s *Service) CommitStarAttempt(ctx context.Context, charID, equipmentID, stoneID int64, count, star int) (*domainitem.Item, *domainitem.Item, bool, error) {
	starAttemptMu.Lock()
	defer starAttemptMu.Unlock()
	equipment, err := s.GetItemByID(ctx, charID, equipmentID)
	if err != nil {
		return nil, nil, false, err
	}
	stone, err := s.GetItemByID(ctx, charID, stoneID)
	if err != nil {
		return nil, nil, false, err
	}
	if equipment.ItemType != domainitem.ItemTypeEquipment || (stone.TemplateID != 5 && stone.TemplateID != 2906) || count < 1 || count > 5 || stone.StackCount < count || star < 0 || star > 10 {
		return nil, nil, false, pkgerrors.ErrInvalidInput
	}
	if stone.SlotType != domainitem.SlotTypeBag && stone.SlotType != domainitem.SlotTypeTempBag && stone.SlotType != domainitem.SlotTypePetItemBag {
		return nil, nil, false, pkgerrors.ErrInvalidInput
	}
	if stone.TemplateID == 2906 {
		tpl := s.GetEquipmentTemplate(equipment.TemplateID)
		if tpl == nil || int(tpl.Kind) != 13 || int(tpl.Position) != 22 {
			return nil, nil, false, pkgerrors.ErrInvalidInput
		}
	}
	original := *equipment
	original.Properties = make(map[string]interface{}, len(equipment.Properties))
	for key, value := range equipment.Properties {
		original.Properties[key] = value
	}
	next := original
	next.Properties = make(map[string]interface{}, len(original.Properties)+1)
	for key, value := range original.Properties {
		next.Properties[key] = value
	}
	next.StarLevel = star
	next.Properties["upgradeNum"] = star
	if atomic, ok := s.itemRepo.(interface {
		CommitEquipmentStarAttempt(context.Context, *domainitem.Item, *domainitem.Item, int) error
	}); ok {
		if err := atomic.CommitEquipmentStarAttempt(ctx, &next, stone, count); err != nil {
			return nil, nil, false, err
		}
		remaining := *stone
		remaining.StackCount -= count
		return &next, &remaining, remaining.StackCount == 0, nil
	}
	if err := s.itemRepo.Update(ctx, &next); err != nil {
		return nil, nil, false, err
	}
	consumed, deleted, err := s.ConsumeItemStack(ctx, charID, stoneID, count)
	if err != nil {
		rollbackErr := s.itemRepo.Update(ctx, &original)
		return nil, nil, false, errors.Join(err, rollbackErr)
	}
	return &next, consumed, deleted, nil
}
