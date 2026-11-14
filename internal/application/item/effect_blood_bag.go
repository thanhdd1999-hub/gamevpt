package item

import (
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

func bloodBagItem(templateID int) (int, int, bool) {
	switch templateID {
	case 2846:
		return 1, 50000, true
	case 2847:
		return 0, 50000, true
	case 2037:
		return 1, 1000000, true
	case 2038:
		return 0, 1000000, true
	}
	return 0, 0, false
}

func applyBloodBag(char *character.Character, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, bool) {
	index, amount, ok := bloodBagItem(int(tpl.ID))
	if !ok {
		return nil, false
	}
	result := NewEffectResult()
	values := char.BloodBag()
	if amount > character.BloodBagCapacity-values[index] {
		result.ConsumeItem = false
		result.Message = "Túi cứu thương không đủ chỗ chứa; không tiêu hao vật phẩm."
		return result, true
	}
	values[index] += amount
	raw := char.SetBloodBag(values)
	result.CharacterUpdates = map[string]interface{}{"ep": raw}
	result.Message = "Đã bổ sung túi cứu thương."
	return result, true
}
