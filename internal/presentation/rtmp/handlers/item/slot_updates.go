package item

import "mcgame-server/internal/infrastructure/rtmp"

type slotUpdateSender interface {
	SendCallback(string, ...interface{}) error
}

func sendSlotUpdate(sender slotUpdateSender, dto map[string]interface{}) {
	if sender == nil || len(dto) == 0 {
		return
	}
	callback := "onCreateItemInstance"
	if table, ok := parseFlexibleInt64(dto["type"]); ok && table == 18 {
		callback = "onChangeEquiptInst"
	}
	_ = sender.SendCallback(callback, dto)
	_ = sender.SendCallback("onAddCharactorSlot", dto)
	if callback == "onCreateItemInstance" {
		_ = sender.SendCallback("onUpChaSlotStackNum", dto["id"], dto["stackNum"], dto["sid"])
	}
}

func (h *Handler) sendSlotUpdate(ctx *rtmp.RPCContext, dto map[string]interface{}) {
	if ctx == nil || ctx.Connection == nil {
		return
	}
	sendSlotUpdate(ctx.Connection, dto)
}
