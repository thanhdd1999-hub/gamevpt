// Open-sourced by BaoLT

// Purchase bag sync helpers.
package shop

import (
	"context"
	"sort"
	"time"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

func (s *shopService) grantPurchase(ctx context.Context, char *character.Character, templateID int, itemType domainitem.ItemType, count int, bound bool, color int) (*domainitem.Item, []map[string]interface{}, error) {
	changed, err := s.itemService.GrantBagRewards(ctx, char.ID, []appitem.BagReward{{TemplateID: templateID, ItemType: itemType, Count: count, IsBound: bound, ColorCode: color}}, func() error {
		return s.charRepo.Update(ctx, char)
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(changed, func(i, j int) bool { return changed[i].SlotIndex < changed[j].SlotIndex })
	return changed[len(changed)-1], s.itemService.BuildClientItemDTOList(changed), nil
}

func newCartBagReward(id int, kind domainitem.ItemType, count int, bound bool, color int) appitem.BagReward {
	return appitem.BagReward{TemplateID: id, ItemType: kind, Count: count, IsBound: bound, ColorCode: color}
}

func shopBalances(char *character.Character, specialIDs ...int) map[string]interface{} {
	balances := map[string]interface{}{"money": char.Money, "moneyBind": char.MoneyBind, "gold": char.Gold, "goldBind": char.GoldBind, "honor": char.Honor, "exPoint": char.ExPoint, "pmExp": char.PMExp, "pmLevel": char.CurrentPMLevel(time.Now())}
	for _, id := range specialIDs {
		if id <= 0 {
			continue
		}
		descriptor, known := character.GameCurrencyDescriptor(id)
		value, readable := char.GetGameCurrencyTotal(id)
		if known && readable {
			balances[descriptor.ClientKey] = value
		}
	}
	return balances
}

func (s *shopService) snapshotPurchaseBagStacks(ctx context.Context, characterID int64, templateID int, itemType domainitem.ItemType, isBound bool) (map[int64]int, error) {
	items, err := s.itemService.GetAllItems(ctx, characterID)
	if err != nil {
		return nil, err
	}

	stacks := make(map[int64]int)
	for _, it := range items {
		if it.SlotType != domainitem.SlotTypeBag {
			continue
		}
		if it.TemplateID != templateID || it.ItemType != itemType || it.IsBound != isBound {
			continue
		}
		stacks[it.ID] = it.StackCount
	}

	return stacks, nil
}

func (s *shopService) changedPurchaseBagDTOs(ctx context.Context, characterID int64, templateID int, itemType domainitem.ItemType, isBound bool, before map[int64]int) ([]map[string]interface{}, error) {
	items, err := s.itemService.GetAllItems(ctx, characterID)
	if err != nil {
		return nil, err
	}

	changed := make([]*domainitem.Item, 0)
	for _, it := range items {
		if it.SlotType != domainitem.SlotTypeBag {
			continue
		}
		if it.TemplateID != templateID || it.ItemType != itemType || it.IsBound != isBound {
			continue
		}

		previousStack, existed := before[it.ID]
		if existed && previousStack == it.StackCount {
			continue
		}

		changed = append(changed, it)
	}

	sort.Slice(changed, func(i, j int) bool {
		if changed[i].SlotIndex != changed[j].SlotIndex {
			return changed[i].SlotIndex < changed[j].SlotIndex
		}
		return changed[i].ID < changed[j].ID
	})

	return s.itemService.BuildClientItemDTOList(changed), nil
}
