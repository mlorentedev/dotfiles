//go:build unix

package evals

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// shellLauncher runs the attempt through the real spec.RunWithLimits, with the
// candidate's argv swapped for a shell script, so the pipe, the sink and the
// process handling are the ones a real run goes through.
func shellLauncher(root, script string, deadline time.Duration) ExecLauncher {
	l := testLauncher(root, &fakeRunner{}, nil)
	l.Timeout = deadline
	l.Deps.Run = func(lim spec.Limits, _ []string, in io.Reader, out, errw io.Writer) (spec.RunResult, error) {
		return spec.RunWithLimits(lim, []string{"sh", "-c", script}, in, out, errw)
	}
	return l
}

// launchWithin fails the test instead of hanging it when Launch never returns.
func launchWithin(t *testing.T, l ExecLauncher, a Attempt, d time.Duration) (LaunchResult, error) {
	t.Helper()
	type out struct {
		res LaunchResult
		err error
	}
	done := make(chan out, 1)
	go func() {
		res, err := l.Launch(context.Background(), a)
		done <- out{res, err}
	}()
	select {
	case o := <-done:
		return o.res, o.err
	case <-time.After(d):
		t.Fatalf("Launch did not return within %s", d)
		return LaunchResult{}, nil
	}
}

func TestLaunchThroughARealRunnerKeepsTheAnswer(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	script := `printf '%s\n' '{"type":"message_end"}'; printf 'the answer\n' > ` + spec.EvalAnswerDir + "/" + spec.EvalSlot.Review
	res, err := launchWithin(t, shellLauncher(root, script, time.Minute), a, 30*time.Second)
	if err != nil || res.ExitErr != nil {
		t.Fatalf("err %v, result %+v", err, res)
	}
	if got, err := os.ReadFile(filepath.Join(a.OutDir, AnswerFile)); err != nil || string(got) != "the answer\n" {
		t.Errorf("answer kept = %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(a.OutDir, spec.EvalSlot.Transcript)); err != nil || !strings.Contains(string(got), "message_end") {
		t.Errorf("transcript = %q, %v", got, err)
	}
}

// A transcript that cannot be written must not leave the runner writing into a
// pipe nobody reads: the attempt ends, as an error, instead of hanging.
func TestLaunchReturnsWhenTheTranscriptCannotBeWritten(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	if err := os.MkdirAll(filepath.Join(a.OutDir, spec.EvalSlot.Transcript), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `while :; do printf '%s\n' '{"type":"message_update","pad":"................................................................"}'; done`
	_, err := launchWithin(t, shellLauncher(root, script, 3*time.Second), a, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "transcript") {
		t.Errorf("err = %v, want the transcript failure", err)
	}
}

// A runner the deadline cuts ends the attempt on time even when a child it
// started still holds its stdout, and the tree goes with it.
func TestLaunchReturnsWhenTheDeadlineCutsARunnerWithChildren(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	res, err := launchWithin(t, shellLauncher(root, "sleep 60 & wait", time.Second), a, 15*time.Second)
	if err != nil || !res.TimedOut {
		t.Errorf("err %v, result %+v, want a timeout", err, res)
	}
	if _, err := os.Stat(a.Tree); !os.IsNotExist(err) {
		t.Errorf("the tree outlived the attempt: %v", err)
	}
}
