package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ReviewRequestFile records what THIS repository asked a reviewer to review,
// written before the reviewer starts and never by the reviewer.
//
// The gate it feeds answers a question `review.md` alone cannot: is the verdict
// on disk the one this launch produced? Two live failures make that question
// load-bearing, and neither is hypothetical:
//
//   - Measured 2026-08-21 (#1157): a review ran 25 turns, resolved the right
//     tree, and was ended by the runner's turn cap before writing anything.
//     `dotf spec review` reported success — correctly, since detached launch
//     means it only ever reports that the runner STARTED — and the previous
//     round's verdict was still sitting in review.md, stamped with a sha that
//     was no longer any branch. A stale verdict left in place is indistinguishable
//     from a fresh one that reached the same conclusion.
//   - The same round's reviewer stamped `date: 2026-08-20` on a review run on the
//     21st. The model authors its own frontmatter, so `reviewed_sha` is a claim,
//     not a measurement, and the staleness gate downstream trusts it completely.
//
// So the launcher writes the sha it is actually reviewing, and the digest of
// whatever review.md held beforehand. Both are facts about the launch rather
// than assertions by the reviewed party, which is the entire point.
const ReviewRequestFile = "review-request.json"

// ReviewRequest is the sidecar's on-disk shape. Field names are snake_case to
// match every other registry in this repo.
type ReviewRequest struct {
	// ReviewedSHA is the repository HEAD at launch — what the reviewer was
	// pointed at, independent of what it later says it looked at.
	ReviewedSHA string `json:"reviewed_sha"`
	// Reviewer is the pool id the launcher resolved, so a verdict signed by a
	// different model is visible even when that model is itself pool-admitted.
	Reviewer string `json:"reviewer"`
	// RequestedAt is for humans reading the folder; nothing gates on it,
	// because a timestamp the launcher writes proves only when it ran.
	RequestedAt string `json:"requested_at"`
	// ReviewDigestBefore is the SHA-256 of review.md at launch, or "" when no
	// review.md existed. A digest that has not moved means the reviewer wrote
	// no verdict — the one case the launcher provably cannot observe itself.
	ReviewDigestBefore string `json:"review_digest_before"`
	// BaseSHA is the commit the spec's work starts FROM, so the reviewer has a
	// diff to read instead of a folder to browse.
	//
	// The skill specifies the review scope as `git diff <base>...HEAD`, and
	// until this field existed the launcher recorded no base at all: the
	// reviewer had to guess it, guessed `main`, and a review launched on `main`
	// after the work merged therefore diffed nothing. Measured on BUG-093 round
	// 4 — it declared `git diff main...HEAD` as its source, that diff was empty,
	// and both of its findings say "Code read" with nothing executed. One of the
	// two was factually wrong, having read a mutation payload out of the spec
	// folder as if it were the implementation.
	//
	// omitempty because specs archived before this field existed carry the old
	// shape and must still validate.
	BaseSHA string `json:"base_sha,omitempty"`
}

// HeadSHA returns the repository HEAD, or "" when repoRoot is not a checkout.
//
// Empty rather than an error on purpose: a missing HEAD must not stop a review
// from launching. It degrades the provenance gate to "not asserted", which is
// honest, where refusing to launch would make the guard a liability.
func HeadSHA(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ResolveReviewBase returns the commit the spec's work starts from: the PARENT
// of the commit that first added specDir.
//
// Why this and not the PR's base branch. The PR route needs the network, `gh`
// auth and PR metadata a human can edit, and it answers differently depending
// on which of a spec's several PRs you ask about. This is local, offline and
// deterministic, and it is correct in BOTH of the situations that matter:
//
//   - review on the work branch — the adding commit is on the branch, so the
//     parent is where the branch left main.
//   - review on main after a squash merge — the adding commit IS the squash
//     commit, so the parent is main immediately before the work landed.
//
// That second case is the whole point: it is the one the old code got wrong by
// having no answer at all.
//
// Returns "" when there is no such commit (a spec folder not yet committed),
// which the caller must treat as "cannot review", not as "review everything".
func ResolveReviewBase(repoRoot, specDir string) string {
	rel, err := filepath.Rel(repoRoot, specDir)
	if err != nil {
		rel = specDir
	}
	// --diff-filter=A finds the commit that ADDED the folder; the last line of
	// a reverse-chronological log is the earliest such commit. `--` guards a
	// path that could be read as a revision.
	out, err := exec.Command("git", "-C", repoRoot,
		"log", "--diff-filter=A", "--format=%H", "--", rel).Output()
	if err != nil {
		return ""
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return ""
	}
	adding := lines[len(lines)-1]

	parent, err := exec.Command("git", "-C", repoRoot, "rev-parse", adding+"^").Output()
	if err != nil {
		// The adding commit is the repository's root commit: there is no parent,
		// and the empty tree is the honest base.
		return ""
	}
	return strings.TrimSpace(string(parent))
}

// fileDigest returns the SHA-256 of path, or "" when it does not exist.
//
// "" is also what an unreadable file yields, and that is deliberate: it makes
// the later comparison fail OPEN into "the digest moved", never into a silent
// pass. A guard that cannot read the file must not conclude the file is fine.
func fileDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// WriteReviewRequest records the launch in specDir, replacing any previous one.
//
// Replacing rather than appending: the sidecar describes the CURRENT outstanding
// request, and a history of requests is what the transcript is for.
func WriteReviewRequest(specDir, reviewedSHA, reviewer, baseSHA string) error {
	req := ReviewRequest{
		ReviewedSHA:        reviewedSHA,
		Reviewer:           reviewer,
		RequestedAt:        time.Now().UTC().Format(time.RFC3339),
		ReviewDigestBefore: fileDigest(filepath.Join(specDir, ReviewFile)),
		BaseSHA:            baseSHA,
	}
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the review request: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(specDir, ReviewRequestFile), data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", ReviewRequestFile, err)
	}
	return nil
}

// ReadReviewRequest loads the sidecar. found is false when there is none, which
// is not an error: reviews predating this file, and hand-written ones, are still
// governed by the verdict and staleness checks.
func ReadReviewRequest(specDir string) (ReviewRequest, bool, error) {
	data, err := os.ReadFile(filepath.Join(specDir, ReviewRequestFile))
	if os.IsNotExist(err) {
		return ReviewRequest{}, false, nil
	}
	if err != nil {
		return ReviewRequest{}, false, fmt.Errorf("reading %s: %w", ReviewRequestFile, err)
	}
	var req ReviewRequest
	if err := json.Unmarshal(data, &req); err != nil {
		// Loud, not skipped. An unparseable sidecar is the shape C15 forbids:
		// treating it as absent would silently drop the guard exactly when the
		// file that carries it is damaged.
		return ReviewRequest{}, true, fmt.Errorf("%s is not valid JSON: %w", ReviewRequestFile, err)
	}
	return req, true, nil
}

// VerifyReviewProduced reports whether the run that has just finished left a
// verdict, and is the launcher's own half of the guard checkReviewProvenance
// enforces later.
//
// Later is the problem it fixes. The archive gate answers the same question, but
// only when somebody next tries to archive -- which can be days on, in another
// session, with the transcript no longer in hand. A FOREGROUND run knows the
// answer the moment the runner exits, and every Windows run is one, because tmux
// is Linux-only.
//
// Measured 2026-08-29 (#1383): AI-042 round 4 ran 40 minutes, made 248 bash calls,
// wrote no review.md, and `dotf spec review` exited 0. A review that produced no
// file is a failed review, not a green one.
//
// Detached launches are out of scope by construction: the command returns while
// the reviewer is still running, so there is nothing yet to verify.
func VerifyReviewProduced(specDir, transcript string) error {
	digest := fileDigest(filepath.Join(specDir, ReviewFile))
	if digest == "" {
		return fmt.Errorf("the reviewer exited without writing %s -- that is a failed review, not a passing one\n"+
			"what it did instead is in the transcript: %s\n"+
			"re-run the review (a run ended by a turn cap, a rate limit, or a reviewer that talked itself out of the job leaves exactly this state)",
			ReviewFile, transcript)
	}

	req, found, err := ReadReviewRequest(specDir)
	if err != nil {
		return err
	}
	if !found {
		// No sidecar means no digest to compare against. The file exists, which
		// is all this check can honestly assert without one.
		return nil
	}
	if req.ReviewDigestBefore != "" && req.ReviewDigestBefore == digest {
		return fmt.Errorf("%s is byte-identical to what it held before this run -- the reviewer wrote no verdict\n"+
			"what is on disk is the PREVIOUS round's, which is not a review of this change\n"+
			"the transcript of the run that wrote nothing: %s",
			ReviewFile, transcript)
	}
	return nil
}
