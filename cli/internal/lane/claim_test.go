package lane

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests run the real git against a local bare remote: the claim is a
// property of the push, so a fake Runner would assert nothing about it.

func claimGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func claimRemote(t *testing.T) string {
	t.Helper()
	remote := t.TempDir()
	claimGit(t, remote, "init", "-q", "--bare")
	return remote
}

// claimerIn is one session: its own clone-less working directory that points
// at the shared remote.
func claimerIn(t *testing.T, remote, holder string) Claimer {
	t.Helper()
	dir := t.TempDir()
	claimGit(t, dir, "init", "-q")
	claimGit(t, dir, "remote", "add", "origin", remote)
	return Claimer{
		Run:    ExecRunner,
		Dir:    dir,
		Remote: "origin",
		As:     holder,
		Now:    func() time.Time { return time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC) },
	}
}

func TestClaimAcquire(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	b := claimerIn(t, remote, "bo@box-b")

	if _, err := a.Acquire(2551); err != nil {
		t.Fatalf("first Acquire of a free issue: %v", err)
	}
	// The same claimant and a different one are both refused: the claim is
	// held, and holding it twice is not a second win.
	for name, c := range map[string]Claimer{"same holder": a, "other holder": b} {
		_, err := c.Acquire(2551)
		var held *ClaimedError
		if !errors.As(err, &held) {
			t.Fatalf("%s: Acquire of a held issue = %v, want *ClaimedError", name, err)
		}
	}
	// Another issue is untouched.
	if _, err := b.Acquire(2552); err != nil {
		t.Fatalf("Acquire of another issue: %v", err)
	}
}

func TestClaimedErrorNamesTheHolder(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	b := claimerIn(t, remote, "bo@box-b")
	if _, err := a.Acquire(2551); err != nil {
		t.Fatal(err)
	}

	_, err := b.Acquire(2551)
	var held *ClaimedError
	if !errors.As(err, &held) {
		t.Fatalf("err = %v, want *ClaimedError", err)
	}
	if held.Holder != "ana@box-a" || held.Issue != 2551 {
		t.Errorf("ClaimedError = %+v, want issue 2551 held by ana@box-a", held)
	}
	if !held.At.Equal(time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("claimed at %v, want the time written in the claim commit", held.At)
	}
	for _, want := range []string{"#2551", "ana@box-a", "2026-10-16"} {
		if !strings.Contains(held.Error(), want) {
			t.Errorf("message %q lacks %q", held.Error(), want)
		}
	}
}

func TestClaimConcurrent(t *testing.T) {
	remote := claimRemote(t)
	const n = 8
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners int
		losers  []error
	)
	for i := 0; i < n; i++ {
		c := claimerIn(t, remote, "session-"+string(rune('a'+i)))
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Acquire(2551)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				winners++
				return
			}
			losers = append(losers, err)
		}()
	}
	wg.Wait()

	if winners != 1 {
		t.Fatalf("winners = %d, want exactly 1 (losers: %v)", winners, losers)
	}
	for _, err := range losers {
		var held *ClaimedError
		if !errors.As(err, &held) {
			t.Errorf("a loser got %v, want *ClaimedError", err)
		}
	}
}

func TestClaimIsInvisibleToADefaultClone(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	// Give the remote a branch so a clone has something to fetch.
	claimGit(t, a.Dir, "-c", "user.name=t", "-c", "user.email=t@x", "commit", "-q", "--allow-empty", "-m", "seed")
	claimGit(t, a.Dir, "push", "-q", "origin", "HEAD:refs/heads/main")
	if _, err := a.Acquire(2551); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	claimGit(t, clone, "clone", "-q", remote, ".")
	if refs := claimGit(t, clone, "for-each-ref", "--format=%(refname)"); strings.Contains(refs, "dotf") {
		t.Errorf("a default clone fetched the claim: %s", refs)
	}
	if heads := claimGit(t, clone, "ls-remote", "--heads", "origin"); strings.Contains(heads, "dotf") {
		t.Errorf("the claim shows up as a branch: %s", heads)
	}
}

func TestClaimRelease(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	b := claimerIn(t, remote, "bo@box-b")
	if _, err := a.Acquire(2551); err != nil {
		t.Fatal(err)
	}

	var held *ClaimedError
	if err := b.Release(2551, false); !errors.As(err, &held) {
		t.Fatalf("release of another holder's claim = %v, want *ClaimedError", err)
	}
	if err := a.Release(2551, false); err != nil {
		t.Fatalf("release of own claim: %v", err)
	}
	if err := a.Release(2551, false); err != nil {
		t.Errorf("a second release must be a no-op, got %v", err)
	}
	if _, err := b.Acquire(2551); err != nil {
		t.Errorf("a released issue must be claimable: %v", err)
	}
	if err := a.Release(2551, true); err != nil {
		t.Errorf("--force release of another holder's claim: %v", err)
	}
	if _, ok, err := a.Holder(2551); err != nil || ok {
		t.Errorf("after a forced release Holder = (%v, %v), want none", ok, err)
	}
}

// TestClaimRefFormOnARemote is the ADR-050 probe: does a forge accept the
// claim ref and refuse a second push to it? It touches a real remote, so it
// runs only when DOTF_CLAIM_PROBE_REMOTE names one (a URL or a path), and it
// uses an issue number no queue can hold. The ref is deleted at the end.
func TestClaimRefFormOnARemote(t *testing.T) {
	remote := os.Getenv("DOTF_CLAIM_PROBE_REMOTE")
	if remote == "" {
		t.Skip("DOTF_CLAIM_PROBE_REMOTE not set")
	}
	issue := 900000000 + int(time.Now().Unix()%1000000)
	a := claimerIn(t, remote, "probe-a")
	b := claimerIn(t, remote, "probe-b")
	t.Cleanup(func() { _ = a.Release(issue, true) })

	if _, err := a.Acquire(issue); err != nil {
		t.Fatalf("the remote refused the claim ref: %v", err)
	}
	var held *ClaimedError
	if _, err := b.Acquire(issue); !errors.As(err, &held) || held.Holder != "probe-a" {
		t.Fatalf("a second push to the claim ref = %v, want *ClaimedError held by probe-a", err)
	}
	if err := a.Release(issue, false); err != nil {
		t.Fatalf("the remote refused deleting the claim ref: %v", err)
	}
	if _, ok, err := a.Holder(issue); err != nil || ok {
		t.Fatalf("after release Holder = (%v, %v), want none", ok, err)
	}
}

func TestClaimedErrorWithoutATimeSaysSo(t *testing.T) {
	msg := (&ClaimedError{Issue: 7, Holder: "ana@box-a"}).Error()
	if !strings.Contains(msg, "unknown time") || strings.Contains(msg, "0001") {
		t.Errorf("message %q must say the time is unknown, not print the zero time", msg)
	}
}

// ls-remote matches a pattern by its tail: a branch that ends like a claim ref
// must not read as one.
func TestClaimIgnoresARefThatOnlyEndsLikeTheClaim(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	claimGit(t, a.Dir, "-c", "user.name=t", "-c", "user.email=t@x", "commit", "-q", "--allow-empty", "-m", "seed")
	claimGit(t, a.Dir, "push", "-q", "origin", "HEAD:refs/heads/x/refs/dotf/claims/7")

	if held, err := a.Held(7); err != nil || held {
		t.Errorf("Held(7) = (%v, %v) with only a look-alike branch, want (false, nil)", held, err)
	}
	if _, ok, err := a.Holder(7); err != nil || ok {
		t.Errorf("Holder(7) = (%v, %v) with only a look-alike branch, want none", ok, err)
	}
	if _, err := a.Acquire(7); err != nil {
		t.Errorf("Acquire(7) = %v, want a win: nobody holds it", err)
	}
}

func TestClaimList(t *testing.T) {
	remote := claimRemote(t)
	a := claimerIn(t, remote, "ana@box-a")
	b := claimerIn(t, remote, "bo@box-b")
	if got, err := a.List(); err != nil || len(got) != 0 {
		t.Fatalf("List on a remote with no claims = %v, %v", got, err)
	}
	if _, err := b.Acquire(2552); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Acquire(2551); err != nil {
		t.Fatal(err)
	}
	// A look-alike branch and a non-numeric claim ref are not claims.
	claimGit(t, a.Dir, "-c", "user.name=t", "-c", "user.email=t@x", "commit", "-q", "--allow-empty", "-m", "seed")
	claimGit(t, a.Dir, "push", "-q", "origin", "HEAD:refs/heads/x/refs/dotf/claims/9", "HEAD:refs/dotf/claims/not-a-number")

	got, err := a.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Issue != 2551 || got[0].Holder != "ana@box-a" || got[1].Issue != 2552 || got[1].Holder != "bo@box-b" {
		t.Errorf("List = %+v, want #2551 by ana and #2552 by bo, in that order", got)
	}
}
