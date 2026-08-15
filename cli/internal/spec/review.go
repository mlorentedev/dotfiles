package spec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReviewFile is the artifact the adversarial-review skill persists into the
// spec folder, and the one Archive gates on.
const ReviewFile = "review.md"

// contractFiles are the spec files a review's verdict is *about*. Staleness is
// measured against these and nothing else.
//
// review.md is excluded because its own commit always postdates the sha it
// records, so including it would make every review stale by construction.
// verification.md is excluded because its archive checklist is ticked AT
// archive time, which would false-positive on every archive. Both exclusions
// are deliberate and specified in the proposal.
var contractFiles = []string{"proposal.md", "tasks.md", "features.json"}

// Verdict is the adversarial-review outcome recorded in review.md frontmatter.
type Verdict string

const (
	VerdictPass         Verdict = "PASS"
	VerdictPassWithGaps Verdict = "PASS-WITH-GAPS"
	VerdictFail         Verdict = "FAIL"
)

// Blocks reports whether this verdict must stop an archive. Anything that is
// not a recognized passing verdict blocks: an unknown value is a malformed
// artifact, and for a gate the safe direction is to refuse.
func (v Verdict) Blocks() bool {
	return v != VerdictPass && v != VerdictPassWithGaps
}

// Review is the parsed review.md frontmatter.
type Review struct {
	Spec        string
	Verdict     Verdict
	ReviewedSHA string
	Reviewer    string
	Date        string
}

// frontmatterFields reads `key: value` pairs from the first `---` block.
//
// Values are unquoted when quoted, and an unquoted value has any trailing
// `# comment` stripped — the scaffolded templates carry those. A quoted value
// is taken verbatim up to its closing quote, so a `#` inside quotes (an issue
// reference like "mlorentedev/dotfiles#875") survives intact.
func frontmatterFields(content string) map[string]string {
	fields := map[string]string{}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, rest, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(key) != key || key == "" {
			continue // body text, a nested mapping, or a list item
		}
		value := strings.TrimSpace(rest)
		if len(value) > 1 && (value[0] == '"' || value[0] == '\'') {
			if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
				value = value[1 : 1+end]
			}
		} else if idx := strings.Index(value, "#"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}
		fields[key] = value
	}
	return fields
}

// ParseReview reads review.md content into a Review. A missing or unrecognized
// verdict is an error: the artifact exists but does not say anything the gate
// can act on, which is worse than absent and must not pass silently.
func ParseReview(content string) (Review, error) {
	f := frontmatterFields(content)
	r := Review{
		Spec:        f["spec"],
		Verdict:     Verdict(strings.ToUpper(f["verdict"])),
		ReviewedSHA: f["reviewed_sha"],
		Reviewer:    f["reviewer"],
		Date:        f["date"],
	}
	if r.Verdict == "" {
		return r, fmt.Errorf("%s has no `verdict:` field in its frontmatter", ReviewFile)
	}
	if r.Verdict != VerdictPass && r.Verdict != VerdictPassWithGaps && r.Verdict != VerdictFail {
		return r, fmt.Errorf("%s has an unrecognized verdict %q (want PASS, PASS-WITH-GAPS or FAIL)", ReviewFile, r.Verdict)
	}
	if r.ReviewedSHA == "" {
		return r, fmt.Errorf("%s has no `reviewed_sha:` field — the review cannot be checked for staleness", ReviewFile)
	}
	return r, nil
}

// FindReview loads specDir/review.md. found is false when the file is absent,
// which the caller reports differently from a malformed one.
func FindReview(specDir string) (review Review, found bool, err error) {
	data, readErr := os.ReadFile(filepath.Join(specDir, ReviewFile))
	if os.IsNotExist(readErr) {
		return Review{}, false, nil
	}
	if readErr != nil {
		return Review{}, false, readErr
	}
	parsed, parseErr := ParseReview(string(data))
	return parsed, true, parseErr
}

// StalenessChecker decides whether a review still describes the spec's
// contract files. It is an interface so tests can drive the decision without
// building a git history for every case.
type StalenessChecker interface {
	// Stale reports whether the review is out of date. known is false when the
	// question cannot be answered at all (no git history to compare against),
	// in which case the caller skips the check rather than guessing.
	Stale(repoRoot, specID, reviewedSHA string) (stale bool, known bool, reason string)
}

// gitStaleness answers the question from the repository's own history.
type gitStaleness struct{}

func (gitStaleness) Stale(repoRoot, specID, reviewedSHA string) (bool, bool, string) {
	if err := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-dir").Run(); err != nil {
		// Not a work tree: there is no history for the review to be stale
		// against. The presence and verdict checks still applied.
		return false, false, ""
	}
	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", reviewedSHA+"^{commit}").Run(); err != nil {
		return true, true, fmt.Sprintf("reviewed_sha %s is not a commit in this history (rewritten by a rebase?)", reviewedSHA)
	}

	args := []string{"-C", repoRoot, "log", "--format=%H", reviewedSHA + "..HEAD", "--"}
	for _, name := range contractFiles {
		args = append(args, filepath.Join("specs", specID, name))
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return true, true, "could not compare the review against the current history"
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return true, true, fmt.Sprintf("a contract file (%s) changed after reviewed_sha %s",
			strings.Join(contractFiles, ", "), reviewedSHA)
	}

	// Committed history is not the whole answer. An uncommitted edit to a
	// contract file is the same defect — the review no longer describes what is
	// on disk — and it is the cheaper bypass of the two: get a passing review,
	// rewrite the acceptance criteria in the working tree, archive. `git log`
	// cannot see it, and "uncommitted" is exactly the state a spec sits in while
	// someone edits it.
	//
	// Scoped to the contract files, not the spec folder: a review in flight
	// writes review.md and its transcript into that same folder, so a directory
	// -wide check would make every review stale by producing its own output.
	dirty := []string{"-C", repoRoot, "status", "--porcelain", "--"}
	for _, name := range contractFiles {
		dirty = append(dirty, filepath.Join("specs", specID, name))
	}
	out, err = exec.Command("git", dirty...).Output()
	if err != nil {
		return true, true, "could not check the working tree for uncommitted contract changes"
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return true, true, fmt.Sprintf("a contract file (%s) has uncommitted changes, so the review does not describe what is on disk",
			strings.Join(contractFiles, ", "))
	}
	return false, true, ""
}

// reviewWaiver reads the declared waiver from proposal.md frontmatter. A waiver
// counts only when it also carries a non-empty reason: requiring the reason is
// what keeps a routine waive visible in the diff instead of invisible in a
// habit.
func reviewWaiver(specDir string) (waived bool, reason string) {
	data, err := os.ReadFile(filepath.Join(specDir, "proposal.md"))
	if err != nil {
		return false, ""
	}
	f := frontmatterFields(string(data))
	if strings.ToLower(f["review"]) != "waived" {
		return false, ""
	}
	return true, strings.TrimSpace(f["review_waived_reason"])
}

// checkReviewGate is Archive's second pre-flight. It returns nil when the spec
// may be archived, and an error naming both declared escapes otherwise.
func checkReviewGate(repoRoot, specID, specDir string, checker StalenessChecker) error {
	if waived, reason := reviewWaiver(specDir); waived {
		if reason == "" {
			return fmt.Errorf("proposal.md declares `review: waived` without a reason\n" +
				"add a non-empty `review_waived_reason:` so the waiver is auditable, or archive with --force-without-review")
		}
		return nil
	}

	review, found, err := FindReview(specDir)
	if !found {
		return fmt.Errorf("no %s in the spec folder — run /adversarial-review before archiving\n"+
			"to proceed without one, declare `review: waived` with a `review_waived_reason:` in proposal.md, or pass --force-without-review",
			ReviewFile)
	}
	if err != nil {
		return fmt.Errorf("%w\nfix the artifact, declare `review: waived` with a reason in proposal.md, or pass --force-without-review", err)
	}
	// A review.md copied from a sibling spec would otherwise satisfy the gate
	// while describing a different change — the copy-paste analogue of the
	// one-line alibi SPEC_FLOOR exists to defeat in check-spec-gate.sh.
	if review.Spec != "" && review.Spec != specID {
		return fmt.Errorf("%s declares spec %q but lives in %q — the review describes a different change\n"+
			"re-run /adversarial-review for this spec, or pass --force-without-review",
			ReviewFile, review.Spec, specID)
	}
	if review.Verdict.Blocks() {
		return fmt.Errorf("%s records verdict %s — address the findings and re-review before archiving\n"+
			"to override, declare `review: waived` with a reason in proposal.md, or pass --force-without-review",
			ReviewFile, review.Verdict)
	}

	if checker == nil {
		checker = gitStaleness{}
	}
	if stale, known, reason := checker.Stale(repoRoot, specID, review.ReviewedSHA); known && stale {
		return fmt.Errorf("%s is stale: %s\nre-run /adversarial-review against the current head, declare `review: waived` with a reason in proposal.md, or pass --force-without-review",
			ReviewFile, reason)
	}

	// Last, because it is the only check that asks WHO reviewed rather than
	// what they concluded — a valid, fresh, passing review signed by the wrong
	// model is still a self-review, and the earlier checks cannot see that.
	return checkReviewerPool(repoRoot, review.Reviewer)
}
