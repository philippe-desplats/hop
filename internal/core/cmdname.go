package core

import (
	"regexp"
	"slices"
)

// cmdNameRe is the shape of a usable daily function name. The pattern is
// deliberately tight: the name is interpolated into the three integration
// templates, so anything outside it could inject shell syntax.
var cmdNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// DefaultCmdName is the daily function name used when nothing valid overrides it.
const DefaultCmdName = "p"

// SupportedShells are the shells `hop init` can emit an integration for.
var SupportedShells = []string{"zsh", "bash", "fish"}

// IsShellName reports whether name is one of the shells hop init knows. Such a
// name is refused as a function name: the integration would shadow the shell's
// own binary, so typing `zsh` would run hop nav instead of starting a subshell.
func IsShellName(name string) bool {
	return slices.Contains(SupportedShells, name)
}

// ValidCmdName reports whether name is usable as the daily shell function name.
// It lives here rather than in the CLI so `hop init` and the config editor share
// one rule: a name the editor accepted and `hop init` refused was saved, then
// dropped for DefaultCmdName at the next shell start.
func ValidCmdName(name string) bool {
	return cmdNameRe.MatchString(name) && !IsShellName(name)
}
