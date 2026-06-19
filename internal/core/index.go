package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/philippe-desplats/hop/internal/i18n"
	"github.com/philippe-desplats/hop/internal/store"
)

const indexVersion = 1

// ErrStaleIndex means the on-disk index has an unknown schema version.
var ErrStaleIndex = errors.New("core: stale index version")

// Index is the cached project list.
type Index struct {
	Version   int       `json:"version"`
	ScannedAt int64     `json:"scanned_at"`
	Projects  []Project `json:"projects"`
}

// IndexPath is the canonical location of the index file.
func IndexPath() string { return filepath.Join(StateDir(), "index.json") }

// HasIndex reports whether an index file exists on disk, used to detect a
// first run and nudge the user toward `hop setup`.
func HasIndex() bool {
	_, err := os.Stat(IndexPath())
	return err == nil
}

// LoadIndex reads the index, returning ErrStaleIndex on a version mismatch and
// store.ErrCorrupt / os.ErrNotExist as appropriate.
func LoadIndex() (*Index, error) {
	idx := &Index{}
	if err := store.Load(IndexPath(), idx); err != nil {
		return nil, err
	}
	if idx.Version != indexVersion {
		return nil, ErrStaleIndex
	}
	return idx, nil
}

// BuildIndex scans the tree without persisting. Folders the user tracked with
// `hop track` are merged in, so they survive every rescan.
func BuildIndex(cfg Config) *Index {
	return &Index{
		Version:   indexVersion,
		ScannedAt: time.Now().Unix(),
		Projects:  mergeExtras(Scan(cfg)),
	}
}

// SaveIndex persists the index atomically.
func SaveIndex(idx *Index) error { return store.Save(IndexPath(), idx) }

// BuildAndSaveIndex rescans and persists, returning the fresh index.
func BuildAndSaveIndex(cfg Config) *Index {
	idx := BuildIndex(cfg)
	_ = SaveIndex(idx)
	return idx
}

// LoadIndexOrBuild returns the cached index, rebuilding it on any miss
// (missing, corrupt or stale version). The rebuilt index is best-effort saved.
func LoadIndexOrBuild(cfg Config, verbose bool) (*Index, error) {
	if idx, err := LoadIndex(); err == nil {
		return idx, nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, i18n.T("cli.indexing"))
	}
	idx := BuildIndex(cfg)
	_ = SaveIndex(idx) // usable even if the save fails
	return idx, nil
}
