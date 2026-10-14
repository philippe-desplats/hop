package core

import "testing"

func TestCmdNameValidation(t *testing.T) {
	valid := []string{"p", "pp", "go_proj", "_x", "Hop2"}
	invalid := []string{"", "2p", "p p", "p-x", "p.x", "p/x"}
	for _, n := range valid {
		if !cmdNameRe.MatchString(n) {
			t.Errorf("%q should be valid", n)
		}
	}
	for _, n := range invalid {
		if cmdNameRe.MatchString(n) {
			t.Errorf("%q should be invalid", n)
		}
	}
	// A valid pattern is a valid name.
	for _, n := range valid {
		if !ValidCmdName(n) {
			t.Errorf("ValidCmdName(%q) = false, want true", n)
		}
	}
	// A shell name passes the pattern, yet naming the function after a shell makes
	// the emitted integration shadow that shell's binary, so it must be refused.
	for _, n := range SupportedShells {
		if !cmdNameRe.MatchString(n) {
			t.Fatalf("precondition: %q is expected to pass the name pattern", n)
		}
		if ValidCmdName(n) {
			t.Errorf("ValidCmdName(%q) = true, want false (collides with a shell name)", n)
		}
	}
}
