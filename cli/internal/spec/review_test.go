package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// passingReview is the review.md an archivable spec carries. Tests that
// exercise archive behaviors OTHER than the gate include it so they describe a
// spec that is legitimately archivable under the CLI-034 contract.
func passingReview(id string) string {
	return "---\nspec: \"" + id + "\"\nverdict: \"PASS\"\nreviewed_sha: \"0000000000000000000000000000000000000000\"\nreviewer: \"test\"\ndate: \"2026-08-09\"\n---\nno blocking findings\n"
}

// fakeStaleness drives the freshness decision without a git history.
type fakeStaleness struct {
	stale  bool
	known  bool
	reason string
}

func (f fakeStaleness) Stale(string, string, string) (bool, bool, string) {
	return f.stale, f.known, f.reason
}

// TestArchiveBlocksOnMissingReview is the proved-red acceptance for CLI-034:
// it must fail on main, where Archive has no review pre-flight at all.
func TestArchiveBlocksOnMissingReview(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: verifying\n---\nclean\n",
	})

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil {
		t.Fatalf("expected a missing review.md to block the archive")
	}
	// The error must name the artifact and BOTH declared escapes, so the human
	// never has to read the source to learn how to proceed.
	for _, want := range []string{"review.md", "review: waived", "--force-without-review"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x")); err != nil {
		t.Errorf("source must remain when blocked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x")); !os.IsNotExist(err) {
		t.Errorf("target must not be created when blocked")
	}
}

// archivableSpec writes a spec whose only interesting property is the review.
func archivableSpec(t *testing.T, root, id, review string) {
	t.Helper()
	files := map[string]string{"proposal.md": "---\nstatus: verifying\n---\nclean\n"}
	if review != "" {
		files["review.md"] = review
	}
	writeSpec(t, root, id, files)
}

func TestParseReviewVerdicts(t *testing.T) {
	cases := []struct {
		name    string
		verdict string
		wantErr bool
		blocks  bool
	}{
		{"pass", "PASS", false, false},
		{"pass with gaps", "PASS-WITH-GAPS", false, false},
		{"fail", "FAIL", false, true},
		{"lowercase is normalized", "pass", false, false},
		{"unrecognized", "LGTM", true, true},
		{"empty", "", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := "---\nspec: \"AI-001-x\"\nverdict: \"" + tc.verdict + "\"\nreviewed_sha: \"abc123\"\n---\n"
			r, err := ParseReview(content)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got err=%v", tc.wantErr, err)
			}
			if err == nil && r.Verdict.Blocks() != tc.blocks {
				t.Errorf("Blocks()=%v, want %v", r.Verdict.Blocks(), tc.blocks)
			}
		})
	}
}

// TestParseReviewRequiresSHA: without reviewed_sha the review cannot be checked
// for freshness, so the artifact is malformed rather than merely incomplete.
func TestParseReviewRequiresSHA(t *testing.T) {
	_, err := ParseReview("---\nspec: \"AI-001-x\"\nverdict: \"PASS\"\n---\n")
	if err == nil || !strings.Contains(err.Error(), "reviewed_sha") {
		t.Fatalf("want a reviewed_sha error, got: %v", err)
	}
}

// TestFrontmatterKeepsHashInQuotedValue guards the comment-stripping rule: a
// quoted value may legitimately contain '#' (an issue reference), and stripping
// it would corrupt the field.
func TestFrontmatterKeepsHashInQuotedValue(t *testing.T) {
	f := frontmatterFields("---\nissue: \"mlorentedev/dotfiles#875\"   # repo#NNN — tracker\nstatus: draft # a comment\n---\n")
	if got := f["issue"]; got != "mlorentedev/dotfiles#875" {
		t.Errorf("quoted value with '#': got %q", got)
	}
	if got := f["status"]; got != "draft" {
		t.Errorf("unquoted value should drop its trailing comment: got %q", got)
	}
}

func TestArchiveBlocksOnFailVerdict(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", "---\nspec: \"AI-001-x\"\nverdict: \"FAIL\"\nreviewed_sha: \"abc\"\n---\n")

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil || !strings.Contains(err.Error(), "FAIL") {
		t.Fatalf("expected a FAIL verdict to block, got: %v", err)
	}
}

func TestArchiveBlocksOnUnknownVerdict(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", "---\nspec: \"AI-001-x\"\nverdict: \"LGTM\"\nreviewed_sha: \"abc\"\n---\n")

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil || !strings.Contains(err.Error(), "unrecognized verdict") {
		t.Fatalf("expected an unrecognized verdict to block, got: %v", err)
	}
}

// TestArchiveBlocksOnForeignReview: a review.md copied from another spec must
// not satisfy the gate.
func TestArchiveBlocksOnForeignReview(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", passingReview("AI-002-y"))

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil || !strings.Contains(err.Error(), "different change") {
		t.Fatalf("expected a foreign review to block, got: %v", err)
	}
}

func TestArchiveProceedsOnPass(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", passingReview("AI-001-x"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err != nil {
		t.Fatalf("a PASS review should archive: %v", err)
	}
}

// TestArchiveProceedsOnPassWithGaps: gaps are already tracked by the review
// itself, so they must not also block the archive.
func TestArchiveProceedsOnPassWithGaps(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x",
		"---\nspec: \"AI-001-x\"\nverdict: \"PASS-WITH-GAPS\"\nreviewed_sha: \"abc\"\n---\n")

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err != nil {
		t.Fatalf("PASS-WITH-GAPS should archive: %v", err)
	}
}

func TestArchiveWaivedWithReason(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: verifying\nreview: waived\nreview_waived_reason: \"doc-only change, no behavior\"\n---\n",
	})

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err != nil {
		t.Fatalf("a declared waiver with a reason should archive: %v", err)
	}
}

// TestArchiveWaivedWithoutReasonRefuses is the load-bearing half of the waiver:
// requiring the reason is what keeps waiving visible in the diff instead of
// invisible in a habit.
func TestArchiveWaivedWithoutReasonRefuses(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: verifying\nreview: waived\n---\n",
	})

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil || !strings.Contains(err.Error(), "without a reason") {
		t.Fatalf("expected a reasonless waiver to block, got: %v", err)
	}
}

func TestArchiveForceWithoutReview(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", "")

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithoutReview: true}); err != nil {
		t.Fatalf("force-without-review should archive despite no review: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x", "proposal.md")); err != nil {
		t.Errorf("expected archived proposal: %v", err)
	}
}

// TestArchiveForceWithoutReviewOverridesFail: the escape covers a FAIL verdict
// too, not only an absent artifact.
func TestArchiveForceWithoutReviewOverridesFail(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", "---\nspec: \"AI-001-x\"\nverdict: \"FAIL\"\nreviewed_sha: \"abc\"\n---\n")

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithoutReview: true}); err != nil {
		t.Fatalf("force-without-review should override a FAIL verdict: %v", err)
	}
}

func TestArchiveBlocksOnStaleReview(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", passingReview("AI-001-x"))

	_, err := Archive(root, "AI-001-x", ArchiveOptions{
		Staleness: fakeStaleness{stale: true, known: true, reason: "proposal.md changed after reviewed_sha"},
	})
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected a stale review to block, got: %v", err)
	}
}

// TestStaleRefusalOffersTheExitThatKeepsTheReview: the three exits the refusal
// used to name — re-review, waive, force — all discard or bypass a verdict that
// may be perfectly good. The common way to reach this refusal is applying a
// PASSing review's own recommended next steps, and there the review is fine and
// the contract edit is the thing to undo. That exit must be on offer, and first:
// an operator reads the list in order, and the overrides are the expensive ones.
func TestStaleRefusalOffersTheExitThatKeepsTheReview(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", passingReview("AI-001-x"))

	_, err := Archive(root, "AI-001-x", ArchiveOptions{
		Staleness: fakeStaleness{stale: true, known: true, reason: "proposal.md changed after reviewed_sha"},
	})
	if err == nil {
		t.Fatal("expected a stale review to block")
	}
	msg := err.Error()

	restore := strings.Index(msg, "restore the contract files")
	if restore < 0 {
		t.Fatalf("refusal must name the exit that keeps the review, got: %s", msg)
	}
	if !strings.Contains(msg, "verification.md") {
		t.Fatalf("refusal must say where the dispositions go, got: %s", msg)
	}
	for _, override := range []string{"re-run /adversarial-review", "review: waived", "--force-without-review"} {
		at := strings.Index(msg, override)
		if at < 0 {
			t.Fatalf("refusal dropped the %q exit, got: %s", override, msg)
		}
		if at < restore {
			t.Fatalf("%q is offered before the exit that keeps the review, got: %s", override, msg)
		}
	}
}

// TestArchiveProceedsWhenStalenessUnknown: outside a git work tree there is no
// history for the review to be stale against, so the check is skipped rather
// than guessed. Presence and verdict still applied.
func TestArchiveProceedsWhenStalenessUnknown(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", passingReview("AI-001-x"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{
		Staleness: fakeStaleness{stale: true, known: false},
	}); err != nil {
		t.Fatalf("an unanswerable staleness question must not block: %v", err)
	}
}

// --- gitStaleness against a real history -------------------------------------
//
// The fake above drives Archive's decision; these exercise the real checker,
// which is where AC6/AC7 actually live.

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitSpecRepo builds a repo with specs/<id>/ committed, returning the root and
// the sha of that first commit.
func gitSpecRepo(t *testing.T, id string) (root, sha string) {
	t.Helper()
	root = t.TempDir()
	gitRun(t, root, "init", "-q", "-b", "main")
	writeSpec(t, root, id, map[string]string{
		"proposal.md":     "---\nstatus: verifying\n---\nclean\n",
		"tasks.md":        "tasks\n",
		"verification.md": "evidence\n",
	})
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "spec")
	return root, gitRun(t, root, "rev-parse", "HEAD")
}

// An independent review of HARNESS-071 found this hole: the checker read
// committed history only, so editing a contract file and NOT committing it left
// the review looking fresh. That is the cheapest possible bypass of the whole
// gate — get a passing review, rewrite the acceptance criteria in the working
// tree, archive. Uncommitted is exactly the state a spec is in while someone is
// editing it, so this was not an exotic path.
//
// Scoped to the three contract files on purpose: a review in flight writes
// review.md and review-transcript.jsonl into the same folder, and a check over
// the whole spec dir would call every review stale the moment it produced its
// own artifacts.
func TestGitStalenessDetectsUncommittedContractChange(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "proposal.md"),
		[]byte("---\nstatus: verifying\n---\nedited but never committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, known, reason := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known || !stale {
		t.Fatalf("an uncommitted contract edit must be stale (known=%v stale=%v)", known, stale)
	}
	if !strings.Contains(reason, "uncommitted") {
		t.Errorf("reason should say the change is uncommitted, got %q", reason)
	}
	if !strings.Contains(reason, "proposal.md") {
		t.Errorf("reason should name the file that actually changed, got %q", reason)
	}
}

// The other side of the scoping: a review writing its own artifacts into the
// spec folder must not make itself stale. Without this, fixing the hole above
// would break every archive instead.
func TestGitStalenessIgnoresUncommittedNonContractFiles(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	for _, name := range []string{"review.md", "review-transcript.jsonl", "verification.md"} {
		if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", name),
			[]byte("written by the reviewer mid-run\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	stale, known, _ := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known {
		t.Fatal("a real repo must give a known answer")
	}
	if stale {
		t.Error("a review's own uncommitted artifacts must not make it stale")
	}
}

func TestGitStalenessDetectsContractChange(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "proposal.md"),
		[]byte("---\nstatus: verifying\n---\nchanged after the review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "commit", "-qam", "amend the proposal")

	stale, known, reason := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known || !stale {
		t.Fatalf("a contract change after reviewed_sha must be stale (known=%v stale=%v)", known, stale)
	}
	if !strings.Contains(reason, "proposal.md") {
		t.Errorf("reason should name the file that actually changed, got %q", reason)
	}
	for _, unmoved := range []string{"tasks.md", "features.json"} {
		if strings.Contains(reason, unmoved) {
			t.Errorf("reason names %s, which did not change: %q", unmoved, reason)
		}
	}
}

// #998. The refusal used to recite every contract file whether or not it had
// changed, so an operator could not tell a rewritten acceptance criterion from a
// ticked checkbox without reconstructing the diff by hand. Naming only what moved
// is what makes the two distinguishable at the point of refusal.
func TestGitStalenessNamesOnlyTheFileThatMoved(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "tasks.md"),
		[]byte("- [x] ticked after the review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "commit", "-qam", "tick a box")

	stale, known, reason := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known || !stale {
		t.Fatalf("a contract change after reviewed_sha must be stale (known=%v stale=%v)", known, stale)
	}
	if !strings.Contains(reason, "tasks.md") {
		t.Errorf("reason should name tasks.md, got %q", reason)
	}
	for _, unmoved := range []string{"proposal.md", "features.json"} {
		if strings.Contains(reason, unmoved) {
			t.Errorf("reason names %s, which did not change: %q", unmoved, reason)
		}
	}
}

// TestGitStalenessIgnoresReviewOwnCommit is the self-defeat guard: review.md is
// always committed AFTER the sha it records, so counting it would make every
// review stale by construction.
func TestGitStalenessIgnoresReviewOwnCommit(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "review.md"),
		[]byte(passingReview("AI-001-x")), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "add the review")

	stale, known, _ := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known {
		t.Fatalf("a git work tree must be answerable")
	}
	if stale {
		t.Errorf("review.md's own commit must not make it stale")
	}
}

// TestGitStalenessIgnoresVerificationMd: verification.md's archive checklist is
// ticked AT archive time, so counting it would false-positive on every archive.
func TestGitStalenessIgnoresVerificationMd(t *testing.T) {
	root, sha := gitSpecRepo(t, "AI-001-x")
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "verification.md"),
		[]byte("evidence\n- [x] folder moved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "commit", "-qam", "tick the archive checklist")

	stale, known, _ := gitStaleness{}.Stale(root, "AI-001-x", sha)
	if !known {
		t.Fatalf("a git work tree must be answerable")
	}
	if stale {
		t.Errorf("a verification.md change must not make the review stale")
	}
}

// TestGitStalenessRebaseWithoutContractChangeIsNotStale verifies that rebasing
// a branch when contract files are byte-identical does not falsely mark the
// review as stale (#1036).
func TestGitStalenessRebaseWithoutContractChangeIsNotStale(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "init.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "init.txt")
	gitRun(t, root, "commit", "-qm", "init")

	// Branch out to feature branch and create/commit the spec
	gitRun(t, root, "checkout", "-b", "feat/my-spec")
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md":     "---\nstatus: verifying\n---\nclean\n",
		"tasks.md":        "tasks\n",
		"verification.md": "evidence\n",
		"features.json":   "[]\n",
	})
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "feat: create spec")
	reviewedSHA := gitRun(t, root, "rev-parse", "HEAD")

	// Advance main with an unrelated change
	gitRun(t, root, "checkout", "main")
	if err := os.WriteFile(filepath.Join(root, "other.txt"), []byte("main work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "other.txt")
	gitRun(t, root, "commit", "-qm", "chore: main commit")

	// Rebase feature branch onto main (rewrites the spec commit SHA)
	gitRun(t, root, "checkout", "feat/my-spec")
	gitRun(t, root, "rebase", "main")

	stale, known, reason := gitStaleness{}.Stale(root, "AI-001-x", reviewedSHA)
	if !known {
		t.Fatalf("git repository must be answerable")
	}
	if stale {
		t.Errorf("review should not be stale after rebase with identical contract files, got reason: %q", reason)
	}
}

func TestGitStalenessUnresolvableShaIsStale(t *testing.T) {
	root, _ := gitSpecRepo(t, "AI-001-x")

	stale, known, reason := gitStaleness{}.Stale(root, "AI-001-x", "0123456789012345678901234567890123456789")
	if !known || !stale {
		t.Fatalf("an unresolvable sha must be treated as stale (known=%v stale=%v)", known, stale)
	}
	if !strings.Contains(reason, "not a commit") {
		t.Errorf("reason should say the sha is unknown, got %q", reason)
	}
}

// TestGitStalenessOutsideRepoIsUnknown: with no history there is nothing for
// the review to be stale against, so the question is unanswerable rather than
// failing — the presence and verdict checks still applied.
func TestGitStalenessOutsideRepoIsUnknown(t *testing.T) {
	root := t.TempDir()
	if _, known, _ := (gitStaleness{}).Stale(root, "AI-001-x", "abc"); known {
		t.Errorf("outside a work tree the staleness question must be unanswerable")
	}
}

// The verdict a reviewer WRITES and the enum the gate MATCHES differ only in
// case and word separators. HARNESS-072's review recorded "PASS WITH GAPS" with
// spaces; the gate wanted "PASS-WITH-GAPS", so Blocks() refused the archive over
// two characters, discarding a six-minute review and a passing verdict.
//
// The repo already carries this lesson about the `reviewer:` field. This pins it
// one field across — and pins the limit too: normalization must not start
// guessing at intent.
func TestNormalizeVerdictAcceptsSpellingsNotGuesses(t *testing.T) {
	accepted := map[string]Verdict{
		"PASS-WITH-GAPS": VerdictPassWithGaps,
		"PASS WITH GAPS": VerdictPassWithGaps, // the observed case
		"pass with gaps": VerdictPassWithGaps,
		"pass_with_gaps": VerdictPassWithGaps,
		"  PASS  ":       VerdictPass,
		"pass":           VerdictPass,
		"FAIL":           VerdictFail,
		"fail":           VerdictFail,
	}
	for raw, want := range accepted {
		if got := normalizeVerdict(raw); got != want {
			t.Errorf("normalizeVerdict(%q) = %q, want %q", raw, got, want)
		}
	}

	// Still rejected: these are not the same words differently punctuated, and a
	// gate that guesses is worse than one that refuses.
	for _, raw := range []string{"PASSWITHGAPS", "PASSED", "OK", "APPROVED", "PASS WITH GAP"} {
		v := normalizeVerdict(raw)
		if !v.Blocks() {
			t.Errorf("normalizeVerdict(%q) = %q, which does not block — normalization is guessing", raw, v)
		}
	}
}

func TestParseReviewAcceptsASpacedVerdict(t *testing.T) {
	// End to end through the real parser, on the exact frontmatter the reviewer
	// produced for HARNESS-072.
	content := "---\n" +
		"spec: \"HARNESS-072-pr-stewardship\"\n" +
		"verdict: \"PASS WITH GAPS\"\n" +
		"reviewed_sha: \"a9b2063d7440c152a1997c20f5d78ee4b5261998\"\n" +
		"reviewer: \"nan/deepseek-v4-flash\"\n" +
		"---\n"
	r, err := ParseReview(content)
	if err != nil {
		t.Fatalf("ParseReview: %v", err)
	}
	if r.Verdict != VerdictPassWithGaps {
		t.Errorf("verdict = %q, want %q", r.Verdict, VerdictPassWithGaps)
	}
	if r.Verdict.Blocks() {
		t.Error("a passing verdict must not block the archive over punctuation")
	}
}
