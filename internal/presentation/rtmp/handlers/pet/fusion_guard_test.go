package pet

import (
	"testing"
	"mcgame-server/internal/infrastructure/rtmp"
)

func TestIncompleteFusionDoesNotReachMutatingService(t *testing.T) {
	h := &Handler{}
	for _, call := range []func(*rtmp.RPCContext, []interface{}) (interface{}, error){h.PetJoin, h.SeniorPetJoin} {
		result, err := call(&rtmp.RPCContext{}, []interface{}{20, 21, 24, 18, 10})
		if err != nil || result.(map[string]interface{})["flag"] != false {
			t.Fatal("incomplete fusion must reject without accessing services")
		}
	}
}
