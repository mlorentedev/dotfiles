package converge

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// completionRun answers `<tool> completion zsh` from a table and counts calls.
func completionRun(out map[string]string) func(string, ...string) ([]byte, error) {
	return func(name string, args ...string) ([]byte, error) {
		if strings.Join(args, " ") != "completion zsh" {
			return nil, errors.New("unexpected args: " + strings.Join(args, " "))
		}
		s, ok := out[name]
		if !ok {
			return nil, errors.New(name + ": not found")
		}
		return []byte(s), nil
	}
}

func completionsStep(out map[string]string, on ...string) shellCompletions {
	have := map[string]bool{"zsh": true}
	for _, n := range on {
		have[n] = true
	}
	return shellCompletions{run: completionRun(out), has: func(c string) bool { return have[c] }}
}

func TestShellCompletions_WritesBothThenReportsNoChange(t *testing.T) {
	home := t.TempDir()
	env := Env{Home: home, GOOS: "darwin"}
	step := completionsStep(map[string]string{"dotf": "#compdef dotf\n", "herdr": "#compdef herdr\n"}, "dotf", "herdr")

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 2 {
		t.Fatalf("first run: changes=%d err=%v detail=%q", res.Changes, err, res.Detail)
	}
	for tool, want := range map[string]string{"dotf": "#compdef dotf\n", "herdr": "#compdef herdr\n"} {
		got, err := os.ReadFile(filepath.Join(home, ".zsh", "completions", "_"+tool))
		if err != nil || string(got) != want {
			t.Errorf("_%s = %q, %v", tool, got, err)
		}
	}
	if res, err = step.Reconcile(env, false); err != nil || res.Changes != 0 {
		t.Errorf("second run: changes=%d err=%v, want changed=0", res.Changes, err)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
}

func TestShellCompletions_ADriftedFileIsRewritten(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".zsh", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_dotf"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	step := completionsStep(map[string]string{"dotf": "#compdef dotf\nnew\n"}, "dotf")
	res, err := step.Reconcile(Env{Home: home}, false)
	if err != nil || res.Changes != 1 {
		t.Fatalf("changes=%d err=%v", res.Changes, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "_dotf")); string(got) != "#compdef dotf\nnew\n" {
		t.Errorf("_dotf = %q", got)
	}
}

func TestShellCompletions_ADryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	step := completionsStep(map[string]string{"dotf": "#compdef dotf\n"}, "dotf")
	res, err := step.Reconcile(Env{Home: home}, true)
	if err != nil || res.Changes != 1 || !strings.Contains(res.Detail, "to write _dotf") {
		t.Fatalf("changes=%d detail=%q err=%v", res.Changes, res.Detail, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zsh")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a dry run created %s/.zsh (%v)", home, err)
	}
}

func TestShellCompletions_ASkippedToolIsNamedAndItsFileKept(t *testing.T) {
	home := t.TempDir()
	step := completionsStep(map[string]string{"dotf": "#compdef dotf\n"}, "dotf")
	res, err := step.Reconcile(Env{Home: home}, false)
	if err != nil || res.Changes != 1 || !strings.Contains(res.Detail, "skipped herdr") {
		t.Fatalf("changes=%d detail=%q err=%v", res.Changes, res.Detail, err)
	}
}

func TestShellCompletions_SkipsWhenNothingCanRun(t *testing.T) {
	env := Env{Home: t.TempDir()}
	for name, step := range map[string]shellCompletions{
		"no zsh":          {run: completionRun(nil), has: func(c string) bool { return c != "zsh" }},
		"neither tool":    completionsStep(nil),
		"no runner wired": {has: func(string) bool { return true }},
	} {
		if res, err := step.Reconcile(env, false); err != nil || res.Skip == "" {
			t.Errorf("%s: skip=%q err=%v, want a skip reason", name, res.Skip, err)
		}
	}
}

func TestShellCompletions_AnEmptyOrFailingCompletionIsAnError(t *testing.T) {
	env := Env{Home: t.TempDir()}
	if _, err := completionsStep(map[string]string{"dotf": " \n"}, "dotf").Reconcile(env, false); err == nil || !strings.Contains(err.Error(), "printed nothing") {
		t.Errorf("empty completion: err=%v", err)
	}
	if _, err := completionsStep(nil, "dotf").Reconcile(env, false); err == nil || !strings.Contains(err.Error(), "dotf completion zsh") {
		t.Errorf("failing completion: err=%v", err)
	}
}

// TestExecStdoutHelperProcess is not a test: re-exec'd with the env var set, it
// writes one line to each stream, the way a tool prints a deprecation warning
// beside its completion script.
func TestExecStdoutHelperProcess(t *testing.T) {
	if os.Getenv("DOTF_EXECSTDOUT_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("#compdef dotf\n")
	_, _ = os.Stderr.WriteString("warning: deprecated flag\n")
	os.Exit(0)
}

// #2686: every other test injects a fake runner, so this is the one that runs
// ExecStdout against a real child. Appending stderr would write the tool's
// warning into the file compinit loads.
func TestExecStdout_KeepsAToolsStderrOutOfTheCompletion(t *testing.T) {
	t.Setenv("DOTF_EXECSTDOUT_HELPER", "1")
	args := []string{"-test.run=^TestExecStdoutHelperProcess$"}

	both, err := exec.Command(os.Args[0], args...).CombinedOutput() //nolint:gosec // the test binary itself
	if err != nil || !strings.Contains(string(both), "warning: deprecated flag") {
		t.Fatalf("helper must write to stderr for this test to mean anything: err=%v out=%q", err, both)
	}

	out, err := ExecStdout(os.Args[0], args...)
	if err != nil {
		t.Fatalf("ExecStdout: %v", err)
	}
	if string(out) != "#compdef dotf\n" {
		t.Errorf("ExecStdout returned %q, want stdout alone", out)
	}
}
