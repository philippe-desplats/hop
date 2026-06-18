package core

import (
	"sort"
	"strings"
	"time"
)

// RankWeights blends match quality (Fuzzy) with frecency in the final score.
type RankWeights struct{ Fuzzy, Frecency float64 }

// DefaultWeights favours match quality slightly over frecency.
func DefaultWeights() RankWeights { return RankWeights{Fuzzy: 0.6, Frecency: 0.4} }

// Match is a project paired with its scores for a query.
type Match struct {
	Project Project
	Score   float64 // raw frecency score
	Final   float64 // blended rank (match quality + normalized frecency)
	kind    int     // last-keyword hit on the name: 3 token, 2 substring, 1 path-only
}

// Resolve ranks projects against an ordered list of keywords (zoxide-style):
// every keyword must appear in the path as an ordered subsequence, so
// `p acme web` narrows to .../acme/.../web-monorepo. Ranking blends a
// match-quality score (token > substring > path) with frecency normalized
// across the candidates, weighted by w. No keywords ranks purely by frecency.
func Resolve(projects []Project, frec *Frecency, keywords []string, now time.Time, w RankWeights) []Match {
	kws := make([]string, 0, len(keywords))
	for _, k := range keywords {
		if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
			kws = append(kws, k)
		}
	}

	var matches []Match
	maxFrec := 0.0
	for _, p := range projects {
		kind, ok := matchKeywords(p, kws)
		if !ok {
			continue
		}
		fs := frec.Score(p.Path, now)
		if fs > maxFrec {
			maxFrec = fs
		}
		matches = append(matches, Match{Project: p, Score: fs, kind: kind})
	}

	var last string
	if len(kws) > 0 {
		last = kws[len(kws)-1]
	}
	for i := range matches {
		quality := kindScore(matches[i].kind)
		if last != "" && strings.HasPrefix(strings.ToLower(matches[i].Project.Name), last) {
			quality = min(1, quality+0.1)
		}
		var normFrec float64
		if maxFrec > 0 {
			normFrec = matches[i].Score / maxFrec
		}
		matches[i].Final = w.Fuzzy*quality + w.Frecency*normFrec
	}

	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.Final != b.Final {
			return a.Final > b.Final
		}
		if a.kind != b.kind {
			return a.kind > b.kind
		}
		if last != "" {
			ap := strings.HasPrefix(strings.ToLower(a.Project.Name), last)
			bp := strings.HasPrefix(strings.ToLower(b.Project.Name), last)
			if ap != bp {
				return ap
			}
		}
		if len(a.Project.Name) != len(b.Project.Name) {
			return len(a.Project.Name) < len(b.Project.Name)
		}
		return a.Project.Name < b.Project.Name
	})
	return matches
}

func kindScore(kind int) float64 {
	switch kind {
	case 3:
		return 1.0
	case 2:
		return 0.6
	default:
		return 0.3
	}
}

// matchKeywords reports whether every keyword appears in the project path as an
// ordered subsequence, and how strongly the last keyword hits the project name
// (3 token, 2 substring, 1 path-only). With no keywords, everything matches.
func matchKeywords(p Project, kws []string) (kind int, ok bool) {
	if len(kws) == 0 {
		return 2, true
	}
	lowerPath := strings.ToLower(p.Path)
	pos := 0
	for _, kw := range kws {
		idx := strings.Index(lowerPath[pos:], kw)
		if idx < 0 {
			return 0, false
		}
		pos += idx + len(kw)
	}
	return nameKind(p.Name, kws[len(kws)-1]), true
}

func nameKind(name, kw string) int {
	lname := strings.ToLower(name)
	for _, tok := range tokenize(lname) {
		if tok == kw {
			return 3
		}
	}
	if strings.Contains(lname, kw) {
		return 2
	}
	return 1
}

func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' '
	})
}
