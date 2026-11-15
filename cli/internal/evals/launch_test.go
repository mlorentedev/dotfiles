package evals

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// fakeRunner stands in for spec.RunWithLimits: it sees what the launcher
// would run and acts as the runner, from inside the tree.
type fakeRunner struct {
	limits spec.Limits
	argv   []string
	act    func(tree string, stdout io.Writer) (spec.RunResult, error)
}

func (f *fakeRunner) run(l spec.Limits, argv []string, _ io.Reader, stdout, _ io.Writer) (spec.RunResult, error) {
	f.limits, f.argv = l, argv
	if f.act == nil {
		return spec.RunResult{}, nil
	}
	return f.act(l.Dir, stdout)
}

// writesAnswer acts as a runner that writes body to the answer file it was told
// about, and streams one event.
func writesAnswer(answerPath *string, body string) func(string, io.Writer) (spec.RunResult, error) {
	return func(_ string, stdout io.Writer) (spec.RunResult, error) {
		_, _ = fmt.Fprintln(stdout, `{"type":"message_end","text":"done"}`)
		return spec.RunResult{}, os.WriteFile(*answerPath, []byte(body), 0o600)
	}
}

const plantedPrompt = "# Review this\n\n## How to answer\n\nverdict and a table.\n"

func plantedRoot(t *testing.T) (string, Case) {
	t.Helper()
	root := t.TempDir()
	rel := "harness/evals/planted/p1/prompt.md"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte(plantedPrompt), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, Case{ID: "p1", Kind: "planted", Prompt: rel, KeyAuthor: "human", ExpectedVerdict: "FAIL"}
}

func testLauncher(root string, r *fakeRunner, checkout func(Case, string, int) (Tree, error)) ExecLauncher {
	return ExecLauncher{
		Root: root, Timeout: time.Minute, Stall: 30 * time.Second,
		Candidates: map[string]spec.ReviewerEntry{"agy/x": {ID: "agy/x", Runner: "agy", Model: "x"}},
		Deps:       LaunchDeps{Run: r.run, Checkout: checkout},
	}
}

func attemptIn(dir string, c Case) Attempt {
	return attemptFor(dir, c, "agy/x", 1)
}

// A planted attempt runs in an empty tree that already holds the answer
// folder, keeps the answer and the transcript in its output directory, and
// leaves no tree behind.
func TestLaunchRunsAPlantedCaseAndKeepsItsAnswer(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	answerPath := filepath.Join(a.Tree, spec.EvalAnswerDir, spec.EvalSlot.Review)
	r := &fakeRunner{}
	r.act = func(tree string, stdout io.Writer) (spec.RunResult, error) {
		if fi, err := os.Stat(filepath.Dir(answerPath)); err != nil || !fi.IsDir() {
			t.Errorf("the answer folder must exist before the runner starts: %v", err)
		}
		return writesAnswer(&answerPath, answer("FAIL"))(tree, stdout)
	}
	res, err := testLauncher(root, r, nil).Launch(context.Background(), a)
	if err != nil || res.ExitErr != nil {
		t.Fatalf("err %v, result %+v", err, res)
	}
	if !strings.Contains(strings.Join(r.argv, " "), "# Review this") {
		t.Errorf("the planted prompt was not sent: %q", r.argv)
	}
	if r.limits.Dir != a.Tree || r.limits.Deadline != time.Minute || r.limits.StallTimeout != 30*time.Second || r.limits.Transcript != filepath.Join(a.OutDir, spec.EvalSlot.Transcript) {
		t.Errorf("limits = %+v", r.limits)
	}
	if got, err := os.ReadFile(filepath.Join(a.OutDir, AnswerFile)); err != nil || !strings.Contains(string(got), "verdict") {
		t.Errorf("answer kept = %q, %v", got, err)
	}
	if got, err := os.ReadFile(r.limits.Transcript); err != nil || !strings.Contains(string(got), "message_end") {
		t.Errorf("transcript = %q, %v", got, err)
	}
	if _, err := os.Stat(a.Tree); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the tree outlived the attempt: %v", err)
	}
}

// A replay attempt checks out its case where the run loop expects the tree,
// and removes the checkout however the runner ended.
func TestLaunchRemovesTheReplayCheckoutHoweverTheRunEnds(t *testing.T) {
	c := Case{ID: "a-r1", Kind: "replay", Spec: "AI-055", ReviewedSHA: strings.Repeat("a", 40), BaseSHA: strings.Repeat("b", 40)}
	for _, tc := range []struct {
		name string
		act  func(string, io.Writer) (spec.RunResult, error)
	}{
		{"an answer", func(string, io.Writer) (spec.RunResult, error) { return spec.RunResult{}, nil }},
		{"a signal", func(string, io.Writer) (spec.RunResult, error) { return spec.RunResult{Code: 130, Signaled: true}, nil }},
		{"a runner that would not start", func(string, io.Writer) (spec.RunResult, error) {
			return spec.RunResult{Code: -1}, errors.New("exec: agy not found")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := attemptIn(t.TempDir(), c)
			removed := false
			checkout := func(got Case, runDir string, epoch int) (Tree, error) {
				dir := filepath.Join(runDir, fmt.Sprintf("%s-%d", got.ID, epoch))
				if dir != a.Tree {
					t.Errorf("checkout at %s, the loop expects %s", dir, a.Tree)
				}
				return Tree{Dir: dir, Remove: func() error { removed = true; return os.RemoveAll(dir) }}, os.MkdirAll(dir, 0o755)
			}
			r := &fakeRunner{act: tc.act}
			_, _ = testLauncher(t.TempDir(), r, checkout).Launch(context.Background(), a)
			if !removed {
				t.Error("the replay checkout was not removed")
			}
			if len(r.argv) > 0 && !strings.Contains(strings.Join(r.argv, " "), "AI-055") {
				t.Errorf("the replay prompt does not name its spec: %q", r.argv)
			}
		})
	}
}

// How the runner ended reaches the loop as the classes it records.
func TestLaunchReportsHowTheRunnerEnded(t *testing.T) {
	root, c := plantedRoot(t)
	for _, tc := range []struct {
		name   string
		res    spec.RunResult
		runErr error
		check  func(LaunchResult, error) bool
	}{
		{"the deadline", spec.RunResult{Code: spec.ExitDeadline, Stop: spec.StopDeadline}, nil,
			func(r LaunchResult, err error) bool { return err == nil && r.TimedOut && !r.Stalled && !r.Interrupted }},
		{"a stall", spec.RunResult{Code: spec.ExitStalled, Stop: spec.StopStall, LastCall: "read_file"}, nil,
			func(r LaunchResult, err error) bool {
				return err == nil && r.Stalled && r.ExitErr != nil && strings.Contains(r.ExitErr.Error(), "read_file")
			}},
		{"a signal", spec.RunResult{Code: 130, Signaled: true}, nil,
			func(r LaunchResult, err error) bool { return err == nil && r.Interrupted && !r.TimedOut }},
		{"its own exit 130", spec.RunResult{Code: 130}, nil,
			func(r LaunchResult, err error) bool { return err == nil && !r.Interrupted && r.ExitErr != nil }},
		{"a runner that would not start", spec.RunResult{Code: -1}, errors.New("exec: agy not found"),
			func(r LaunchResult, err error) bool {
				return err != nil && strings.Contains(err.Error(), "agy not found")
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &fakeRunner{act: func(string, io.Writer) (spec.RunResult, error) { return tc.res, tc.runErr }}
			res, err := testLauncher(root, r, nil).Launch(context.Background(), attemptIn(t.TempDir(), c))
			if !tc.check(res, err) {
				t.Errorf("result %+v, err %v", res, err)
			}
		})
	}
}

// A runner that leaves a symlink where its answer goes does not get the file it
// points at read as its answer.
func TestLaunchDoesNotFollowAnAnswerSymlink(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte(answer("PASS")), 0o600); err != nil {
		t.Fatal(err)
	}
	answerPath := filepath.Join(a.Tree, spec.EvalAnswerDir, spec.EvalSlot.Review)
	r := &fakeRunner{act: func(string, io.Writer) (spec.RunResult, error) {
		if err := os.Symlink(outside, answerPath); err != nil {
			t.Skipf("cannot create a symlink here: %v", err)
		}
		return spec.RunResult{}, nil
	}}
	res, err := testLauncher(root, r, nil).Launch(context.Background(), a)
	if err != nil || res.ExitErr == nil || !strings.Contains(res.ExitErr.Error(), "not a regular file") {
		t.Fatalf("result %+v, err %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(a.OutDir, AnswerFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the symlink's target was kept as the answer: %v", err)
	}
}

func TestLaunchRefusesWhatItCannotRun(t *testing.T) {
	root, c := plantedRoot(t)
	l := testLauncher(root, &fakeRunner{}, nil)
	a := attemptIn(t.TempDir(), c)
	a.Candidate = "nan/unknown"
	if _, err := l.Launch(context.Background(), a); err == nil || !strings.Contains(err.Error(), "nan/unknown") {
		t.Errorf("an unknown candidate: %v", err)
	}
	c.Prompt = "../outside.md"
	if _, err := l.Launch(context.Background(), attemptIn(t.TempDir(), c)); err == nil {
		t.Error("a prompt outside the root was read")
	}
}

// An answer we fail to keep is our failure, not the model's.
func TestLaunchReportsAnAnswerItCouldNotKeepAsItsOwnFailure(t *testing.T) {
	root, c := plantedRoot(t)
	a := attemptIn(t.TempDir(), c)
	answerPath := filepath.Join(a.Tree, spec.EvalAnswerDir, spec.EvalSlot.Review)
	r := &fakeRunner{act: func(tree string, stdout io.Writer) (spec.RunResult, error) {
		// the kept answer's place is taken by a directory, so it cannot be written
		if err := os.MkdirAll(filepath.Join(a.OutDir, AnswerFile), 0o755); err != nil {
			t.Fatal(err)
		}
		return writesAnswer(&answerPath, answer("FAIL"))(tree, stdout)
	}}
	if _, err := testLauncher(root, r, nil).Launch(context.Background(), a); err == nil {
		t.Fatal("a failure to keep the answer must be returned as the launcher's own error")
	}
}

// A launcher with no limits set runs under the defaults `spec review` uses: a
// zero deadline would stop every runner the moment it started.
func TestLaunchDefaultsLimitsLeftUnset(t *testing.T) {
	root, c := plantedRoot(t)
	r := &fakeRunner{}
	l := testLauncher(root, r, nil)
	l.Timeout, l.Stall = 0, 0
	if _, err := l.Launch(context.Background(), attemptIn(t.TempDir(), c)); err != nil {
		t.Fatal(err)
	}
	if r.limits.Deadline != spec.DefaultReviewerTimeout || r.limits.StallTimeout != spec.DefaultStallTimeout {
		t.Errorf("limits = %+v, want the review defaults", r.limits)
	}
	if !strings.Contains(strings.Join(r.argv, " "), "Time budget:") {
		t.Errorf("the prompt names no budget: %q", r.argv)
	}
}

// A tree that will not go away is the operator's problem, not the model's: the
// answer the run paid for is still graded, and the record says what was left.
func TestATreeThatWillNotGoAwayKeepsTheGrade(t *testing.T) {
	c := Case{ID: "a-r1", Kind: "replay", Spec: "AI-055", ReviewedSHA: strings.Repeat("a", 40), BaseSHA: strings.Repeat("b", 40),
		ExpectedVerdict: "FAIL", KeyAuthor: "human"}
	a := attemptIn(t.TempDir(), c)
	answerPath := filepath.Join(a.Tree, spec.EvalAnswerDir, spec.EvalSlot.Review)
	checkout := func(_ Case, _ string, _ int) (Tree, error) {
		return Tree{Dir: a.Tree, Remove: func() error { return errors.New("device busy") }}, os.MkdirAll(a.Tree, 0o755)
	}
	r := &fakeRunner{act: writesAnswer(&answerPath, answer("FAIL"))}
	rec := grade(context.Background(), a, testLauncher(t.TempDir(), r, checkout))
	if rec.Completion != CompletionAnswered || rec.Grade == nil {
		t.Fatalf("record = %+v, want the answer graded", rec)
	}
	if !strings.Contains(rec.Detail, "device busy") || !strings.Contains(rec.Detail, a.Tree) {
		t.Errorf("detail = %q, want the tree left behind named", rec.Detail)
	}
}
