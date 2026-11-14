package item

import "testing"

type slotUpdateRecorder struct {
	names    []string
	payloads []interface{}
}

func (r *slotUpdateRecorder) SendCallback(name string, args ...interface{}) error {
	r.names = append(r.names, name)
	r.payloads = append(r.payloads, args[0])
	return nil
}

func TestSlotUpdateHydratesBeforeDisplaying(t *testing.T) {
	r := &slotUpdateRecorder{}
	dto := map[string]interface{}{"id": int64(123), "sid": 2101, "stackNum": 2}
	sendSlotUpdate(r, dto)
	if len(r.names) != 3 || r.names[0] != "onCreateItemInstance" || r.names[1] != "onAddCharactorSlot" {
		t.Fatal(r.names)
	}
	for _, payload := range r.payloads[:2] {
		p := payload.(map[string]interface{})
		if p["id"] != int64(123) || p["stackNum"] != 2 {
			t.Fatal("stale or mismatched payload")
		}
	}
	r = &slotUpdateRecorder{}
	dto["type"] = 18
	sendSlotUpdate(r, dto)
	if r.names[0] != "onChangeEquiptInst" || r.names[1] != "onAddCharactorSlot" {
		t.Fatal(r.names)
	}
}
