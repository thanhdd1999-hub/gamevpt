package character

import (
	"fmt"
	"strconv"
	"strings"
)

const BloodBagCapacity = 10000000

func (c *Character) BloodBag() [4]int {
	var values [4]int
	if c == nil {
		return values
	}
	raw, _ := c.PMProcessData["localBloodBag"].(string)
	parts := strings.Split(raw, "|")
	if len(parts) != 5 {
		return values
	}
	for i := range values {
		value, err := strconv.Atoi(parts[i+1])
		if err == nil && value > 0 {
			if value > BloodBagCapacity {
				value = BloodBagCapacity
			}
			values[i] = value
		}
	}
	return values
}

func (c *Character) SetBloodBag(values [4]int) string {
	state := make(map[string]interface{}, len(c.PMProcessData)+1)
	for key, value := range c.PMProcessData {
		state[key] = value
	}
	raw := fmt.Sprintf("|%d|%d|%d|%d", values[0], values[1], values[2], values[3])
	state["localBloodBag"] = raw
	c.PMProcessData = state
	return raw
}

func (c *Character) BloodBagString() string {
	v := c.BloodBag()
	return fmt.Sprintf("|%d|%d|%d|%d", v[0], v[1], v[2], v[3])
}

func RecoverFromBloodBag(current *int, maximum int, reserve *int) int {
	if *current <= 0 || maximum <= *current || *reserve <= 0 {
		return 0
	}
	amount := maximum - *current
	if amount > *reserve {
		amount = *reserve
	}
	*current += amount
	*reserve -= amount
	return amount
}
