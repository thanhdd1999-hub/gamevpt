package item

import "mcgame-server/internal/infrastructure/rtmp"

func (h *Handler) JewelUpdateOne(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runStackUpgrade(ctx, args, true, false)
}
func (h *Handler) JewelUpdateAll(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	return h.runStackUpgrade(ctx, args, true, true)
}
