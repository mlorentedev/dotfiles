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
	if !strings.Contains(reason, "contract file") {
		t.Errorf("reason should name the contract files, got %q", reason)
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
