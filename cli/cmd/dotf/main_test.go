package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/mlorentedev/dotfiles/cli/internal/errors"
	"github.com/spf13/cobra"
)

func TestRunTerminalFailure(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.NewTerminalFailure("test terminal failure")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if !strings.HasPrefix(out, errors.HandoffPrefix) {
		t.Errorf("expected output to start with %q, got %q", errors.HandoffPrefix, out)
	}
	if strings.Contains(out, "Error: ") {
		t.Errorf("expected output to not contain cobra 'Error: ' prefix, got %q", out)
	}
}

func TestRunNormalError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("regular error")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if !strings.HasPrefix(out, "Error: regular error") {
		t.Errorf("expected output to start with 'Error: regular error', got %q", out)
	}
}

func TestRunSilentError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:           "testcmd",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("silent error")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if out != "" {
		t.Errorf("expected silent error output to be empty, got %q", out)
	}
}

func TestRunWrappedTerminalFailure(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			tfe := errors.NewTerminalFailure("test terminal failure")
			return fmt.Errorf("some wrapper context: %w", tfe)
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	code := run(rootCmd, &stderr)

	if code == 0 {
		t.Errorf("expected non-zero exit code, got %d", code)
	}

	out := stderr.String()
	if !strings.HasPrefix(out, errors.HandoffPrefix) {
		t.Errorf("expected output to start with %q, got %q", errors.HandoffPrefix, out)
	}
	if strings.Contains(out, "some wrapper context") {
		t.Errorf("expected output to not contain wrapper text, got %q", out)
	}
}

// TestRunDoesNotDiscardDeliberateStderr is the guard for a regression that
// silenced nine command files at once.
//
// `run` used to do `rootCmd.SetErr(io.Discard)` to suppress Cobra's automatic
// "Error: ..." wrapper. But `cmd.ErrOrStderr()` resolves through the root, so it
// discarded every diagnostic any subcommand deliberately wrote — measured on
// d4ea0f5: 17 call sites across 9 files, all four `dotf secrets` subcommands
// among them. The sharpest case was `dotf harness gate`, which then blocked a
// tool call with exit 2 and NOTHING on stderr, leaving the operator no way to
// learn which skill to invoke: an enforcement mechanism whose refusal is
// indistinguishable from a crash.
//
// The two concerns share a sink, so they cannot be separated by writer. Cobra's
// SilenceErrors is the mechanism for the first, and this pins that using it did
// not cost the second.
func TestRunDoesNotDiscardDeliberateStderr(t *testing.T) {
	const diagnostic = "[cmd] a deliberate diagnostic"

	rootCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), diagnostic)
			return nil
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	if code := run(rootCmd, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), diagnostic) {
		t.Errorf("a command's deliberate stderr write was discarded; got %q.\n"+
			"Every `dotf` subcommand that explains itself on stderr depends on this.", stderr.String())
	}
}

// A command that writes a diagnostic AND fails must still get both out, with no
// duplicated Cobra wrapper.
func TestRunKeepsDiagnosticsAlongsideASilencedError(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:           "testcmd",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "[cmd] why it failed")
			return fmt.Errorf("boom")
		},
	}
	rootCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	if code := run(rootCmd, &stderr); code == 0 {
		t.Fatal("expected a non-zero exit")
	}
	out := stderr.String()
	if !strings.Contains(out, "[cmd] why it failed") {
		t.Errorf("the diagnostic was discarded: %q", out)
	}
	if strings.Contains(out, "Error: boom") {
		t.Errorf("a SilenceErrors command must not get Cobra's wrapper: %q", out)
	}
}

func TestResolveVersion(t *testing.T) {
	pseudo := "v0.0.0-20260930012845-db2b904794b4"
	vcs := []debug.BuildSetting{{Key: "vcs.revision", Value: "db2b904794b4700b1e9b5c784d7b570bdc46fae0"}}
	cases := []struct {
		name, ldflag string
		info         *debug.BuildInfo
		want         string
	}{
		{"release stamp wins", "0.61.1", &debug.BuildInfo{Main: debug.Module{Version: pseudo}}, "0.61.1"},
		{"no build info", "dev", nil, "dev"},
		{"checkout build", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: vcs}, "dev"},
		{"checkout build without vcs", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev"},
		{"empty module version", "dev", &debug.BuildInfo{}, "dev"},
		{"versioned checkout build", "dev", &debug.BuildInfo{Main: debug.Module{Version: pseudo}, Settings: vcs}, "dev"},
		{"module install", "dev", &debug.BuildInfo{Main: debug.Module{Version: pseudo}}, "0.0.0-20260930012845-db2b904794b4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.ldflag, tc.info); got != tc.want {
				t.Errorf("resolveVersion(%q) = %q, want %q", tc.ldflag, got, tc.want)
			}
		})
	}
}

// `dotf agent run` exits 3 when no pool could serve the dispatch, and composers
// branch on that number: 3 means "another machine may run it", 1 means "the task
// failed". The in-process tests in internal/cmd see the code on the returned
// error; only run() turns it into what main() exits with, so a regression there
// (every error exiting 1) leaves them green.
//
// It stands in for a bats case that ran the compiled binary with an empty PATH.
// What that case also covered, os.Exit(run(...)) being wired in main(), is the
// one-liner tests/dotf-agent-run.bats still pins through its refusal case, which
// reads a non-zero status out of the real binary.
func TestRunExitsWithTheCodeAnExhaustedChainCarries(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	// An identified machine, and a PATH with no harness binary: the walk finds no
	// transport for any entry. Without PATH emptied this dispatches for real.
	cfg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cfg, "dotfiles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	machine := `{"machine": {"id": "main-test"}, "pools": {"deny": []}}`
	if err := os.WriteFile(filepath.Join(cfg, "dotfiles", "machine.json"), []byte(machine), 0o600); err != nil {
		t.Fatalf("seed machine.json: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("PATH", t.TempDir())

	rootCmd := cmd.New("dev", "")
	rootCmd.SetArgs([]string{
		"agent", "run", "--role", "r", "--task", "t", "--tier", "mid",
		"--timeout", "1m", "--repo-root", root, "--semaphore-dir", t.TempDir(),
	})
	rootCmd.SetOut(&bytes.Buffer{})

	var stderr bytes.Buffer
	if code := run(rootCmd, &stderr); code != 3 {
		t.Errorf("exit code = %d, want 3 (chain exhausted); stderr: %s", code, stderr.String())
	}
}
