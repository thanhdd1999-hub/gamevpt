package item

import (
	"context"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type hmlvPendantService interface {
	UseHMLVPendant(context.Context, int64, int64, int64) (*domainpet.Pet, *domainitem.Item, string, error)
}

func (s *Service) applyHMLVPendant(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item) (*ItemEffectResult, error) {
	result := NewEffectResult()
	result.ConsumeItem = false
	if uctx == nil || uctx.TargetType != 2 || uctx.TargetID <= 0 {
		result.Message = "CTRL + nhấp đôi Vạn Linh Bội để dùng cho pet đang chiến đấu (Huyền Minh Linh Vũ)."
		return result, nil
	}
	service, ok := s.petService.(hmlvPendantService)
	if !ok {
		return nil, pkgerrors.ErrSystemError
	}
	pet, source, message, err := service.UseHMLVPendant(ctx, char.ID, uctx.TargetID, it.ID)
	if err != nil {
		return nil, err
	}
	if source != nil {
		it.StackCount = source.StackCount
	}
	result.Message = message
	if pet != nil {
		result.UpdatedPets = []*domainpet.Pet{pet}
	}
	return result, nil
}
