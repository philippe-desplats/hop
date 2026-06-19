package core

// Config controls how the project tree is scanned. It is derived from the loaded
// TOML Settings via ScanConfig; the defaults live in DefaultSettings.
type Config struct {
	Roots    []string
	MaxDepth int
	Ignore   []string
}

// ScanConfig builds the scan Config from user settings, expanding ~ in roots.
// Callers pass LoadSettings(), which already coerces empty roots and a
// non-positive depth back to their defaults, so no fallback is needed here.
func ScanConfig(s Settings) Config {
	c := Config{MaxDepth: s.Scan.MaxDepth, Ignore: s.Scan.Ignore}
	for _, r := range s.Scan.Roots {
		c.Roots = append(c.Roots, expandHome(r))
	}
	return c
}
