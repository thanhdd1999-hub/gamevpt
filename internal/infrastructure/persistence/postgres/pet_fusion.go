package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *PetRepository) CommitPetFusion(ctx context.Context, main *pet.Pet, ids []int64) error {
	data, err := json.Marshal(main)
	if err != nil {
		return err
	}
	var committed bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_pet_fusion($1,$2,$3,$4::jsonb)", main.CharacterID, main.ID, ids, data).Scan(&committed); err != nil {
		return err
	}
	if !committed {
		return pkgerrors.ErrSystemError
	}
	return nil
}
