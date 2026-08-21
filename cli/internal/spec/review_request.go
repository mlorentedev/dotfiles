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
func WriteReviewRequest(specDir, reviewedSHA, reviewer string) error {
	req := ReviewRequest{
		ReviewedSHA:        reviewedSHA,
		Reviewer:           reviewer,
		RequestedAt:        time.Now().UTC().Format(time.RFC3339),
		ReviewDigestBefore: fileDigest(filepath.Join(specDir, ReviewFile)),
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
