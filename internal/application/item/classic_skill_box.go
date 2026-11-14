package item

import (
	"mcgame-server/internal/gamedata"
	pkgerrors "mcgame-server/pkg/errors"
)

func classicSkillBox(gameData *gamedata.Manager) (*ItemEffectResult, error) {
	if gameData == nil {
		return nil, pkgerrors.ErrCannotUseItem
	}
	choices := make([]*GiftBoxChoice, 0, 5)
	for _, id := range []int{3467, 3468, 3469, 3466, 2709} {
		tpl := gameData.GetItem(id)
		if tpl == nil || int(tpl.Type) != 506 {
			return nil, pkgerrors.ErrCannotUseItem
		}
		choices = append(choices, &GiftBoxChoice{TableType: 29, ID: id, StackNum: 1})
	}
	return &ItemEffectResult{ConsumeItem: false, GiftBox: &GiftBoxData{Choices: choices, GetNum: 1}}, nil
}
