package character

import (
	"math"
	"time"
)

func (c *Character) AwardSystemGoldSpend(cost int64) int64 {
	if c == nil || cost <= 0 || !c.HasActiveVIP(time.Now()) {
		return 0
	}
	if cost > math.MaxInt64-c.PMExp {
		cost = math.MaxInt64 - c.PMExp
	}
	c.PMExp += cost
	return cost
}
