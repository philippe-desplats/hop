package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This file drives the CLI from the outside: the parent re-executes the test
// binary, the child points os.Args at the argv under test and calls main, so a
// real argument vector goes through the real dispatch in a real process where
// os.Exit takes effect. Nothing here asserts on an internal function: only on
// what a user observes, which is stdout, stderr, the exit code and the files on
// disk. That is what makes these tests outlive the shape of the dispatch code.

// TestHopCLIChild is the child half of every test below. It has nothing to do in
// the parent pass, where the Go test runner also calls it.
func TestHopCLIChild(t *testing.T) {
	if os.Getenv("HOP_CLI_CHILD") != "1" {
		t.Skip("child half of the CLI tests, driven by runHopCLI")
	}
	os.Args = append([]string{"hop"}, decodeArgv(os.Getenv("HOP_CLI_ARGV"))...)
	main()
	os.Exit(0) // exit before the test runner prints PASS to stdout
}

// englishConfig pins the child's language, so the assertions below can quote the
// English catalog rather than whatever locale the test host runs under.
const englishConfig = "[ui]\nlanguage = \"en\"\n"

// hopHome is one child's whole filesystem: its home, its config directory and
// its state directory, each in a temporary folder. A test that checks for damage
// needs to own the files hop could damage.
type hopHome struct {
	home, configHome, stateHome string
}

// newHopHome lays out an isolated home, writing config as the child's
// config.toml when it is non-empty (an empty config leaves the file absent, the
// first-run case).
func newHopHome(t *testing.T, config string) hopHome {
	t.Helper()
	h := hopHome{
		home:       t.TempDir(),
		configHome: t.TempDir(),
		stateHome:  filepath.Join(t.TempDir(), "state"),
	}
	if config == "" {
		return h
	}
	if err := os.MkdirAll(filepath.Join(h.configHome, "hop"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.configFile(), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	return h
}

// configFile and indexFile are the two files hop writes that a help request must
// never create or rewrite.
func (h hopHome) configFile() string { return filepath.Join(h.configHome, "hop", "config.toml") }
func (h hopHome) indexFile() string  { return filepath.Join(h.stateHome, "hop", "index.json") }

// hopResult is everything the parent gets to observe of a child run.
type hopResult struct {
	stdout, stderr string
	code           int
}

func runHopCLI(t *testing.T, h hopHome, argv ...string) hopResult {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHopCLIChild")
	cmd.Dir = h.home
	cmd.Env = []string{
		"HOP_CLI_CHILD=1",
		"HOP_CLI_ARGV=" + encodeArgv(t, argv),
		"HOME=" + h.home,
		"XDG_CONFIG_HOME=" + h.configHome,
		"XDG_STATE_HOME=" + h.stateHome,
		"PATH=" + t.TempDir(),       // empty: printing a help page never shells out
		"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("subprocess failed to run: %v\nstderr: %s", err, stderr.String())
		}
		code = ee.ExitCode()
	}
	return hopResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// helpSubcommands are the subcommands that answered a help request by running.
// Six took their arguments and discarded them, so `--help` ran the command;
// init rejected the flag outright, so the one subcommand with an option of its
// own was the one whose options could not be read.
var helpSubcommands = []string{"setup", "init", "scan", "clean", "prompt", "config", "doctor"}

// TestSubcommandHelpPrintsHelp holds every one of them to the same answer, for
// both spellings of the request: help on stdout and exit 0.
func TestSubcommandHelpPrintsHelp(t *testing.T) {
	for _, sub := range helpSubcommands {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(sub+" "+flag, func(t *testing.T) {
				got := runHopCLI(t, newHopHome(t, englishConfig), sub, flag)
				if got.code != 0 {
					t.Errorf("exit = %d, want 0 (a question answered)\nstdout: %s\nstderr: %s", got.code, got.stdout, got.stderr)
				}
				if !strings.Contains(got.stdout, "Usage:") {
					t.Errorf("stdout must carry a help page, got:\n%s", got.stdout)
				}
				if got.stderr != "" {
					t.Errorf("a help request has nothing to report on stderr, got:\n%s", got.stderr)
				}
			})
		}
	}
}

// TestSetupHelpWritesNothing is the assertion that would have caught the
// reported data loss: `hop setup --help` used to run the guided setup, which
// saves the configuration and builds the index, so a user asking a question paid
// for it with their settings.
func TestSetupHelpWritesNothing(t *testing.T) {
	t.Run("an existing config comes out byte-identical", func(t *testing.T) {
		h := newHopHome(t, englishConfig+"\n[actions]\neditor = \"my-chosen-editor\"\n")
		before, err := os.ReadFile(h.configFile())
		if err != nil {
			t.Fatal(err)
		}

		got := runHopCLI(t, h, "setup", "--help")
		if got.code != 0 {
			t.Errorf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
		}

		after, err := os.ReadFile(h.configFile())
		if err != nil {
			t.Fatalf("the config file is gone: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("setup --help rewrote the config\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})

	t.Run("nothing is created when there was nothing", func(t *testing.T) {
		h := newHopHome(t, "")

		got := runHopCLI(t, h, "setup", "--help")
		if got.code != 0 {
			t.Errorf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
		}
		if got.stdout == "" {
			t.Error("stdout must carry a help page")
		}

		if _, err := os.Stat(h.configFile()); !os.IsNotExist(err) {
			t.Errorf("setup --help created a config file at %s (stat err = %v)", h.configFile(), err)
		}
		if _, err := os.Stat(h.indexFile()); !os.IsNotExist(err) {
			t.Errorf("setup --help built an index at %s (stat err = %v)", h.indexFile(), err)
		}
	})
}

// TestDedicatedHelpAnswersItsOwnSubcommand asserts the two dedicated pages on
// what a user needs from them, and that neither is the general help: a
// regression silently routing them to the general text fails here.
func TestDedicatedHelpAnswersItsOwnSubcommand(t *testing.T) {
	general := runHopCLI(t, newHopHome(t, englishConfig), "--help")
	if general.code != 0 || general.stdout == "" {
		t.Fatalf("hop --help must print the general help, got exit %d and:\n%s", general.code, general.stdout)
	}

	t.Run("init names the supported shells and documents --cmd", func(t *testing.T) {
		got := runHopCLI(t, newHopHome(t, englishConfig), "init", "--help")
		for _, want := range []string{"zsh", "bash", "fish", "--cmd"} {
			if !strings.Contains(got.stdout, want) {
				t.Errorf("init --help must mention %q, got:\n%s", want, got.stdout)
			}
		}
		if got.stdout == general.stdout {
			t.Error("init --help must print its own page, not the general help")
		}
	})

	t.Run("setup describes the guided setup", func(t *testing.T) {
		got := runHopCLI(t, newHopHome(t, englishConfig), "setup", "--help")
		for _, want := range []string{"folders", "editor", "assistant", "config"} {
			if !strings.Contains(got.stdout, want) {
				t.Errorf("setup --help must mention %q, got:\n%s", want, got.stdout)
			}
		}
		if got.stdout == general.stdout {
			t.Error("setup --help must print its own page, not the general help")
		}
	})
}

// TestTopLevelFormsAreUnchanged pins the invocations the interception must not
// disturb: the general help, the version, and a name hop does not know, which
// stays an error even when a help flag follows it.
func TestTopLevelFormsAreUnchanged(t *testing.T) {
	general := runHopCLI(t, newHopHome(t, englishConfig), "--help").stdout

	t.Run("the general help", func(t *testing.T) {
		for _, argv := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
			got := runHopCLI(t, newHopHome(t, englishConfig), argv...)
			if got.code != 0 {
				t.Errorf("hop %v exit = %d, want 0\nstderr: %s", argv, got.code, got.stderr)
			}
			if got.stdout != general {
				t.Errorf("hop %v must print the general help, got:\n%s", argv, got.stdout)
			}
		}
	})

	t.Run("the version", func(t *testing.T) {
		for _, argv := range [][]string{{"version"}, {"--version"}, {"-v"}} {
			got := runHopCLI(t, newHopHome(t, englishConfig), argv...)
			if got.code != 0 {
				t.Errorf("hop %v exit = %d, want 0\nstderr: %s", argv, got.code, got.stderr)
			}
			if !strings.HasPrefix(got.stdout, "hop ") || strings.Contains(got.stdout, "Usage:") {
				t.Errorf("hop %v must print the version, got:\n%s", argv, got.stdout)
			}
		}
	})

	t.Run("an unknown command, with or without a help flag", func(t *testing.T) {
		for _, argv := range [][]string{{"nope"}, {"nope", "--help"}, {"nope", "-h"}} {
			got := runHopCLI(t, newHopHome(t, englishConfig), argv...)
			if got.code != 2 {
				t.Errorf("hop %v exit = %d, want 2 (usage error)\nstdout: %s", argv, got.code, got.stdout)
			}
			if !strings.Contains(got.stderr, "unknown command") {
				t.Errorf("hop %v must report an unknown command, got:\n%s", argv, got.stderr)
			}
			if strings.Contains(got.stdout, "Usage:") {
				t.Errorf("hop %v must not be swallowed as a help request, got:\n%s", argv, got.stdout)
			}
		}
	})
}

// TestMachineFacingSubcommandsKeepTheirStdout pins the exemption. nav and
// complete reserve stdout for a protocol: nav's sentinel lines are captured by
// the generated shell function, and complete's lines go straight into the
// completion engine. A help page on either stdout would be swallowed or offered
// as a candidate, so both keep the argument handling they have today.
func TestMachineFacingSubcommandsKeepTheirStdout(t *testing.T) {
	nav := runHopCLI(t, newHopHome(t, englishConfig), "nav", "--help")
	if nav.code != 2 {
		t.Errorf("nav --help exit = %d, want 2 (usage error)\nstdout: %s\nstderr: %s", nav.code, nav.stdout, nav.stderr)
	}
	if !strings.Contains(nav.stderr, "unknown option") {
		t.Errorf("nav --help must keep answering on stderr, got:\n%s", nav.stderr)
	}
	if strings.Contains(nav.stdout, "Usage:") {
		t.Errorf("nav's stdout is a protocol, not a help page, got:\n%s", nav.stdout)
	}

	complete := runHopCLI(t, newHopHome(t, englishConfig), "complete", "--help")
	if strings.Contains(complete.stdout, "Usage:") {
		t.Errorf("complete's stdout feeds the completion engine, not a help page, got:\n%s", complete.stdout)
	}
}
