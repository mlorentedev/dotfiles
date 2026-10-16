package lane

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// fixtureRepo is a repository with one merged, expired-lease linked worktree
// and one unadopted one: enough for plan to propose an auto action and an
// ambiguous one.
func fixtureRepo(t *testing.T, now time.Time) (repo string, worktrees []string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo = filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-q", "-b", "main")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "base")
	// The lease file is untracked; dotf excludes it, or every leased worktree reads dirty.
	exclude := filepath.Join(repo, ".git", "info", "exclude")
	if err := os.WriteFile(exclude, []byte(worktree.MetadataFileName+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"merged", "loose"} {
		wt := filepath.Join(root, name)
		git(t, repo, "worktree", "add", "-q", "-b", "fix/"+name, wt)
		worktrees = append(worktrees, wt)
	}
	err = worktree.SaveMetadata(worktrees[0], worktree.Metadata{
		Issue: 7, CreatedAt: now.Add(-72 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour), ReapOK: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return repo, worktrees
}

// fingerprint hashes everything plan could have touched: every ref, the
// worktree registry, each worktree's status and each lease file.
func fingerprint(t *testing.T, repo string, worktrees []string) string {
	t.Helper()
	h := sha256.New()
	h.Write([]byte(git(t, repo, "for-each-ref")))
	h.Write([]byte(git(t, repo, "worktree", "list", "--porcelain")))
	for _, wt := range worktrees {
		h.Write([]byte(git(t, wt, "status", "--porcelain", "--ignored")))
		if b, err := os.ReadFile(filepath.Join(wt, worktree.MetadataFileName)); err == nil {
			h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TestPlanMutatesNothing runs the whole read path, status then plan, against a
// real repository and shows its refs, worktrees and leases byte-identical
// afterwards (AC2). The plan it checks is not empty: an empty plan changes
// nothing trivially.
func TestPlanMutatesNothing(t *testing.T) {
	now := time.Now()
	repo, wts := fixtureRepo(t, now)
	forge := CachedForge(&fakeForge{prs: map[string]PRInfo{"fix/merged": {Number: 10, State: PRMerged}}})
	src := Sources{
		Now: now, Repo: repo,
		List: func() ([]worktree.Info, error) {
			return worktree.ListWithRunner(repo, ForgeGit{RealGitRunner: &worktree.RealGitRunner{}, Forge: forge}, now)
		},
		Forge: forge,
		Live:  func(string) Liveness { return LiveIdle },
		Herdr: fakeFinder{},
	}
	before := fingerprint(t, repo, wts)
	rows, err := Status(src)
	if err != nil {
		t.Fatal(err)
	}
	acts := Plan(rows, nil, now)
	if after := fingerprint(t, repo, wts); after != before {
		t.Errorf("plan changed the repository: fingerprint %s -> %s", before, after)
	}
	var classes []string
	for _, a := range acts {
		classes = append(classes, string(a.Kind)+"/"+string(a.Class))
	}
	got := strings.Join(classes, " ")
	if !strings.Contains(got, "reap/auto") || !strings.Contains(got, "flag/ambiguous") {
		t.Errorf("plan = %q, want a reap/auto for the merged worktree and a flag/ambiguous for the unadopted one: %+v", got, rows)
	}
}
