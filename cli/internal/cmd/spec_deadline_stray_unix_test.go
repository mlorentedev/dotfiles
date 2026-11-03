//go:build !windows

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// strayRepo is a git repository with one commit and a spec folder, and the
// review path a launch would pass as --review.
func strayRepo(t *testing.T) (repo, review string) {
	t.Helper()
	repo = t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		c := exec.Command("git", args...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	spec := filepath.Join(repo, "specs", "AI-001-x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, filepath.Join(spec, "review.md")
}

// HARNESS-142 (#1649): a reviewer that leaves files outside its spec folder is
// reported by path, and a run that otherwise exited 0 does not exit clean.
// What it writes under specs/<id>/ is its sanctioned output and is not listed.
func TestSpecDeadlineReportsWhatTheReviewWroteOutsideTheSpec(t *testing.T) {
	repo, review := strayRepo(t)
	script := `mkdir -p "$1/tests" && touch "$1/tests/probe_test.py" "$1/main.py.txt" && echo verdict > "$2"`

	_, stderr, err := execute(t, "spec", "deadline", "1m", "--review", review, "--",
		"sh", "-c", script, "sh", repo, review)

	if got := ExitCode(err); got != 4 {
		t.Fatalf("exit %d, want 4, the stray-write status (err %v)\n%s", got, err, stderr)
	}
	for _, want := range []string{"tests/probe_test.py", "main.py.txt", "outside specs/AI-001-x/"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr must mention %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "review.md") {
		t.Errorf("the review's own output was reported as stray:\n%s", stderr)
	}
}

// A file that was already untracked before the run is the operator's, not the
// reviewer's, and a run that writes only its verdict exits with its own status.
func TestSpecDeadlineIgnoresWhatWasThereBeforeTheReview(t *testing.T) {
	repo, review := strayRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := execute(t, "spec", "deadline", "1m", "--review", review, "--",
		"sh", "-c", `echo verdict > "$1"`, "sh", review)

	if err != nil {
		t.Fatalf("a clean review must exit with the runner's status: %v\n%s", err, stderr)
	}
	if strings.Contains(stderr, "notes.txt") {
		t.Errorf("a file present before the run was reported:\n%s", stderr)
	}
}
