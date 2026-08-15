package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// poolWasTracked reports whether the pool file is part of this repo's committed
// state. It is how "never enabled" is told apart from "enabled, then lost".
//
// The distinction matters because absence alone is ambiguous, and the two
// meanings need opposite answers: a repo that never had a pool must keep
// working, while a repo whose pool vanished — a bad merge, a stray delete, a
// .gitignore mistake — must not silently stop checking who reviews. Reading git
// rather than adding an "enabled" key keeps the pool one file with one job, and
// follows the precedent already in this package: gitStaleness answers its own
// question from the repository's history too.
//
// It asks HEAD's tree, not the log. `git log -- <path>` also returns the commit
// that DELETED the path, so the original implementation answered true forever
// once the file had ever existed — including after a deliberate, committed
// removal. The error message told the reader to retire the gate "in a commit
// that says so", and doing exactly that left every future archive blocked with
// no way out but --force-without-review. An independent review of this spec
// found it and it reproduces in four git commands.
//
// So the boundary is now: present in HEAD but missing from the working tree →
// lost, fail closed. Absent from HEAD → either never added or deliberately
// removed, and both are recorded in history where a reader can see them. That
// makes a committed deletion the sanctioned off switch; the case this guards
// against was never a documented removal but a silent one.
func poolWasTracked(repoRoot string) bool {
	if err := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-dir").Run(); err != nil {
		return false // not a work tree: no history to ask
	}
	out, err := exec.Command("git", "-C", repoRoot, "ls-tree", "HEAD", "--", ReviewerPoolFile).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

// loadReviewerPool reads the allow-list.
//
// It returns (nil, nil) only when this repo NEVER had a pool — the default
// disabled state, which has to keep working because dotf runs in repos that
// never opted into this gate at all.
//
// A pool that git knows about but that is missing from the tree is the opposite
// case, and it FAILS CLOSED. Deleting the file is not a sanctioned way to switch
// the gate off: that would let a safety check disappear with no signal at the
// moment it stops applying, and it is the "blank the config to hide the work"
// shape this repo's own guidelines reject. The declared escapes are per-spec and
// auditable — `review: waived` with a reason in proposal.md, or
// --force-without-review — and they stay available here.
//
// Every other failure is an error too. A pool that exists but cannot be read is
// the state where the gate cannot tell an allowed reviewer from a forbidden one,
// and passing there would silently downgrade to no gate while looking like one.
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
		if poolWasTracked(repoRoot) {
			return nil, fmt.Errorf("%s is in HEAD but missing from the working tree — the reviewer gate was enabled and the file is gone\n"+
				"restore it (`git checkout -- %s`), or, if the gate is genuinely being retired, record that: `git rm %s` and commit the removal",
				ReviewerPoolFile, ReviewerPoolFile, ReviewerPoolFile)
		}
		return nil, nil // never enabled here
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
