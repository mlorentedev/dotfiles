package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

func detachedOpts(t *testing.T) DetachedOptions {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "myrepo")
	if err := os.MkdirAll(filepath.Join(repo, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	return DetachedOptions{RepoRoot: repo, WorktreeRoot: t.TempDir(), Name: "harness-142-first-ab12",
		Commit: "4ed8cff5", TTL: 2 * time.Hour, Creator: "test", Kind: KindReview}
}

// HARNESS-142: a review worktree is detached at the reviewed commit, holds no
// issue lease and is reapable once its lease passes, so a crashed cleanup is
// something `list` shows and `sweep` removes.
func TestAddDetachedSavesALeaseFreeReapableLease(t *testing.T) {
	opts := detachedOpts(t)
	now := time.Now()
	var gotRepo, gotTarget, gotCommit string
	runner := &MockAddRunner{OnWorktreeAddDetached: func(repo, target, commit string) error {
		gotRepo, gotTarget, gotCommit = repo, target, commit
		return fakeAdd(repo, target)
	}}

	info, err := AddDetachedWithRunner(opts, runner, now)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(opts.WorktreeRoot, "myrepo", opts.Name)
	if info.Path != want || gotTarget != want || gotRepo != opts.RepoRoot || gotCommit != opts.Commit {
		t.Errorf("added %q in %q at %q, info %q; want %q at %q", gotTarget, gotRepo, gotCommit, info.Path, want, opts.Commit)
	}
	meta, err := LoadMetadata(want)
	if err != nil || meta == nil {
		t.Fatalf("no metadata: %v", err)
	}
	if meta.Issue != 0 || !meta.ReapOK || meta.Kind != KindReview || !meta.LeaseExpiresAt.Equal(now.Add(opts.TTL)) {
		t.Errorf("metadata %+v: want issue 0, reap_ok, kind %q, lease %s", meta, KindReview, now.Add(opts.TTL))
	}
	if runner.WorktreeListRan || runner.DefaultBaseRan {
		t.Error("a detached add scanned leases or fetched a base: it holds no issue and names its commit")
	}
	exclude, _ := os.ReadFile(filepath.Join(opts.RepoRoot, ".git", "info", "exclude"))
	if !containsLine(string(exclude), MetadataFileName) {
		t.Errorf("%s not excluded", MetadataFileName)
	}
}

func TestAddDetachedRefusesAnExistingTargetAndMissingInputs(t *testing.T) {
	runner := &MockAddRunner{OnWorktreeAddDetached: func(repo, target, _ string) error { return fakeAdd(repo, target) }}

	taken := detachedOpts(t)
	if err := os.MkdirAll(filepath.Join(taken.WorktreeRoot, "myrepo", taken.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := AddDetachedWithRunner(taken, runner, time.Now()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("an existing target: err = %v, want it refused", err)
	}

	for field, blank := range map[string]func(*DetachedOptions){
		"commit": func(o *DetachedOptions) { o.Commit = "" },
		"name":   func(o *DetachedOptions) { o.Name = "" },
		"root":   func(o *DetachedOptions) { o.WorktreeRoot = "" },
		"kind":   func(o *DetachedOptions) { o.Kind = "" },
	} {
		opts := detachedOpts(t)
		blank(&opts)
		if _, err := AddDetachedWithRunner(opts, runner, time.Now()); err == nil {
			t.Errorf("no %s: added anyway", field)
		}
	}
}

// The real runner checks the commit out detached, so the reviewer's tree has
// no branch to push and nothing to merge.
func TestRealRunnerAddsADetachedWorktree(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "myrepo")
	gitT(t, tmp, "init", "-q", "-b", "main", repo)
	gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	sha := gitT(t, repo, "rev-parse", "HEAD")

	target := filepath.Join(tmp, "review")
	if err := (&RealAddRunner{}).WorktreeAddDetached(repo, target, sha); err != nil {
		t.Fatal(err)
	}
	if got := gitT(t, target, "rev-parse", "HEAD"); got != sha {
		t.Errorf("HEAD %s, want %s", got, sha)
	}
	if got := gitT(t, target, "rev-parse", "--abbrev-ref", "HEAD"); got != "HEAD" {
		t.Errorf("worktree is on branch %q, want detached", got)
	}
}

// A review worktree holds no issue, so it never stands in the way of the
// author's own `worktree add --issue`, even while its lease is live.
func TestAddWithIssueIgnoresALiveReviewWorktree(t *testing.T) {
	now := time.Now()
	review := linkedTempDir(t)
	if err := SaveMetadata(review, Metadata{Kind: KindReview, ReapOK: true, LeaseExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	runner := &MockAddRunner{
		OnWorktreeList: func(string) (string, error) { return "worktree " + review + "\nHEAD abc\ndetached\n\n", nil },
		OnWorktreeAdd:  func(repo, target, _, _ string) error { return fakeAdd(repo, target) },
	}
	if _, err := AddWithRunner(heldIssueOpts(t, 1649), runner, now); err != nil {
		t.Fatalf("a review worktree blocked add --issue: %v", err)
	}
}

// A worktree whose lease could not be written carries no kind and no expiry,
// so sweep never reaps it: the add removes it again rather than return it
// (#3223).
func TestAddDetachedRemovesTheWorktreeWhenItsLeaseCannotBeSaved(t *testing.T) {
	opts := detachedOpts(t)
	var removed string
	runner := &MockAddRunner{
		OnWorktreeAddDetached: func(repo, target, _ string) error {
			// A directory where the lease goes in the admin dir makes SaveMetadata fail.
			admin, err := gittest.LinkWorktree(repo, target)
			if err != nil {
				return err
			}
			return os.MkdirAll(filepath.Join(admin, LeaseFileName), 0o755)
		},
		OnWorktreeRemoveForce: func(_, target string) error {
			removed = target
			return os.RemoveAll(target)
		},
	}

	_, err := AddDetachedWithRunner(opts, runner, time.Now())
	if err == nil || !strings.Contains(err.Error(), "saving worktree metadata") {
		t.Fatalf("err = %v, want the metadata failure", err)
	}
	want := filepath.Join(opts.WorktreeRoot, "myrepo", opts.Name)
	if removed != want {
		t.Errorf("removed %q, want the worktree %q", removed, want)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("the unlabelled worktree is still there: %v", err)
	}
}
