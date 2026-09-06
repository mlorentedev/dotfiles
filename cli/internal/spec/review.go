package spec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// verdictSeparators matches any run of the characters a reviewer might use
// between the words of a multi-word verdict.
var verdictSeparators = regexp.MustCompile(`[\s_-]+`)

// normalizeVerdict maps what a reviewer WROTE onto the enum the gate matches,
// differing only in case and word separators.
//
// Observed: an otherwise complete review of HARNESS-072 recorded
// `verdict: "PASS WITH GAPS"`, with spaces. The gate matches `PASS-WITH-GAPS`
// exactly, so `Blocks()` refused the archive over punctuation — a six-minute
// review and a passing verdict, discarded on two characters. The repo already
// carries this lesson about the `reviewer:` field ("the right model spelled
// differently is refused over punctuation, which costs a whole review round");
// this is the same defect one field across.
//
// Deliberately narrow. Only case and the separators between words are
// normalized, so `PASS WITH GAPS`, `pass_with_gaps` and `PASS-WITH-GAPS` are one
// verdict, while `PASSWITHGAPS`, `PASSED` or anything else is still rejected as
// unrecognized. Widening it further would start guessing at intent, and a gate
// that guesses is worse than one that refuses: the safe direction here is to
// accept only spellings that are unambiguously the same words.
func normalizeVerdict(raw string) Verdict {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	return Verdict(strings.ToUpper(verdictSeparators.ReplaceAllString(s, "-")))
}

func ParseReview(content string) (Review, error) {
	f := frontmatterFields(content)
	r := Review{
		Spec:        f["spec"],
		Verdict:     normalizeVerdict(f["verdict"]),
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

	args := []string{"-C", repoRoot, "diff", "--name-only", reviewedSHA, "HEAD", "--"}
	for _, name := range contractFiles {
		args = append(args, filepath.Join("specs", specID, name))
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return true, true, "could not compare the review against the current history"
	}
	if moved := namesIn(string(out)); len(moved) > 0 {
		return true, true, fmt.Sprintf("%s changed after reviewed_sha %s",
			strings.Join(moved, ", "), reviewedSHA)
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
	if dirtyNames := porcelainNames(string(out)); len(dirtyNames) > 0 {
		return true, true, fmt.Sprintf("%s %s uncommitted changes, so the review does not describe what is on disk",
			strings.Join(dirtyNames, ", "), plural(len(dirtyNames), "has", "have"))
	}
	return false, true, ""
}

// namesIn reduces `git log --name-only` output to the distinct base names it
// mentions, in first-seen order.
//
// Naming the files that actually moved is the point (#998): the refusal used
// to recite all of contractFiles whether or not they changed, so an operator
// reading it could not tell a rewritten acceptance criterion from a ticked
// checkbox, and had to reconstruct the diff by hand to find out.
func namesIn(out string) []string {
	var names []string
	seen := map[string]bool{}
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		base := filepath.Base(line)
		if !seen[base] {
			seen[base] = true
			names = append(names, base)
		}
	}
	return names
}

// porcelainNames reduces `git status --porcelain` output to the distinct base
// names it mentions. Each line is "XY <path>", and a rename is "XY <old> -> <new>";
// the path after the arrow is the one on disk now, so that is the one reported.
func porcelainNames(out string) []string {
	var paths []string
	for line := range strings.SplitSeq(out, "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if _, after, found := strings.Cut(path, " -> "); found {
			path = after
		}
		paths = append(paths, path)
	}
	return namesIn(strings.Join(paths, "\n"))
}

// plural picks the verb form for n subjects.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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
	// Provenance before verdict, deliberately. Both later checks read the file's
	// CLAIMS; this one asks whether the file is the answer to the question this
	// repository asked. Put it after, and a stale PASS left behind by a reviewer
	// that wrote nothing would be accepted before anything looked.
	if err := checkReviewProvenance(specDir, review); err != nil {
		return err
	}
	if review.Verdict.Blocks() {
		return fmt.Errorf("%s records verdict %s — address the findings and re-review before archiving\n"+
			"to override, declare `review: waived` with a reason in proposal.md, or pass --force-without-review",
			ReviewFile, review.Verdict)
	}

	if checker == nil {
		checker = gitStaleness{}
	}
	// The first exit named is the one that keeps the review, and it is named
	// first deliberately. The other three all discard or bypass a verdict that
	// may be perfectly good, so an operator offered only those reaches for an
	// override on a spec that has a passing review sitting right there.
	//
	// The common way to land here is applying a passing review's own recommended
	// next steps, which the review skill used to head "(before archive)" — see
	// BUG-093 (#1516), where four of them targeted the contract set. Restoring
	// the contract and recording the dispositions is the correct answer there,
	// and it was not previously on offer.
	if stale, known, reason := checker.Stale(repoRoot, specID, review.ReviewedSHA); known && stale {
		return fmt.Errorf("%s is stale: %s\n"+
			"keeps the review:\n"+
			"  restore the contract files to reviewed_sha and record what changed as dispositions in verification.md (excluded from this check)\n"+
			"discards it — only if the review is genuinely no longer the right one:\n"+
			"  re-run /adversarial-review against the current head\n"+
			"  declare `review: waived` with a reason in proposal.md\n"+
			"  pass --force-without-review",
			ReviewFile, reason)
	}

	// Last, because it is the only check that asks WHO reviewed rather than
	// what they concluded — a valid, fresh, passing review signed by the wrong
	// model is still a self-review, and the earlier checks cannot see that.
	return checkReviewerPool(repoRoot, review.Reviewer)
}

// checkReviewProvenance compares review.md against the sidecar the LAUNCHER
// wrote, which is the only part of the evidence the reviewed party did not
// author.
//
// Absent sidecar → no finding. Reviews predating this guard, and hand-written
// ones, remain governed by the verdict, staleness and pool checks; refusing them
// would invalidate every review already on disk to close a hole none of them
// demonstrably has.
func checkReviewProvenance(specDir string, review Review) error {
	req, found, err := ReadReviewRequest(specDir)
	if err != nil {
		return fmt.Errorf("%w\nrepair or delete it and re-run /adversarial-review, or pass --force-without-review", err)
	}
	if !found {
		return nil
	}

	// The digest is the no-verdict case, and it is checked first because it
	// explains the sha mismatch that would otherwise be reported instead: a
	// reviewer that wrote nothing leaves the PREVIOUS round's sha in place, and
	// "the shas differ" would send the reader hunting for a rebase.
	if req.ReviewDigestBefore != "" && req.ReviewDigestBefore == fileDigest(filepath.Join(specDir, ReviewFile)) {
		return fmt.Errorf("%s has not changed since the review was launched — the reviewer wrote no verdict\n"+
			"what is on disk is the PREVIOUS round's, which is not a review of this change\n"+
			"re-run /adversarial-review (a run ended by a turn limit or a rate limit leaves exactly this state), or pass --force-without-review",
			ReviewFile)
	}

	if req.ReviewedSHA != "" && review.ReviewedSHA != "" && req.ReviewedSHA != review.ReviewedSHA {
		return fmt.Errorf("%s claims reviewed_sha %s but the review was launched against %s\n"+
			"the launcher records the head it pointed the reviewer at; the frontmatter is the reviewer's own claim about it\n"+
			"re-run /adversarial-review against the current head, or pass --force-without-review",
			ReviewFile, short(review.ReviewedSHA), short(req.ReviewedSHA))
	}

	// Reviewer identity, last of the three: checkReviewerPool already refuses a
	// signature from outside the pool, so what this adds is narrower — a verdict
	// signed by a DIFFERENT pool member than the one that was launched, which
	// that check cannot see because both are admitted.
	if req.Reviewer != "" && review.Reviewer != "" && req.Reviewer != review.Reviewer {
		return fmt.Errorf("%s is signed by %q but %q was launched — the verdict is not from the run that was requested\n"+
			"re-run /adversarial-review, or pass --force-without-review",
			ReviewFile, review.Reviewer, req.Reviewer)
	}
	return nil
}

// short renders a sha for a human-readable error without hiding a short input.
func short(sha string) string {
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}
