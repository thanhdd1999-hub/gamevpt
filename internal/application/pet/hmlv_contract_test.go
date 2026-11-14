package pet

import (
	"context"
	"go.uber.org/zap"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestHuyenMinhContractIncludesCanonicalInnateSkill(t *testing.T) {
	raw, err := os.ReadFile("../../../supabase/seeds/data_tbl_creature_skill.sql")
	if err != nil {
		t.Fatal(err)
	}
	rows := regexp.MustCompile(`\(\s*(\d+),\s*2281,\s*(NULL|-?\d+),\s*(\d+)\s*\)`).FindAllStringSubmatch(string(raw), -1)
	skills := make([]models.CreatureSkillTemplate, 0, len(rows))
	for _, row := range rows {
		id, _ := strconv.Atoi(row[1])
		sid, _ := strconv.Atoi(row[3])
		position, _ := strconv.Atoi(row[2])
		skills = append(skills, models.CreatureSkillTemplate{ID: int64(id), Cid: 2281, Position: float64(position), Sid: float64(sid)})
	}
	repo := &contractTestPetRepo{pets: map[int64]*domainpet.Pet{}}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(newContractPetGameDataWithSkills(t, []models.CreatureTemplate{{ID: 2281, Name: "Huyền Minh Linh Vũ", GrowBase: 1.5}}, skills))
	pets, err := service.ContractPets(context.Background(), 1, []int{2281})
	if err != nil || len(pets) != 1 {
		t.Fatalf("contract failed: %v", err)
	}
	found := map[int]bool{}
	for slot := 1; slot <= 5; slot++ {
		if id := petSkillSlotValue(pets[0], slot); id > 0 {
			found[id] = true
		}
	}
	for _, id := range []int{7002, 7011, 7014} {
		if !found[id] {
			t.Fatalf("missing Huyền Minh default skill %d; actual=%v", id, found)
		}
	}
	if len(found) != 3 {
		t.Fatalf("unexpected Huyền Minh skill pool %v", found)
	}
}
