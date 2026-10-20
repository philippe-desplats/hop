package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// StatePermStatus is the verdict on one path hop reads to build its output.
type StatePermStatus int

const (
	// StatePermOK: owned by the current user, with no group or other write bit.
	StatePermOK StatePermStatus = iota
	// StatePermAbsent: nothing sits at that path, which is not a failure. hop
	// creates its state on first write, and the settings file stays optional.
	StatePermAbsent
	// StatePermWritable: a group or other write bit is set, so another local
	// account can rewrite the content hop reads.
	StatePermWritable
	// StatePermForeign: owned by another user, who can chmod it back whatever
	// the owner of the hop session does to it.
	StatePermForeign
	// StatePermSymlink: an entry of the state directory is a symbolic link. It
	// is a failure on its own and the link is never followed: it can point hop's
	// next write at a file the planter chose, and reading its target to judge it
	// would already be doing what it was planted for.
	StatePermSymlink
	// StatePermUnreadable: the path exists, or may exist, but cannot be
	// inspected. Reporting it as absent would hide it.
	StatePermUnreadable
)

// StatePermCheck is the verdict on one checked item.
type StatePermCheck struct {
	Path   string // absolute path, as hop uses it
	IsDir  bool   // drives the suggested mode: 700 for a directory, 600 for a file
	Status StatePermStatus
	Mode   fs.FileMode // permission bits only, zero when the path is absent
}

// CheckStatePerms inspects the files whose content reaches the shell. The
// function `hop init` generates evaluates what `hop nav` prints, and that output
// is built from the state directory (StateDir) and from the settings file
// (ConfigPath), so an account that can write one of them, or the directory
// holding it, can plant a command the owner's shell runs. The checked items are
// the state directory, every entry directly inside it, the settings directory
// and the settings file, in that order.
//
// It only looks: no path is created, no mode is changed, no file is opened.
func CheckStatePerms() []StatePermCheck {
	stateDir := StateDir()
	checks := []StatePermCheck{checkStatePath(stateDir, true)}
	// ReadDir reports a symlink as a symlink, and the FileInfo of its entries
	// describes the link rather than its target, so no planted link is followed.
	entries, err := os.ReadDir(stateDir)
	for _, e := range entries {
		checks = append(checks, checkStateEntry(filepath.Join(stateDir, e.Name()), e))
	}
	// A directory that exists but cannot be listed hides whatever it holds, so it
	// is reported rather than passed as clean. A verdict already raised on the
	// directory itself is the more precise one and stands.
	if err != nil && !errors.Is(err, fs.ErrNotExist) && checks[0].Status == StatePermOK {
		checks[0].Status = StatePermUnreadable
	}
	cfg := ConfigPath()
	return append(checks, checkStatePath(filepath.Dir(cfg), true), checkStatePath(cfg, false))
}

// checkStatePath judges a path hop addresses by name. Such a path is resolved
// the way hop resolves it when it reads or writes it, symlinks included: what
// matters there is the mode of what hop would end up touching.
func checkStatePath(path string, isDir bool) StatePermCheck {
	c := StatePermCheck{Path: path, IsDir: isDir}
	fi, err := os.Stat(path)
	if err != nil {
		c.Status = StatePermAbsent
		if !errors.Is(err, fs.ErrNotExist) {
			c.Status = StatePermUnreadable
		}
		return c
	}
	return judge(c, fi)
}

// checkStateEntry judges an entry of the state directory from the information
// ReadDir already gathered, which describes the entry itself.
func checkStateEntry(path string, e os.DirEntry) StatePermCheck {
	c := StatePermCheck{Path: path, IsDir: e.IsDir()}
	fi, err := e.Info()
	if err != nil {
		c.Status = StatePermUnreadable
		return c
	}
	if e.Type()&fs.ModeSymlink != 0 {
		c.Mode = fi.Mode().Perm()
		c.Status = StatePermSymlink
		return c
	}
	return judge(c, fi)
}

// judge completes a check from the FileInfo its caller gathered. An item is OK
// when it belongs to the user running hop and carries no group or other write
// bit, which is the only combination no other local account can turn into a
// command the owner's shell runs.
func judge(c StatePermCheck, fi fs.FileInfo) StatePermCheck {
	c.IsDir = fi.IsDir()
	c.Mode = fi.Mode().Perm()
	c.Status = StatePermOK
	switch {
	case !ownedByCurrentUser(fi):
		c.Status = StatePermForeign
	case c.Mode&0o022 != 0:
		c.Status = StatePermWritable
	}
	return c
}

// ownedByCurrentUser compares the owner of fi with the uid hop runs as. hop
// ships for darwin and linux only, so the Unix stat is always there. On a
// platform that has no uid, the owner cannot be judged and is taken as ours,
// which leaves the mode bits to speak for themselves.
func ownedByCurrentUser(fi fs.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	return int64(st.Uid) == int64(os.Getuid())
}
