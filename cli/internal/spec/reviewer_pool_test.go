package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// reviewBy is passingReview with a chosen reviewer id — the field the pool gate
// is about, and the one every other test leaves at "test".
func reviewBy(id, reviewer string) string {
	return "---\nspec: \"" + id + "\"\nverdict: \"PASS\"\n" +
		"reviewed_sha: \"0000000000000000000000000000000000000000\"\n" +
		"reviewer: \"" + reviewer + "\"\ndate: \"2026-08-13\"\n---\nno blocking findings\n"
}

// writePool lays down harness/reviewer-pool.json with the given raw content.
func writePool(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reviewer-pool.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const testPool = `{"pool":[{"id":"nan/deepseek-v4-flash"},{"id":"agy/gemini-3.1-pro-high"}]}`

// The reason this gate exists: BUG-074 was reviewed twice by claude-opus-5, the
// same model family that implemented it, which is the single-agent self-review
// the skill forbids. Nothing stopped it, because reviewer identity was recorded
// and then ignored.
func TestArchiveBlocksReviewerOutsideThePool(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))

	_, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}})
	if err == nil {
		t.Fatal("a reviewer outside the pool must block the archive")
	}
	// The refusal has to be diagnosable in one read: who signed it, and who was
	// allowed to. Otherwise the operator has to open the source to find out.
	for _, want := range []string{"claude-opus-5", "nan/deepseek-v4-flash", "agy/gemini-3.1-pro-high"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name %q, got: %v", want, err)
		}
	}
	// Same contract as every other refusal in this gate: name the declared
	// escapes rather than leaving the human to guess or to force blindly.
	for _, want := range []string{"review: waived", "--force-without-review"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name the escape %q, got: %v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x")); !os.IsNotExist(statErr) {
		t.Error("target must not be created when blocked")
	}
}

func TestArchiveAllowsReviewerInThePool(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "nan/deepseek-v4-flash"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("a pooled reviewer must archive cleanly: %v", err)
	}
}

// A non-primary pool entry is just as valid: the array is an allow-list, and
// only its FIRST element carries the extra meaning of being the launcher's
// default. A fallback-signed review must not be second-class.
func TestArchiveAllowsAnyPoolEntryNotOnlyTheFirst(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "agy/gemini-3.1-pro-high"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("the fallback reviewer must archive cleanly: %v", err)
	}
}

// dotf runs in repos that have no pool. Requiring one everywhere would break
// them, so an absent pool skips the check when the repo NEVER had one — which is
// also why the pre-existing tests (reviewer "test") still pass, since a t.TempDir
// has no git history to contradict them.
//
// Deleting a pool the repo does have is a different case entirely and fails
// closed: see TestArchiveFailsClosedWhenATrackedPoolGoesMissing.
func TestArchiveSkipsThePoolCheckWhenNoPoolExists(t *testing.T) {
	root := t.TempDir()
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("no pool means no opinion, so the archive must proceed: %v", err)
	}
}

// A gate fails toward refusing. An unreadable pool is the state where the check
// cannot tell an allowed reviewer from a forbidden one, so passing would be a
// silent downgrade to no gate at all.
func TestArchiveRefusesAMalformedPool(t *testing.T) {
	cases := map[string]string{
		"not json":     `{"pool": [`,
		"empty pool":   `{"pool": []}`,
		"missing pool": `{}`,
		"blank id":     `{"pool":[{"id":"   "}]}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writePool(t, root, content)
			archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "nan/deepseek-v4-flash"))

			_, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}})
			if err == nil {
				t.Fatal("an unusable pool must refuse, not silently allow")
			}
			if !strings.Contains(err.Error(), "reviewer-pool.json") {
				t.Errorf("the error must name the file at fault, got: %v", err)
			}
		})
	}
}

// Reviewer ids are self-reported free text, so surrounding whitespace is a
// realistic way to fail a match that should have succeeded. Matching is exact
// on the trimmed value: forgiving about spacing, strict about identity.
func TestArchiveMatchesReviewerExactlyAfterTrimming(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "  nan/deepseek-v4-flash  "))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("padding around a pooled id must not refuse: %v", err)
	}
}

// A prefix is not a member. `agy` also serves claude-opus-4-6-thinking, so a
// substring match would let "nan/deepseek-v4-flash-lite" — or anything merely
// starting with a pooled id — sign a review.
func TestArchiveRejectsAReviewerThatMerelyResemblesAPoolEntry(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "nan/deepseek-v4-flash-lite"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err == nil {
		t.Fatal("a near-miss id must refuse; membership is exact, not prefix")
	}
}

// The escape must actually work, or the refusal is a wall rather than a gate.
func TestArchiveForceOverridesThePoolCheck(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{
		Staleness:          fakeStaleness{},
		ForceWithoutReview: true,
	}); err != nil {
		t.Fatalf("--force-without-review must bypass the pool check too: %v", err)
	}
}

// gitInit makes root a real repository with one commit, so history questions
// have real answers instead of stubbed ones.
func gitInit(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v\n%s", err, out)
		}
	}
}

func gitCommitAll(t *testing.T, root, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// Absence is ambiguous, and the two meanings need opposite answers.
//
// A repo that NEVER had a pool must keep working — dotf runs in repos that never
// opted into this gate. But a repo whose pool vanished (bad merge, stray delete,
// .gitignore mistake) must not silently stop checking who reviews: that is a
// safety check disappearing with no signal at the moment it stops applying.
func TestArchiveFailsClosedWhenATrackedPoolGoesMissing(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))
	gitCommitAll(t, root, "add the pool")

	// The pool is now part of this repo's history. Delete it from the tree.
	if err := os.Remove(filepath.Join(root, "harness", "reviewer-pool.json")); err != nil {
		t.Fatal(err)
	}

	_, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}})
	if err == nil {
		t.Fatal("deleting a tracked pool must not silently disable the gate")
	}
	if !strings.Contains(err.Error(), "HEAD") || !strings.Contains(err.Error(), "working tree") {
		t.Errorf("the error must explain that the gate is in HEAD and the file is gone, got: %v", err)
	}
}

// The third case, and the one an independent review of this spec found missing:
// a pool removed ON PURPOSE, in a commit.
//
// The original poolWasTracked asked `git log -- <path>`, which also returns the
// commit that DELETED the path. So once the file had ever existed the answer was
// true forever, and a deliberate removal left every future archive blocked —
// while the error message told the reader to retire the gate by committing
// exactly that removal. The instruction and the code disagreed, and the code won.
//
// Asking HEAD's tree instead splits the two absences the way the rule intends:
// gone from the tree but present in HEAD is a loss (fail closed, above), absent
// from HEAD is a decision someone recorded. Silent disabling is what this gate
// exists to prevent, and a committed `git rm` is the opposite of silent.
func TestArchiveAllowsAPoolRetiredInACommit(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writePool(t, root, testPool)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))
	gitCommitAll(t, root, "add the pool")

	// Retire it the way the error message prescribes: remove AND commit.
	if err := os.Remove(filepath.Join(root, "harness", "reviewer-pool.json")); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root, "retire the reviewer pool gate")

	// claude-opus-5 was refused by the pool while it existed; with the pool
	// formally retired there is no pool to have an opinion, so this archives.
	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("a pool retired in a commit must not block archives forever: %v", err)
	}
}

// The other half of the same rule: a repo that never had a pool archives freely,
// including inside a git work tree. Without this, adopting dotf would require
// every repo to carry a pool it never asked for.
func TestArchiveStillSkipsInARepoThatNeverHadAPool(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	archivableSpec(t, root, "AI-001-x", reviewBy("AI-001-x", "claude-opus-5"))
	gitCommitAll(t, root, "no pool here, and never was")

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}}); err != nil {
		t.Fatalf("a repo that never enabled the gate must archive: %v", err)
	}
}
