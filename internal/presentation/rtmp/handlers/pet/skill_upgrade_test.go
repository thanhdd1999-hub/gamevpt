package pet

import (
	"errors"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
	"testing"
)

func TestPetUpSkillRejectsMalformedRequests(t *testing.T) {
	h := &Handler{}
	for _, tc := range []struct {
		name string
		ctx  *rtmp.RPCContext
		args []interface{}
		want error
	}{
		{"missing arguments", &rtmp.RPCContext{}, nil, pkgerrors.ErrInvalidArgs},
		{"missing character", &rtmp.RPCContext{}, []interface{}{float64(101), float64(1)}, pkgerrors.ErrUnauthorized},
		{"invalid pet id", &rtmp.RPCContext{CharacterID: "1"}, []interface{}{false, float64(1)}, pkgerrors.ErrInvalidArgs},
		{"invalid slot", &rtmp.RPCContext{CharacterID: "1"}, []interface{}{float64(101), false}, pkgerrors.ErrInvalidArgs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.PetUpSkill(tc.ctx, tc.args)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
		})
	}
}
