// Package store is the single owner of on-disk state for hop.
// It provides version-aware, atomic, lock-guarded JSON persistence so that
// index.json, frecency.json and bookmarks.json never reimplement the
// write/lock/recovery dance separately.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ErrCorrupt signals that a data file exists but could not be decoded.
var ErrCorrupt = errors.New("store: corrupt data file")

// Load reads JSON from path into v. It returns a wrapped os.ErrNotExist when the
// file is missing, or ErrCorrupt when it exists but cannot be decoded. Reads are
// lock-free: writes are atomic renames, so a reader always sees a whole file.
func Load(path string, v any) error {
	data, err := os.ReadFile(path) //nolint:gosec // internal state file path, not untrusted input
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	return nil
}

// Save writes v atomically to path under a dedicated lock file. It blocks until
// the lock is acquired.
func Save(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	unlock, ok, err := acquire(path+".lock", true)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("store: could not acquire lock")
	}
	defer unlock()
	return writeAtomic(path, v)
}

// Update locks path, loads the current content into target (tolerating a missing
// or corrupt file, in which case target keeps its caller-provided value, which
// acts as a reset), runs mutate, then writes target back atomically. When
// blocking is false and the lock is busy, it returns (false, nil) without writing.
func Update(path string, target any, blocking bool, mutate func() error) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	unlock, ok, err := acquire(path+".lock", blocking)
	if err != nil || !ok {
		return false, err
	}
	defer unlock()

	if data, rerr := os.ReadFile(path); rerr == nil { //nolint:gosec // internal state file path
		// A decode error means corruption: ignore it and let target's initial
		// value (plus mutate's own guards) perform the reset.
		_ = json.Unmarshal(data, target)
	}
	if err := mutate(); err != nil {
		return false, err
	}
	if err := writeAtomic(path, target); err != nil {
		return false, err
	}
	return true, nil
}

func writeAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename succeeds
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// acquire takes an exclusive flock on a dedicated lock file (never on the data
// file itself, since os.Rename swaps the inode and would orphan the lock).
func acquire(lockPath string, blocking bool) (unlock func(), ok bool, err error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // lock file path derived internally from the state path
	if err != nil {
		return nil, false, err
	}
	how := syscall.LOCK_EX
	if !blocking {
		how |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		_ = f.Close()
		if !blocking && (errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, true, nil
}
