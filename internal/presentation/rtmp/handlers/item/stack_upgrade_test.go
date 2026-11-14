package item

import "testing"

func TestStackUpgradeRates(t *testing.T) {
	for _, tc := range []struct {
		count, colour int
		jewel         bool
		vip, want     int
	}{{2, 0, false, 0, 40}, {5, 3, false, 10, 100}, {5, 4, false, 0, 30}, {5, 4, false, 5, 35}, {2, 4, true, 20, 40}} {
		if got := stackUpgradeRate(tc.count, tc.colour, tc.jewel, tc.vip); got != tc.want {
			t.Fatalf("%+v got %d", tc, got)
		}
	}
}
