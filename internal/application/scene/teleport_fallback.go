package scene

import (
	"strings"

	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func (s *Service) resolveUnlinkedGate(source *models.SceneitemInstanceTemplate) *TeleportDestination {
	if source == nil || source.Tid != 38 || strings.TrimSpace(source.Name) == "" {
		return nil
	}
	var target *models.MapTemplate
	for _, candidate := range s.gameDataManager.GetAllMaps() {
		if candidate.Name != source.Name {
			continue
		}
		if target != nil {
			return nil
		}
		target = candidate
	}
	if target == nil || target.ID == int64(source.PosMapID) || target.CopyFlag != 0 || target.T <= 0 {
		return nil
	}
	if target.SafeX <= 0 || target.SafeY <= 0 || target.SafeX >= target.Width || target.SafeY >= target.Height {
		return nil
	}
	s.logger.Info("Resolved unlinked portal to named map safe point",
		zap.Int64("source_gate_id", source.ID),
		zap.Int64("destination_map_id", target.ID))
	return &TeleportDestination{MapID: int(target.ID), X: int(target.SafeX), Y: int(target.SafeY)}
}
