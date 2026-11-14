// Open-sourced by BaoLT

// Buff effect handler consumes UseType==2 items and returns a usage message.
package item

import (
	"context"

	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type buffEffectHandler struct{}

func (h *buffEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool {
	return int(tpl.UseType) == 2
}

func (h *buffEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
	return nil, pkgerrors.ErrCannotUseItem
}
