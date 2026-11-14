package item

import "testing"

func TestStarRateUsesTargetStarIndex(t *testing.T) {
	for _, tc := range []struct{ star, stones, want int }{{0, 5, 100}, {2, 5, 90}, {7, 5, 30}, {8, 1, 6}, {10, 5, 0}} {
		if got := starCalcSuccessRate(tc.star, tc.stones); got != tc.want {
			t.Fatalf("star %d stones %d: %d want %d", tc.star, tc.stones, got, tc.want)
		}
	}
}

func TestWingStarFailurePenalties(t *testing.T) {
	for current, want := range []int{0, 0, 1, 2, 3, 4, 4, 5, 0, 0} {
		if got := wingStarFailureLevel(current); got != want {
			t.Fatalf("star %d: %d want %d", current, got, want)
		}
	}
}
