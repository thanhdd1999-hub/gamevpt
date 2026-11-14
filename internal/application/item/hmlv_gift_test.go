package item

import (
	"context"
	"errors"
	"go.uber.org/zap"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

type hmlvGiftTestPets struct {
	petBagTestPetService
	live    map[int64]*domainpet.Pet
	calls   int
	deleted int
}

func (p *hmlvGiftTestPets) ContractPetsWithQuality(ctx context.Context, charID int64, ids []int, q int) ([]*domainpet.Pet, error) {
	p.calls++
	p.quality = q
	p.templateIDs = append([]int(nil), ids...)
	if p.err != nil {
		return nil, p.err
	}
	pet := &domainpet.Pet{ID: int64(100 + p.calls), CharacterID: charID, TemplateID: ids[0], GrowRate: float64(q) / 10, Property: map[string]interface{}{"state": 3}}
	p.live[pet.ID] = pet
	return []*domainpet.Pet{pet}, nil
}
func (p *hmlvGiftTestPets) Save(_ context.Context, pet *domainpet.Pet) error {
	p.live[pet.ID] = pet
	return nil
}
func (p *hmlvGiftTestPets) DeletePet(_ context.Context, _ int64, id int64) error {
	delete(p.live, id)
	p.deleted++
	return nil
}

type hmlvGiftFailRepo struct {
	*itemServiceTestRepo
	failGrant, failConsume bool
}

func (r *hmlvGiftFailRepo) Create(ctx context.Context, it *domainitem.Item) error {
	if r.failGrant {
		return errors.New("bonus grant failed")
	}
	return r.itemServiceTestRepo.Create(ctx, it)
}
func (r *hmlvGiftFailRepo) Delete(ctx context.Context, id int64) error {
	if r.failConsume && id == 50 {
		return errors.New("box consumption failed")
	}
	return r.itemServiceTestRepo.Delete(ctx, id)
}

func hmlvGiftTestService(t *testing.T, repo domainitem.Repository) (*Service, *hmlvGiftTestPets) {
	t.Helper()
	svc := NewService(repo, zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 6861, Name: "Túi Quà Huyền Minh Linh Vũ", Description: "Chứa 1 Huyền Minh Linh Vũ xanh dương 1.5", Type: 550, UseType: 1, StackMax: 99999})
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 6859, Name: "Vạn Linh Bội", Type: 550, UseType: 2, StackMax: 99999})
	loadCreatureTemplate(t, manager, models.CreatureTemplate{ID: 2281, Name: "Huyền Minh Linh Vũ", GrowBase: 0.8})
	svc.SetGameDataManager(manager)
	pets := &hmlvGiftTestPets{live: map[int64]*domainpet.Pet{}}
	svc.SetPetService(pets)
	return svc, pets
}

func TestHMLVGiftAlwaysGrantsOnePendantAndQuality15Pet(t *testing.T) {
	for _, bound := range []bool{false, true} {
		repo := newItemServiceTestRepo()
		repo.nextID = 1000
		repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 6861, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 3, IsBound: bound}
		svc, pets := hmlvGiftTestService(t, repo)
		for attempt := 0; attempt < 3; attempt++ {
			used, result, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 1, Level: 50}, 50, nil)
			if err != nil || used.StackCount != 2-attempt || len(pets.live) != attempt+1 {
				t.Fatalf("attempt%d err%v pets%d", attempt, err, len(pets.live))
			}
			awarded, ok := result["granted_items"].([]*GrantedItemResult)
			if !ok || len(awarded) != 1 || awarded[0].TemplateID != 6859 || awarded[0].Count != 1 || awarded[0].IsBound != bound {
				t.Fatalf("bad pendant callbacks %v", result)
			}
			for _, pet := range pets.live {
				if pet.GrowRate != 1.5 || pet.IsBound() != bound {
					t.Fatal("quality/binding wrong")
				}
			}
		}
		total := 0
		for _, it := range repo.items {
			if it.TemplateID == 6859 {
				total += it.StackCount
				if it.IsBound != bound {
					t.Fatal("pendant binding")
				}
			}
		}
		if total != 3 || repo.items[50] != nil {
			t.Fatalf("totalpendants%d source%v", total, repo.items[50])
		}
	}
}

func TestHMLVGiftTechnicalFailureDoesNotDuplicatePetOrConsumeBox(t *testing.T) {
	for _, consumeFailure := range []bool{false, true} {
		repo := &hmlvGiftFailRepo{itemServiceTestRepo: newItemServiceTestRepo(), failGrant: !consumeFailure, failConsume: consumeFailure}
		repo.nextID = 1000
		repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 6861, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
		svc, pets := hmlvGiftTestService(t, repo)
		_, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 1, Level: 50}, 50, nil)
		if err == nil || len(pets.live) != 0 || len(repo.items) != 1 || repo.items[50].StackCount != 1 || repo.items[50].SlotType != domainitem.SlotTypeBag || repo.items[50].SlotIndex != 0 {
			t.Fatalf("compensation failed err%v pets%v items%v", err, pets.live, repo.items)
		}
		if consumeFailure && pets.deleted != 1 {
			t.Fatal("created pet not rolled back")
		}
		if !consumeFailure && pets.calls != 0 {
			t.Fatal("pet created before bonus grant succeeded")
		}
	}
}

func TestHMLVGiftPetCreationFailureRestoresPendantAndBox(t *testing.T) {
	repo := newItemServiceTestRepo()
	repo.nextID = 1000
	repo.items[50] = &domainitem.Item{ID: 50, CharacterID: 1, TemplateID: 6861, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, SlotIndex: 0, StackCount: 1}
	svc, pets := hmlvGiftTestService(t, repo)
	pets.err = errors.New("pet capacity failed")
	_, _, err := svc.UseItem(context.Background(), &domainchar.Character{ID: 1, Level: 50}, 50, nil)
	if err == nil || len(repo.items) != 1 || repo.items[50].StackCount != 1 || repo.items[50].SlotType != domainitem.SlotTypeBag || len(pets.live) != 0 {
		t.Fatal("pet creation failure was not compensated")
	}
}
