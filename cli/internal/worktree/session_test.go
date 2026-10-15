package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// fakeTranscript writes a Claude Code transcript for a session started in cwd,
// last modified age ago, and returns the projects directory holding it.
func fakeTranscript(t *testing.T, projects, cwd string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(projects, memlink.ClaudeProjectKey(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "0b6c-session.jsonl")
	if err := os.WriteFile(f, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := time.Now().Add(-age)
	if err := os.Chtimes(f, mod, mod); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRecentSession(t *testing.T) {
	wt := filepath.Join(t.TempDir(), "repo", "feat-x")
	for _, tc := range []struct {
		name  string
		cwd   string // where the fake session started; "" writes none
		age   time.Duration
		found bool
	}{
		{"an hour-old session in the worktree", wt, time.Hour, true},
		{"a session started in a subdirectory", filepath.Join(wt, "cli"), time.Hour, true},
		{"a session older than the window", wt, 7 * time.Hour, false},
		{"another worktree's session", filepath.Join(filepath.Dir(wt), "feat-y"), time.Hour, false},
		{"no transcript at all", "", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projects := t.TempDir()
			if tc.cwd != "" {
				fakeTranscript(t, projects, tc.cwd, tc.age)
			}
			s, found := RecentSession(projects, wt, time.Now(), SessionWindow)
			if found != tc.found {
				t.Errorf("found = %v (%+v), want %v", found, s, tc.found)
			}
		})
	}
	if _, found := RecentSession("", wt, time.Now(), SessionWindow); found {
		t.Error("an unknown projects directory reported a session")
	}
}

func TestDoneRefusesARecentSessionUnlessIgnored(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	stubMergedPRHeads(t, nil, nil)
	projects := t.TempDir()
	fakeTranscript(t, projects, wtDir, time.Hour)
	opts := DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, ClaudeProjectsDir: projects}

	_, err := Done(opts)
	if err == nil || !strings.Contains(err.Error(), "--ignore-recent-session") {
		t.Fatalf("got %v, want a refusal naming --ignore-recent-session", err)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Fatal("the worktree was removed despite the refusal")
	}

	// --force is about uncommitted work and does not override this check.
	opts.Force = true
	if _, err := Done(opts); err == nil {
		t.Fatal("--force overrode the recent-session refusal")
	}

	opts.Force, opts.IgnoreRecentSession = false, true
	if _, err := Done(opts); err != nil {
		t.Fatalf("with --ignore-recent-session: %v", err)
	}
}

func TestSweepSkipsAWorktreeWithARecentSession(t *testing.T) {
	withHostProcessInside(t, false)
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	live := filepath.Join(tmpDir, "myrepo-wt-live")
	idle := filepath.Join(tmpDir, "myrepo-wt-idle")
	now := time.Now()
	for _, wt := range []string{mainRepo, live, idle} {
		_ = os.MkdirAll(wt, 0o755)
	}
	for _, wt := range []string{live, idle} {
		_ = SaveMetadata(wt, Metadata{ReapOK: true, CreatedAt: now.Add(-2 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour)})
	}
	projects := t.TempDir()
	fakeTranscript(t, projects, live, time.Hour)

	var reaped []string
	runner := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
				"worktree " + live + "\nHEAD 2222\nbranch refs/heads/feat/live\n\n" +
				"worktree " + idle + "\nHEAD 3333\nbranch refs/heads/feat/idle\n\n",
			MergedBranches: map[string]bool{"feat/live": true, "feat/idle": true},
		},
		OnWorktreeRemove: func(_, path string) error { reaped = append(reaped, path); return nil },
	}
	opts := SweepOptions{RepoRoot: mainRepo, LockPath: filepath.Join(tmpDir, "sweep.lock"), ClaudeProjectsDir: projects}
	if _, err := SweepWithRunner(opts, runner, now); err != nil {
		t.Fatal(err)
	}
	if len(reaped) != 1 || reaped[0] != idle {
		t.Errorf("reaped %v, want only %s: %s has a session from an hour ago", reaped, idle, live)
	}
}
