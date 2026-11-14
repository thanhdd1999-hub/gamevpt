package shop

import "mcgame-server/internal/infrastructure/rtmp"

func (h *Handler) GetLimitTimeShop(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if catalog, ok := h.shopService.(interface{ GetLimitTimeCatalog() map[string]interface{} }); ok {
		return catalog.GetLimitTimeCatalog(), nil
	}
	return map[string]interface{}{}, nil
}

func (h *Handler) BuyLimitTimeItem(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}
	id := parseIntVal(args[0])
	if id != 9003012 && id != 9003755 {
		return nil, nil
	}
	return h.BuySystemItemClient(ctx, args)
}
