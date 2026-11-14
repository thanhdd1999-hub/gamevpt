// Open-sourced by BaoLT

// VIP-shop-specific item purchase: gold-only (currencyType=4), single quantity,
// server applies the VIP-tier discount. Bound gold (Gt=1) and PType1/2 special
// currencies are intentionally not supported — VIP slots are always priced
// from TBL_SHOP_SLOT.gold and paid from char.Gold.
package shop

import (
	"context"
	"errors"
	"math"
	"time"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	pkgitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

const vipShopCurrencyType = 4

func (s *shopService) buyVIPSlotItem(
	ctx context.Context,
	char *character.Character,
	slot *models.ShopSlotTemplate,
	amount int,
	now time.Time,
	rotationEpoch int64,
) (*BuyResult, error) {
	if s.itemService == nil {
		return nil, errors.New("item service not initialized")
	}

	itType, itemName, itemKind, itemTableType, err := s.resolveVIPSlotItemMetadata(slot)
	if err != nil {
		return nil, err
	}

	if amount != 1 || slot == nil || math.IsNaN(slot.Gold) || math.IsInf(slot.Gold, 0) || slot.Gold != math.Trunc(slot.Gold) || slot.Gold > float64(math.MaxInt64/100) {
		return nil, errors.New("invalid vip shop purchase price or quantity")
	}
	pmLevel := char.CurrentPMLevel(now)
	pricePerItem := applyDiscount(int64(slot.Gold), pmLevel)
	if pricePerItem <= 0 {
		return nil, errors.New("vip shop item has no price defined")
	}

	cost := pricePerItem * int64(amount)
	if char.Gold < cost {
		return nil, ErrVIPShopInsufficientGold
	}
	original := *char
	next := original
	next.Gold -= cost
	pmDelta := next.AwardSystemGoldSpend(cost)

	templateID := int(slot.ItemID)

	colorCode := int(math.Ceil(slot.Quality / 5))
	changed, err := s.itemService.GrantBagRewards(ctx, char.ID, []appitem.BagReward{{
		TemplateID: templateID, ItemType: itType, Count: amount, IsBound: false, ColorCode: colorCode,
	}}, func() error {
		if err := s.charRepo.Update(ctx, &next); err != nil {
			return err
		}
		inserted, recordErr := s.vipShopRepo.RecordPurchase(ctx, char.ID, rotationEpoch, int32(slot.ID), int32(amount))
		if recordErr == nil && !inserted {
			recordErr = ErrVIPShopSlotAlreadyPurchased
		}
		if recordErr != nil {
			return errors.Join(recordErr, s.charRepo.Update(ctx, &original))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	addedItem := changed[len(changed)-1]
	addedItemDTOs := s.itemService.BuildClientItemDTOList(changed)
	if itType == pkgitem.ItemTypeEquipment {
		itemName = pkgitem.FormatEquipmentDisplayName(itemName, addedItem.ColorCode)
	}

	return &BuyResult{
		PMExpDelta:   pmDelta,
		Balances:     shopBalances(&next),
		NewMoney:     next.Gold,
		Cost:         cost,
		CurrencyType: vipShopCurrencyType,
		ItemData: map[string]any{
			"i":  addedItem.TemplateID,
			"q":  amount,
			"s":  "",
			"c":  pkgitem.PopupNoticeColor(itType, addedItem.ColorCode),
			"t":  itemTableType,
			"n":  itemName,
			"tt": itemKind,
		},
		AddedItemDTO:  s.itemService.BuildClientItemDTO(addedItem),
		AddedItemDTOs: addedItemDTOs,
	}, nil
}

func (s *shopService) resolveVIPSlotItemMetadata(slot *models.ShopSlotTemplate) (pkgitem.ItemType, string, int, int, error) {
	if slot == nil || (int(slot.Type) != 19 && int(slot.Type) != 29) {
		return 0, "", 0, 0, errors.New("invalid vip shop item table")
	}
	if int(slot.Type) == 19 {
		equipTpl := s.gameDataRec.GetEquipment(int(slot.ItemID))
		if equipTpl == nil {
			return 0, "", 0, 0, errors.New("equipment template not found")
		}
		return pkgitem.ItemTypeEquipment, equipTpl.Name, int(equipTpl.Kind), 19, nil
	}

	itemTpl := s.gameDataRec.GetItem(int(slot.ItemID))
	if itemTpl == nil {
		return 0, "", 0, 0, errors.New("item template not found")
	}
	itType := pkgitem.ItemTypeConsumable
	if int(itemTpl.Kind) == 6 || int(itemTpl.Kind) == 14 {
		itType = pkgitem.ItemTypeMaterial
	}
	return itType, itemTpl.Name, int(itemTpl.Kind), 29, nil
}
