package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdoptWritesALeaseOnce(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	now := time.Now()

	res, err := Adopt(AdoptOptions{RepoRoot: repoDir, WorktreePath: wtDir, TTL: time.Hour, Creator: "herdr"}, now)
	if err != nil || res.AlreadyAdopted {
		t.Fatalf("first adopt: %+v, %v", res, err)
	}
	meta, err := LoadMetadata(wtDir)
	if err != nil || meta == nil || !meta.ReapOK || !meta.LeaseExpiresAt.Equal(now.Add(time.Hour)) || meta.Creator != "herdr" {
		t.Fatalf("lease = %+v, %v; want reap_ok, expiring in 1h, creator herdr", meta, err)
	}
	exclude, _ := os.ReadFile(filepath.Join(repoDir, ".git", "info", "exclude"))
	if !strings.Contains(string(exclude), MetadataFileName) {
		t.Error("the lease file is not excluded, so it would show as untracked")
	}

	// A second adopt changes nothing: the lease it found stands.
	res, err = Adopt(AdoptOptions{RepoRoot: repoDir, WorktreePath: wtDir, TTL: 48 * time.Hour}, now.Add(time.Minute))
	if err != nil || !res.AlreadyAdopted {
		t.Fatalf("second adopt: %+v, %v; want already adopted", res, err)
	}
	if again, _ := LoadMetadata(wtDir); !again.LeaseExpiresAt.Equal(meta.LeaseExpiresAt) {
		t.Errorf("second adopt moved the lease from %s to %s", meta.LeaseExpiresAt, again.LeaseExpiresAt)
	}
}

func TestAdoptRefusesWhatIsNotALinkedWorktree(t *testing.T) {
	repoDir, _, _ := setupTestGitRepoAndWorktree(t)
	plain := t.TempDir()
	for _, target := range []string{repoDir, plain, filepath.Join(plain, "missing")} {
		if _, err := Adopt(AdoptOptions{RepoRoot: repoDir, WorktreePath: target}, time.Now()); err == nil {
			t.Errorf("adopted %s", target)
		}
		if meta, _ := LoadMetadata(target); meta != nil {
			t.Errorf("wrote a lease into %s", target)
		}
	}
}

// The whole point of adopt: sweep refuses a worktree without a lease, and reaps
// it once adopted, merged, clean and past the lease.
func TestSweepReapsAnAdoptedWorktreeOnly(t *testing.T) {
	withHostProcessInside(t, false)
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	now := time.Now()
	runner := func(reaped *[]string) *MockSweepRunner {
		return &MockSweepRunner{
			MockGitRunner: MockGitRunner{
				PorcelainOutput: "worktree " + repoDir + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
					"worktree " + wtDir + "\nHEAD 2222\nbranch refs/heads/feat/test\n\n",
				MergedBranches: map[string]bool{"feat/test": true},
			},
			OnWorktreeRemove: func(_, path string) error { *reaped = append(*reaped, path); return nil },
		}
	}
	lock := filepath.Join(t.TempDir(), "sweep.lock")

	var before []string
	if _, err := SweepWithRunner(SweepOptions{RepoRoot: repoDir, LockPath: lock}, runner(&before), now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("sweep reaped an unadopted worktree: %v", before)
	}

	if _, err := Adopt(AdoptOptions{RepoRoot: repoDir, WorktreePath: wtDir, TTL: time.Hour}, now); err != nil {
		t.Fatal(err)
	}
	var after []string
	if _, err := SweepWithRunner(SweepOptions{RepoRoot: repoDir, LockPath: lock}, runner(&after), now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 {
		t.Errorf("reaped %v after adopt, want %s", after, wtDir)
	}
}
