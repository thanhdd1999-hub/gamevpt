package item

import (
	"testing"

	"mcgame-server/internal/domain/character"
)

func TestUseMultiItemRejectsMalformedArgumentsWithoutPanic(t *testing.T) {
	handler, ctx := buildItemTestHandler(t, &character.Character{ID: 1})
	for _, args := range [][]interface{}{
		{1.0, -1.0, 1.0, -1.0},
		{1.0, -1.0, 1.0, 0.0},
		{1.0, -1.0, 1.0, 10001.0},
		{nil, -1.0, 1.0, 1.0},
		{1.0, map[string]interface{}{}, 1.0, 1.0},
		{1.0, -1.0, "not-an-item", 1.0},
		{1.0, -1.0, 1.0, true},
	} {
		if _, err := handler.UseMultiItem(ctx, args); err == nil {
			t.Fatalf("expected rejection for %v", args)
		}
	}
}
