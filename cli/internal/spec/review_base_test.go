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
	// A commit can fork `git maintenance run --auto` or a detached gc that keeps
	// writing into .git after the test returns, so t.TempDir's RemoveAll finds
	// the directory not empty and fails a test whose assertions passed.
	git(t, repoRoot, "config", "gc.auto", "0")
	git(t, repoRoot, "config", "maintenance.auto", "false")

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

// writeAndCommit writes each path with fixed content and commits them.
func writeAndCommit(t *testing.T, repoRoot, msg string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(repoRoot, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# "+p+"\n\nenough text for rename detection to pair the file\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, repoRoot, "add", ".")
	git(t, repoRoot, "commit", "-qm", msg)
}

// BUG-108 (#1829): a renamed spec folder looks newly added under its new path,
// so the base used to be the parent of the RENAME, after the implementation.
// The base must be the parent of the commit that first added the folder under
// any of its names.
func TestResolveReviewBaseFollowsARenamedSpecFolder(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "base", "README.md")
	wantBase := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
	git(t, repoRoot, "mv", "specs/OLD-001-x", "specs/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "rename the spec")
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "mv", "specs/NEW-001-x", "specs/archive/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "archive the spec")

	// Two hops: archive's move, then the rename before it.
	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "archive", "NEW-001-x")); got != wantBase {
		t.Errorf("ResolveReviewBase = %q, want %q (the commit before the spec first appeared)", got, wantBase)
	}
}

// #2642: a spec added and renamed on a branch that later merged with --no-ff. The
// old name does not exist at HEAD and the merge is TREESAME to its first parent
// for that path, so history simplification pruned the side branch where the folder
// was added and the resolver found nothing. The base is the commit the branch left.
func TestResolveReviewBaseFollowsARenameOnAMergedBranch(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "base", "README.md")
	wantBase := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	git(t, repoRoot, "checkout", "-q", "-b", "work")
	writeAndCommit(t, repoRoot, "add the spec", "specs/OLD-001-x/proposal.md")
	git(t, repoRoot, "mv", "specs/OLD-001-x", "specs/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "rename the spec")
	git(t, repoRoot, "checkout", "-q", "main")
	git(t, repoRoot, "merge", "-q", "--no-ff", "work", "-m", "merge the work")
	writeAndCommit(t, repoRoot, "implement", "impl.go")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != wantBase {
		t.Errorf("ResolveReviewBase = %q, want %q (the commit the merged branch left)", got, wantBase)
	}
}

// The projects-toolkit shape from #1829: the implementation and the spec are in
// the root commit and the spec is renamed later. There is no commit before the
// work, so no base resolves, and the caller refuses rather than reviewing a
// diff that holds only the rename.
func TestResolveReviewBaseRefusesASpecRenamedOutOfTheRootCommit(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "root: spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
	git(t, repoRoot, "mv", "specs/OLD-001-x", "specs/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "rename the spec")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != "" {
		t.Errorf("ResolveReviewBase = %q, want \"\": the spec was born in the root commit, so the only "+
			"base after it excludes the implementation", got)
	}
}

// A move that renames a file in the same commit (proposal.md to spec.md) is
// still traced by the file's depth in the folder rather than its name.
func TestResolveReviewBaseFollowsAMoveThatAlsoRenamedAFile(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "base", "README.md")
	wantBase := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "NEW-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "mv", "specs/OLD-001-x/proposal.md", "specs/NEW-001-x/spec.md")
	git(t, repoRoot, "commit", "-qm", "move and rename")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != wantBase {
		t.Errorf("ResolveReviewBase = %q, want %q", got, wantBase)
	}
}

// A rename into the folder whose source cannot be placed (here, from directly
// under specs/, too shallow to have held it) is refused, never answered with
// the parent of the move: that base would exclude the implementation.
func TestResolveReviewBaseRefusesAnUntraceableRename(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "base", "README.md")
	writeAndCommit(t, repoRoot, "draft and implementation", "specs/draft.md", "impl.go")
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "NEW-001-x", "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "mv", "specs/draft.md", "specs/NEW-001-x/notes/draft.md")
	git(t, repoRoot, "commit", "-qm", "file the draft as a spec")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != "" {
		t.Errorf("ResolveReviewBase = %q, want \"\" for a rename it cannot trace", got)
	}
}

// A move that also rewrites the spec past git's rename threshold is not paired:
// diff-tree reports a delete and an add. That unpaired move is refused, not
// read as a folder born here, whose parent would exclude the implementation.
func TestResolveReviewBaseRefusesAMoveGitDidNotPair(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	writeAndCommit(t, repoRoot, "base", "README.md")
	writeAndCommit(t, repoRoot, "spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
	git(t, repoRoot, "rm", "-q", "-r", "specs/OLD-001-x")
	newSpec := filepath.Join(repoRoot, "specs", "NEW-001-x", "proposal.md")
	if err := os.MkdirAll(filepath.Dir(newSpec), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSpec, []byte("rewritten from scratch, nothing in common\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "add", ".")
	git(t, repoRoot, "commit", "-qm", "move and rewrite")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != "" {
		t.Errorf("ResolveReviewBase = %q, want \"\" for a move git did not pair", got)
	}
}

// A spec created fresh in a commit that also moves an unrelated file into it,
// or deletes a same-named file elsewhere, was born in that commit. Neither the
// move nor the delete is the folder moving, so the base is that commit's parent.
func TestResolveReviewBaseIgnoresFilesMovedInFromOutsideTheSpecsRoot(t *testing.T) {
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	// The deleted ADR's content shares nothing with the new proposal, so git
	// reports a plain delete rather than pairing the two as a rename.
	adr := filepath.Join(repoRoot, "docs", "adr", "proposal.md")
	if err := os.MkdirAll(filepath.Dir(adr), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adr, []byte("status: superseded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAndCommit(t, repoRoot, "base", "README.md", "docs/notes.md")
	wantBase := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "NEW-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repoRoot, "mv", "docs/notes.md", "specs/NEW-001-x/notes.md")
	git(t, repoRoot, "rm", "-q", "docs/adr/proposal.md")
	writeAndCommit(t, repoRoot, "new spec", "specs/NEW-001-x/proposal.md")

	if got := ResolveReviewBase(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x")); got != wantBase {
		t.Errorf("ResolveReviewBase = %q, want %q (the spec was created in that commit)", got, wantBase)
	}
}

// The fixture must not leave a background git process behind: one that is still
// writing into .git when t.TempDir is removed fails the test on cleanup.
func TestNewRepoWithSpecDisablesBackgroundMaintenance(t *testing.T) {
	repoRoot, _ := newRepoWithSpec(t, "FIX-001")
	for key, want := range map[string]string{"gc.auto": "0", "maintenance.auto": "false"} {
		if got := trim(git(t, repoRoot, "config", "--get", key)); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

// CLI-132 (#2566): the resolver names WHY it found no base, because the launcher
// accepts --initial for exactly one reason -- a spec born in the root commit --
// and must keep refusing every other one.

// initQuietRepo is an empty repository whose commits fork no background git
// process that could outlive the test.
func initQuietRepo(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	git(t, repoRoot, "init", "-q", "-b", "main")
	git(t, repoRoot, "config", "gc.auto", "0")
	git(t, repoRoot, "config", "maintenance.auto", "false")
	return repoRoot
}

func TestResolveReviewBaseDetailReportsADirectRootCommitSpec(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "root: spec and implementation", "specs/NEW-001-x/proposal.md", "impl.go")
	root := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "correct the implementation", "fix.go")

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x"))
	if want := (ReviewBase{Refusal: BaseRootCommit, RootSHA: root}); got != want {
		t.Errorf("ResolveReviewBaseDetail = %+v, want %+v", got, want)
	}
}

// The projects-toolkit shape: the root commit adds the spec, a later commit
// renames it. Following the rename must land on the root commit, not on the
// rename's parent, which would exclude the implementation.
func TestResolveReviewBaseDetailReportsASpecRenamedOutOfTheRootCommit(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "root: spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
	root := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	git(t, repoRoot, "mv", "specs/OLD-001-x", "specs/NEW-001-x")
	git(t, repoRoot, "commit", "-qm", "rename the spec")

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x"))
	if want := (ReviewBase{Refusal: BaseRootCommit, RootSHA: root}); got != want {
		t.Errorf("ResolveReviewBaseDetail = %+v, want %+v", got, want)
	}
}

func TestResolveReviewBaseDetailReportsAnUncommittedSpec(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "base", "README.md")
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "NEW-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x"))
	if want := (ReviewBase{Refusal: BaseNotCommitted}); got != want {
		t.Errorf("ResolveReviewBaseDetail = %+v, want %+v", got, want)
	}
}

// Both untraceable shapes the plain resolver already refuses: a rename whose
// source cannot be placed, and a move git did not pair. Neither is a root-commit
// spec, so neither may be reviewed with --initial.
func TestResolveReviewBaseDetailReportsAnUntraceableMove(t *testing.T) {
	cases := map[string]func(t *testing.T, repoRoot string){
		"rename from a source too shallow to place": func(t *testing.T, repoRoot string) {
			writeAndCommit(t, repoRoot, "draft and implementation", "specs/draft.md", "impl.go")
			if err := os.MkdirAll(filepath.Join(repoRoot, "specs", "NEW-001-x", "notes"), 0o755); err != nil {
				t.Fatal(err)
			}
			git(t, repoRoot, "mv", "specs/draft.md", "specs/NEW-001-x/notes/draft.md")
			git(t, repoRoot, "commit", "-qm", "file the draft as a spec")
		},
		"move git did not pair": func(t *testing.T, repoRoot string) {
			writeAndCommit(t, repoRoot, "spec and implementation", "specs/OLD-001-x/proposal.md", "impl.go")
			git(t, repoRoot, "rm", "-q", "-r", "specs/OLD-001-x")
			newSpec := filepath.Join(repoRoot, "specs", "NEW-001-x", "proposal.md")
			if err := os.MkdirAll(filepath.Dir(newSpec), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(newSpec, []byte("rewritten from scratch, nothing in common\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			git(t, repoRoot, "add", ".")
			git(t, repoRoot, "commit", "-qm", "move and rewrite")
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			repoRoot := initQuietRepo(t)
			writeAndCommit(t, repoRoot, "base", "README.md")
			build(t, repoRoot)

			got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, "specs", "NEW-001-x"))
			if want := (ReviewBase{Refusal: BaseUntraceable}); got != want {
				t.Errorf("ResolveReviewBaseDetail = %+v, want %+v", got, want)
			}
		})
	}
}

func TestResolveReviewBaseDetailKeepsTheBaseOfAnOrdinarySpec(t *testing.T) {
	repoRoot, wantBase := newRepoWithSpec(t, "FOO-001-thing")

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, "specs", "FOO-001-thing"))
	if want := (ReviewBase{SHA: wantBase, Refusal: BaseResolved}); got != want {
		t.Errorf("ResolveReviewBaseDetail = %+v, want %+v", got, want)
	}
}

// The baseline of an initial review is the empty tree in the repository's own
// object format. A hard-coded SHA-1 id would name a missing object in a SHA-256
// repository.
func TestEmptyTreeIsGitsEmptyTree(t *testing.T) {
	cases := []struct{ format, want string }{
		{"sha1", "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
		{"sha256", "6ef19b41225c5369f1c104d45d8d85efa9b057b53b14b4b9b939dd74decc5321"},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			repoRoot := t.TempDir()
			initCmd := exec.Command("git", "-C", repoRoot, "init", "-q", "--object-format="+tc.format)
			if out, err := initCmd.CombinedOutput(); err != nil {
				t.Skipf("this git cannot create a %s repository: %v\n%s", tc.format, err, out)
			}

			got, err := EmptyTree(repoRoot)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("EmptyTree = %q, want %q", got, tc.want)
			}
		})
	}
}

// The initial scope must hold the root commit's implementation AND every later
// correction. The triple-dot form is pinned as invalid from a tree, which is
// why the prompt has to state a two-dot diff.
func TestInitialScopeCoversTheRootImplementationAndLaterCorrections(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "root: spec and implementation", "specs/NEW-001-x/proposal.md", "impl.go")
	writeAndCommit(t, repoRoot, "correct the implementation", "fix.go")
	tree, err := EmptyTree(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	scope := git(t, repoRoot, "diff", "--name-only", tree, "HEAD")
	for _, want := range []string{"impl.go", "fix.go", "specs/NEW-001-x/proposal.md"} {
		if !contains(scope, want) {
			t.Errorf("the initial scope misses %s:\n%s", want, scope)
		}
	}
	if out, err := exec.Command("git", "-C", repoRoot, "diff", "--name-only", tree+"...HEAD").CombinedOutput(); err == nil {
		t.Errorf("a triple-dot diff from a tree succeeded, so the two-dot rule is unfounded:\n%s", out)
	}
}
