// Open-sourced by BaoLT

package item

import (
	"context"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type skillBookEffectHandler struct{}

func (h *skillBookEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.Type) == 505
}

func (h *skillBookEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	if uctx == nil || !uctx.BookPrepared {
		return nil, pkgerrors.ErrCannotUseItem
	}
	return NewEffectResult(), nil
}
