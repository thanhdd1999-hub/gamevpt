package utils

import (
	"mcgame-server/internal/infrastructure/rtmp"
	"strconv"
)

func SendPMGoldSpendUpdate(ctx *rtmp.RPCContext, delta int64, total interface{}, level interface{}) {
	if ctx == nil || ctx.Connection == nil || delta <= 0 {
		return
	}
	_ = ctx.Connection.SendCallback("addOrMinusPmExp", map[string]interface{}{"type": 1, "pnt": delta, "nExp": total, "pmLevel": level})
	id, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err == nil {
		_ = ctx.Connection.SendCallback("updateScenePmLevel", map[string]interface{}{"cid": id, "pmLevel": level})
	}
}
