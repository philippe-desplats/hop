//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests drive `hop doctor` through runHopCLI (help_test.go): what the
// `state` lines have to prove is a report about real permission bits on real
// files, which only a real process looking at a real directory can show.
//
// The shell function `hop init` generates evaluates what `hop nav` prints, and
// that output is built from the state directory and the settings file. A file
// there that another local account can write is a command the owner's shell
// runs, so these lines are a security report, not a cosmetic one.

// doctorHome lays out an isolated home with hop's own default modes: 0700 for
// the state and config directories, 0600 for the files inside them. It is the
// baseline every failure case below departs from by exactly one mode.
func doctorHome(t *testing.T) hopHome {
	t.Helper()
	h := newHopHome(t, englishConfig)
	if err := os.Chmod(filepath.Dir(h.configFile()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(h.stateFile(""), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"index.json":    `{"version":1,"projects":[]}`,
		"frecency.json": `{"version":1,"entries":{}}`,
	} {
		if err := os.WriteFile(h.stateFile(name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// stateLine is one `state` line of the report: the path of its second column
// and the verdict of its third.
type stateLine struct {
	path, status string
}

// stateLines picks the `state` lines out of a doctor report, in the order they
// were printed. The other lines (roots, binaries, index) belong to checks this
// feature does not touch, and a test that asserted on the whole report would
// break every time one of them changes.
func stateLines(t *testing.T, h hopHome, argv ...string) []stateLine {
	t.Helper()
	got := runHopCLI(t, h, argv...)
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0 (doctor reports, it never fails)\nstdout: %s\nstderr: %s",
			got.code, got.stdout, got.stderr)
	}
	var lines []stateLine
	for _, line := range strings.Split(got.stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "state" {
			continue
		}
		lines = append(lines, stateLine{path: fields[1], status: strings.Join(fields[2:], " ")})
	}
	if len(lines) == 0 {
		t.Fatalf("the report carries no `state` line, got:\n%s", got.stdout)
	}
	return lines
}

// lookup returns an empty verdict for a path the report does not mention, which
// is what proves an absence. Callers asserting on a line use find instead.
func lookup(lines []stateLine, path string) string {
	for _, l := range lines {
		if l.path == path {
			return l.status
		}
	}
	return ""
}

// find returns the verdict reported for path, failing the test when the report
// has no line for it.
func find(t *testing.T, lines []stateLine, path string) string {
	t.Helper()
	status := lookup(lines, path)
	if status == "" {
		t.Fatalf("no `state` line for %s, got %v", path, lines)
	}
	return status
}

// TestDoctorStateDefaultModesAreOK holds hop's own defaults to a clean report.
// If the modes hop writes were themselves flagged, the check would be noise a
// user learns to ignore, and the one line that matters would be lost in it.
func TestDoctorStateDefaultModesAreOK(t *testing.T) {
	h := doctorHome(t)
	lines := stateLines(t, h, "doctor")

	for _, want := range []string{
		h.stateFile(""),              // the state directory
		h.stateFile("index.json"),    // an entry directly inside it
		h.stateFile("frecency.json"), // and another
		filepath.Dir(h.configFile()), // the settings directory
		h.configFile(),               // the settings file
	} {
		if status := find(t, lines, want); status != "OK" {
			t.Errorf("%s is owned by the test user at a default mode, want OK, got %q", want, status)
		}
	}
	for _, l := range lines {
		if strings.Contains(l.status, "✗") {
			t.Errorf("no default mode may be flagged, got %s %q", l.path, l.status)
		}
	}
}

// TestDoctorStateFlagsWritablePaths is the report a user asks doctor for: the
// mark, the mode as it is on disk, and the command that fixes it. A line that
// said only "insecure" would leave them guessing which bit and which chmod.
func TestDoctorStateFlagsWritablePaths(t *testing.T) {
	for _, tc := range []struct {
		name     string
		target   func(hopHome) string
		mode     os.FileMode
		wantMode string
		wantFix  string
	}{
		{"a state file anyone can write", func(h hopHome) string { return h.stateFile("index.json") }, 0o666, "0666", "chmod 600"},
		{"the state directory anyone can write", func(h hopHome) string { return h.stateFile("") }, 0o777, "0777", "chmod 700"},
		{"a settings file anyone can write", func(h hopHome) string { return h.configFile() }, 0o666, "0666", "chmod 600"},
		{"a settings file the group can write", func(h hopHome) string { return h.configFile() }, 0o664, "0664", "chmod 600"},
		{"the settings directory the group can write", func(h hopHome) string { return filepath.Dir(h.configFile()) }, 0o770, "0770", "chmod 700"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := doctorHome(t)
			target := tc.target(h)
			if err := os.Chmod(target, tc.mode); err != nil {
				t.Fatal(err)
			}

			lines := stateLines(t, h, "doctor")
			status := find(t, lines, target)
			if !strings.Contains(status, "✗") {
				t.Errorf("%s at %04o must be flagged, got %q", target, tc.mode.Perm(), status)
			}
			if !strings.Contains(status, tc.wantMode) {
				t.Errorf("the line must show the mode %s, got %q", tc.wantMode, status)
			}
			if want := tc.wantFix + " " + target; !strings.Contains(status, want) {
				t.Errorf("the line must carry the fix %q, got %q", want, status)
			}
		})
	}
}

// TestDoctorStateFlagsSymlinksWithoutFollowingThem covers the entry that is
// dangerous whatever its own mode says. A symlink in the state directory
// redirects hop's next write to a file its planter chose, so it is a failure on
// sight, and judging it by its target would mean touching that target, which is
// exactly what it was planted for.
func TestDoctorStateFlagsSymlinksWithoutFollowingThem(t *testing.T) {
	h := doctorHome(t)
	outside := t.TempDir()

	target := filepath.Join(outside, "owner-secret")
	const secret = "the target's own bytes\n"
	if err := os.WriteFile(target, []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	planted := h.stateFile("pins.json")
	if err := os.Symlink(target, planted); err != nil {
		t.Fatal(err)
	}
	// A link to nothing proves the verdict is the link's own: following it would
	// find no file and report the entry absent instead of flagging it.
	dangling := h.stateFile("extras.json")
	if err := os.Symlink(filepath.Join(outside, "no-such-file"), dangling); err != nil {
		t.Fatal(err)
	}

	lines := stateLines(t, h, "doctor")
	for _, link := range []string{planted, dangling} {
		status := find(t, lines, link)
		if !strings.Contains(status, "✗") {
			t.Errorf("the symlink %s must be flagged, got %q", link, status)
		}
	}
	if status := find(t, lines, planted); !strings.Contains(status, "symlink") {
		t.Errorf("the line must name what it found, got %q", status)
	}
	if status := lookup(lines, target); status != "" {
		t.Errorf("the target %s must not appear in the report at all, got %q", target, status)
	}

	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("the symlink target is gone: %v", err)
	}
	if string(after) != secret {
		t.Errorf("the target was rewritten: %q", after)
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the target's mode changed to %04o, doctor must chmod nothing", fi.Mode().Perm())
	}
}

// TestDoctorStateEscapesPlantedNames closes the hole the report would otherwise
// open by existing. The name of an entry in the state directory is chosen by
// whoever can write that directory, and a directory another account can write is
// precisely what these lines are here to denounce. A name carrying ESC [1A and
// CR moves the cursor back over the verdicts already printed and rewrites them,
// so the account the report accuses could make the whole block read OK. doctor
// must emit no byte a terminal acts on, and no name may forge a line of its own.
func TestDoctorStateEscapesPlantedNames(t *testing.T) {
	h := doctorHome(t)
	if err := os.Chmod(h.stateFile(""), 0o777); err != nil { // what lets the name be planted at all
		t.Fatal(err)
	}
	// Cursor up, erase the line, write a clean verdict over it, twice, then a
	// newline that would forge a whole line on its own.
	// A name cannot hold a slash, so the forged verdict names no path: what it
	// imitates is the shape of a clean line, which is all a reader scans for.
	forged := "state    every check below reads clean          OK"
	planted := h.stateFile("a\x1b[1A\x1b[2K\r" + forged + "\n" + forged)
	if err := os.WriteFile(planted, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(planted, 0o666); err != nil {
		t.Fatal(err)
	}

	got := runHopCLI(t, h, "doctor")
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0\nstdout: %q\nstderr: %q", got.code, got.stdout, got.stderr)
	}
	for _, b := range []string{"\x1b", "\r", "\x00"} {
		if strings.Contains(got.stdout, b) {
			t.Errorf("the report carries %q, which the terminal would act on:\n%q", b, got.stdout)
		}
	}

	lines := stateLines(t, h, "doctor")
	// 6 items exist: the state directory, its 3 entries, the settings directory
	// and the settings file. A 7th line would be one the planted name forged.
	if len(lines) != 6 {
		t.Errorf("the report must hold 6 `state` lines, got %d: %v", len(lines), lines)
	}
	// Both real failures have to survive the escaping, since hiding them is what
	// the planted name was for.
	var flagged int
	for _, l := range lines {
		if strings.Contains(l.status, "✗") {
			flagged++
		}
	}
	if flagged != 2 {
		t.Errorf("the 0777 directory and the 0666 file must both stay flagged, got %d of %d: %v",
			flagged, len(lines), lines)
	}
	if !strings.Contains(got.stdout, `\x1b[1A`) {
		t.Errorf("the escaped name must still be readable in the report:\n%s", got.stdout)
	}
}

// TestDoctorStateKeepsReadableModesOK draws the line the check must not cross.
// 0644 is the mode a careless umask leaves behind: it lets another account read
// the index, which is a privacy matter, but not write it, so it cannot plant a
// command. Flagging it would make the check cry wolf on most machines.
func TestDoctorStateKeepsReadableModesOK(t *testing.T) {
	h := doctorHome(t)
	for _, p := range []string{h.configFile(), h.stateFile("index.json")} {
		if err := os.Chmod(p, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(h.stateFile(""), 0o755); err != nil {
		t.Fatal(err)
	}

	lines := stateLines(t, h, "doctor")
	for _, p := range []string{h.configFile(), h.stateFile("index.json"), h.stateFile("")} {
		if status := find(t, lines, p); status != "OK" {
			t.Errorf("%s has no group or other write bit, want OK, got %q", p, status)
		}
	}
}

// TestDoctorStateReportsAbsentPathsWithoutFailing covers the first run: hop
// creates its state on the first write and the settings file stays optional, so
// a user who has only just installed hop must not be told their machine is
// compromised. Nothing may be created either, the report is read-only.
func TestDoctorStateReportsAbsentPathsWithoutFailing(t *testing.T) {
	h := newHopHome(t, "") // no config file, no config directory, no state directory

	lines := stateLines(t, h, "doctor")
	for _, p := range []string{h.stateFile(""), filepath.Dir(h.configFile()), h.configFile()} {
		if status := find(t, lines, p); status != "absent" {
			t.Errorf("%s does not exist, want absent, got %q", p, status)
		}
	}
	for _, l := range lines {
		if strings.Contains(l.status, "✗") {
			t.Errorf("an absent path is not a failure, got %s %q", l.path, l.status)
		}
	}
	for _, p := range []string{h.stateFile(""), filepath.Dir(h.configFile()), h.configFile()} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("doctor created %s (lstat err = %v)", p, err)
		}
	}
}

// TestDoctorStateShowsHomeRelativePaths holds the report to the path form the
// rest of the CLI uses. A $HOME spelled out in full pushes the verdict off the
// right edge of the terminal, which hides the very column the line is for.
func TestDoctorStateShowsHomeRelativePaths(t *testing.T) {
	h := doctorHome(t)
	// The default state directory sits under $HOME, while the fixture keeps it
	// in a temporary folder of its own, so point it back inside the home here.
	h.stateHome = filepath.Join(h.home, ".local", "state")
	if err := os.MkdirAll(h.stateFile(""), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.stateFile("index.json"), []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(h.stateFile("index.json"), 0o666); err != nil { // Chmod, so the umask cannot soften it
		t.Fatal(err)
	}

	lines := stateLines(t, h, "doctor")
	status := find(t, lines, "~/.local/state/hop/index.json")
	if !strings.Contains(status, "chmod 600 ~/.local/state/hop/index.json") {
		t.Errorf("the fix must quote the same ~ form as the path column, got %q", status)
	}
	if status := find(t, lines, "~/.local/state/hop"); status != "OK" {
		t.Errorf("the state directory line must use the ~ form too, got %q", status)
	}
}

// TestDoctorChangesNothingOnDisk is the promise the whole feature rests on:
// doctor reports and hands over a command, it never repairs. A doctor that
// chmod'ed what it found would take that decision away from the one person who
// knows why a file is shared, and would silently break a deliberate setup.
func TestDoctorChangesNothingOnDisk(t *testing.T) {
	h := doctorHome(t)
	// Every shape doctor has an opinion about, so the run has something to fix
	// in each branch of the report.
	if err := os.Chmod(h.stateFile("index.json"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(h.configFile(), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(h.configFile(), h.stateFile("pins.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(h.stateFile(""), 0o777); err != nil {
		t.Fatal(err)
	}

	before := snapshot(t, h)
	lines := stateLines(t, h, "doctor")
	// A fixture doctor had nothing to say about would prove nothing here, so the
	// four paths planted above are checked to be the ones it flagged.
	for _, p := range []string{
		h.stateFile(""), h.stateFile("index.json"), h.stateFile("pins.json"), h.configFile(),
	} {
		if status := find(t, lines, p); !strings.Contains(status, "✗") {
			t.Fatalf("%s was planted to be flagged, got %q", p, status)
		}
	}

	after := snapshot(t, h)
	for path, was := range before {
		switch now, ok := after[path]; {
		case !ok:
			t.Errorf("%s disappeared", path)
		case now != was:
			t.Errorf("%s changed\nbefore: %s\nafter:  %s", path, was, now)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("doctor created %s", path)
		}
	}
}

// snapshot records the mode and the content of every file and directory under
// the child's state and config directories, keyed by path. Modes come from
// Lstat and contents are read without following a link, so a symlink is
// compared as a symlink.
func snapshot(t *testing.T, h hopHome) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range []string{h.stateFile(""), filepath.Dir(h.configFile())} {
		err := filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			fi, err := os.Lstat(path)
			if err != nil {
				return err
			}
			entry := fi.Mode().String()
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				dest, err := os.Readlink(path)
				if err != nil {
					return err
				}
				entry += " -> " + dest
			case fi.Mode().IsRegular():
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				entry += " " + string(data)
			}
			out[path] = entry
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}
