package mem

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The shape of the file that was actually clobbered, twice, on 2026-08-26/27.
const memoryWithTwoThreads = `# Project Memory

## Index

- [something](x.md)

## Session Handoff

### thread: wt-cli-023 (feat/cli-050-crystallize-cutover)

**Last task:** CLI-050 shipped, PR #1276 open.
**Next action:** merge #1276.

### thread: wt-pi-harness (fix/harness-045-reviewer-findings)

**Last task:** #561 binding core, PR #1272 merged.
**Next action:** AC7 guards before migrating the 35 skills.

## Findings and measurements

Moved to a topic file.
`

// THE REGRESSION THIS FILE EXISTS FOR. Two sessions wrote the handoff in
// sequence and each replaced the other's block; neither noticed, because
// last-writer-wins produces a well-formed file and a successful edit.
func TestWriteThreadLeavesEveryOtherThreadByteIdentical(t *testing.T) {
	out, changed, err := WriteThread(memoryWithTwoThreads, "wt-pi-harness",
		"**Last task:** rewritten by this session.\n**Next action:** merge #1275.")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("new content must report changed")
	}

	// The foreign thread, verbatim.
	if !strings.Contains(out, "**Last task:** CLI-050 shipped, PR #1276 open.") {
		t.Error("the other session's thread was lost — this is the exact clobber")
	}
	if !strings.Contains(out, "### thread: wt-cli-023 (feat/cli-050-crystallize-cutover)") {
		t.Error("the other session's heading was lost")
	}
	// Ours, replaced not appended.
	if !strings.Contains(out, "rewritten by this session") {
		t.Error("our own content was not written")
	}
	if strings.Contains(out, "#561 binding core") {
		t.Error("our old content survived alongside the new — appended instead of replaced")
	}
	if n := strings.Count(out, "### thread: wt-pi-harness"); n != 1 {
		t.Errorf("our thread appears %d times, want 1", n)
	}
	// Everything outside the section is untouched.
	for _, keep := range []string{"# Project Memory", "## Index", "## Findings and measurements", "Moved to a topic file."} {
		if !strings.Contains(out, keep) {
			t.Errorf("content outside the section was lost: %q", keep)
		}
	}
}

func TestWriteThreadIsIdempotent(t *testing.T) {
	body := "**Last task:** x.\n**Next action:** y."
	once, _, err := WriteThread(memoryWithTwoThreads, "wt-pi-harness", body)
	if err != nil {
		t.Fatal(err)
	}
	twice, changed, err := WriteThread(once, "wt-pi-harness", body)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("writing identical content must report changed=false")
	}
	if once != twice {
		t.Error("a second identical write altered the file")
	}
}

// A thread that is not present yet is APPENDED at the end of the section, so
// existing threads keep their positions: reordering a foreign entry is a diff
// its author did not make and would have to review.
func TestWriteThreadAppendsANewThreadWithoutReordering(t *testing.T) {
	out, changed, err := WriteThread(memoryWithTwoThreads, "wt-new", "**Last task:** fresh.")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("a new thread must report changed")
	}
	iCli := strings.Index(out, "### thread: wt-cli-023")
	iPi := strings.Index(out, "### thread: wt-pi-harness")
	iNew := strings.Index(out, "### thread: wt-new")
	if iCli < 0 || iPi < 0 || iNew < 0 {
		t.Fatal("a thread went missing")
	}
	if iCli >= iPi || iPi >= iNew {
		t.Errorf("threads were reordered: cli=%d pi=%d new=%d", iCli, iPi, iNew)
	}
	if strings.Contains(out[iNew:], "## Findings") && !strings.Contains(out, "**Last task:** fresh.") {
		t.Error("the new thread's body landed outside the section")
	}
}

// The heading may carry the branch, which moves; identity is the worktree, which
// does not.
func TestWriteThreadMatchesAThreadWhoseBranchChanged(t *testing.T) {
	out, _, err := WriteThread(memoryWithTwoThreads, "wt-pi-harness", "**Last task:** new branch now.")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "### thread: wt-pi-harness"); n != 1 {
		t.Errorf("a changed branch created a second thread (%d headings)", n)
	}
}

// Refusing to invent the section: a handoff written where nothing reads it is
// worse than none, because it looks like it was recorded.
func TestWriteThreadRefusesWhenTheSectionIsAbsent(t *testing.T) {
	if _, _, err := WriteThread("# Memory\n\n## Index\n\n- x\n", "wt-a", "body"); err == nil {
		t.Fatal("a missing handoff section must be an error, not a silent append")
	}
}

func TestWriteThreadRejectsAnEmptyKey(t *testing.T) {
	if _, _, err := WriteThread(memoryWithTwoThreads, "  ", "body"); err == nil {
		t.Fatal("an empty thread key is the shared slot this replaces")
	}
}

// A heading is matched by its first word, so a key with whitespace could never
// find its own block again: every write appended a copy (#1885). A branch name
// cannot hold whitespace, so only an explicit --thread could reach this.
func TestWriteThreadRejectsAKeyWithWhitespace(t *testing.T) {
	for _, key := range []string{"my work", "my\twork"} {
		if _, _, err := WriteThread(memoryWithTwoThreads, key, "body"); err == nil {
			t.Errorf("WriteThread accepted %q", key)
		}
		if _, err := WriteThreadAs(memoryWithTwoThreads, key, "claude", "body"); err == nil {
			t.Errorf("WriteThreadAs accepted %q", key)
		}
	}
}

// The written file must still archive correctly: extractHandoffBlock stops at
// the next "## ", so `###` sub-blocks fall inside it. Asserted rather than
// assumed — a second boundary rule here would be the fifth silently-divergent
// parser found in a week.
func TestWrittenThreadsStayInsideTheArchivedBlock(t *testing.T) {
	out, _, err := WriteThread(memoryWithTwoThreads, "wt-pi-harness", "**Last task:** archived too.")
	if err != nil {
		t.Fatal(err)
	}
	block := extractHandoffBlock(out)
	for _, want := range []string{"### thread: wt-cli-023", "### thread: wt-pi-harness", "archived too"} {
		if !strings.Contains(block, want) {
			t.Errorf("archival lost %q — the writer and the archiver disagree about the section boundary", want)
		}
	}
	if strings.Contains(block, "## Findings") {
		t.Error("archival ran past the section end")
	}
}

// gitFixture writes a repo the way git writes one: a linked worktree whose
// `.git` is a FILE pointing at `<repo>/.git/worktrees/<name>`, with HEAD naming
// the branch. Returns the worktree path.
func gitFixture(t *testing.T, repo, worktree, branch string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), repo)
	gitdir := filepath.Join(root, ".git", "worktrees", worktree)
	if err := os.MkdirAll(gitdir, 0o755); err != nil {
		t.Fatal(err)
	}
	head := "ref: refs/heads/" + branch + "\n"
	if branch == "" {
		head = "0123456789abcdef0123456789abcdef01234567\n" // detached
	}
	if err := os.WriteFile(filepath.Join(gitdir, "HEAD"), []byte(head), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), worktree)
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+gitdir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return wt
}

func mainFixture(t *testing.T, repo, branch string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), repo)
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/"+branch+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// THE THREAD IS THE LINE OF WORK — THE BRANCH — NOT THE WORKTREE.
//
// The vault is the SSOT across machines, so the same branch continued on a
// second box must resolve to the SAME thread rather than forking one. Keying on
// the worktree correlates on one machine and decorrelates the moment the work
// moves, which is exactly what "work from different machines" breaks.
//
// It reads git's own on-disk state, so a worktree created by any tool — Claude
// Code, Orca, opencode, pi, agy, copilot, or a bare `git worktree add` — behaves
// identically. A key derived from one tool's path pattern resolved every other
// tool's worktree to "main" and reintroduced the clobber for everyone else.
func TestThreadKeyIsTheBranchSoItTravelsBetweenMachines(t *testing.T) {
	wt := gitFixture(t, "dotfiles", "whatever-this-tool-calls-it", "feat/harness-088")
	if got := ThreadKey(wt); got != "feat-harness-088" {
		t.Errorf("want the branch, got %q — the worktree name must not decide the thread", got)
	}
	sub := filepath.Join(wt, "cli", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ThreadKey(sub); got != "feat-harness-088" {
		t.Errorf("a subdirectory got %q", got)
	}
	// The SAME branch in a DIFFERENT worktree — the cross-machine case — is the
	// same thread. This is the property the whole design turns on.
	other := gitFixture(t, "dotfiles", "a-totally-different-directory", "feat/harness-088")
	if ThreadKey(other) != ThreadKey(wt) {
		t.Errorf("the same branch in two places produced two threads: %q vs %q", ThreadKey(other), ThreadKey(wt))
	}
}

// The one genuine collision: the default branch is ambient work, and two
// machines' `main` are not one thread. The hostname enters ONLY there.
func TestThreadKeyQualifiesOnlyTheDefaultBranchWithTheHost(t *testing.T) {
	for _, b := range []string{"main", "master"} {
		got := ThreadKey(mainFixture(t, "dotfiles", b))
		if !strings.HasPrefix(got, b+"@") {
			t.Errorf("branch %q must be host-qualified, got %q", b, got)
		}
	}
	if got := ThreadKey(gitFixture(t, "dotfiles", "w", "fix/thing")); strings.Contains(got, "@") {
		t.Errorf("a feature branch must not carry the host, got %q", got)
	}
}

// A detached HEAD has no line of work to name, and must say so rather than
// collapsing into a plausible "main" that would silently share somebody's thread.
func TestThreadKeyNamesADetachedHeadRatherThanGuessing(t *testing.T) {
	got := ThreadKey(gitFixture(t, "dotfiles", "wt-detached", ""))
	if !strings.HasPrefix(got, "wt-detached@") {
		t.Errorf("a detached HEAD must be named after its worktree and host, got %q", got)
	}
}

// THE DEBT FIX: the project is the REPOSITORY, not the basename of wherever the
// session happens to be standing.
//
// `SessionEnd` resolved it as filepath.Base(cwd), so a session in a subdirectory
// — `cli/`, where most work here happens — resolved the wrong project, found no
// MEMORY.md, and SILENTLY ARCHIVED NOTHING. Same class as the thread-key defect,
// in a different function; both read RepoIdentity now, so they cannot disagree.
func TestRepoIdentityResolvesTheProjectFromAnywhereInTheTree(t *testing.T) {
	wt := gitFixture(t, "dotfiles", "wt-x", "feat/y")
	for _, dir := range []string{wt, filepath.Join(wt, "cli"), filepath.Join(wt, "cli", "internal", "mem")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		id, ok := RepoIdentity(dir)
		if !ok {
			t.Fatalf("RepoIdentity failed at %s", dir)
		}
		if id.Project != "dotfiles" {
			t.Errorf("at %s: project = %q, want dotfiles", dir, id.Project)
		}
		if id.Branch != "feat/y" {
			t.Errorf("at %s: branch = %q, want feat/y", dir, id.Branch)
		}
	}
	root := mainFixture(t, "knowledge", "master")
	sub := filepath.Join(root, "10_projects", "dotfiles")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	id, ok := RepoIdentity(sub)
	if !ok || id.Project != "knowledge" || id.Branch != "master" {
		t.Errorf("main checkout subdirectory: got %+v ok=%v", id, ok)
	}
}

func TestRepoIdentityReportsWhenGitKnowsNothing(t *testing.T) {
	if _, ok := RepoIdentity(t.TempDir()); ok {
		t.Error("a path outside any repository must report ok=false, not a guessed identity")
	}
}

// The naming collision this replaces: two WORKTREES on one day produced `-2` and
// `-3` suffixes that encode nothing, so no session could derive its own journal.
func TestJournalNameIsDerivableAndDistinctPerWorktree(t *testing.T) {
	a := JournalName("2026-08-27", "dotfiles", "claude", "wt-pi-harness")
	b := JournalName("2026-08-27", "dotfiles", "claude", "wt-cli-023")
	if a == b {
		t.Fatal("two worktrees on one day still collide")
	}
	if a != "2026-08-27-dotfiles-claude-wt-pi-harness.md" {
		t.Errorf("unexpected name %q", a)
	}
	// A single-checkout session keeps the historical name: "main" carries no
	// information, and suffixing it would rename every archive on every
	// single-session machine for nothing.
	for _, thread := range []string{"", "main"} {
		if got := JournalName("2026-08-27", "dotfiles", "claude", thread); got != "2026-08-27-dotfiles-claude.md" {
			t.Errorf("thread %q must produce the unsuffixed name, got %q", thread, got)
		}
	}
}

// HARNESS-174 (#1980): the filename takes only what vault-validate's session
// rule accepts ([A-Za-z0-9._@-]); a fork's '+' and anything else becomes '-'.
func TestJournalNameKeepsOnlyCharactersTheVaultAccepts(t *testing.T) {
	for thread, want := range map[string]string{
		"main@msi+copilot": "2026-10-01-dotfiles-copilot-main@msi-copilot.md",
		"wt-a_b.c":         "2026-10-01-dotfiles-copilot-wt-a_b.c.md",
		"feat/x y:z#1":     "2026-10-01-dotfiles-copilot-feat-x-y-z-1.md",
	} {
		if got := JournalName("2026-10-01", "dotfiles", "copilot", thread); got != want {
			t.Errorf("thread %q: got %q, want %q", thread, got, want)
		}
	}
	// --agent is free text and a project is a directory name: neither is
	// validated upstream, so the sanitiser covers the whole name.
	if got, want := JournalName("2026-10-01", "my+proj", "claude code", "main"), "2026-10-01-my-proj-claude-code.md"; got != want {
		t.Errorf("project and agent: got %q, want %q", got, want)
	}
}

// REVIEWER FINDING (#1279): a `###` heading inside a thread's BODY truncated its
// span, so a replacement rewrote only the part before it and orphaned the rest.
// Handoff bodies legitimately carry `### Next Actions` and the like, so this is
// data loss on ordinary content.
func TestWriteThreadSurvivesASubheadingInsideAThreadBody(t *testing.T) {
	doc := "# M\n\n## Session Handoff\n\n" +
		"### thread: wt-a\n\nintro for a\n\n### Next Actions\n\n- do the thing\n\n" +
		"### thread: wt-b\n\nbody for b\n\n## Findings\n\ntail\n"

	out, _, err := WriteThread(doc, "wt-a", "replaced body for a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "- do the thing") {
		t.Error("the thread's own subheading content survived — the replacement covered only part of the thread")
	}
	if !strings.Contains(out, "body for b") {
		t.Fatal("the NEXT thread was destroyed, absorbed into the replaced span")
	}
	if !strings.Contains(out, "### thread: wt-b") {
		t.Fatal("the next thread's heading was destroyed")
	}
	if !strings.Contains(out, "replaced body for a") {
		t.Error("the new body was not written")
	}
	if !strings.Contains(out, "tail") {
		t.Error("content after the section was lost")
	}
}

// REVIEWER FINDING (#1279): the naming fallback could claim a directory that
// merely CONTAINS `-wt-` while sitting inside an ordinary checkout. git is
// authoritative, so it must be consulted across the whole walk-up before the
// convention is consulted at all.
func TestThreadKeyPrefersGitOverAConventionalLookingSubdirectory(t *testing.T) {
	root := mainFixture(t, "dotfiles", "main")
	inner := filepath.Join(root, "vendor-wt-cache")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ThreadKey(inner); !strings.HasPrefix(got, "main@") {
		t.Errorf("a look-alike directory inside a main checkout got its own thread %q — git said main", got)
	}
}

// A THREAD KEY NAMES A LINE OF WORK IN ONE REPOSITORY (#1606).
//
// handoff-write took the key from the current directory's branch whatever
// MEMORY.md it was told to write. Run from the vault checkout (branch master)
// with dotfiles' MEMORY.md, it keyed the thread `master@<host>` and replaced
// another session's block of that name, reporting success. A key read from one
// repository says nothing about work in another, so HandoffThread refuses it and
// names the key it would have used; an explicit --thread is always taken.
func TestHandoffThreadRefusesAKeyFromAnotherRepository(t *testing.T) {
	vault := mainFixture(t, "knowledge", "master")
	memory := filepath.Join(vault, "10_projects", "dotfiles", "memory", "MEMORY.md")

	got, err := HandoffThread("", memory, vault)
	if err == nil {
		t.Fatalf("a vault checkout keyed a dotfiles thread %q; want a refusal", got)
	}
	for _, want := range []string{"--thread", `"knowledge"`, `"dotfiles"`, ThreadKey(vault)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %s:\n%v", want, err)
		}
	}
}

func TestHandoffThreadRefusesOutsideAnyRepository(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "10_projects", "dotfiles", "memory", "MEMORY.md")
	if got, err := HandoffThread("", memory, t.TempDir()); err == nil {
		t.Fatalf("a directory in no repository keyed a dotfiles thread %q; want a refusal", got)
	}
}

func TestHandoffThreadTakesAnExplicitKeyFromAnywhere(t *testing.T) {
	vault := mainFixture(t, "knowledge", "master")
	memory := filepath.Join(vault, "10_projects", "dotfiles", "memory", "MEMORY.md")
	got, err := HandoffThread("fix-harness-remediation", memory, vault)
	if err != nil || got != "fix-harness-remediation" {
		t.Fatalf("an explicit --thread must be taken as given: got %q, %v", got, err)
	}
}

func TestHandoffThreadIsTheBranchInsideTheProject(t *testing.T) {
	wt := gitFixture(t, "dotfiles", "dotfiles-wt-x", "feat/x")
	memory := filepath.Join(t.TempDir(), "10_projects", "dotfiles", "memory", "MEMORY.md")
	got, err := HandoffThread("", memory, wt)
	if err != nil || got != "feat-x" {
		t.Fatalf("a worktree of the project must key its own branch: got %q, %v", got, err)
	}
}

// A MEMORY.md outside the vault's project layout names no project, so there is
// nothing to compare the repository against; the key stays the cwd's, as before.
func TestHandoffThreadKeepsTheCwdKeyOutsideTheProjectLayout(t *testing.T) {
	vault := mainFixture(t, "knowledge", "master")
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	got, err := HandoffThread("", memory, vault)
	if err != nil || got != ThreadKey(vault) {
		t.Fatalf("want the cwd key %q for a path outside the layout, got %q, %v", ThreadKey(vault), got, err)
	}
}

// A journal name is <project>-<agent>[-<thread>], so its first word is always
// the project's. The writer scan used to read it as a candidate too, so a
// project named like an agent (`pi`, `codex`, `gemini`) attributed every block
// to that agent (HARNESS-088 review, round 1).
func TestJournalWriterSkipsTheProjectWord(t *testing.T) {
	cases := []struct{ line, want string }{
		{"Journal: sessions/2026-09-30-pi-claude-main.md", "claude"},
		{"Journal: sessions/2026-09-30-codex-copilot.md", "copilot"},
		{"Journal: sessions/2026-09-30-dotfiles-claude-wt-pi.md", "claude"},
		{"Journal: sessions/2026-09-30-yt-metrics-cli-agy-feat-x.md", "agy"},
	}
	for _, c := range cases {
		if got := journalWriter(c.line, ""); got != c.want {
			t.Errorf("journalWriter(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

// MEMORY-014 (#1928): a body line the parser reads as structure is refused, and
// the document is left as it was. Reproduced first without the check: the
// marker line split wt-a in two, and the next write of wt-a replaced only the
// part above it, stranding the tail under a thread named "smuggled".
func TestWriteThreadRefusesABodyLineThatIsStructure(t *testing.T) {
	for name, body := range map[string]string{
		"thread marker":         "**Last task:** x<br>\n### thread: smuggled\n**Next action:** y<br>",
		"stamped thread marker": "### thread: main@msi (writer: pi)\ncopied from another block",
		"marker with CR":        "intro\r\n### thread: smuggled\r\n",
		"level-2 heading":       "**Last task:** x<br>\n## Findings\nmore",
		"handoff heading":       "## Session Handoff\nnested",
	} {
		t.Run(name, func(t *testing.T) {
			out, changed, err := WriteThread(memoryWithTwoThreads, "wt-cli-023", body)
			if err == nil {
				t.Fatalf("accepted a structural body line; the document became:\n%s", out)
			}
			if changed || out != "" {
				t.Errorf("a refused write still produced a document (changed=%v)", changed)
			}
		})
	}
}

// The refusal is narrow: ordinary headings and text that only mentions a marker
// stay content, as they always were.
func TestWriteThreadKeepsOrdinaryHeadingsAsContent(t *testing.T) {
	for name, body := range map[string]string{
		"level-3 heading":     "### Next Actions\n- do it",
		"marker mid-line":     "see the `### thread: wt-a` block",
		"indented marker":     "  ### thread: wt-a",
		"bare level-2 marker": "##",
	} {
		t.Run(name, func(t *testing.T) {
			out, _, err := WriteThread(memoryWithTwoThreads, "wt-cli-023", body)
			if err != nil {
				t.Fatalf("refused ordinary content: %v", err)
			}
			if !strings.Contains(out, body) {
				t.Errorf("body not written:\n%s", out)
			}
			// Every byte outside the written thread is unchanged: the document up
			// to its marker, and from the foreign thread's marker to the end.
			// Containment alone would pass a foreign block emitted twice or moved.
			own := strings.Index(memoryWithTwoThreads, "### thread: wt-cli-023")
			foreign := strings.Index(memoryWithTwoThreads, "### thread: wt-pi-harness")
			if !strings.HasPrefix(out, memoryWithTwoThreads[:own]) {
				t.Errorf("the document before the written thread changed:\n%s", out)
			}
			if !strings.HasSuffix(out, memoryWithTwoThreads[foreign:]) {
				t.Errorf("the foreign thread or the tail after it changed:\n%s", out)
			}
			if n := strings.Count(out, "### thread: wt-pi-harness"); n != 1 {
				t.Errorf("the foreign thread appears %d times:\n%s", n, out)
			}
		})
	}
}

// MEMORY-016 (#1930): git writes a relative gitdir pointer under
// worktree.useRelativePaths, relative to the `.git` file. Resolved against the
// process's working directory instead, the project read as ".." and HEAD came
// from the wrong place. The process sits in an unrelated directory here, so a
// cwd-relative read cannot pass by accident.
func TestThreadKeyResolvesARelativeGitdirAgainstThePointerFile(t *testing.T) {
	base := t.TempDir()
	gitdir := filepath.Join(base, "dotfiles", ".git", "worktrees", "rel")
	if err := os.MkdirAll(gitdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitdir, "HEAD"), []byte("ref: refs/heads/feat/relative\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(base, "dotfiles-wt-rel")
	sub := filepath.Join(wt, "cli", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	pointer := "gitdir: " + filepath.Join("..", "dotfiles", ".git", "worktrees", "rel") + "\n"
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte(pointer), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	for _, dir := range []string{wt, sub} {
		if got := ThreadKey(dir); got != "feat-relative" {
			t.Errorf("ThreadKey(%s) = %q, want the branch the relative pointer names", dir, got)
		}
		id, ok := RepoIdentity(dir)
		if !ok || id.Project != "dotfiles" || id.Worktree != "rel" {
			t.Errorf("RepoIdentity(%s) = %+v, %v; want project dotfiles, worktree rel", dir, id, ok)
		}
	}
}

// MEMORY-016 (#1930): an unreadable working directory used to resolve to
// "main", the ambient thread, so an unrelated failure wrote into somebody
// else's handoff. It is an error now.
func TestThreadKeyForCwdFailsWhenTheWorkingDirectoryIsGone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows refuses to remove a process's working directory")
	}
	gone := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(gone)
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	key, err := ThreadKeyForCwd()
	if err == nil {
		t.Fatalf("got key %q from a working directory that no longer exists; want an error", key)
	}
	if key != "" {
		t.Errorf("an error came with key %q; a caller that ignores the error must not get a usable key", key)
	}
}

// setOriginHead records the remote's default branch the way `git clone` does,
// in the common git dir of the repository root returned by the fixtures.
func setOriginHead(t *testing.T, commonDir, branch string) {
	t.Helper()
	dir := filepath.Join(commonDir, "refs", "remotes", "origin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("ref: refs/remotes/origin/"+branch+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// MEMORY-013 (#1921): the ambient branch is the remote's default, not only the
// two names hardcoded before. A repository whose default is `develop` used one
// key on every machine, and two machines overwrote each other's handoff.
func TestThreadKeyQualifiesTheRemotesDefaultBranchWithTheHost(t *testing.T) {
	host := "@" + shortHost()
	for _, tc := range []struct {
		name, branch, originHead string
		linked, qualified        bool
	}{
		{"main, no origin HEAD", "main", "", false, true},
		{"master, origin HEAD develop", "master", "develop", false, true},
		{"develop as the default, main checkout", "develop", "develop", false, true},
		{"trunk as the default, linked worktree", "trunk", "trunk", true, true},
		{"develop when the default is main", "develop", "main", true, false},
		{"a feature branch", "feat/x", "develop", true, false},
		{"a slashed default", "release/1.2", "release/1.2", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cwd, common string
			if tc.linked {
				cwd = gitFixture(t, "proj", "wt", tc.branch)
				raw, err := os.ReadFile(filepath.Join(cwd, ".git"))
				if err != nil {
					t.Fatal(err)
				}
				gitdir := strings.TrimSpace(strings.TrimPrefix(string(raw), "gitdir:"))
				common = filepath.Dir(filepath.Dir(gitdir))
			} else {
				cwd = mainFixture(t, "proj", tc.branch)
				common = filepath.Join(cwd, ".git")
			}
			if tc.originHead != "" {
				setOriginHead(t, common, tc.originHead)
			}
			got := ThreadKey(cwd)
			if qualified := strings.HasSuffix(got, host); qualified != tc.qualified {
				t.Errorf("ThreadKey = %q; host-qualified = %v, want %v", got, qualified, tc.qualified)
			}
			// The key is a filename component: no derived default may put a
			// separator in it.
			if strings.ContainsAny(got, `/\`) {
				t.Errorf("ThreadKey = %q carries a path separator", got)
			}
		})
	}
}
