//go:build !windows

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubBin writes an executable shell script named `name` into a dir prepended
// to PATH, and returns a path the script appends its argv and stdin to.
//
// Stubs rather than mocks: this exercises the real fork/exec, the real kill on
// deadline, real exit codes and real stdout capture, through the same code path
// production takes — and spends no quota. Mocking runProcess would leave every
// one of those untested, which is the half most likely to be wrong.
func stubBin(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, name+".log")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"argv: $*\" >> " + log + "\n" +
		"cat >> " + log + " 2>/dev/null\n" +
		script + "\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func TestSubprocess_RunsTheHarnessForThePoolAndPassesTheTaskOnStdin(t *testing.T) {
	log := stubBin(t, "pi", `printf 'the answer'`)

	got := Subprocess{}.Dispatch(context.Background(), Request{
		Pool: "nan", Model: "deepseek-v4-flash", Task: "what is 2+2", Timeout: time.Minute,
	})

	if got.Status != StatusOK {
		t.Errorf("status = %q, want ok (output %q)", got.Status, got.Output)
	}
	if got.Output != "the answer" {
		t.Errorf("output = %q", got.Output)
	}
	logged := readLog(t, log)
	if !strings.Contains(logged, "--print") || !strings.Contains(logged, "--model deepseek-v4-flash") {
		t.Errorf("argv wrong: %q", logged)
	}
	// The task must NOT be on argv: it is world-readable through `ps` and
	// bounded by ARG_MAX, and it is arbitrary user text.
	if strings.Contains(strings.SplitN(logged, "\n", 2)[0], "what is 2+2") {
		t.Errorf("the task text was passed on argv: %q", logged)
	}
	if !strings.Contains(logged, "what is 2+2") {
		t.Errorf("the task text never reached the process at all: %q", logged)
	}
}

func TestSubprocess_ServesOnlyPoolsWhoseBinaryIsPresent(t *testing.T) {
	// A PATH with pi but no claude.
	stubBin(t, "pi", `printf x`)
	t.Setenv("PATH", filepath.Dir(mustLook(t, "pi")))

	s := Subprocess{}
	if !s.Serves("nan") {
		t.Error("nan not served with pi present")
	}
	if s.Serves("claude") {
		t.Error("claude served with no claude binary on PATH")
	}
	if s.Serves("copilot") {
		t.Error("copilot served; no harness is mapped to it")
	}
}

// The measured case, and the reason this rule exists: on a machine without the
// NaN credential, `pi --print` exits 0, warns that no model matches, and
// answers nothing. Reporting that as `ok` hands the caller a successful record
// containing no answer.
func TestSubprocess_ExitZeroWithNoOutputIsATaskFailure(t *testing.T) {
	stubBin(t, "pi", `printf 'No models match pattern "nan/deepseek-v4-flash"' >&2; exit 0`)

	got := Subprocess{}.Dispatch(context.Background(), Request{
		Pool: "nan", Model: "deepseek-v4-flash", Task: "t", Timeout: time.Minute,
	})

	if got.Status != StatusTaskFailed {
		t.Errorf("status = %q, want task_failed: a dispatch that answers nothing has not done the task", got.Status)
	}
	if !strings.Contains(got.Output, "No models match") {
		t.Errorf("output drops the stderr that explains the failure: %q", got.Output)
	}
}

// A harness that ran and failed is a TASK failure, never an unavailable pool:
// advancing the chain would retry a real failure on a different model.
func TestSubprocess_ANonZeroExitIsATaskFailureNotAnUnavailablePool(t *testing.T) {
	stubBin(t, "pi", `printf 'boom' >&2; exit 7`)

	got := Subprocess{}.Dispatch(context.Background(), Request{
		Pool: "nan", Model: "m", Task: "t", Timeout: time.Minute,
	})

	if got.Status != StatusTaskFailed {
		t.Errorf("status = %q, want task_failed", got.Status)
	}
	if got.Exit != 7 {
		t.Errorf("exit = %d, want the child's 7", got.Exit)
	}
}

// The deadline kills a real process. Without WaitDelay a child whose output
// pipe is held open would block Wait past the kill, which is what would quietly
// undo AC3's release-before-reap in the real case rather than the fake one.
func TestSubprocess_IsKilledOnTheDeadline(t *testing.T) {
	stubBin(t, "pi", `sleep 30; printf 'too late'`)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	got := Subprocess{}.Dispatch(ctx, Request{Pool: "nan", Model: "m", Task: "t", Timeout: time.Minute})
	elapsed := time.Since(start)

	if elapsed > 10*time.Second {
		t.Errorf("took %s: the child was not killed on the deadline", elapsed)
	}
	if strings.Contains(got.Output, "too late") {
		t.Error("the killed child's answer was kept")
	}
}

func TestHive_MapsTheDocumentedExitCodes(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   Status
	}{
		{name: "0 is an answer", script: `printf 'answered'; exit 0`, want: StatusOK},
		{name: "3 is the pool declining", script: `printf 'busy'; exit 3`, want: StatusPoolUnavailable},
		{name: "1 is the worker failing", script: `printf 'bad'; exit 1`, want: StatusTaskFailed},
		{
			name:   "an unrecognised code fails closed, never as unavailable",
			script: `printf 'who knows'; exit 42`, want: StatusTaskFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubBin(t, "hive", tc.script)
			got := Hive{}.Dispatch(context.Background(), Request{
				Pool: "nan", Model: "m", Task: "t", Timeout: time.Minute,
			})
			if got.Status != tc.want {
				t.Errorf("status = %q, want %q (output %q)", got.Status, tc.want, got.Output)
			}
		})
	}
}

func TestHive_PassesTheContractArguments(t *testing.T) {
	log := stubBin(t, "hive", `printf 'ok'`)

	Hive{}.Dispatch(context.Background(), Request{
		Pool: "nan", Model: "deepseek-v4-flash", Task: "the task", Timeout: 90 * time.Second,
	})

	logged := readLog(t, log)
	for _, want := range []string{"delegate", "--model deepseek-v4-flash", "--timeout 90", "--prompt the task"} {
		if !strings.Contains(logged, want) {
			t.Errorf("argv missing %q: %q", want, logged)
		}
	}
}

// The verb takes whole seconds, so a sub-second deadline must not truncate to
// zero — a zero timeout is not the deadline anyone asked for.
func TestHive_TimeoutIsWholeSecondsRoundedUp(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{90 * time.Second, "90"},
		{500 * time.Millisecond, "1"},
		{1500 * time.Millisecond, "2"},
		{0, "1"},
	}
	for _, tc := range tests {
		if got := hiveTimeoutSeconds(tc.in); got != tc.want {
			t.Errorf("hiveTimeoutSeconds(%s) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestHive_ServesOnlyNan(t *testing.T) {
	stubBin(t, "hive", `exit 0`)
	h := Hive{}
	if !h.Serves("nan") {
		t.Error("nan not served with hive present")
	}
	if h.Serves("claude") {
		t.Error("hive claims to serve claude; its worker is NaN-only")
	}
}

func mustLook(t *testing.T, bin string) string {
	t.Helper()
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		p := filepath.Join(dir, bin)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	t.Fatalf("%s not on PATH", bin)
	return ""
}
