//go:build unix

package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/evals"
)

// The command's real composition, end to end, with PATH holding only stubs: a
// `dotf` that verifies every secret and runs what follows `--`, and a `pi`
// that answers in the file the prompt names. Nothing real can be reached, so
// this is the one test that runs `dotf eval run` without --plan.
func TestEvalRunLaunchesThroughStubRunners(t *testing.T) {
	bin := t.TempDir()
	if git, err := exec.LookPath("git"); err == nil {
		if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
			t.Fatal(err)
		}
	}
	stub := func(name, body string) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub("dotf", `if [ "$1 $2" = "secrets verify" ]; then exit 0; fi
while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do shift; done
shift
exec "$@"
`)
	cwdFile := filepath.Join(t.TempDir(), "pi-cwd")
	stub("pi", `pwd -P > '`+cwdFile+`'
printf '%s\n' '{"type":"message_end"}'
printf '%s\n' '| # | Severity | Finding |' '|---|---|---|' '| 1 | Major | count never increments |' > .dotf-eval/review.md
`)
	t.Setenv("PATH", bin)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	stdout, _, err := execute(t, "eval", "run", "--repo-root", evalRepoRoot(t),
		"--candidate", "nan/glm5.3-flash", "--case", "bash-count-increment", "--epochs", "1")
	if err != nil {
		t.Fatalf("eval run: %v\n%s", err, stdout)
	}
	runs, err := filepath.Glob(filepath.Join(state, "dotfiles", "evals", "*", "runs", "*", "bash-count-increment", "1", "record.json"))
	if err != nil || len(runs) != 1 {
		t.Fatalf("record.json: %v %v\n%s", runs, err, stdout)
	}
	raw, err := os.ReadFile(runs[0])
	if err != nil {
		t.Fatal(err)
	}
	var rec evals.Record
	if err := json.Unmarshal(raw, &rec); err != nil || rec.Completion != evals.CompletionAnswered {
		t.Errorf("record = %s (%v)", raw, err)
	}
	// The tree is the directory the runner actually ran in, as the stub saw
	// it, so the check cannot pass by looking somewhere the tree never was.
	cwd, err := os.ReadFile(cwdFile)
	if err != nil || strings.TrimSpace(string(cwd)) == "" {
		t.Fatalf("the stub runner recorded no working directory: %v", err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(cwd))); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the attempt's tree was left behind: %s (%v)", cwd, err)
	}
	if !strings.Contains(stdout, "1 attempts: 1 answered") {
		t.Errorf("the report lacks the count:\n%s", stdout)
	}
}

// The command's own signal wiring, in a child process that receives real
// interrupts: the first cancels the run, and a later one must kill the
// process instead of being swallowed while an attempt runs to its deadline.
func TestEvalRunInterruptContextLetsASecondInterruptKill(t *testing.T) {
	if os.Getenv("DOTF_EVAL_INTERRUPT_CHILD") == "1" {
		ctx, stop := interruptContext(context.Background())
		defer stop()
		fmt.Println("ready")
		<-ctx.Done()
		fmt.Println("cancelled")
		time.Sleep(time.Minute)
		os.Exit(3)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestEvalRunInterruptContextLetsASecondInterruptKill$")
	child.Env = append(os.Environ(), "DOTF_EVAL_INTERRUPT_CHILD=1")
	out, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	lines := bufio.NewScanner(out)
	expectLine(t, lines, "ready")
	if err := child.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	expectLine(t, lines, "cancelled")
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	interruptUntilExit(t, child, done)
}

func expectLine(t *testing.T, s *bufio.Scanner, want string) {
	t.Helper()
	if !s.Scan() || s.Text() != want {
		t.Fatalf("child printed %q, want %q (%v)", s.Text(), want, s.Err())
	}
}

// interruptUntilExit keeps interrupting, since the release runs just after
// the cancel, and fails if the child outlives five seconds of interrupts.
func interruptUntilExit(t *testing.T, child *exec.Cmd, done <-chan error) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			var exit *exec.ExitError
			if !errors.As(err, &exit) || !exit.Sys().(syscall.WaitStatus).Signaled() {
				t.Fatalf("the child did not die by the signal: %v", err)
			}
			return
		case <-tick.C:
			_ = child.Process.Signal(os.Interrupt)
		case <-deadline:
			_ = child.Process.Kill()
			t.Fatal("a second interrupt did not kill the run: the handler still swallows it")
		}
	}
}
