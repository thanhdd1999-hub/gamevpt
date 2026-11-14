package item

import (
	"context"
	"errors"
	"go.uber.org/zap"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

type wingFusionTestRepo struct {
	*itemServiceTestRepo
	fail    bool
	commits int
}

func (r *wingFusionTestRepo) CommitWingFusion(ctx context.Context, main *domainitem.Item, ids []int64) error {
	if r.fail {
		return errors.New("commit failed")
	}
	r.commits++
	if err := r.Update(ctx, main); err != nil {
		return err
	}
	for _, id := range ids {
		delete(r.items, id)
	}
	return nil
}
func TestWingFusionSuccessFailureAndValidation(t *testing.T) {
	for _, mode := range []string{"success", "random failure", "commit failure", "duplicate", "wrong colour", "equipped", "senior"} {
		t.Run(mode, func(t *testing.T) {
			r := &wingFusionTestRepo{itemServiceTestRepo: newItemServiceTestRepo(), fail: mode == "commit failure"}
			for id := int64(1); id <= 5; id++ {
				r.items[id] = &domainitem.Item{ID: id, CharacterID: 7, TemplateID: 1676, ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeBag, SlotIndex: int(id), StackCount: 1, ColorCode: 3, StarLevel: 4, IsBound: id == 2, Properties: map[string]interface{}{"upgradeNum": 4}}
			}
			s := NewService(r, zap.NewNop())
			m := gamedata.NewManager(nil, zap.NewNop())
			loadEquipmentTemplate(t, m, models.EquiptTemplateTemplate{ID: 1676, Kind: 13, Position: 22})
			s.SetGameDataManager(m)
			ids := []int64{1, 2, 3, 4, 5}
			if mode == "duplicate" {
				ids[4] = 2
			}
			if mode == "wrong colour" {
				r.items[2].ColorCode = 2
			}
			if mode == "equipped" {
				r.items[2].SlotType = domainitem.SlotTypeEquipped
			}
			result, err := s.fuseWings(context.Background(), 7, ids, mode == "senior", 0, func(n int) int {
				if mode == "random failure" {
					return n - 1
				}
				return 0
			})
			invalid := mode == "duplicate" || mode == "wrong colour" || mode == "equipped" || mode == "commit failure"
			if invalid {
				if err == nil || len(r.items) != 5 || r.items[1].ColorCode != 3 {
					t.Fatal("invalid attempt changed inventory")
				}
				return
			}
			if err != nil || len(r.items) != 1 || result.Main.StarLevel != 4 {
				t.Fatalf("bad result %v", err)
			}
			if mode == "random failure" {
				if result.Success || result.Main.ColorCode != 3 || result.Main.IsBound {
					t.Fatal("failure changed main")
				}
			} else if !result.Success || result.Main.ColorCode != 4 || !result.Main.IsBound {
				t.Fatal("success failed to upgrade/bind")
			}
		})
	}
}
