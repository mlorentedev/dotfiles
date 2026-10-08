package main

import (
	"bytes"
	"encoding/json"
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

// Every command reports a mistyped flag, whether or not it silences its own
// errors (#2090). Measured before the fix: eight commands, `doctor` and
// `pr land` among them, exited 1 with nothing on stderr.
func TestEveryCommandReportsAnUnknownFlag(t *testing.T) {
	for _, path := range commandPaths(cmd.New("test", "")) {
		name := strings.Join(path, " ")
		t.Run(name, func(t *testing.T) {
			root := cmd.New("test", "")
			found, _, err := root.Find(path)
			if err != nil {
				t.Fatal(err)
			}
			if found.DisableFlagParsing || found.FParseErrWhitelist.UnknownFlags {
				t.Skipf("dotf %s does not parse flags, so an unknown one would reach its code", name)
			}
			root.SetArgs(append(append([]string{}, path...), "--no-such-flag"))
			var stderr bytes.Buffer
			root.SetOut(&stderr)
			code := run(root, &stderr)
			if code == 0 {
				t.Errorf("dotf %s --no-such-flag exited 0", name)
			}
			if !strings.Contains(stderr.String(), "unknown flag: --no-such-flag") {
				t.Errorf("dotf %s --no-such-flag did not report the flag; stderr: %q", name, stderr.String())
			}
		})
	}
}

// A silenced command's Args refusal is reported too, and its own diagnostics
// are still not printed twice.
func TestASilencedCommandReportsArgumentsItRefuses(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"pr", "land"}, "requires at least 1 arg(s)"},
		{[]string{"pr", "land", "abc"}, `"abc" is not a PR number`},
		{[]string{"pr", "land", "7", "#7"}, "#7 is listed twice"},
	} {
		root := cmd.New("test", "")
		root.SetArgs(tc.args)
		var stderr bytes.Buffer
		if code := run(root, &stderr); code == 0 {
			t.Errorf("dotf %v exited 0", tc.args)
		}
		if got := stderr.String(); strings.Count(got, tc.want) != 1 {
			t.Errorf("dotf %v: want %q reported exactly once, stderr: %q", tc.args, tc.want, got)
		}
	}
}

// An unknown top-level command is refused by Cobra's own root check, before
// any Args validator, so markUsageErrors never sees it. It is printed only
// because the root does not silence its errors; this pins that.
func TestAnUnknownTopLevelCommandIsReported(t *testing.T) {
	root := cmd.New("test", "")
	root.SetArgs([]string{"nosuch"})
	var stderr bytes.Buffer
	if code := run(root, &stderr); code == 0 {
		t.Error("dotf nosuch exited 0")
	}
	if !strings.Contains(stderr.String(), `unknown command "nosuch"`) {
		t.Errorf("dotf nosuch did not report the command; stderr: %q", stderr.String())
	}
}

// Every command group refuses an unknown subcommand, and still prints its help
// with no argument (#2091). Measured before the fix: `dotf pr nosuch`,
// `dotf vault nosuch` and the rest printed help and exited 0.
//
// A group that declares its own Args takes positional arguments on purpose
// (`dotf init [path]` scaffolds a project), so it is never run here. The rest
// run in a scratch cwd and HOME, and with every path the env contract declares
// pointed at scratch, so a group whose RunE ever does more than print help
// cannot write into the checkout or into the real vault.
// sandboxContractPaths points every variable env-contract.json declares, and the
// XDG roots, at its own scratch directory. A scratch HOME alone does not do
// this: env.ResolvePath returns a set variable before it consults HOME, so a
// developer shell's VAULT_PATH reached the real vault. That is how this walk's
// first version left two scaffolded projects there on 2026-10-07 (lesson 350).
// The names come from the contract, so a path variable added later is covered
// without anyone remembering this test.
func sandboxContractPaths(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		EnvVars []struct {
			Name string `json:"name"`
		} `json:"env_vars"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	names := []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"}
	for _, v := range contract.EnvVars {
		names = append(names, v.Name)
	}
	if len(names) < 5 || !strings.Contains(strings.Join(names, " "), "VAULT_PATH") {
		t.Fatalf("env-contract.json declared no VAULT_PATH; parsed %v", names)
	}
	for _, name := range names {
		t.Setenv(name, t.TempDir())
	}
}

func TestEveryCommandGroupRefusesAnUnknownSubcommand(t *testing.T) {
	sandboxContractPaths(t)
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	groups := 0
	for _, path := range commandPaths(cmd.New("test", "")) {
		root := cmd.New("test", "")
		found, _, err := root.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		if !found.HasSubCommands() || found.Args != nil {
			continue
		}
		groups++
		name := strings.Join(path, " ")
		t.Run(name, func(t *testing.T) {
			root.SetArgs(append(append([]string{}, path...), "nosuch"))
			var stderr bytes.Buffer
			root.SetOut(&stderr)
			if code := run(root, &stderr); code == 0 {
				t.Errorf("dotf %s nosuch exited 0", name)
			}
			if want := fmt.Sprintf("unknown command %q for %q", "nosuch", "dotf "+name); !strings.Contains(stderr.String(), want) {
				t.Errorf("dotf %s nosuch: want %q, stderr: %q", name, want, stderr.String())
			}

			bare := cmd.New("test", "")
			bare.SetArgs(path)
			var out bytes.Buffer
			bare.SetOut(&out)
			if code := run(bare, &out); code != 0 {
				t.Errorf("dotf %s with no argument exited %d; want its help and 0", name, code)
			}
			if !strings.Contains(out.String(), "Available Commands:") {
				t.Errorf("dotf %s with no argument did not print its help: %q", name, out.String())
			}
		})
	}
	if groups == 0 {
		t.Fatal("found no command groups; the walk is broken")
	}
}
