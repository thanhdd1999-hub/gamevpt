package character

import (
	"testing"
	"time"
)

func TestSystemGoldSpendRequiresActiveVIP(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		kind   int
		expiry *time.Time
		cost   int64
		want   int64
	}{{3, &future, 20, 20}, {0, &future, 20, 0}, {3, &past, 20, 0}, {3, &future, 0, 0}, {3, &future, -1, 0}} {
		char := &Character{VIPType: tc.kind, VIPExpiresAt: tc.expiry, PMExp: 47000}
		delta := char.AwardSystemGoldSpend(tc.cost)
		if delta != tc.want || char.PMExp != 47000+tc.want {
			t.Fatalf("delta=%d exp=%d want delta%d", delta, char.PMExp, tc.want)
		}
	}
}
