package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The review's scope is `git diff <base>...HEAD`, and until this base existed
// the launcher recorded none: the reviewer guessed `main`, and a review
// launched ON main after the work merged diffed nothing and degraded into
// browsing the spec folder. These pin that the base is resolved the same way in
// both situations that matter — on the work branch, and on main after a squash.

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newRepoWithSpec builds a repo whose history is: base commit, then a commit
// that adds specs/<id>/, then one more. The base the resolver must find is the
// FIRST commit -- the parent of the one that added the folder.
func newRepoWithSpec(t *testing.T, id string) (repoRoot, wantBase string) {
	t.Helper()
	repoRoot = t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")

	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "add", "README.md")
	git(t, repoRoot, "commit", "-qm", "base")
	wantBase = trim(git(t, repoRoot, "rev-parse", "HEAD"))

	specDir := filepath.Join(repoRoot, "specs", id)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "proposal.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "add", ".")
	git(t, repoRoot, "commit", "-qm", "add spec")

	if err := os.WriteFile(filepath.Join(repoRoot, "impl.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "add", ".")
	git(t, repoRoot, "commit", "-qm", "implement")

	return repoRoot, wantBase
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestResolveReviewBaseIsTheParentOfTheCommitThatAddedTheSpec(t *testing.T) {
	repoRoot, wantBase := newRepoWithSpec(t, "FOO-001-thing")

	got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "FOO-001-thing"))
	if got != wantBase {
		t.Errorf("ResolveReviewBase = %q, want %q (the commit before the spec folder appeared)", got, wantBase)
	}
}

// The case the old code got wrong by having no answer at all: HEAD is main,
// the work already merged, and `git diff main...HEAD` is therefore empty. The
// base must still point before the work.
func TestResolveReviewBaseGivesANonEmptyDiffOnMainAfterTheWorkLanded(t *testing.T) {
	repoRoot, _ := newRepoWithSpec(t, "FOO-001-thing")
	specDir := filepath.Join(repoRoot, "specs", "FOO-001-thing")

	base := ResolveReviewBase(repoRoot, specDir)
	if base == "" {
		t.Fatal("no base resolved")
	}

	// The anchor: the naive base really is empty here, so this test is about
	// something. Without it the assertion below could pass on a repo where the
	// old behaviour also happened to work.
	naive := git(t, repoRoot, "diff", "--name-only", "main...HEAD")
	if trim(naive) != "" {
		t.Fatalf("fixture does not reproduce the defect: `git diff main...HEAD` returned %q", naive)
	}

	scoped := git(t, repoRoot, "diff", "--name-only", base+"...HEAD")
	if trim(scoped) == "" {
		t.Error("the resolved base produced an EMPTY diff, which is the whole defect: the reviewer " +
			"then browses the spec folder and reports findings with nothing executed behind them")
	}
}

func TestResolveReviewBaseIsEmptyWhenTheSpecIsNotCommitted(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "add", ".")
	git(t, repoRoot, "commit", "-qm", "base")

	specDir := filepath.Join(repoRoot, "specs", "UNCOMMITTED-001")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := ResolveReviewBase(repoRoot, specDir); got != "" {
		t.Errorf("ResolveReviewBase = %q for an uncommitted spec, want \"\" — the caller must "+
			"refuse to launch rather than review the whole history", got)
	}
}

// The sidecar has to survive reading a spec archived before base_sha existed.
func TestReviewRequestWithoutBaseSHAStillParses(t *testing.T) {
	dir := t.TempDir()
	old := `{"reviewed_sha":"abc","reviewer":"nan/x","requested_at":"2026-01-01T00:00:00Z","review_digest_before":"d"}`
	if err := os.WriteFile(filepath.Join(dir, ReviewRequestFile), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	req, found, err := ReadReviewRequest(dir)
	if err != nil || !found {
		t.Fatalf("reading an old sidecar: found=%v err=%v", found, err)
	}
	if req.ReviewedSHA != "abc" {
		t.Errorf("ReviewedSHA = %q, want abc", req.ReviewedSHA)
	}
	if req.BaseSHA != "" {
		t.Errorf("BaseSHA = %q, want empty for a pre-base sidecar", req.BaseSHA)
	}
}

func TestWriteReviewRequestRecordsTheBase(t *testing.T) {
	dir := t.TempDir()
	if err := WriteReviewRequest(dir, "head123", "nan/x", "base456"); err != nil {
		t.Fatal(err)
	}
	req, found, err := ReadReviewRequest(dir)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if req.BaseSHA != "base456" {
		t.Errorf("BaseSHA = %q, want base456 — without it the reviewer has no scope", req.BaseSHA)
	}
}

// The prompt must state the base, because the skill tells the reviewer to diff
// against one and the reviewer cannot derive it.
func TestReviewPromptStatesTheResolvedBase(t *testing.T) {
	p := ReviewPrompt("FOO-001", "/repo", "nan/x", "pi", "/skill.md", "base456")
	if !contains(p, "base456") {
		t.Error("the prompt does not name the base, so the reviewer will guess `main` — which is " +
			"empty on a post-merge review")
	}
	if !contains(p, "do NOT substitute") {
		t.Error("the prompt does not tell the reviewer to stop guessing a base")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
