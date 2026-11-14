// Open-sourced by BaoLT

// Shop purchase slot update helpers.
package shop

import (
	appshop "mcgame-server/internal/application/shop"
	"mcgame-server/internal/infrastructure/rtmp"
	rtmputils "mcgame-server/internal/presentation/rtmp/utils"
)

func sendPurchaseSlotUpdates(ctx *rtmp.RPCContext, res *appshop.BuyResult) {
	if res == nil {
		return
	}

	if len(res.AddedItemDTOs) > 0 {
		for _, itemDTO := range res.AddedItemDTOs {
			_ = ctx.Connection.SendCallback("onCreateItemInstance", itemDTO)
			_ = ctx.Connection.SendCallback("onAddCharactorSlot", itemDTO)
			_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", itemDTO["id"], itemDTO["stackNum"], itemDTO["sid"])
		}
		return
	}

	if len(res.AddedItemDTO) == 0 {
		return
	}

	_ = ctx.Connection.SendCallback("onCreateItemInstance", res.AddedItemDTO)
	_ = ctx.Connection.SendCallback("onAddCharactorSlot", res.AddedItemDTO)
	_ = ctx.Connection.SendCallback("onUpChaSlotStackNum", res.AddedItemDTO["id"], res.AddedItemDTO["stackNum"], res.AddedItemDTO["sid"])
}

func sendPurchaseWalletUpdates(ctx *rtmp.RPCContext, res *appshop.BuyResult) {
	if res == nil {
		return
	}
	if len(res.Balances) > 0 {
		_ = ctx.Connection.SendCallback("onUPP", res.Balances)
		rtmputils.SendPMGoldSpendUpdate(ctx, res.PMExpDelta, res.Balances["pmExp"], res.Balances["pmLevel"])
		return
	}
	key := getCurrencyKey(res.CurrencyType)
	if key != "" {
		_ = ctx.Connection.SendCallback("onUPP", map[string]interface{}{key: res.NewMoney})
	}
}
