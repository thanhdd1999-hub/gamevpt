package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
)

func (r *ItemRepository) CommitEquipmentStarAttempt(ctx context.Context, equipment, stone *item.Item, count int) error {
	e, err := json.Marshal(equipment)
	if err != nil {
		return err
	}
	s, err := json.Marshal(stone)
	if err != nil {
		return err
	}
	var ok bool
	return r.db.pool.QueryRow(ctx, "select player.commit_equipment_star($1,$2::jsonb,$3::jsonb,$4)", equipment.CharacterID, e, s, count).Scan(&ok)
}
