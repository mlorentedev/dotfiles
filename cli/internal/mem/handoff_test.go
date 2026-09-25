package mem

import (
	"os"
	"path/filepath"
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

// A section written before threads existed: un-threaded text, then a thread.
const memoryWithLegacyBlock = `# Project Memory

## Session Handoff
> Updated: 2026-09-05  
**Last task:** shipped #410.  
**Next action:** merge #412.

### thread: master@msi

**Last task:** live work.
**Next action:** keep going.

## Findings

Moved to a topic file.
`

// THE LEGACY BLOCK HAS NO OWNER, SO NO WRITE EVER REPLACED IT (#1651).
//
// Every real thread is left byte-identical, which is right for threads. But
// text written before threads existed is not a thread: nobody's handoff-write
// will ever replace it, and it sits first under the heading, so the next
// session read a stale Next action ("merge #412", merged weeks earlier) as the
// current handoff. The command is the only way the file may change, so the
// command moves that text into a thread named by its own Updated date, at the
// end of the section, where it reads as history rather than as the handoff.
func TestWriteThreadMovesTheUnthreadedBlockIntoALegacyThread(t *testing.T) {
	out, changed, err := WriteThread(memoryWithLegacyBlock, "feat-x", "**Next action:** new work.")
	if err != nil || !changed {
		t.Fatalf("WriteThread: changed=%v err=%v", changed, err)
	}
	lines := strings.Split(out, "\n")
	start, end := handoffSection(lines)
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if _, ok := threadHeadingKey(lines[i]); !ok {
			t.Fatalf("the section still opens with un-threaded text %q:\n%s", lines[i], out)
		}
		break
	}
	legacy := strings.Index(out, "### thread: legacy-2026-09-05\n")
	if legacy < 0 {
		t.Fatalf("no legacy thread named by the block's Updated date:\n%s", out)
	}
	if !strings.Contains(out[legacy:], "**Next action:** merge #412.") {
		t.Errorf("the legacy text was lost instead of moved:\n%s", out)
	}
	for _, live := range []string{"### thread: master@msi", "### thread: feat-x"} {
		if i := strings.Index(out, live); i < 0 || i > legacy {
			t.Errorf("%q must precede the legacy thread:\n%s", live, out)
		}
	}
	if !strings.Contains(out, "### thread: master@msi\n\n**Last task:** live work.\n**Next action:** keep going.\n") {
		t.Errorf("a real thread changed:\n%s", out)
	}
	if !strings.Contains(out, "## Findings\n\nMoved to a topic file.\n") {
		t.Errorf("content after the section changed:\n%s", out)
	}
}

func TestWriteThreadMigratesEvenWhenItsOwnThreadIsUnchanged(t *testing.T) {
	out, changed, err := WriteThread(memoryWithLegacyBlock, "master@msi", "**Last task:** live work.\n**Next action:** keep going.")
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !strings.Contains(out, "### thread: legacy-2026-09-05") {
		t.Fatalf("an unchanged thread must not skip the migration: changed=%v\n%s", changed, out)
	}
	again, changed, err := WriteThread(out, "master@msi", "**Last task:** live work.\n**Next action:** keep going.")
	if err != nil || changed || again != out {
		t.Errorf("a second write must find nothing left to migrate: changed=%v err=%v", changed, err)
	}
}

func TestWriteThreadNamesAnUndatedLegacyBlockAndNeverCollides(t *testing.T) {
	undated := strings.Replace(memoryWithLegacyBlock, "> Updated: 2026-09-05  \n", "", 1)
	if key, ok := LegacyThreadKey(undated); !ok || key != "legacy-undated" {
		t.Errorf("an undated block: got %q, %v; want legacy-undated", key, ok)
	}
	taken := strings.Replace(memoryWithLegacyBlock, "### thread: master@msi", "### thread: legacy-2026-09-05\n\nmigrated earlier\n\n### thread: master@msi", 1)
	if key, ok := LegacyThreadKey(taken); !ok || key != "legacy-2026-09-05-2" {
		t.Errorf("a taken name: got %q, %v; want legacy-2026-09-05-2", key, ok)
	}
	if _, ok := LegacyThreadKey(memoryWithTwoThreads); ok {
		t.Error("a section with no un-threaded text reported a legacy block")
	}
}
