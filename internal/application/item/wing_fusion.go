// Wing colour/chance rules follow the Flash contract; within-colour stat rolls use the local equipment quality generator.
package item

import (
	"context"
	"math/rand"
	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"
	"strconv"
	"sync"
)

var wingFusionMu sync.Mutex

type WingFusionResult struct {
	Main     *domainitem.Item
	Consumed []*domainitem.Item
	Success  bool
}

func (s *Service) FuseWings(ctx context.Context, charID int64, ids []int64, senior bool, bonus int) (*WingFusionResult, error) {
	return s.fuseWings(ctx, charID, ids, senior, bonus, rand.Intn)
}

func (s *Service) fuseWings(ctx context.Context, charID int64, ids []int64, senior bool, bonus int, draw func(int) int) (*WingFusionResult, error) {
	wingFusionMu.Lock()
	defer wingFusionMu.Unlock()
	atomic, ok := s.itemRepo.(interface {
		CommitWingFusion(context.Context, *domainitem.Item, []int64) error
	})
	if !ok || len(ids) < 5 || len(ids) > 17 || (!senior && len(ids) != 5) {
		return nil, pkgerrors.ErrInvalidInput
	}
	seen := map[int64]bool{}
	inputs := make([]*domainitem.Item, 0, len(ids))
	colour := 0
	penalty := 0
	bound := false
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return nil, pkgerrors.ErrInvalidInput
		}
		seen[id] = true
		it, err := s.GetItemByID(ctx, charID, id)
		if err != nil {
			return nil, err
		}
		tpl := s.GetEquipmentTemplate(it.TemplateID)
		if tpl == nil || int(tpl.Kind) != 13 || int(tpl.Position) != 22 || it.ItemType != domainitem.ItemTypeEquipment || it.SlotType != domainitem.SlotTypeBag || it.StackCount != 1 || it.ColorCode < 1 || it.ColorCode > 5 {
			return nil, pkgerrors.ErrInvalidInput
		}
		c := domainitem.EquipmentDisplayColorFromColorCode(it.ColorCode)
		if len(inputs) == 0 {
			colour = c
		} else {
			if c != colour {
				return nil, pkgerrors.ErrInvalidInput
			}
			if it.TemplateID != inputs[0].TemplateID {
				if senior {
					return nil, pkgerrors.ErrInvalidInput
				}
				penalty += 5
			}
		}
		bound = bound || it.IsBound
		inputs = append(inputs, it)
	}
	if colour < 0 || colour >= 4 {
		return nil, pkgerrors.ErrInvalidInput
	}
	rate := []int{100, 80, 50, 30}[colour] - penalty
	if senior {
		var rates []int
		switch colour {
		case 2:
			rates = []int{50, 53, 58, 65, 74, 85, 100}
		case 3:
			rates = []int{30, 33, 36, 40, 44, 49, 54, 60, 66, 73, 81, 90, 100}
		default:
			return nil, pkgerrors.ErrInvalidInput
		}
		index := len(ids) - 5
		if index >= len(rates) {
			return nil, pkgerrors.ErrInvalidInput
		}
		rate = rates[index]
	} else if colour == 3 {
		rate += bonus
	}
	if rate < 0 {
		rate = 0
	}
	if rate > 100 {
		rate = 100
	}
	next := *inputs[0]
	next.Properties = make(map[string]interface{}, len(inputs[0].Properties))
	for key, value := range inputs[0].Properties {
		next.Properties[key] = value
	}
	success := draw(100) < rate
	if success {
		next.ColorCode = domainitem.EquipmentColorCodeFromDisplayColor(colour + 1)
		quality := (next.ColorCode-1)*5 + 1 + draw(5)
		s.ApplyRandomizedEquipmentPropertyRolls(&next, s.GetEquipmentTemplate(next.TemplateID), quality)
		next.Properties["color"] = strconv.Itoa(colour + 1)
		next.Properties["q"] = quality
		next.Properties["quality"] = quality
		next.Properties["preNameType"] = domainitem.EquipmentPrefixTypeFromQuality(quality)
		next.IsBound = bound
		if bound {
			next.Properties["binded"] = "1"
		}
	}
	if err := atomic.CommitWingFusion(ctx, &next, ids[1:]); err != nil {
		return nil, err
	}
	return &WingFusionResult{Main: &next, Consumed: inputs[1:], Success: success}, nil
}
