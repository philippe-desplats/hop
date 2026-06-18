package core

// Config controls how the project tree is scanned. In v0 it is built from
// sensible defaults; a TOML file lands in v1.0.
type Config struct {
	Roots    []string
	MaxDepth int
	Ignore   []string
}

// DefaultConfig scans ~/Projects deeply enough to reach repos nested several
// levels down (some layouts go ~6 levels deep).
func DefaultConfig() Config {
	return Config{
		Roots:    []string{expandHome("~/Projects")},
		MaxDepth: 7,
		Ignore:   []string{"node_modules", "vendor", "_archives"},
	}
}

// ScanConfig builds the scan Config from user settings (roots get ~ expanded).
func ScanConfig(s Settings) Config {
	c := Config{MaxDepth: s.Scan.MaxDepth, Ignore: s.Scan.Ignore}
	for _, r := range s.Scan.Roots {
		c.Roots = append(c.Roots, expandHome(r))
	}
	if len(c.Roots) == 0 || c.MaxDepth <= 0 {
		return DefaultConfig()
	}
	return c
}
