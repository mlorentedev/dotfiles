package evals

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// replayRemote is a remote whose main holds one commit and whose
// refs/pull/7/head holds a second that no branch reaches, the shape a
// squash-merged pull request leaves. It returns a clone that lacks the second.
func replayRemote(t *testing.T) (clone, onMain, onPullRef string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	base := t.TempDir()
	bare, seed := filepath.Join(base, "remote.git"), filepath.Join(base, "seed")
	git(t, base, "init", "-q", "--bare", bare)
	git(t, base, "init", "-q", "-b", "main", seed)
	if err := os.WriteFile(filepath.Join(seed, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, seed, "add", "a.txt")
	git(t, seed, "commit", "-q", "-m", "a")
	onMain = git(t, seed, "rev-parse", "HEAD")
	git(t, seed, "push", "-q", bare, "main")
	if err := os.WriteFile(filepath.Join(seed, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, seed, "add", "b.txt")
	git(t, seed, "commit", "-q", "-m", "b")
	onPullRef = git(t, seed, "rev-parse", "HEAD")
	git(t, seed, "push", "-q", bare, "HEAD:refs/pull/7/head")
	clone = filepath.Join(base, "clone")
	// --no-local: a local clone copies the whole object store, the second
	// commit with it, and nothing would ever be fetched.
	git(t, base, "clone", "-q", "--no-local", bare, clone)
	return clone, onMain, onPullRef
}

func TestReplayCheckoutFetchesAReviewedCommitFromItsRef(t *testing.T) {
	clone, _, onPullRef := replayRemote(t)
	refsBefore := git(t, clone, "for-each-ref")
	runDir := filepath.Join(t.TempDir(), "run-1")

	r, err := CheckoutReplay(clone, Case{ID: "c1", ReviewedSHA: onPullRef, FetchRef: "refs/pull/7/head"}, runDir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if r.Dir != filepath.Join(runDir, "c1-2") {
		t.Errorf("replay dir = %s, want <run>/<case>-<epoch>", r.Dir)
	}
	if head := git(t, r.Dir, "rev-parse", "HEAD"); head != onPullRef {
		t.Errorf("replay HEAD = %s, want the reviewed commit %s", head, onPullRef)
	}
	if branch := git(t, r.Dir, "branch", "--show-current"); branch != "" {
		t.Errorf("replay is on branch %q, want a detached HEAD", branch)
	}
	if refs := git(t, clone, "for-each-ref"); refs != refsBefore {
		t.Errorf("the fetch created refs in the clone:\nbefore:\n%s\nafter:\n%s", refsBefore, refs)
	}
	if err := r.Remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.Dir); !os.IsNotExist(err) {
		t.Errorf("the replay tree survived its removal: %v", err)
	}
	if list := git(t, clone, "worktree", "list", "--porcelain"); strings.Count(list, "worktree ") != 1 {
		t.Errorf("the clone still registers the replay worktree:\n%s", list)
	}
}

// A commit the clone already holds is checked out without asking the remote,
// so a case runs offline once its commit is local.
func TestReplayCheckoutUsesALocalCommitWithoutFetching(t *testing.T) {
	clone, onMain, _ := replayRemote(t)
	git(t, clone, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	r, err := CheckoutReplay(clone, Case{ID: "c2", ReviewedSHA: onMain}, filepath.Join(t.TempDir(), "run"), 1)
	if err != nil {
		t.Fatalf("a local commit needed the remote: %v", err)
	}
	t.Cleanup(func() { _ = r.Remove() })
}

func TestReplayCheckoutNamesACaseItCannotResolve(t *testing.T) {
	clone, _, _ := replayRemote(t)
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := CheckoutReplay(clone, Case{ID: "lost-case", ReviewedSHA: strings.Repeat("d", 40), FetchRef: "refs/pull/9/head"}, runDir, 1)
	if err == nil || !strings.Contains(err.Error(), "lost-case") {
		t.Fatalf("want a failure naming the case, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(runDir, "lost-case-1")); !os.IsNotExist(statErr) {
		t.Errorf("an unresolved case left a replay directory: %v", statErr)
	}
}

func TestReplayCheckoutRefusesARunDirectoryInsideTheRepository(t *testing.T) {
	clone, onMain, _ := replayRemote(t)
	_, err := CheckoutReplay(clone, Case{ID: "c3", ReviewedSHA: onMain}, filepath.Join(clone, "runs"), 1)
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("want a refusal of a run directory in the repository, got %v", err)
	}
}

// A server that refuses a want for an unadvertised commit is reached only
// through the case's fetch_ref. Here the reviewed commit sits below the tip of
// refs/pull/7/head, and the clone speaks protocol v0, which asks by
// advertisement: fetching the sha fails, and the fetch_ref brings it.
func TestReplayCheckoutFallsBackToTheFetchRef(t *testing.T) {
	clone, _, reviewed := replayRemote(t)
	bare := git(t, clone, "remote", "get-url", "origin")
	seed := filepath.Join(filepath.Dir(bare), "seed")
	if err := os.WriteFile(filepath.Join(seed, "c.txt"), []byte("c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, seed, "add", "c.txt")
	git(t, seed, "commit", "-q", "-m", "a later push to the same pull request")
	git(t, seed, "push", "-q", "-f", bare, "HEAD:refs/pull/7/head")
	git(t, clone, "config", "protocol.version", "0")

	without, err := CheckoutReplay(clone, Case{ID: "no-ref", ReviewedSHA: reviewed}, filepath.Join(t.TempDir(), "run"), 1)
	if err == nil {
		_ = without.Remove()
		t.Fatal("the commit was fetched by sha, so this fixture does not reach the fetch_ref")
	}
	r, err := CheckoutReplay(clone, Case{ID: "with-ref", ReviewedSHA: reviewed, FetchRef: "refs/pull/7/head"}, filepath.Join(t.TempDir(), "run"), 1)
	if err != nil {
		t.Fatalf("the fetch_ref did not resolve the commit: %v", err)
	}
	t.Cleanup(func() { _ = r.Remove() })
	if head := git(t, r.Dir, "rev-parse", "HEAD"); head != reviewed {
		t.Errorf("replay HEAD = %s, want %s", head, reviewed)
	}
}

// The case id names a directory, so one that could leave the run directory is
// refused before anything is checked out.
func TestReplayCheckoutRefusesACaseIdThatIsNotAName(t *testing.T) {
	clone, onMain, _ := replayRemote(t)
	for _, id := range []string{"", "../escape", "a/b", "Upper"} {
		_, err := CheckoutReplay(clone, Case{ID: id, ReviewedSHA: onMain}, filepath.Join(t.TempDir(), "run"), 1)
		if err == nil || !strings.Contains(err.Error(), "case id") {
			t.Errorf("id %q: want a refusal naming the case id, got %v", id, err)
		}
	}
}
