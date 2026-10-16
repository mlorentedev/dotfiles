package lane

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// A claim is the ref refs/dotf/claims/<issue>, pointing at an empty-tree commit
// whose message names the holder (ADR-050 decision 2). Taking it is one push
// with an empty expected value, so the server decides the winner atomically:
// a read-then-write on a forge assignee cannot.
//
// The ref lives outside refs/heads: branch protection does not see it, no push
// workflow runs for it, and a default clone does not fetch it.
const (
	claimRefPrefix  = "refs/dotf/claims/"
	claimSeenPrefix = "refs/dotf/seen/"
	// emptyTree is the one tree every git repository knows without storing it.
	emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
)

// Claim is a held claim, as read back from its commit.
type Claim struct {
	Issue  int
	Holder string
	At     time.Time
}

// ClaimedError reports that an issue is already claimed, and by whom.
type ClaimedError struct {
	Issue  int
	Holder string
	At     time.Time
}

func (e *ClaimedError) Error() string {
	// A claim commit whose time cannot be read still names its holder, and
	// refusing to release it over that would strand it: say the time is unknown.
	since := "an unknown time"
	if !e.At.IsZero() {
		since = e.At.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("#%d is claimed by %s since %s", e.Issue, e.Holder, since)
}

// Claimer takes, reads and releases claims on Remote, from the repository in
// Dir. As is who a claim is taken as; it is written into the claim commit,
// which is the only record of it.
type Claimer struct {
	Run    Runner
	Dir    string
	Remote string
	As     string
	Now    func() time.Time
}

// Claims is what a hand-off needs of the claim store: take a claim, and ask
// whether one is held without taking it (a dry run changes nothing).
type Claims interface {
	Acquire(issue int) (Claim, error)
	Held(issue int) (bool, error)
}

func claimRef(issue int) string  { return claimRefPrefix + strconv.Itoa(issue) }
func claimSeen(issue int) string { return claimSeenPrefix + strconv.Itoa(issue) }

func (c Claimer) git(args ...string) (string, error) {
	return c.Run("git", append([]string{"-C", c.Dir}, args...)...)
}

// push publishes the claim commit, or deletes the ref when sha is empty. The
// hooks are skipped: a claim publishes no branch, and the pre-push gates read
// one.
func (c Claimer) push(issue int, expect, sha string) error {
	ref := claimRef(issue)
	src := sha + ":" + ref
	if sha == "" {
		src = ":" + ref
	}
	_, err := c.git("push", "--quiet", "--no-verify", "--force-with-lease="+ref+":"+expect, c.Remote, src)
	return err
}

// Acquire takes the claim on issue. It returns a *ClaimedError naming the
// holder when somebody else got there first; any other error means the claim
// state is unknown, and the caller must start nothing.
func (c Claimer) Acquire(issue int) (Claim, error) {
	now := c.Now().UTC()
	// The nonce makes every claim commit unique. Without it two claims by the
	// same holder inside one second are the same object, and pushing an object
	// the ref already points at is "up to date", not a refusal: both would win.
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return Claim{}, fmt.Errorf("claim #%d: %w", issue, err)
	}
	msg := fmt.Sprintf("claim #%d\n\nholder: %s\nat: %s\nnonce: %s\n", issue, c.As, now.Format(time.RFC3339), hex.EncodeToString(nonce))
	sha, err := c.git("-c", "user.name=dotf", "-c", "user.email=dotf@localhost", "commit-tree", emptyTree, "-m", msg)
	if err != nil {
		return Claim{}, fmt.Errorf("claim #%d: build the claim commit: %w", issue, err)
	}
	sha = strings.TrimSpace(sha)

	pushErr := c.push(issue, "", sha)
	if pushErr == nil {
		return Claim{Issue: issue, Holder: c.As, At: now}, nil
	}
	// A refused push is either a lost race or a failure. The ref says which.
	// An empty read right after a refusal is the lock another claimant held
	// while it was writing: try once more before calling it a failure.
	for attempt := 0; attempt < 2; attempt++ {
		held, ok, err := c.Holder(issue)
		if err != nil {
			return Claim{}, fmt.Errorf("claim #%d: push refused (%v) and the holder is unreadable: %w", issue, pushErr, err)
		}
		if ok {
			return Claim{}, &ClaimedError{Issue: issue, Holder: held.Holder, At: held.At}
		}
		if attempt == 0 {
			if pushErr = c.push(issue, "", sha); pushErr == nil {
				return Claim{Issue: issue, Holder: c.As, At: now}, nil
			}
		}
	}
	return Claim{}, fmt.Errorf("claim #%d: %w", issue, pushErr)
}

// Held says whether anybody holds the claim on issue, without reading who.
func (c Claimer) Held(issue int) (bool, error) {
	out, err := c.git("ls-remote", c.Remote, claimRef(issue))
	if err != nil {
		return false, fmt.Errorf("ls-remote %s: %w", claimRef(issue), err)
	}
	return strings.TrimSpace(out) != "", nil
}

// Holder reads the claim on issue. The bool is false when nobody holds it.
func (c Claimer) Holder(issue int) (Claim, bool, error) {
	ref := claimRef(issue)
	out, err := c.git("ls-remote", c.Remote, ref)
	if err != nil {
		return Claim{}, false, fmt.Errorf("ls-remote %s: %w", ref, err)
	}
	if strings.TrimSpace(out) == "" {
		return Claim{}, false, nil
	}
	// The claim commit carries the holder, so read it into a private ref
	// rather than keep a second record somewhere.
	if _, err := c.git("fetch", "--quiet", "--no-tags", c.Remote, "+"+ref+":"+claimSeen(issue)); err != nil {
		return Claim{}, false, fmt.Errorf("fetch %s: %w", ref, err)
	}
	body, err := c.git("cat-file", "commit", claimSeen(issue))
	if err != nil {
		return Claim{}, false, fmt.Errorf("read the claim on #%d: %w", issue, err)
	}
	cl := Claim{Issue: issue}
	for _, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(line, "holder: "):
			cl.Holder = strings.TrimPrefix(line, "holder: ")
		case strings.HasPrefix(line, "at: "):
			cl.At, _ = time.Parse(time.RFC3339, strings.TrimPrefix(line, "at: "))
		}
	}
	if cl.Holder == "" {
		return Claim{}, false, fmt.Errorf("the claim on #%d names no holder", issue)
	}
	return cl, true, nil
}

// Release deletes the claim on issue. Releasing a claim nobody holds is a
// no-op; releasing another holder's takes force, because a stale claim is a
// human decision and never an expiry.
func (c Claimer) Release(issue int, force bool) error {
	held, ok, err := c.Holder(issue)
	if err != nil || !ok {
		return err
	}
	if held.Holder != c.As && !force {
		return &ClaimedError{Issue: issue, Holder: held.Holder, At: held.At}
	}
	sha, err := c.git("rev-parse", claimSeen(issue))
	if err != nil {
		return fmt.Errorf("release #%d: %w", issue, err)
	}
	// The lease pins the sha that was read: a claim retaken in between is not
	// deleted by a release that never saw it.
	if err := c.push(issue, strings.TrimSpace(sha), ""); err != nil {
		return fmt.Errorf("release #%d: %w", issue, err)
	}
	return nil
}
