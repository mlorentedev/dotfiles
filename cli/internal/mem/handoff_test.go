package mem

import (
	"strings"
	"testing"
)

// The shape of the file that was actually clobbered, twice, on 2026-08-26/27.
const memoryWithTwoThreads = `# Project Memory

## Index

- [something](x.md)

## Session Handoff

### wt-cli-023 (feat/cli-050-crystallize-cutover)

**Last task:** CLI-050 shipped, PR #1276 open.
**Next action:** merge #1276.

### wt-pi-harness (fix/harness-045-reviewer-findings)

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
	if !strings.Contains(out, "### wt-cli-023 (feat/cli-050-crystallize-cutover)") {
		t.Error("the other session's heading was lost")
	}
	// Ours, replaced not appended.
	if !strings.Contains(out, "rewritten by this session") {
		t.Error("our own content was not written")
	}
	if strings.Contains(out, "#561 binding core") {
		t.Error("our old content survived alongside the new — appended instead of replaced")
	}
	if n := strings.Count(out, "### wt-pi-harness"); n != 1 {
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
	iCli := strings.Index(out, "### wt-cli-023")
	iPi := strings.Index(out, "### wt-pi-harness")
	iNew := strings.Index(out, "### wt-new")
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
	if n := strings.Count(out, "### wt-pi-harness"); n != 1 {
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
	for _, want := range []string{"### wt-cli-023", "### wt-pi-harness", "archived too"} {
		if !strings.Contains(block, want) {
			t.Errorf("archival lost %q — the writer and the archiver disagree about the section boundary", want)
		}
	}
	if strings.Contains(block, "## Findings") {
		t.Error("archival ran past the section end")
	}
}

func TestThreadKeyDerivesFromTheWorktree(t *testing.T) {
	for _, tc := range []struct{ cwd, want string }{
		{"/home/x/Projects/dotfiles-wt-pi-harness", "wt-pi-harness"},
		{"/home/x/Projects/dotfiles-wt-cli-023/", "wt-cli-023"},
		{"/home/x/Projects/dotfiles", "main"},
		{"/home/x/Projects/knowledge", "main"},
		{"", "main"},
		// SUBDIRECTORIES. The first version read only the basename, so a session
		// in `cli/` — where most work in this repository happens — resolved to
		// "main", and every such session would have shared one key and clobbered
		// the others exactly as before. The tests passed because they all passed
		// worktree roots; running `dotf mem thread` from `cli/` did not.
		{"/home/x/Projects/dotfiles-wt-pi-harness/cli", "wt-pi-harness"},
		{"/home/x/Projects/dotfiles-wt-pi-harness/cli/internal/mem", "wt-pi-harness"},
		{"/home/x/Projects/dotfiles/cli/internal/mem", "main"},
	} {
		if got := ThreadKey(tc.cwd); got != tc.want {
			t.Errorf("ThreadKey(%q) = %q, want %q", tc.cwd, got, tc.want)
		}
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
	if got := JournalName("2026-08-27", "dotfiles", "claude", ""); !strings.HasSuffix(got, "-main.md") {
		t.Errorf("an empty thread must fall back to main, got %q", got)
	}
}
