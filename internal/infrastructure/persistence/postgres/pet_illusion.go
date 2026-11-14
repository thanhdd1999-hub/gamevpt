package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *PetRepository) CommitPetIllusionAttempt(ctx context.Context, p *pet.Pet, sub int64, it *item.Item, count int) error {
	pj, err := json.Marshal(p)
	if err != nil {
		return err
	}
	ij, err := json.Marshal(it)
	if err != nil {
		return err
	}
	var done bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_pet_illusion_attempt($1,$2::jsonb,$3,$4::jsonb,$5)", p.CharacterID, pj, sub, ij, count).Scan(&done); err != nil {
		return err
	}
	if !done {
		return pkgerrors.ErrSystemError
	}
	return nil
}
