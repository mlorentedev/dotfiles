//go:build !windows

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// reviewWorktree is a real repository with a spec folder, and a review
// worktree detached at its HEAD under a temporary worktree root.
func reviewWorktree(t *testing.T) (repo, callerSpec, tree string) {
	t.Helper()
	// deadline enters the worktree and leaves it on removal, so a test using
	// one must get its own working directory back.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(wd)
	repo, _ = strayRepo(t)
	callerSpec = filepath.Join(repo, "specs", "AI-001-x")
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "main.go"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "main"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	info, err := worktree.AddDetached(worktree.DetachedOptions{
		RepoRoot: repo, WorktreeRoot: t.TempDir(), Name: "ai-001-x-first-abc123",
		Commit: strings.TrimSpace(string(head)), TTL: time.Hour, Kind: worktree.KindReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(info.Path, "specs", "AI-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, callerSpec, info.Path
}

// HARNESS-142 AC3: however the run ends, the slot's verdict is copied back to
// the caller's spec folder and the review worktree is removed.
func TestDeadlineCopiesBackAndRemovesReviewWorktree(t *testing.T) {
	for _, c := range []struct {
		name, limit, slotFile, script string
		stall                         bool
		code                          int
	}{
		{"exit 0", "1m", "review.md", `echo verdict > "$1"`, false, 0},
		{"stray writes, exit 4", "1m", "review.md", `echo verdict > "$1" && touch "$2/probe.txt"`, false, 4},
		{"runner failed", "1m", "review.md", `echo verdict > "$1"; exit 3`, false, 3},
		{"deadline, 124", "2s", "review.md", `echo provisional > "$1"; sleep 30`, false, 124},
		{"stall, 125", "1m", "review.md", `echo provisional > "$1"; sleep 30`, true, 125},
		{"second signer", "1m", "review-second.md", `echo verdict > "$1"`, false, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, callerSpec, tree := reviewWorktree(t)
			review := filepath.Join(tree, "specs", "AI-001-x", c.slotFile)

			args := []string{"spec", "deadline", c.limit, "--review", review, "--worktree", tree, "--copy-back", callerSpec}
			if c.stall {
				transcript := filepath.Join(callerSpec, "review-transcript.jsonl")
				if err := os.WriteFile(transcript, []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--stall-timeout", "2s", "--transcript", transcript)
			}
			args = append(args, "--", "sh", "-c", c.script, "sh", review, tree)
			_, stderr, err := execute(t, args...)

			if got := ExitCode(err); got != c.code {
				t.Fatalf("exit %d, want %d (err %v)\n%s", got, c.code, err, stderr)
			}
			if _, err := os.Stat(filepath.Join(callerSpec, c.slotFile)); err != nil {
				t.Errorf("%s was not copied back: %v\n%s", c.slotFile, err, stderr)
			}
			if _, err := os.Stat(tree); !os.IsNotExist(err) {
				t.Errorf("the review worktree is still on disk: %v\n%s", err, stderr)
			}
			list, _ := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
			if strings.Contains(string(list), tree) {
				t.Errorf("git still lists the review worktree:\n%s", list)
			}
			if _, err := os.Stat(filepath.Join(repo, "probe.txt")); !os.IsNotExist(err) {
				t.Errorf("a stray write reached the caller's tree: %v", err)
			}
		})
	}
}

// A verdict that cannot be copied back keeps the worktree: it is then the only
// copy, and sweep keeps it too, since its output was never saved.
func TestDeadlineKeepsTheWorktreeWhenTheCopyBackFails(t *testing.T) {
	_, _, tree := reviewWorktree(t)
	review := filepath.Join(tree, "specs", "AI-001-x", "review.md")
	missing := filepath.Join(t.TempDir(), "gone", "specs", "AI-001-x")

	_, stderr, _ := execute(t, "spec", "deadline", "1m", "--review", review,
		"--worktree", tree, "--copy-back", missing, "--",
		"sh", "-c", `echo verdict > "$1"`, "sh", review)

	if _, err := os.Stat(review); err != nil {
		t.Fatalf("the only copy of the verdict was removed: %v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "kept") {
		t.Errorf("stderr does not say the worktree was kept:\n%s", stderr)
	}
	meta, err := worktree.LoadMetadata(tree)
	if err != nil || meta == nil || meta.OutputSaved {
		t.Errorf("metadata %+v (err %v): a worktree whose output was not copied back must not be marked saved", meta, err)
	}
}

// fakeReviewerRun runs a reviewer that writes a probe test, edits a tracked
// file and writes its verdict, inside a review worktree under deadline.
func fakeReviewerRun(t *testing.T) (repo, callerSpec, tree, stderr string, err error) {
	t.Helper()
	repo, callerSpec, tree = reviewWorktree(t)
	review := filepath.Join(tree, "specs", "AI-001-x", "review.md")
	script := `mkdir -p "$2/tests" && echo probe > "$2/tests/probe_test.go" && echo edited >> "$2/main.go" && echo verdict > "$1"`
	_, stderr, err = execute(t, "spec", "deadline", "1m", "--review", review,
		"--worktree", tree, "--copy-back", callerSpec, "--",
		"sh", "-c", script, "sh", review, tree)
	return repo, callerSpec, tree, stderr, err
}

// HARNESS-142 AC1: what the reviewer writes stays in its worktree. The caller's
// status outside specs/<id>/ is unchanged, and inside it only the verdict
// arrives. The transcript and its .stderr are written caller-side by the
// launcher's pipeline, not by the reviewer, so this run does not produce them.
func TestReviewInWorktreeLeavesCallerTreeUntouched(t *testing.T) {
	repo, callerSpec, _, stderr, _ := fakeReviewerRun(t)
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(status)); got != "?? specs/AI-001-x/review.md" {
		t.Errorf("caller status = %q, want only the copied-back verdict\n%s", got, stderr)
	}
	if b, err := os.ReadFile(filepath.Join(callerSpec, "review.md")); err != nil || strings.TrimSpace(string(b)) != "verdict" {
		t.Errorf("verdict not copied back: %q, %v", b, err)
	}
}

// HARNESS-142 AC2: the leftovers are still named, now in the worktree, and the
// run still exits 4.
func TestReviewInWorktreeStillReportsStrayWrites(t *testing.T) {
	_, _, _, stderr, err := fakeReviewerRun(t)
	if got := ExitCode(err); got != 4 {
		t.Fatalf("exit %d, want 4 (err %v)\n%s", got, err, stderr)
	}
	for _, want := range []string{"tests/probe_test.go", "main.go"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not name %s:\n%s", want, stderr)
		}
	}
}

// #3220: the reviewer starts in the review worktree whatever directory deadline
// was started from. tmux's -c was the only thing putting it there, and a tmux
// server with an unreadable working directory ignored it.
func TestDeadlineRunsTheReviewerInTheReviewWorktree(t *testing.T) {
	_, callerSpec, tree := reviewWorktree(t)
	t.Chdir(t.TempDir())
	review := filepath.Join(tree, "specs", "AI-001-x", "review.md")
	want, err := filepath.EvalSymlinks(tree)
	if err != nil {
		t.Fatal(err)
	}

	_, stderr, err := execute(t, "spec", "deadline", "1m", "--review", review, "--worktree", tree, "--copy-back", callerSpec,
		"--", "sh", "-c", `pwd -P > "$1"`, "sh", review)
	if err != nil {
		t.Fatalf("deadline failed: %v\n%s", err, stderr)
	}
	got, err := os.ReadFile(filepath.Join(callerSpec, "review.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != want {
		t.Errorf("the reviewer ran in %q, want the review worktree %q", strings.TrimSpace(string(got)), want)
	}
}

// A review worktree deadline cannot enter is refused before the reviewer runs,
// naming the directory, rather than left to the reviewer to fail on.
func TestDeadlineRefusesAReviewWorktreeItCannotEnter(t *testing.T) {
	_, callerSpec, _ := reviewWorktree(t)
	missing := filepath.Join(t.TempDir(), "gone")
	ran := filepath.Join(t.TempDir(), "ran")

	_, stderr, err := execute(t, "spec", "deadline", "1m", "--worktree", missing, "--copy-back", callerSpec,
		"--", "sh", "-c", `touch "$1"`, "sh", ran)
	if err == nil {
		t.Fatalf("deadline ran in a review worktree that does not exist\n%s", stderr)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("the error does not name the review worktree: %v", err)
	}
	if _, statErr := os.Stat(ran); !os.IsNotExist(statErr) {
		t.Errorf("the reviewer ran although deadline could not enter the worktree: %v", statErr)
	}
}
