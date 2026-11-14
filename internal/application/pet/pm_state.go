package pet

import (
	"context"
	"time"

	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) GetSpendingPMState(ctx context.Context, charID int64) (int64, int, error) {
	if s.charRepo == nil {
		return 0, 0, pkgerrors.ErrSystemError
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return 0, 0, err
	}
	if char == nil {
		return 0, 0, pkgerrors.ErrNotFound
	}
	return char.PMExp, char.CurrentPMLevel(time.Now()), nil
}
