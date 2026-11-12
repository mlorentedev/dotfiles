package spec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// slotOfReview is the signature whose review file is named name; the first
// signer's when no slot names it.
func slotOfReview(name string) ReviewSlot {
	if name == SecondSigner.Review {
		return SecondSigner
	}
	return FirstSigner
}

// WriteErrorStub writes the review file a run left absent, with verdict ERROR
// (BUG-168, #1511), and reports whether it wrote one.
//
// A detached launch only knows the reviewer started. Before this, a run that
// died on a spent quota left no review.md, and an absent file reads the same
// as a review nobody launched: the gate looked passed until `spec archive`
// refused, much later. The stub is the failure on disk the moment the run
// ends: who was launched, at which commit, how the run ended, and what the
// reviewer said. A file the run left, signed or provisional, is the
// reviewer's own and is never replaced.
//
// ended says how the run ended; transcript is where it wrote, read for its
// last words. The reviewer and commit come from the launch request beside the
// review file, and the runner from the pool of the checkout the spec folder
// is in, for pi's turn-cap line.
func WriteErrorStub(review, transcript, ended string) (bool, error) {
	if _, err := os.Lstat(review); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	specDir := filepath.Dir(review)
	specID := filepath.Base(specDir)
	slot := slotOfReview(filepath.Base(review))
	req, _, reqErr := ReadReviewRequestIn(specDir, slot)

	var b strings.Builder
	fmt.Fprintf(&b, "---\nspec: %q\nverdict: %s\nreviewer: %q\nreviewed_sha: %q\ndate: %q\n---\n\n",
		specID, VerdictError, req.Reviewer, req.ReviewedSHA, time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "# Review ended without a verdict\n\n"+
		"Written by `dotf spec deadline`, not by the reviewer: the run ended and left no %s. "+
		"Nothing was reviewed, and `dotf spec archive` refuses this file. "+
		"Run `dotf spec review %s%s` again, or choose another member with `--reviewer`.\n\n",
		slot.Review, specID, slot.Flag())
	if reqErr != nil {
		// Loud, as the request's own reader is: an empty reviewer with no
		// reason looks like a launcher bug, not a damaged sidecar.
		fmt.Fprintf(&b, "The launch request could not be read, so the reviewer and commit above are empty: %v\n\n", reqErr)
	}
	fmt.Fprintf(&b, "## How the run ended\n\n%s\n", ended)
	if hint := TurnCapHint(runnerOf(filepath.Dir(filepath.Dir(specDir)), req.Reviewer), transcript); hint != "" {
		fmt.Fprintf(&b, "\n%s\n", hint)
	}
	words, ok := ReviewerLastWords(transcript)
	if !ok {
		words = "It wrote nothing on stderr or in the transcript."
	}
	fmt.Fprintf(&b, "\n## What the reviewer wrote\n\n%s\n", words)
	if err := os.WriteFile(review, []byte(b.String()), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
