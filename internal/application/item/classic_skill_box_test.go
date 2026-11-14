package item

import (
	"go.uber.org/zap"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"testing"
)

func TestClassicSkillBoxRequiresCompleteFiveBookChoice(t *testing.T) {
	m := gamedata.NewManager(nil, zap.NewNop())
	if _, err := classicSkillBox(m); err == nil {
		t.Fatal("missing books accepted")
	}
	for _, id := range []int64{3467, 3468, 3469, 3466, 2709} {
		loadItemTemplate(t, m, models.ItemTemplateTemplate{ID: id, Type: 506})
	}
	r, err := classicSkillBox(m)
	if err != nil {
		t.Fatal(err)
	}
	if r.ConsumeItem || r.GiftBox.GetNum != 1 || len(r.GiftBox.Choices) != 5 {
		t.Fatal("incorrect choice box")
	}
	for _, choice := range r.GiftBox.Choices {
		if choice.TableType != 29 || choice.StackNum != 1 {
			t.Fatal("wrong reward type/count")
		}
	}
}
