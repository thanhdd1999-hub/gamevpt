// Open-sourced by BaoLT

package item

import (
	"strconv"

	domainpet "mcgame-server/internal/domain/pet"
)

type itemCallbackSender interface {
	SendCallback(method string, args ...interface{}) error
}

func sendUpdatedPetCallbacks(sender itemCallbackSender, pets []*domainpet.Pet) {
	if sender == nil {
		return
	}
	for _, pet := range pets {
		if pet == nil {
			continue
		}
		_ = sender.SendCallback("onUpdatePet", float64(pet.ID), "life", strconv.Itoa(pet.Life))
		hasSkills := false
		for slot := 1; slot <= domainpet.TotalPetSkillSlots; slot++ {
			key := "skill" + strconv.Itoa(slot)
			if value, exists := pet.Property[key]; exists && value != nil {
				hasSkills = true
				_ = sender.SendCallback("onUpdatePet", float64(pet.ID), key, value)
			}
		}
		if hasSkills {
			dto := pet.ToDTO()
			if data, ok := dto["data"].(map[string]interface{}); ok {
				if prop, ok := data["property"]; ok {
					_ = sender.SendCallback("onRefreshPetProp", map[string]interface{}{"id": pet.ID, "s": prop})
				}
			}
		}
	}
}
