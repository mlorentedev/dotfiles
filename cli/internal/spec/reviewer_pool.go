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

// reviewerPool is the file's shape. `id` is what the GATE matches on; the
// runner/provider/model fields are what the LAUNCHER needs. The `why` prose is
// for humans and is deliberately not parsed, so editorial changes to it can
// never break either consumer.
type reviewerPool struct {
	Pool []ReviewerEntry `json:"pool"`
}

// LoadReviewerPoolEntries returns the full pool, in file order, for callers that
// need to RUN a reviewer rather than merely validate one. The first entry is the
// launcher's primary; the rest are fallbacks, and all of them are equally valid
// signatures as far as the gate is concerned.
//
// Returns (nil, nil) when the repo has no pool, on the same reasoning as
// loadReviewerPool: absent means "no opinion", not "nobody".
func LoadReviewerPoolEntries(repoRoot string) ([]ReviewerEntry, error) {
	return loadReviewerPoolEntries(repoRoot)
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
	entries, err := loadReviewerPoolEntries(repoRoot)
	if err != nil || entries == nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, strings.TrimSpace(e.ID))
	}
	return ids, nil
}

// loadReviewerPoolEntries is the single reader both consumers share, so the gate
// and the launcher can never disagree about what the pool says.
func loadReviewerPoolEntries(repoRoot string) ([]ReviewerEntry, error) {
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

	for i, entry := range p.Pool {
		if strings.TrimSpace(entry.ID) == "" {
			return nil, fmt.Errorf("%s entry %d has a blank id", ReviewerPoolFile, i)
		}
	}
	if len(p.Pool) == 0 {
		return nil, fmt.Errorf("%s declares an empty pool — remove the file to disable the check, or list the models allowed to review", ReviewerPoolFile)
	}
	return p.Pool, nil
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
