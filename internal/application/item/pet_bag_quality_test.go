package item

import (
	"context"
	"testing"

	"go.uber.org/zap"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

func TestPetBagAdvertisedGrowth(t *testing.T) {
	for _, tc := range []struct {
		id                int64
		name, description string
		want              int
	}{
		{1326, "Túi Quà Pet Lễ Tình Nhân", "Có thể nhận được pet Thiên Sứ Mít \nƯớt hoặc Ác Ma Quấy Phá bảo \nbảo phẩm chất 1.5 (cấp độ sử dụng 10)", 15},
		{1814, "Túi Quà Thiên Sứ Mít Ướt Cam", "Chứa 1 pet Thiên Sứ Mít Ướt Bảo Bảo phẩm chất 2.3", 23},
		{5937, "Túi Quà Đại Thánh", "Chứa pet Đại Thánh phẩm chất 1.5\n(Cấp độ mang theo cấp 50)\nXác suất nhận 1 Quả Đào Ngàn Năm", 15},
		{6861, "Túi Quà Huyền Minh Linh Vũ", "Chứa 1 Huyền Minh Linh Vũ xanh dương 1.5.\nCấp độ mang theo 50. Mở túi xác suất nhận Vạn Linh Bội", 15},
		{8000, "Túi Quà Pet", "Chứa pet Đại Thánh cấp độ mang theo 50", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop()
			manager := gamedata.NewManager(nil, logger)
			for i, name := range []string{"Thiên Sứ Mít Ướt", "Ác Ma Quấy Phá", "Đại Thánh", "Huyền Minh Linh Vũ"} {
				loadCreatureTemplate(t, manager, models.CreatureTemplate{ID: int64(i + 19), Name: name, GrowBase: 0.8})
			}
			svc := NewService(newItemServiceTestRepo(), logger)
			svc.SetGameDataManager(manager)
			pets := &petBagTestPetService{}
			svc.SetPetService(pets)
			tpl := &models.ItemTemplateTemplate{ID: tc.id, Name: tc.name, Description: tc.description, Type: 550, UseType: 1}
			_, handled, err := svc.tryApplyPetBagEffect(context.Background(), &domainchar.Character{ID: 1}, tpl)
			if err != nil || !handled {
				t.Fatalf("handled=%v error=%v", handled, err)
			}
			if pets.quality != tc.want {
				t.Fatalf("quality=%d want%d", pets.quality, tc.want)
			}
			if len(pets.templateIDs) != 1 {
				t.Fatalf("wrong pet selection:%v", pets.templateIDs)
			}
		})
	}
}

func TestDescriptionGrowthParser(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int
	}{
		{"phẩm\nchất\n1,5", 15}, {"xanh dương\n1,5", 15}, {"XANH LAM 1.5", 15},
		{"Cấp độ mang theo 50, cần 10 vàng", 0}, {"xanh dương cấp 50", 0}, {"xanh dương 50", 0}, {"Chứa 1 pet cấp 50", 0},
	} {
		if got := parseDescriptionQuality(tc.text); got != tc.want {
			t.Errorf("%q got%d want%d", tc.text, got, tc.want)
		}
	}
}

func TestPetBagAllowlistDoesNotRecognizeWingBox(t *testing.T) {
	if isPetSummonBoxTemplate(&models.ItemTemplateTemplate{ID: 3376, Name: "Túi Quà Cánh", Description: "Chứa 1 cánh Ác Ma xanh dương 1.5", Type: 550, UseType: 1}) {
		t.Fatal("wing box recognized as pet")
	}
}
