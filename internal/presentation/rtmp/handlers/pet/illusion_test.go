package pet

import (
	"testing"

	apppet "mcgame-server/internal/application/pet"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"
)

func TestIllusionResponderPreservesTwoPhaseContract(t *testing.T) {
	preview := illusionResponse(&apppet.PetIllusionResult{Pet: &domainpet.Pet{ID: 1}, DonorID: 2, CostItem: 2, IsHint: true, AptType: 1, Field: "aptStrengthIllusion", Value: 150})
	if preview["succ"] != false || preview["isHint"] != true || preview["costPet"] != true || preview["costItem"] != 2 || preview["propName"] != "aptStrengthEx" {
		t.Fatalf("wrong preview response:%v", preview)
	}
	applied := illusionResponse(&apppet.PetIllusionResult{Pet: &domainpet.Pet{ID: 1}, Success: true, AptType: 1, Field: "aptStrengthIllusion", Value: 150})
	if applied["succ"] != true || applied["isHint"] != false || applied["costPet"] != false || applied["costItem"] != 0 {
		t.Fatalf("wrong apply response:%v", applied)
	}
}

func TestEnsurePetEatDoesNotSucceedForMalformedRequest(t *testing.T) {
	h := &Handler{}
	response, err := h.EnsurePetEatClient(&rtmp.RPCContext{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.(map[string]interface{})["succ"] != false {
		t.Fatal("malformed confirmation succeeded")
	}
}
