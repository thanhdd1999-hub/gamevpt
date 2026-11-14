package item

import (
	"context"
	"errors"
	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	"testing"
)

type starAttemptRepo struct {
	*itemServiceTestRepo
	failEquipment, failConsume bool
}

func (r *starAttemptRepo) Update(ctx context.Context, it *domainitem.Item) error {
	if r.failEquipment {
		return errors.New("save equipment failed")
	}
	return r.itemServiceTestRepo.Update(ctx, it)
}
func (r *starAttemptRepo) UpdateStack(ctx context.Context, id int64, count int) error {
	if r.failConsume {
		return errors.New("consume failed")
	}
	return r.itemServiceTestRepo.UpdateStack(ctx, id, count)
}
func TestStarAttemptConsumesOnRollFailureButNotTechnicalFailure(t *testing.T) {
	for _, mode := range []string{"success", "random failure", "equipment error", "consume error"} {
		t.Run(mode, func(t *testing.T) {
			r := &starAttemptRepo{itemServiceTestRepo: newItemServiceTestRepo(), failEquipment: mode == "equipment error", failConsume: mode == "consume error"}
			r.items[1] = &domainitem.Item{ID: 1, CharacterID: 7, ItemType: domainitem.ItemTypeEquipment, StarLevel: 2, Properties: map[string]interface{}{"upgradeNum": 2}}
			r.items[2] = &domainitem.Item{ID: 2, CharacterID: 7, TemplateID: 5, StackCount: 10}
			s := NewService(r, zap.NewNop())
			next := 3
			if mode == "random failure" {
				next = 2
			}
			_, _, _, err := s.CommitStarAttempt(context.Background(), 7, 1, 2, 5, next)
			if mode == "equipment error" || mode == "consume error" {
				if err == nil || r.items[1].StarLevel != 2 || r.items[2].StackCount != 10 {
					t.Fatalf("technical failure changed state: %v", err)
				}
			} else if err != nil || r.items[1].StarLevel != next || r.items[2].StackCount != 5 {
				t.Fatalf("bad committed attempt: %v", err)
			}
		})
	}
}
