package gamedata

import (
	"context"
	"go.uber.org/zap"
	apppet "mcgame-server/internal/application/pet"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"
	"testing"
)

type livePetRepo struct {
	domainpet.Repository
	p *domainpet.Pet
}

func (r livePetRepo) FindByID(context.Context, int64) (*domainpet.Pet, error) { return r.p, nil }

func TestGDC39ReturnsCompleteLiveFusionPetData(t *testing.T) {
	p := &domainpet.Pet{ID: 77, CharacterID: 1, TemplateID: 19, Name: "test", Level: 1, Property: map[string]interface{}{"state": 3}}
	h := NewHandler(nil, nil, zap.NewNop())
	h.SetPetService(apppet.NewService(livePetRepo{p: p}, zap.NewNop()))
	result, err := h.GetDataConfig(&rtmp.RPCContext{Context: context.Background(), CharacterID: "1"}, []interface{}{float64(39), float64(77)})
	if err != nil {
		t.Fatal(err)
	}
	dto := result.(map[string]interface{})["data"].(map[string]interface{})
	soul, ok := dto["soulInfo"].(map[string]interface{})
	if !ok || soul["data"] == nil {
		t.Fatal("fusion UI dereferences missing soulInfo.data")
	}
	if dto["id"] != int64(77) || dto["binded"] == nil || dto["property"] == nil {
		t.Fatal("incomplete pet DTO")
	}
}
