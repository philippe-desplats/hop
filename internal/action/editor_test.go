package action

import "testing"

func TestDetectEditors(t *testing.T) {
	t.Run("returns installed editors in preference order", func(t *testing.T) {
		stubLookPath(t, "zed", "cursor") // installed out of preference order
		got := DetectEditors()
		if len(got) != 2 || got[0].Bin != "cursor" || got[1].Bin != "zed" {
			t.Fatalf("got %+v, want [cursor zed] in that order", got)
		}
	})
	t.Run("none installed returns empty", func(t *testing.T) {
		stubLookPath(t)
		if got := DetectEditors(); len(got) != 0 {
			t.Fatalf("got %+v, want empty", got)
		}
	})
}
