package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *PetRepository) CommitPetStarAttempt(ctx context.Context, p *pet.Pet, items []*item.Item) error {
	pj, err := json.Marshal(p)
	if err != nil {
		return err
	}
	ij, err := json.Marshal(items)
	if err != nil {
		return err
	}
	var done bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_pet_star_attempt($1,$2::jsonb,$3::jsonb)", p.CharacterID, pj, ij).Scan(&done); err != nil {
		return err
	}
	if !done {
		return pkgerrors.ErrSystemError
	}
	return nil
}
