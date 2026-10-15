package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The rows are the herdr 0.9.3 measurements recorded on #2503: `herdr worktree
// create --branch <b>` placed each checkout at <root>/<repo>/<slug>. dotf must
// produce the same path, or `done <slug>` and adoption miss what herdr made.
func TestResolveWorktreePath(t *testing.T) {
	cases := []struct {
		branch string
		want   string
	}{
		{"feat/bar-baz", "/w/repo/feat-bar-baz"},
		{"a/b/c", "/w/repo/a-b-c"},
		{"Feat/Up_Case.x", "/w/repo/feat-up-case-x"},
		{"fix/x--y", "/w/repo/fix-x-y"},
	}
	for _, tc := range cases {
		got := ResolveWorktreePath(filepath.FromSlash("/w"), "repo", tc.branch)
		if want := filepath.FromSlash(tc.want); got != want {
			t.Errorf("ResolveWorktreePath(/w, repo, %q) = %q, want %q", tc.branch, got, want)
		}
	}
}

// With a worktree root, add places the checkout where herdr would for the same
// branch, and the default branch feat/<slug> decides the directory name.
func TestAddUnderWorktreeRoot(t *testing.T) {
	tmp := t.TempDir()
	mainRepo := filepath.Join(tmp, "src", "myrepo")
	if err := os.MkdirAll(filepath.Join(mainRepo, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(tmp, "worktrees")
	var gotTarget, gotBranch string
	runner := &MockAddRunner{OnWorktreeAdd: func(_, target, branch, _ string) error {
		gotTarget, gotBranch = target, branch
		return os.MkdirAll(target, 0o755)
	}}

	info, err := AddWithRunner(AddOptions{RepoRoot: mainRepo, Slug: "one-root", WorktreeRoot: root}, runner, time.Now())
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	want := filepath.Join(root, "myrepo", "feat-one-root")
	if info.Path != want || gotTarget != want {
		t.Errorf("path = %q (git got %q), want %q", info.Path, gotTarget, want)
	}
	if gotBranch != "feat/one-root" {
		t.Errorf("branch = %q, want feat/one-root", gotBranch)
	}
}

// An explicit --branch names the directory too, exactly as herdr would.
func TestAddUnderWorktreeRootFollowsBranch(t *testing.T) {
	tmp := t.TempDir()
	mainRepo := filepath.Join(tmp, "myrepo")
	_ = os.MkdirAll(filepath.Join(mainRepo, ".git", "info"), 0o755)
	root := filepath.Join(tmp, "worktrees")
	runner := &MockAddRunner{OnWorktreeAdd: func(_, target, _, _ string) error { return os.MkdirAll(target, 0o755) }}

	info, err := AddWithRunner(AddOptions{RepoRoot: mainRepo, Slug: "x", Branch: "fix/Some_Bug", WorktreeRoot: root}, runner, time.Now())
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if want := filepath.Join(root, "myrepo", "fix-some-bug"); info.Path != want {
		t.Errorf("path = %q, want %q", info.Path, want)
	}
}

// Gate f's own cwd check refuses a worktree the caller stands anywhere inside,
// through a symlink too, without leaning on the process scan: under one root a
// session usually works in a subdirectory, and macOS reaches $TMPDIR through
// /var -> /private/var.
func TestGateFRefusesACwdInsideTheWorktreeWithoutTheProcessScan(t *testing.T) {
	orig := hostProcessInside
	hostProcessInside = func(string) GateFReading { return GateFReading{} }
	t.Cleanup(func() { hostProcessInside = orig })

	wt := filepath.Join(t.TempDir(), "repo", "feat-x")
	sub := filepath.Join(wt, "cli")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(wt, link); err != nil {
		t.Skipf("symlink: %v", err)
	}
	info := Info{Path: wt, State: StateReapable}
	for _, cwd := range []string{wt, sub, filepath.Join(link, "cli")} {
		if _, ok := gateF(info, cwd); ok {
			t.Errorf("gateF allowed reaping %s with the caller in %s", wt, cwd)
		}
	}
	if _, ok := gateF(info, t.TempDir()); !ok {
		t.Errorf("gateF refused a caller outside %s", wt)
	}
}
