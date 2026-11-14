package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

// Until the run path lands, a run without --plan says so rather than doing
// nothing quietly.
func TestEvalRunWithoutPlanSaysItCannotRunYet(t *testing.T) {
	_, _, err := execute(t, "eval", "run", "--repo-root", evalRepoRoot(t), "--candidate", "nan/glm5.3-flash")
	if err == nil || !strings.Contains(err.Error(), "--plan") {
		t.Fatalf("a run without --plan: %v", err)
	}
}
