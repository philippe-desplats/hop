package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZshIntegrationDefaultName(t *testing.T) {
	out := zshIntegration("p")
	for _, want := range []string{"unalias p 2>/dev/null", "function p {", "add-zsh-hook chpwd _hop_chpwd"} {
		if !strings.Contains(out, want) {
			t.Errorf("integration missing %q", want)
		}
	}
}

func TestZshIntegrationCustomName(t *testing.T) {
	out := zshIntegration("pp")
	if !strings.Contains(out, "function pp {") || !strings.Contains(out, "unalias pp 2>/dev/null") {
		t.Errorf("custom name not applied:\n%s", out)
	}
	if strings.Contains(out, "function p {") {
		t.Error("default name leaked into custom integration")
	}
	// Internal helpers stay fixed regardless of the public name.
	if !strings.Contains(out, "function _hop_chpwd {") {
		t.Error("internal hook name should not change")
	}
}

// TestCmdInitSubprocess re-executes the test binary so cmdInit's os.Exit paths
// run for real, and so stdout and stderr can be told apart: the contract here is
// that the integration lands on stdout (the rc file evals it) while a diagnostic
// about the config lands on stderr.
func TestCmdInitSubprocess(t *testing.T) {
	if os.Getenv("HOP_INIT_CHILD") == "1" {
		cmdInit(decodeArgv(os.Getenv("HOP_INIT_ARGV")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	// run writes configCmd as the [shell] command value, then runs `hop init argv`
	// in a child against that config. An empty configCmd writes no config at all.
	run := func(t *testing.T, configCmd string, argv ...string) (stdout, stderr string, code int, configPath string) {
		t.Helper()
		configHome := t.TempDir()
		configPath = filepath.Join(configHome, "hop", "config.toml")
		if configCmd != "" {
			if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
				t.Fatal(err)
			}
			toml := "[shell]\ncommand = \"" + configCmd + "\"\n"
			if err := os.WriteFile(configPath, []byte(toml), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdInitSubprocess")
		cmd.Env = []string{
			"HOP_INIT_CHILD=1",
			"HOP_INIT_ARGV=" + encodeArgv(t, argv),
			"HOME=" + t.TempDir(),
			"XDG_CONFIG_HOME=" + configHome,
			"PATH=" + t.TempDir(),       // empty: init never shells out
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
		err := cmd.Run()
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v\n%s", err, errBuf.String())
		}
		return outBuf.String(), errBuf.String(), code, configPath
	}

	t.Run("an invalid config command warns and falls back to p", func(t *testing.T) {
		stdout, stderr, code, configPath := run(t, "my-p", "zsh")
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (a bad config value must not break init)\nstderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "function p {") {
			t.Errorf("stdout must carry the zsh integration for the default name p, got:\n%s", stdout)
		}
		if !strings.Contains(stderr, configPath) {
			t.Errorf("stderr must name the config file %q, got:\n%s", configPath, stderr)
		}
		if !strings.Contains(stderr, "my-p") {
			t.Errorf("stderr must name the rejected value, got:\n%s", stderr)
		}
	})

	t.Run("an invalid --cmd value is still a usage error", func(t *testing.T) {
		stdout, stderr, code, _ := run(t, "", "zsh", "--cmd", "my-p")
		if code != 2 {
			t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr)
		}
		if stdout != "" {
			t.Errorf("a refused invocation must print nothing on stdout, got:\n%s", stdout)
		}
		if !strings.Contains(stderr, "invalid function name") {
			t.Errorf("stderr must keep its current message, got:\n%s", stderr)
		}
	})

	t.Run("a valid config command is honoured", func(t *testing.T) {
		stdout, stderr, code, _ := run(t, "myjump", "zsh")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "function myjump {") {
			t.Errorf("a valid config value must still be honoured, got:\n%s", stdout)
		}
	})
}

func TestCmdNameValidation(t *testing.T) {
	valid := []string{"p", "pp", "go_proj", "_x", "Hop2"}
	invalid := []string{"", "2p", "p p", "p-x", "p.x", "p/x"}
	for _, n := range valid {
		if !cmdNameRe.MatchString(n) {
			t.Errorf("%q should be valid", n)
		}
	}
	for _, n := range invalid {
		if cmdNameRe.MatchString(n) {
			t.Errorf("%q should be invalid", n)
		}
	}
}
