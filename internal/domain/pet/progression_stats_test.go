package pet

import (
	"math"
	"testing"
)

func TestRerolledAptitudeChangesCombatStats(t *testing.T) {
	p := &Pet{Level: 1, GrowRate: 1.5, AptStrength: 1000, AptStamina: 1000, CreatureData: map[string]interface{}{"attStrength": 30, "attStamina": 30, "aptStrength": 1000, "aptStamina": 1000}}
	p.RecalculateStats()
	oldHP := p.MaxHP
	oldAttack := p.GetCombatStats()["attack"]
	p.AptStrength = 2000
	p.AptStamina = 2000
	p.RecalculateStats()
	if p.MaxHP <= oldHP || p.GetCombatStats()["attack"] <= oldAttack {
		t.Fatal("saved aptitude ignored in combat formula")
	}
}

func TestHighStarGrowthUsesClientBonusWithoutChangingStoredAdd(t *testing.T) {
	p := &Pet{Level: 1, GrowRate: 1.5, GrowRateAdd: 0.08}
	for i, want := range []float64{1.68, 1.88, 1.93, 1.98} {
		p.UpgradeNum = i + 9
		if math.Abs(p.petGrowthScale()-want) > 1e-8 || p.GrowRateAdd != 0.08 {
			t.Fatal("wrong high-star effective growth")
		}
	}
}
