package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A branch someone else named, here kubelab's release workflow, fails the
// creation shape rule. add attaches to it under its own name, never a
// rewritten one, and only the directory is slugified (#2822).
func TestAddAttachesToAnExistingOriginBranchWhateverItsName(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "myrepo")
	_ = os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o755)
	root := filepath.Join(t.TempDir(), "worktrees")
	var attached []string
	runner := &MockAddRunner{
		OnExistingBranch: func(_, branch string) (string, bool) { return "origin", branch == "deploy/errors-1.3.0" },
		OnWorktreeAttach: func(_, target, branch, remote string) error {
			attached = append(attached, branch, remote)
			return os.MkdirAll(target, 0o755)
		},
		OnWorktreeAdd: func(_, _, _, _ string) error { t.Fatal("add created a branch that exists"); return nil },
	}
	info, err := AddWithRunner(AddOptions{RepoRoot: repo, WorktreeRoot: root, Slug: "x", Branch: "deploy/errors-1.3.0", Issue: 7}, runner, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "myrepo", "deploy-errors-1-3-0"); info.Path != want {
		t.Errorf("path = %s, want %s", info.Path, want)
	}
	if strings.Join(attached, " ") != "deploy/errors-1.3.0 origin" || info.Branch != "deploy/errors-1.3.0" {
		t.Errorf("attached %v as %q, want deploy/errors-1.3.0 from origin", attached, info.Branch)
	}
	if runner.DefaultBaseRan {
		t.Error("an existing branch has no base to resolve")
	}
	if meta, err := LoadMetadata(info.Path); err != nil || meta == nil || meta.Issue != 7 {
		t.Errorf("lease metadata = %+v, %v; want issue 7", meta, err)
	}
}

// --base says where a new branch starts. An existing branch already starts
// somewhere, so the flag would do nothing: refuse rather than ignore it.
func TestAddRefusesABaseForAnExistingBranch(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "myrepo")
	_ = os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o755)
	runner := &MockAddRunner{OnExistingBranch: func(string, string) (string, bool) { return "", true }}
	_, err := AddWithRunner(AddOptions{RepoRoot: repo, WorktreeRoot: t.TempDir(), Slug: "x", Branch: "fix/x", BaseRef: "origin/main"}, runner, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--base") {
		t.Fatalf("got %v, want a refusal naming --base", err)
	}
}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// The real runner: a branch that exists only on origin becomes a local branch
// of the same name tracking it; one that exists locally is checked out as is.
func TestRealRunnerAttachesTrackingOrigin(t *testing.T) {
	tmp := t.TempDir()
	seed := filepath.Join(tmp, "seed")
	gitT(t, tmp, "init", "-q", "-b", "main", seed)
	gitT(t, seed, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	gitT(t, seed, "branch", "deploy/errors-1.3.0")
	clone := filepath.Join(tmp, "myrepo")
	gitT(t, tmp, "clone", "-q", seed, clone)
	gitT(t, clone, "branch", "fix/local-only")

	r := &RealAddRunner{}
	for _, c := range []struct{ branch, remote string }{{"deploy/errors-1.3.0", "origin"}, {"fix/local-only", ""}} {
		remote, ok := r.ExistingBranch(clone, c.branch)
		if !ok || remote != c.remote {
			t.Fatalf("%s: ExistingBranch = %q, %v; want %q, true", c.branch, remote, ok, c.remote)
		}
		target := filepath.Join(tmp, "wt-"+slugifyBranch(c.branch))
		if err := r.WorktreeAttach(clone, target, c.branch, remote); err != nil {
			t.Fatal(err)
		}
		if got := gitT(t, target, "rev-parse", "--abbrev-ref", "HEAD"); got != c.branch {
			t.Errorf("worktree is on %q, want %q", got, c.branch)
		}
	}
	if up := gitT(t, clone, "rev-parse", "--abbrev-ref", "deploy/errors-1.3.0@{upstream}"); up != "origin/deploy/errors-1.3.0" {
		t.Errorf("upstream = %q, want origin/deploy/errors-1.3.0", up)
	}
	if _, ok := r.ExistingBranch(clone, "fix/nowhere"); ok {
		t.Error("a branch on neither side reads as existing")
	}
	// deleted on origin after the last fetch: the stale remote ref is not it
	gitT(t, seed, "branch", "fix/gone")
	gitT(t, clone, "fetch", "-q", "origin")
	gitT(t, seed, "branch", "-D", "fix/gone")
	if _, ok := r.ExistingBranch(clone, "fix/gone"); ok {
		t.Error("a branch origin deleted reads as existing through its stale remote ref")
	}
}
