package spec

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EvalReviewerPrefix marks a reviewer id recorded by an evaluation run. No pool
// id carries it, and the archive gate refuses any signature that does.
const EvalReviewerPrefix = "eval:"

// IsEvalReviewer reports whether a recorded reviewer is an evaluation run's.
func IsEvalReviewer(reviewer string) bool {
	return strings.HasPrefix(strings.TrimSpace(reviewer), EvalReviewerPrefix)
}

// refuseEvalRecord keeps an evaluation answer from signing a spec. It reads
// both the answer's own claim and the launcher's request, because either one
// copied into a spec folder is enough to show where the record came from. It
// runs before the pool check, which has no opinion in a repository without a
// pool.
func refuseEvalRecord(specDir string, slot ReviewSlot, review Review) error {
	recorded := ""
	if IsEvalReviewer(review.Reviewer) {
		recorded = review.Reviewer
	} else if req, found, err := ReadReviewRequestIn(specDir, slot); err == nil && found && IsEvalReviewer(req.Reviewer) {
		recorded = req.Reviewer
	}
	if recorded == "" {
		return nil
	}
	return fmt.Errorf("%s records reviewer %q: it is an evaluation answer, graded against a key, not a review of this spec\n"+
		"an evaluation run never signs a spec; launch `dotf spec review %s%s` for a review that does",
		slot.Review, strings.TrimSpace(recorded), filepath.Base(specDir), slot.Flag())
}
