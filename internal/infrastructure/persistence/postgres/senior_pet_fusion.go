package postgres

import (
	"context"
	"encoding/json"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
	"sort"
)

func (r *PetRepository) CommitSeniorPetFusion(ctx context.Context, main *pet.Pet, ids []int64, cost int) error {
	items, err := NewItemRepository(r.db).FindByCharacterID(ctx, main.CharacterID)
	if err != nil {
		return err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	selected := make([]*item.Item, 0)
	needed := cost
	for _, it := range items {
		if it.TemplateID != 3014 || (it.SlotType != item.SlotTypeBag && it.SlotType != item.SlotTypePetItemBag) || it.StackCount <= 0 {
			continue
		}
		selected = append(selected, it)
		needed -= it.StackCount
		if needed <= 0 {
			break
		}
	}
	if needed > 0 {
		return pkgerrors.ErrInsufficientItems
	}
	return r.CommitSeniorPetFusionWithItems(ctx, main, ids, cost, selected)
}

func (r *PetRepository) CommitSeniorPetFusionWithItems(ctx context.Context, main *pet.Pet, ids []int64, cost int, items []*item.Item) error {
	mainJSON, err := json.Marshal(main)
	if err != nil {
		return err
	}
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return err
	}
	var committed bool
	if err := r.db.pool.QueryRow(ctx, "select player.commit_senior_pet_fusion($1,$2,$3,$4,$5::jsonb,$6::jsonb)", main.CharacterID, main.ID, ids, cost, mainJSON, itemsJSON).Scan(&committed); err != nil {
		return err
	}
	if !committed {
		return pkgerrors.ErrSystemError
	}
	return nil
}
