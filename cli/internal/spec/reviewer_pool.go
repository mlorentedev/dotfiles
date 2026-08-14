package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReviewerPoolFile is the repo-relative allow-list of models permitted to sign
// a review.md (HARNESS-071, #955).
const ReviewerPoolFile = "harness/reviewer-pool.json"

// reviewerPool is the subset of that file this gate consumes. The file carries
// more per-entry prose (runner, role, why) for humans; only the id is a
// contract, so parsing just that keeps the gate indifferent to editorial
// changes in the rest.
type reviewerPool struct {
	Pool []struct {
		ID string `json:"id"`
	} `json:"pool"`
}

// loadReviewerPool reads the allow-list.
//
// It returns (nil, nil) when the file does not exist, which means "this repo has
// no opinion about who may review" rather than "nobody may". dotf runs in repos
// that have no pool at all, and a gate that refused everywhere a pool is absent
// would break them. Deleting the pool therefore disables the check — deliberately,
// because that deletion is a visible diff, the same auditable-escape philosophy
// as `review: waived` needing a stated reason.
//
// Every other failure is an error. A pool that exists but cannot be read is the
// state where the gate cannot tell an allowed reviewer from a forbidden one, and
// passing there would silently downgrade to no gate while looking like one.
func loadReviewerPool(repoRoot string) ([]string, error) {
	path := filepath.Join(repoRoot, ReviewerPoolFile)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s exists but could not be read: %w", ReviewerPoolFile, err)
	}

	var p reviewerPool
	if jErr := json.Unmarshal(raw, &p); jErr != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", ReviewerPoolFile, jErr)
	}

	ids := make([]string, 0, len(p.Pool))
	for i, entry := range p.Pool {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			return nil, fmt.Errorf("%s entry %d has a blank id", ReviewerPoolFile, i)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s declares an empty pool — remove the file to disable the check, or list the models allowed to review", ReviewerPoolFile)
	}
	return ids, nil
}

// checkReviewerPool refuses a review signed by a model outside the pool.
//
// This is the layer that makes reviewer independence true when nobody is
// watching. The rest of the mechanism — a launcher that picks the right model, a
// standing rule that the reviewer is never the implementer — is convention, and
// convention is exactly what failed: BUG-074 was reviewed twice by
// claude-opus-5, the same model family that wrote it, not by decision but by
// path of least resistance. Both reviews happened to be rigorous, which is what
// makes this worth fixing before a cheap PASS rather than after one.
//
// Matching is exact on the trimmed id — forgiving about spacing, strict about
// identity. A prefix match would be actively wrong here: `agy` serves
// claude-opus-4-6-thinking alongside the Gemini family, so ids, not tool names,
// are what carry the guarantee.
//
// The guarantee's real bound, stated plainly because the spec must not imply a
// stronger one: `reviewer:` is SELF-REPORTED. This defends against habit and
// accident, not against an agent that lies about its identity — the same trust
// level as the verdict field sitting three lines above it.
func checkReviewerPool(repoRoot, reviewer string) error {
	pool, err := loadReviewerPool(repoRoot)
	if err != nil {
		return fmt.Errorf("%w\nfix the file, or archive with --force-without-review", err)
	}
	if pool == nil {
		return nil // no pool, no opinion
	}

	found := strings.TrimSpace(reviewer)
	for _, id := range pool {
		if found == id {
			return nil
		}
	}

	shown := found
	if shown == "" {
		shown = "(none recorded)"
	}
	return fmt.Errorf("%s records reviewer %q, which is not in %s\n"+
		"the models allowed to review are: %s\n"+
		"re-run /adversarial-review on one of them, declare `review: waived` with a reason in proposal.md, or pass --force-without-review",
		ReviewFile, shown, ReviewerPoolFile, strings.Join(pool, ", "))
}
