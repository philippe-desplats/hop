package core

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/philippe-desplats/hop/internal/i18n"
)

// GitInfo is the lightweight git state shown in the Hub preview.
type GitInfo struct {
	IsRepo     bool
	Branch     string
	Dirty      bool
	LastCommit string
	LastWhen   string // relative age, e.g. "il y a 2 heures"
}

// LoadGitInfo reads branch, dirty state and last commit for a project. It is
// meant to run off the UI thread (one exec per project, cached by the caller).
func LoadGitInfo(path string) GitInfo {
	gi := GitInfo{IsRepo: isRepo(path)} // isRepo (scanner.go): stat path/.git
	if !gi.IsRepo {
		return gi
	}
	run := func(args ...string) string {
		//nolint:gosec // fixed "git" binary; args are internal constants and a project path from our own index
		out, err := exec.Command("git", append([]string{"-C", path}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	gi.Branch = run("rev-parse", "--abbrev-ref", "HEAD")
	gi.Dirty = run("status", "--porcelain") != ""
	if last := run("log", "-1", "--format=%s|%ct"); last != "" {
		if parts := strings.SplitN(last, "|", 2); len(parts) == 2 {
			gi.LastCommit = parts[0]
			if sec, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				gi.LastWhen = i18n.RelAge(time.Since(time.Unix(sec, 0)))
			}
		}
	}
	return gi
}
