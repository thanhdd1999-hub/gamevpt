package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *ItemRepository) CommitWingFusion(ctx context.Context, main *item.Item, ids []int64) error {
	data, err := json.Marshal(main)
	if err != nil {
		return err
	}
	var committed bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_wing_fusion($1,$2,$3,$4::jsonb)", main.CharacterID, main.ID, ids, data).Scan(&committed); err != nil {
		return err
	}
	if !committed {
		return pkgerrors.ErrSystemError
	}
	return nil
}
