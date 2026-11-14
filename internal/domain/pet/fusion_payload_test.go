package pet

import "testing"

func TestPetWrapperRetainsSoulInfoWhenFlashFlattensData(t *testing.T) {
	p := &Pet{ID: 7, Level: 1, Property: map[string]interface{}{}}
	wrapper := p.ToDTO()
	for key, value := range wrapper["data"].(map[string]interface{}) {
		if key != "pi" && key != "soulInfo" {
			wrapper[key] = value
		}
	}
	delete(wrapper, "data")
	soul, ok := wrapper["soulInfo"].(map[string]interface{})
	if !ok || soul["data"] == nil {
		t.Fatal("Flash onAddPet/init flatten loses soulInfo and advancedJoin cannot run")
	}
	if wrapper["pi"] == nil {
		t.Fatal("Flash expects root pi")
	}
}
