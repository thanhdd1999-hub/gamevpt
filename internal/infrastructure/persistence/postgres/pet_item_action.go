package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *PetRepository) CommitPetItemAction(ctx context.Context, p *pet.Pet, it *item.Item) error {
	pj, err := json.Marshal(p)
	if err != nil {
		return err
	}
	ij, err := json.Marshal(it)
	if err != nil {
		return err
	}
	var done bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_pet_item_action($1,$2::jsonb,$3::jsonb)", p.CharacterID, pj, ij).Scan(&done); err != nil {
		return err
	}
	if !done {
		return pkgerrors.ErrSystemError
	}
	return nil
}
