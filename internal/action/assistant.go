package action

import (
	"os/exec"
	"strings"
)

// lookPath resolves a binary on PATH. It is a package var so tests can stub it.
var lookPath = exec.LookPath

// Assistant is an AI coding CLI hop can launch from the Hub. Run and Resume are
// argv slices; Resume is empty when the tool has no resume mode (the r action is
// then hidden).
type Assistant struct {
	Name   string
	Bin    string
	Run    []string
	Resume []string
}

// knownAssistants is the auto-detection preference order. Resume is only set for
// tools whose resume invocation is verified, to avoid shipping a wrong command.
var knownAssistants = []Assistant{
	{Name: "claude", Bin: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}},
	{Name: "codex", Bin: "codex", Run: []string{"codex"}},
	{Name: "aider", Bin: "aider", Run: []string{"aider"}},
	{Name: "gemini", Bin: "gemini", Run: []string{"gemini"}},
}

// ResolveAssistant picks the assistant bound to c/r. pref is "auto" (or empty) to
// take the first installed in preference order, or a name to force one. A named
// tool that is not installed falls back to auto-detection. ok is false when none
// is installed.
func ResolveAssistant(pref string) (Assistant, bool) {
	if pref != "" && pref != "auto" {
		for _, a := range knownAssistants {
			if a.Name == pref {
				if _, err := lookPath(a.Bin); err == nil {
					return a, true
				}
				break // named but missing: fall through to auto-detection
			}
		}
	}
	for _, a := range knownAssistants {
		if _, err := lookPath(a.Bin); err == nil {
			return a, true
		}
	}
	return Assistant{}, false
}

// DetectAssistants returns the names of every known assistant found on PATH, in
// preference order (used by hop doctor).
func DetectAssistants() []string {
	var found []string
	for _, a := range knownAssistants {
		if _, err := lookPath(a.Bin); err == nil {
			found = append(found, a.Name)
		}
	}
	return found
}

func (a Assistant) runCmd() string    { return strings.Join(a.Run, " ") }
func (a Assistant) resumeCmd() string { return strings.Join(a.Resume, " ") }
func (a Assistant) hasResume() bool   { return len(a.Resume) > 0 }
