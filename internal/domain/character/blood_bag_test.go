package character

import (
	"encoding/json"
	"testing"
)

func TestBloodBagPersistenceAndRecovery(t *testing.T) {
	c := &Character{PMProcessData: map[string]interface{}{"other": 42}}
	c.SetBloodBag([4]int{50, 100, 0, 0})
	raw, _ := json.Marshal(c.PMProcessData)
	var loaded Character
	if err := json.Unmarshal(raw, &loaded.PMProcessData); err != nil {
		t.Fatal(err)
	}
	values := loaded.BloodBag()
	hp := 10
	if amount := RecoverFromBloodBag(&hp, 100, &values[0]); amount != 50 || hp != 60 || values[0] != 0 {
		t.Fatal("incorrect reserve debit")
	}
	if amount := RecoverFromBloodBag(&hp, 60, &values[1]); amount != 0 || values[1] != 100 {
		t.Fatal("full HP consumed reserve")
	}
	hp = 0
	if RecoverFromBloodBag(&hp, 100, &values[1]) != 0 {
		t.Fatal("must not revive dead character")
	}
	if loaded.PMProcessData["other"] != float64(42) {
		t.Fatal("unrelated state lost")
	}
}
