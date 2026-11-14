package scene

import (
	"encoding/json"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func teleportService(t *testing.T, maps []models.MapTemplate, gates []models.SceneitemInstanceTemplate) *Service {
	t.Helper()
	manager := gamedata.NewManager(nil, zap.NewNop())
	for table, values := range map[string]interface{}{models.TableMap: maps, models.TableSceneitemInstance: gates} {
		raw, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		if err := manager.GetCache().LoadTable(table, rows); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(nil, nil, nil, nil, zap.NewNop())
	svc.SetGameDataManager(manager)
	return svc
}

func TestUnlinked757UsesUniqueOrdinaryMapSafePoint(t *testing.T) {
	svc := teleportService(t, []models.MapTemplate{{ID: 1, Name: "Xuất Vân Thôn", T: 1, SafeX: 1000, SafeY: 1000, Width: 2000, Height: 1500}}, []models.SceneitemInstanceTemplate{{ID: 757, Name: "Xuất Vân Thôn", Tid: 38, PosMapID: 3}})
	dest, err := svc.GetTeleportDestination(757)
	if err != nil || dest == nil || *dest != (TeleportDestination{MapID: 1, X: 1000, Y: 1000}) {
		t.Fatalf("destination=%v error=%v", dest, err)
	}
}

func TestUnlinkedPortalRejectsAmbiguousInstancedOrInvalidDestinations(t *testing.T) {
	valid := models.MapTemplate{ID: 1, Name: "Target", T: 1, SafeX: 100, SafeY: 100, Width: 500, Height: 500}
	for _, mode := range []string{"missing", "ambiguous", "instance", "invalid-point", "same-map"} {
		t.Run(mode, func(t *testing.T) {
			target := valid
			maps := []models.MapTemplate{target}
			switch mode {
			case "missing":
				maps = nil
			case "ambiguous":
				other := valid
				other.ID = 2
				maps = append(maps, other)
			case "instance":
				maps[0].CopyFlag = 3
			case "invalid-point":
				maps[0].SafeX = 600
			case "same-map":
				maps[0].ID = 3
			}
			svc := teleportService(t, maps, []models.SceneitemInstanceTemplate{{ID: 757, Name: "Target", Tid: 38, PosMapID: 3}})
			dest, err := svc.GetTeleportDestination(757)
			if err != nil || dest != nil {
				t.Fatalf("must reject unsafe fallback: %v %v", dest, err)
			}
		})
	}
}

func TestExplicitPortalLinkStillTakesPriority(t *testing.T) {
	svc := teleportService(t, nil, []models.SceneitemInstanceTemplate{{ID: 757, From: 900, Tid: 38}, {ID: 900, PosMapID: 2, PosX: 120, PosY: 63}})
	dest, err := svc.GetTeleportDestination(757)
	if err != nil || dest == nil || *dest != (TeleportDestination{MapID: 2, X: 220, Y: 163}) {
		t.Fatalf("explicit link changed: %v %v", dest, err)
	}
}
