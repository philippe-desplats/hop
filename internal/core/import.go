package core

import (
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// ZoxideEntry is one parsed line of `zoxide query --list --score`.
type ZoxideEntry struct {
	Path  string
	Score float64
}

// parseZoxideLine reads a single `zoxide query --list --score` line. The format
// is "<score> <path>": a right-aligned float, whitespace, then the path (which
// may itself contain spaces). The leading numeric field is the score, the
// trimmed remainder is the path. Lines that do not parse return ok=false.
func parseZoxideLine(s string) (path string, score float64, ok bool) {
	trimmed := strings.TrimLeft(strings.TrimRight(s, "\r\n"), " \t")
	i := strings.IndexAny(trimmed, " \t")
	if i <= 0 {
		return "", 0, false
	}
	sc, err := strconv.ParseFloat(trimmed[:i], 64)
	if err != nil {
		return "", 0, false
	}
	rest := strings.TrimLeft(trimmed[i:], " \t")
	if rest == "" {
		return "", 0, false
	}
	return rest, sc, true
}

// ParseZoxideOutput parses the full text output of `zoxide query --list --score`,
// skipping any line that does not match the expected format.
func ParseZoxideOutput(out string) []ZoxideEntry {
	var entries []ZoxideEntry
	for _, line := range strings.Split(out, "\n") {
		if path, score, ok := parseZoxideLine(line); ok {
			entries = append(entries, ZoxideEntry{Path: path, Score: score})
		}
	}
	return entries
}

// ImportZoxide seeds hop's frecency from zoxide entries and tracks git repos hop
// does not already index, so a zoxide user keeps their learned ranking when they
// switch. indexed is the set of canonical paths hop already knows from scanning
// (it already includes tracked folders). Per entry whose directory exists:
//   - already indexed   -> seed frecency only
//   - a git repo        -> AddExtra (so it joins the search list) + seed frecency
//   - anything else      -> skipped (an arbitrary cd'd-into dir is not a project)
//
// Writes are batched into a single extras write and a single frecency write, so
// aging runs once over the whole import rather than per entry. With dryRun it
// reports the same counts without writing anything.
func ImportZoxide(entries []ZoxideEntry, indexed map[string]bool, now time.Time, dryRun bool) (imported, tracked, skipped int, err error) {
	seeds := map[string]float64{}
	trackSet := map[string]bool{}
	for _, e := range entries {
		path := CanonicalDir(e.Path)
		if HasControlChars(path) {
			skipped++
			continue
		}
		fi, statErr := os.Stat(path)
		if statErr != nil || !fi.IsDir() {
			skipped++
			continue
		}
		if !indexed[path] {
			if !isRepo(path) {
				skipped++
				continue
			}
			trackSet[path] = true
		}
		rank := math.Round(e.Score)
		switch {
		case rank > maxAge:
			rank = maxAge
		case rank < 1:
			rank = 1 // an indexed or tracked target is worth at least one visit
		}
		if cur, ok := seeds[path]; !ok || rank > cur {
			seeds[path] = rank
		}
	}
	imported = len(seeds)
	tracked = len(trackSet)
	if dryRun {
		return imported, tracked, skipped, nil
	}
	paths := make([]string, 0, len(trackSet))
	for p := range trackSet {
		paths = append(paths, p)
	}
	if tracked, err = AddExtras(paths); err != nil {
		return imported, tracked, skipped, err
	}
	if err = SeedFrecency(seeds, now); err != nil {
		return imported, tracked, skipped, err
	}
	return imported, tracked, skipped, nil
}
