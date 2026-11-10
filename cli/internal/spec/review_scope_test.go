package spec

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// HARNESS-158 (#1727): a spec whose slices merge to main one at a time is
// reviewed over `<base>...HEAD`, which also holds everything other work merged
// in between. These pin which commits in that range are the spec's own.

func subjects(set OwnCommitSet) []string {
	out := make([]string, 0, len(set.Commits))
	for _, c := range set.Commits {
		out = append(out, c.Subject)
	}
	return out
}

func commitMsg(t *testing.T, repoRoot, msg string, paths ...string) {
	t.Helper()
	writeAndCommit(t, repoRoot, msg, paths...)
}

// sliceHistory is two slices merged at different times with unrelated work
// between and after them, an issue-only slice, a decoy whose number only starts
// with the spec's, and a merge commit that names the issue.
func sliceHistory(t *testing.T) (repoRoot, base string) {
	t.Helper()
	repoRoot = initQuietRepo(t)
	commitMsg(t, repoRoot, "base", "README.md")
	base = trim(git(t, repoRoot, "rev-parse", "HEAD"))
	commitMsg(t, repoRoot, "slice one (#10)", "specs/X-001-a/proposal.md", "a.go")
	commitMsg(t, repoRoot, "unrelated work (#11)", "other.go")
	commitMsg(t, repoRoot, "implement part two (#12)\n\nPart of #1727", "b.go")
	commitMsg(t, repoRoot, "decoy (#13)\n\nCloses #17270", "c.go")
	git(t, repoRoot, "checkout", "-q", "-b", "side")
	commitMsg(t, repoRoot, "side work", "side.go")
	git(t, repoRoot, "checkout", "-q", "main")
	git(t, repoRoot, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", "merge side, see #1727", "side")
	commitMsg(t, repoRoot, "slice three (#14)", "specs/X-001-a/tasks.md", "d.go")
	commitMsg(t, repoRoot, "trailing unrelated (#15)", "e.go")
	return repoRoot, base
}

func TestOwnCommitsHoldsOnlyTheSlicesOfTheSpec(t *testing.T) {
	repoRoot, base := sliceHistory(t)
	got, err := OwnCommits(repoRoot, base, "HEAD", []string{"specs/X-001-a"}, 1727)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"slice one (#10)", "implement part two (#12)", "slice three (#14)"}
	if !reflect.DeepEqual(subjects(got), want) {
		t.Errorf("own commits = %q, want %q (oldest first)", subjects(got), want)
	}
	// Seven non-merge commits after base, side work included: the merge itself is not counted.
	if got.RangeSize != 7 {
		t.Errorf("RangeSize = %d, want 7 non-merge commits in base..HEAD", got.RangeSize)
	}
	for _, c := range got.Commits {
		if len(c.SHA) != 40 && len(c.SHA) != 64 {
			t.Errorf("commit %q has SHA %q, want a full object name", c.Subject, c.SHA)
		}
	}
}

func TestOwnCommitsWithoutAnIssueUsesTheFolderAlone(t *testing.T) {
	repoRoot, base := sliceHistory(t)
	got, err := OwnCommits(repoRoot, base, "HEAD", []string{"specs/X-001-a"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"slice one (#10)", "slice three (#14)"}
	if !reflect.DeepEqual(subjects(got), want) {
		t.Errorf("own commits = %q, want %q", subjects(got), want)
	}
}

// A slice committed before the folder was renamed touched the old name.
func TestOwnCommitsFollowsARenamedSpecFolder(t *testing.T) {
	repoRoot := initQuietRepo(t)
	commitMsg(t, repoRoot, "base", "README.md")
	commitMsg(t, repoRoot, "unrelated before", "u.go")
	commitMsg(t, repoRoot, "slice under the old name", "specs/OLD-001-x/proposal.md", "a.go")
	commitMsg(t, repoRoot, "unrelated between", "v.go")
	git(t, repoRoot, "mv", "specs/OLD-001-x", "specs/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "rename the spec")
	commitMsg(t, repoRoot, "slice under the new name", "specs/NEW-001-x/tasks.md", "b.go")

	specDir := filepath.Join(repoRoot, "specs", "NEW-001-x")
	base := ResolveReviewBase(repoRoot, specDir)
	if base == "" {
		t.Fatal("no base resolved for the renamed fixture")
	}
	names := SpecFolderNames(repoRoot, specDir)
	if want := []string{"specs/NEW-001-x", "specs/OLD-001-x"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("SpecFolderNames = %q, want %q", names, want)
	}
	got, err := OwnCommits(repoRoot, base, "HEAD", names, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"slice under the old name", "rename the spec", "slice under the new name"}
	if !reflect.DeepEqual(subjects(got), want) {
		t.Errorf("own commits = %q, want %q", subjects(got), want)
	}
}

func TestOwnCommitsReportsAGitFailure(t *testing.T) {
	repoRoot := initQuietRepo(t)
	commitMsg(t, repoRoot, "base", "README.md")
	if _, err := OwnCommits(repoRoot, "0000000000000000000000000000000000000bad", "HEAD", []string{"specs/X"}, 0); err == nil {
		t.Error("OwnCommits over an unknown base returned no error; a broken range must not read as a spec with no commits")
	}
}

func TestReviewPromptNamesTheOwnCommits(t *testing.T) {
	scope := ReviewScope{
		BaseSHA:  "basebasebase",
		OwnIssue: 1727,
		Own: OwnCommitSet{RangeSize: 14, Commits: []OwnCommit{
			{SHA: "7f4c76fc0000000000000000000000000000aaaa", Subject: "spec: run the spec reviewer in an isolated worktree (#3178)"},
			{SHA: "bbeb4d980000000000000000000000000000bbbb", Subject: "feat(worktree): add a detached, lease-free review worktree (#3195)"},
		}},
	}
	p := ReviewPromptForScope(FirstSigner, "X-001-a", "/repo", "nan/x", "pi", "/skill.md", scope)
	for _, want := range []string{
		"`git diff basebasebase...HEAD`",
		"2 of 14",
		"or reference #1727",
		"- 7f4c76fc0000 spec: run the spec reviewer in an isolated worktree (#3178)",
		"- bbeb4d980000 feat(worktree): add a detached, lease-free review worktree (#3195)",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt must carry %q:\n%s", want, p)
		}
	}
}

func TestReviewPromptSaysTheFolderAloneDecidedWithoutAnIssue(t *testing.T) {
	scope := ReviewScope{BaseSHA: "b", Own: OwnCommitSet{RangeSize: 3, Commits: []OwnCommit{{SHA: "abc", Subject: "s"}}}}
	p := ReviewPromptForScope(FirstSigner, "X-001-a", "/repo", "nan/x", "pi", "/skill.md", scope)
	if !strings.Contains(p, "(they touch the spec folder under any of its names):") {
		t.Errorf("a folder-only list must say so:\n%s", p)
	}
	if strings.Contains(p, "or reference #") {
		t.Errorf("no issue was matched, yet the prompt names one:\n%s", p)
	}
}

// An initial review (CLI-132) has no base commit, so no range to list from.
func TestInitialReviewPromptCarriesNoOwnCommitList(t *testing.T) {
	scope := ReviewScope{BaseTree: "tree", RootSHA: "root", Own: OwnCommitSet{RangeSize: 1, Commits: []OwnCommit{{SHA: "abc", Subject: "s"}}}}
	p := ReviewPromptForScope(FirstSigner, "X-001-a", "/repo", "nan/x", "pi", "/skill.md", scope)
	if strings.Contains(p, "own commits") {
		t.Errorf("an initial review must not carry an own-commit list:\n%s", p)
	}
}

// AC5: the derivation against the real HARNESS-142 history. The commits exist
// only in a full clone of this repository, so a shallow checkout skips.
func TestOwnCommitsRecoversHarness142(t *testing.T) {
	const base, head = "adf49c93bb006b6322bb9bc6b20f1a44303e16e9", "c4d0e6c8b891159a2fb7267140d76a7fa9421689"
	repoRoot := strings.TrimSpace(gitOut(t, ".", "rev-parse", "--show-toplevel"))
	for _, c := range []string{base, head} {
		if exec.Command("git", "-C", repoRoot, "cat-file", "-e", c+"^{commit}").Run() != nil {
			t.Skipf("commit %s is not in this clone (shallow checkout)", c)
		}
	}
	got, err := OwnCommits(repoRoot, base, head, []string{"specs/HARNESS-142-reviewer-isolated-worktree"}, 1649)
	if err != nil {
		t.Fatal(err)
	}
	var prs []string
	for _, c := range got.Commits {
		prs = append(prs, c.Subject[strings.LastIndex(c.Subject, "(#"):])
	}
	if want := []string{"(#3178)", "(#3195)", "(#3196)", "(#3205)", "(#3213)"}; !reflect.DeepEqual(prs, want) || got.RangeSize != 14 {
		t.Errorf("HARNESS-142 own commits = %q of %d, want %q of 14", prs, got.RangeSize, want)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
