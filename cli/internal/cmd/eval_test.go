package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/evals"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

func TestEvalCheckPassesARepositoryWithoutCases(t *testing.T) {
	stdout, _, err := execute(t, "eval", "check", "--repo-root", t.TempDir())
	if err != nil || !strings.Contains(stdout, "no eval cases") {
		t.Fatalf("a repository without cases: err %v\n%s", err, stdout)
	}
}

func TestEvalCheckFailsAndNamesABrokenCase(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".dotf", "evals", "cases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"id": "bad", "kind": "x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, "eval", "check", "--repo-root", root)
	if err == nil {
		t.Fatalf("a broken case passed:\n%s", stdout)
	}
	if !strings.Contains(stdout, ".dotf/evals/cases/bad.json") || !strings.Contains(stdout, "schema") {
		t.Fatalf("the report must name the case file and its problem:\n%s", stdout)
	}
}

func evalRepoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	return filepath.Join(filepath.Dir(here), "..", "..", "..")
}

// --plan prints the matrix and each candidate's estimate, and launches nothing.
func TestEvalRunPlanPrintsTheMatrixAndEstimate(t *testing.T) {
	stdout, _, err := execute(t, "eval", "run", "--plan", "--repo-root", evalRepoRoot(t),
		"--candidate", "nan/glm5.3-flash", "--candidate", "agy/gemini-4-hydrogen-high", "--max-runs", "100")
	if err != nil {
		t.Fatalf("--plan: %v\n%s", err, stdout)
	}
	for _, want := range []string{"× 2 candidates × 3 epochs", "harness-088-r2", "nan/glm5.3-flash", "NaN credits", "agy runs"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the plan lacks %q:\n%s", want, stdout)
		}
	}
}

// Above the cap the plan is still printed, so the operator sees what was
// refused, and the command exits non-zero.
func TestEvalRunRefusesAPlanAboveTheCap(t *testing.T) {
	for _, plan := range []bool{true, false} {
		args := []string{"eval", "run", "--repo-root", evalRepoRoot(t), "--candidate", "nan/glm5.3-flash", "--max-runs", "2"}
		if plan {
			args = append(args, "--plan")
		}
		stdout, _, err := execute(t, args...)
		if err == nil || !strings.Contains(err.Error(), "--max-runs") {
			t.Fatalf("plan=%v: a matrix above the cap was not refused: %v", plan, err)
		}
		if !strings.Contains(stdout, "harness-088-r2") {
			t.Errorf("plan=%v: the refused plan was not printed:\n%s", plan, stdout)
		}
	}
}

func TestEvalRunPlanNarrowsToTheNamedCases(t *testing.T) {
	stdout, _, err := execute(t, "eval", "run", "--plan", "--repo-root", evalRepoRoot(t),
		"--candidate", "nan/glm5.3-flash", "--case", "harness-088-r2", "--epochs", "1")
	if err != nil {
		t.Fatalf("--case: %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "1 cases × 1 candidates × 1 epochs = 1 runs") || strings.Contains(stdout, "memory-017") {
		t.Errorf("--case did not narrow the matrix:\n%s", stdout)
	}
}

// The command's filesystem reads go through evalRunDeps, so a failing loader
// is reported without touching a corpus on disk.
func TestEvalRunReportsAFailingLoader(t *testing.T) {
	var out strings.Builder
	deps := evalRunDeps{
		loadCases: func(string) ([]evals.Case, error) { return nil, errors.New("corpus unreadable") },
		loadCandidates: func(string) (evals.Candidates, error) {
			t.Error("candidates read after the corpus failed")
			return evals.Candidates{}, nil
		},
	}
	err := runEvalRun(context.Background(), &out, deps, evalRunOpts{root: "/nowhere", candidates: []string{"nan/x"}, epochs: 1, maxRuns: 30, planOnly: true})
	if err == nil || !strings.Contains(err.Error(), "corpus unreadable") || out.Len() != 0 {
		t.Fatalf("err %v, output %q", err, out.String())
	}
}

// launchSpy stands in for every step past the plan, records the order they
// ran in, and launches nothing.
type launchSpy struct {
	calls      []string
	checkErr   []string // problems the corpus check reports, on one case
	preflight  error
	runRecords []evals.Record
	runErr     error
}

func (s *launchSpy) deps(t *testing.T) evalRunDeps {
	t.Helper()
	return evalRunDeps{
		loadCases:      evals.LoadCases,
		loadCandidates: evals.LoadCandidates,
		check: func(string) ([]evals.CheckResult, error) {
			s.calls = append(s.calls, "check")
			return []evals.CheckResult{{Path: ".dotf/evals/cases/harness-088-r2.json", ID: "harness-088-r2", Problems: s.checkErr}}, nil
		},
		preflight: func(_ string, p evals.Plan, _ []evals.Case, cands map[string]spec.ReviewerEntry) error {
			s.calls = append(s.calls, "preflight")
			if _, ok := cands["nan/glm5.3-flash"]; !ok || p.Runs == 0 {
				t.Errorf("preflight got plan %+v and candidates %v", p, cands)
			}
			return s.preflight
		},
		newRunDir: func() (string, error) {
			s.calls = append(s.calls, "run-dir")
			return "/state/evals/20261115T150000Z", nil
		},
		run: func(_ context.Context, _ string, _ evals.Plan, _ []evals.Case, runDir string, _ map[string]spec.ReviewerEntry) ([]evals.Record, error) {
			s.calls = append(s.calls, "run:"+runDir)
			return s.runRecords, s.runErr
		},
	}
}

func spyOpts(t *testing.T, plan bool, maxRuns int) evalRunOpts {
	return evalRunOpts{root: evalRepoRoot(t), candidates: []string{"nan/glm5.3-flash"}, caseIDs: []string{"harness-088-r2"},
		epochs: 1, maxRuns: maxRuns, planOnly: plan}
}

// --plan and a matrix above the cap stop before anything that could spend:
// no corpus check, no preflight, no run directory, no launch.
func TestEvalRunLaunchesNothingForAPlanOrAboveTheCap(t *testing.T) {
	for name, o := range map[string]evalRunOpts{"--plan": spyOpts(t, true, 30), "above the cap": spyOpts(t, false, 0)} {
		t.Run(name, func(t *testing.T) {
			s := &launchSpy{}
			var out strings.Builder
			_ = runEvalRun(context.Background(), &out, s.deps(t), o)
			if len(s.calls) != 0 {
				t.Errorf("ran %v", s.calls)
			}
		})
	}
}

// A run goes check, preflight, run directory, launch, and prints each record.
func TestEvalRunChecksThenLaunchesAndReportsEachAttempt(t *testing.T) {
	s := &launchSpy{runRecords: []evals.Record{
		{Case: "harness-088-r2", Candidate: "nan/glm5.3-flash", Epoch: 1, Completion: evals.CompletionAnswered,
			Grade: &evals.Grade{Format: evals.FormatOK, Verdict: "FAIL", VerdictMatch: true, Recall: 1}},
	}}
	var out strings.Builder
	if err := runEvalRun(context.Background(), &out, s.deps(t), spyOpts(t, false, 30)); err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if got := strings.Join(s.calls, ","); got != "check,preflight,run-dir,run:/state/evals/20261115T150000Z" {
		t.Errorf("steps = %s", got)
	}
	for _, want := range []string{"/state/evals/20261115T150000Z", "harness-088-r2", "answered", "recall 1.00"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the report lacks %q:\n%s", want, out.String())
		}
	}
}

// A corpus whose keys fail `dotf eval check` is not run, and the failure is
// printed; a failed preflight launches nothing and names every problem.
func TestEvalRunRefusesABrokenCorpusOrPreflight(t *testing.T) {
	t.Run("corpus", func(t *testing.T) {
		s := &launchSpy{checkErr: []string{"expected anchor gone"}}
		var out strings.Builder
		err := runEvalRun(context.Background(), &out, s.deps(t), spyOpts(t, false, 30))
		if err == nil || strings.Join(s.calls, ",") != "check" || !strings.Contains(out.String(), "expected anchor gone") {
			t.Errorf("err %v, steps %v, output:\n%s", err, s.calls, out.String())
		}
	})
	t.Run("preflight", func(t *testing.T) {
		s := &launchSpy{preflight: errors.New("eval run preflight:\n  nan/glm5.3-flash: runner pi is not on PATH")}
		var out strings.Builder
		err := runEvalRun(context.Background(), &out, s.deps(t), spyOpts(t, false, 30))
		if err == nil || !strings.Contains(err.Error(), "runner pi is not on PATH") || strings.Join(s.calls, ",") != "check,preflight" {
			t.Errorf("err %v, steps %v", err, s.calls)
		}
	})
}

// An interrupted run still prints what it recorded, and exits non-zero.
func TestEvalRunReportsAnInterruptedRun(t *testing.T) {
	s := &launchSpy{
		runRecords: []evals.Record{{Case: "harness-088-r2", Candidate: "nan/glm5.3-flash", Epoch: 1, Completion: evals.CompletionInfra, Detail: "interrupted"}},
		runErr:     evals.ErrInterrupted,
	}
	var out strings.Builder
	err := runEvalRun(context.Background(), &out, s.deps(t), spyOpts(t, false, 30))
	if !errors.Is(err, evals.ErrInterrupted) || !strings.Contains(out.String(), "infra-error") {
		t.Errorf("err %v, output:\n%s", err, out.String())
	}
}

// A run's records live under dotf's state directory, in a directory of its own.
func TestEvalRunDirIsAFreshDirectoryUnderTheStateDir(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	dir, err := newEvalRunDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(dir) != filepath.Join(state, "dotfiles", "evals") {
		t.Errorf("run dir %s is not under %s", dir, state)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Errorf("run dir not created: %v", err)
	}
}
