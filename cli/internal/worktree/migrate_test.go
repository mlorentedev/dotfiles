package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// migrateFixture is a repository with one legacy sibling worktree,
// <repo>-wt-foo on feat/foo, and an empty worktree root.
func migrateFixture(t *testing.T) (repo, legacy, root string) {
	t.Helper()
	tmp := t.TempDir()
	repo = filepath.Join(tmp, "repo")
	legacy = filepath.Join(tmp, "repo-wt-foo")
	root = filepath.Join(t.TempDir(), "worktrees")
	run := func(dir string, args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	_ = os.MkdirAll(repo, 0o755)
	run(repo, "init", "-q", "-b", "main")
	run(repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	run(repo, "worktree", "add", "-q", "-b", "feat/foo", legacy)
	return repo, legacy, root
}

func TestMigrateDryRunPlansWithoutMoving(t *testing.T) {
	repo, legacy, root := migrateFixture(t)
	rep, err := Migrate(MigrateOptions{RepoRoot: repo, WorktreeRoot: root, DryRun: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "repo", "feat-foo")
	if len(rep.Moves) != 1 || rep.Moves[0].From != resolvedPath(legacy) || rep.Moves[0].To != want || rep.Moves[0].Skip != "" {
		t.Fatalf("plan = %+v, want one move %s -> %s", rep.Moves, legacy, want)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Error("dry run moved the worktree")
	}
}

func TestMigrateMovesAndASecondRunHasNothingToDo(t *testing.T) {
	repo, legacy, root := migrateFixture(t)
	if _, err := Migrate(MigrateOptions{RepoRoot: repo, WorktreeRoot: root}, time.Now()); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "repo", "feat-foo")
	list, _ := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if _, err := os.Stat(legacy); !os.IsNotExist(err) || !strings.Contains(string(list), resolvedPath(target)) {
		t.Fatalf("after migrate: legacy stat %v, worktree list:\n%s", err, list)
	}
	rep, err := Migrate(MigrateOptions{RepoRoot: repo, WorktreeRoot: root}, time.Now())
	if err != nil || len(rep.Moves) != 0 {
		t.Errorf("second run: %+v, %v; want nothing to do", rep.Moves, err)
	}
}

func TestMigrateSkipsWhatItMustNotMove(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, repo, legacy, root string) MigrateOptions
		skip  string
	}{
		{"target exists", func(t *testing.T, repo, legacy, root string) MigrateOptions {
			_ = os.MkdirAll(filepath.Join(root, "repo", "feat-foo"), 0o755)
			return MigrateOptions{}
		}, "already exists"},
		{"caller inside", func(t *testing.T, repo, legacy, root string) MigrateOptions {
			return MigrateOptions{Cwd: filepath.Join(legacy)}
		}, "current directory"},
		{"recent session", func(t *testing.T, repo, legacy, root string) MigrateOptions {
			projects := t.TempDir()
			fakeTranscript(t, projects, resolvedPath(legacy), time.Hour)
			return MigrateOptions{ClaudeProjectsDir: projects}
		}, "Claude Code session"},
		{"sessions unreadable", func(t *testing.T, repo, legacy, root string) MigrateOptions {
			notADir := filepath.Join(t.TempDir(), "projects")
			if err := os.WriteFile(notADir, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return MigrateOptions{ClaudeProjectsDir: notADir}
		}, "could not be inspected"},
		{"locked", func(t *testing.T, repo, legacy, root string) MigrateOptions {
			_ = exec.Command("git", "-C", repo, "worktree", "lock", legacy).Run()
			// A dry run: git worktree move refuses a locked worktree on its
			// own, and the plan must not promise a move it cannot make.
			return MigrateOptions{DryRun: true}
		}, "locked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, legacy, root := migrateFixture(t)
			opts := tc.setup(t, repo, legacy, root)
			opts.RepoRoot, opts.WorktreeRoot = repo, root
			rep, err := Migrate(opts, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if len(rep.Moves) != 1 || !strings.Contains(rep.Moves[0].Skip, tc.skip) {
				t.Fatalf("plan = %+v, want a skip naming %q", rep.Moves, tc.skip)
			}
			// migrate has no override flag: a reason naming done's would send
			// the operator to a flag this command rejects.
			if strings.Contains(rep.Moves[0].Skip, "--ignore-recent-session") {
				t.Errorf("skip %q names a flag migrate does not take", rep.Moves[0].Skip)
			}
			if _, err := os.Stat(legacy); err != nil {
				t.Errorf("a skipped worktree was moved")
			}
		})
	}
}

func TestMigrateRefusesWithoutARoot(t *testing.T) {
	repo, _, _ := migrateFixture(t)
	if _, err := Migrate(MigrateOptions{RepoRoot: repo}, time.Now()); err == nil || !strings.Contains(err.Error(), "WORKTREE_ROOT") {
		t.Errorf("got %v, want a refusal naming WORKTREE_ROOT", err)
	}
}
