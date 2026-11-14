package item

import "mcgame-server/internal/infrastructure/rtmp"

func (h *Handler) MaterialMixOne(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runStackUpgrade(ctx, args, false, false)
}
func (h *Handler) MaterialMixAll(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runStackUpgrade(ctx, args, false, true)
}
