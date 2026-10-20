//go:build unix

package core

import (
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"
)

// ownedInfo is an fs.FileInfo whose owner and mode the test chooses. A file
// belonging to another account cannot be created without root, so handing judge
// the stat such a file would carry is the only way to exercise the verdict that
// reports one, and that verdict is the whole reason the check looks at the owner
// instead of trusting the mode bits.
type ownedInfo struct {
	mode fs.FileMode
	stat syscall.Stat_t
}

func (o ownedInfo) Name() string       { return "index.json" }
func (o ownedInfo) Size() int64        { return 0 }
func (o ownedInfo) Mode() fs.FileMode  { return o.mode }
func (o ownedInfo) ModTime() time.Time { return time.Time{} }
func (o ownedInfo) IsDir() bool        { return o.mode.IsDir() }
func (o ownedInfo) Sys() any           { return &o.stat }

func TestJudgeReadsOwnerThenMode(t *testing.T) {
	ours := uint32(os.Getuid())
	theirs := ours + 1
	for _, tc := range []struct {
		name string
		uid  uint32
		mode fs.FileMode
		want StatePermStatus
	}{
		// 0600 is the mode hop writes, and the mode that makes the owner the only
		// thing left to judge: another user's file stays theirs to chmod back.
		{"another user's file, tight mode", theirs, 0o600, StatePermForeign},
		{"another user's directory", theirs, fs.ModeDir | 0o700, StatePermForeign},
		{"our own file at hop's own mode", ours, 0o600, StatePermOK},
		{"our own file the world can write", ours, 0o666, StatePermWritable},
		// The owner is read first: a mode the user cannot keep is not a mode issue.
		{"another user's world-writable file", theirs, 0o666, StatePermForeign},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fi := ownedInfo{mode: tc.mode, stat: syscall.Stat_t{Uid: tc.uid}}
			got := judge(StatePermCheck{Path: "/tmp/hop/index.json"}, fi)
			if got.Status != tc.want {
				t.Errorf("status = %d, want %d", got.Status, tc.want)
			}
			if got.Mode != tc.mode.Perm() {
				t.Errorf("mode = %04o, want %04o", got.Mode, tc.mode.Perm())
			}
			if got.IsDir != tc.mode.IsDir() {
				t.Errorf("IsDir = %t, want %t (it picks the chmod to suggest)", got.IsDir, tc.mode.IsDir())
			}
		})
	}
}
