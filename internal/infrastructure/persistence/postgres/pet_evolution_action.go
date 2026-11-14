package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

func (r *PetRepository) CommitPetEvolutionAction(ctx context.Context, p *pet.Pet, char *character.Character, items []*item.Item, counts []int, npCost int, goldCost int64) error {
	if len(items) != len(counts) {
		return pkgerrors.ErrInvalidInput
	}
	pj, err := json.Marshal(p)
	if err != nil {
		return err
	}
	cj, err := json.Marshal(char)
	if err != nil {
		return err
	}
	charges := make([]map[string]interface{}, 0, len(items))
	for i, it := range items {
		charges = append(charges, map[string]interface{}{"Item": it, "Count": counts[i]})
	}
	ij, err := json.Marshal(charges)
	if err != nil {
		return err
	}
	var done bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_pet_evolution_action($1,$2::jsonb,$3::jsonb,$4::jsonb,$5,$6)", char.ID, pj, cj, ij, npCost, goldCost).Scan(&done); err != nil {
		return err
	}
	if !done {
		return pkgerrors.ErrSystemError
	}
	return nil
}
