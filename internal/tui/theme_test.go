package tui

import "testing"

func TestColorFGBGDark(t *testing.T) {
	cases := []struct {
		env      string
		wantDark bool
		wantOK   bool
	}{
		{"", false, false},            // unset
		{"15;0", true, true},          // light fg on dark bg
		{"0;15", false, true},         // dark fg on light bg
		{"7;0", true, true},           // bg 0 -> dark
		{"15;7", false, true},         // bg 7 -> light
		{"1;2;8", true, true},         // 3-field form, bg 8 -> dark
		{"0;default;15", false, true}, // 3-field form, bg 15 -> light
		{"15;notanumber", false, false},
	}
	for _, tc := range cases {
		t.Setenv("COLORFGBG", tc.env)
		dark, ok := colorFGBGDark()
		if dark != tc.wantDark || ok != tc.wantOK {
			t.Errorf("COLORFGBG=%q -> (dark=%v ok=%v), want (%v %v)", tc.env, dark, ok, tc.wantDark, tc.wantOK)
		}
	}
}

func TestResolveDarkExplicitWins(t *testing.T) {
	t.Setenv("COLORFGBG", "0;15") // light hint, must be ignored when theme is explicit
	if resolveDark("light", nil) {
		t.Error(`resolveDark("light") = dark, want light`)
	}
	if !resolveDark("dark", nil) {
		t.Error(`resolveDark("dark") = light, want dark`)
	}
}
