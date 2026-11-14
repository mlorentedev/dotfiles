package mem

import (
	"strings"
	"testing"
)

const finishedBody = `> Updated: 2026-11-14<br>
**Last task:** Landed #3479.<br>
**Next action:** None for this thread.<br>
Journal: none (#3479 is the record)`

const archiveMemory = `# M

## Session Handoff

### thread: feat-a (writer: claude)

> Updated: 2026-11-10<br>
**Next action:** Keep going.<br>
[full thread](threads/feat-a.md)

### thread: feat-c (writer: pi)

> Updated: 2026-11-11<br>
**Next action:** Also keep going.<br>
[full thread](threads/feat-c.md)
`

// A finished thread leaves MEMORY.md as if it had never been written there, and
// the block it hands back is the one this write would have placed (MEMORY-019 AC1).
func TestArchiveThreadDropsTheEntryAndReturnsTheBlockItWouldHavePlaced(t *testing.T) {
	stored, err := StoreThreadAs(archiveMemory, "MEMORY.md", "feat-b", "claude", finishedBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ArchiveThreadAs(archiveMemory, "MEMORY.md", "feat-b", "claude", finishedBody)
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != archiveMemory {
		t.Errorf("MEMORY.md after the archive is\n%s\nwant it unchanged:\n%s", res.Content, archiveMemory)
	}
	lines := strings.Split(stored.Content, "\n")
	start, end := handoffSection(lines)
	s, e := threadSpan(lines, start, end, "feat-b")
	want := strings.Join(trimBlankEdges(lines[s:e]), "\n")
	if res.Block != want {
		t.Errorf("archived block is\n%s\nwant the block the write would have placed:\n%s", res.Block, want)
	}
	if res.FileBody != stored.FileBody || res.File != stored.File {
		t.Errorf("thread file is %q holding %q, want what a plain write stores", res.File, res.FileBody)
	}
	if res.Key != "feat-b" {
		t.Errorf("key is %q, want feat-b", res.Key)
	}
}

// An existing entry is removed and every other thread stays byte-identical.
func TestArchiveThreadRemovesAnExistingEntryOnly(t *testing.T) {
	res, err := ArchiveThreadAs(archiveMemory, "MEMORY.md", "feat-a", "claude", finishedBody)
	if err != nil {
		t.Fatal(err)
	}
	want := `# M

## Session Handoff

### thread: feat-c (writer: pi)

> Updated: 2026-11-11<br>
**Next action:** Also keep going.<br>
[full thread](threads/feat-c.md)
`
	if res.Content != want {
		t.Errorf("MEMORY.md is\n%s\nwant\n%s", res.Content, want)
	}
	if !res.Changed {
		t.Error("an entry was removed, but Changed is false")
	}
	if !strings.HasPrefix(res.Block, "### thread: feat-a (writer: claude)\n\n> Updated: 2026-11-14<br>") {
		t.Errorf("the block is not the new entry:\n%s", res.Block)
	}
}

// Archiving removes the owner's open decisions from the surface a session start
// loads, so a thread still awaiting one is refused (MEMORY-019 AC2).
func TestArchiveThreadRefusesAThreadAwaitingManu(t *testing.T) {
	body := finishedBody + "\n**Awaiting Manu:**\n- A or B on #3406"
	if _, err := ArchiveThreadAs(archiveMemory, "MEMORY.md", "feat-b", "claude", body); err == nil ||
		!strings.Contains(err.Error(), "Awaiting Manu") {
		t.Fatalf("err = %v, want a refusal naming Awaiting Manu", err)
	}
}

// Another agent's finished write goes to its fork, and only the fork is
// archived: the first agent's thread stays (MEMORY-019 AC3).
func TestArchiveThreadByAnotherAgentArchivesOnlyItsFork(t *testing.T) {
	withFork, err := StoreThreadAs(archiveMemory, "MEMORY.md", "feat-a", "pi", "**Next action:** Fork work.")
	if err != nil {
		t.Fatal(err)
	}
	if withFork.Key != "feat-a+pi" {
		t.Fatalf("setup: fork key is %q", withFork.Key)
	}
	res, err := ArchiveThreadAs(withFork.Content, "MEMORY.md", "feat-a", "pi", finishedBody)
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "feat-a+pi" || res.Kept != "claude" {
		t.Errorf("key %q kept %q, want the fork feat-a+pi kept from claude", res.Key, res.Kept)
	}
	if res.Content != archiveMemory {
		t.Errorf("MEMORY.md is\n%s\nwant claude's feat-a and pi's feat-c in place:\n%s", res.Content, archiveMemory)
	}
}

// The forked prune is exact: removing feat-a never takes feat-a+pi with it.
func TestPruneExactKeepsForks(t *testing.T) {
	withFork, err := StoreThreadAs(archiveMemory, "MEMORY.md", "feat-a", "pi", "**Next action:** Fork work.")
	if err != nil {
		t.Fatal(err)
	}
	out, changed := pruneThreadFromMemory(withFork.Content, "feat-a", false)
	if !changed || !strings.Contains(out, "### thread: feat-a+pi") || strings.Contains(out, "### thread: feat-a (writer") {
		t.Errorf("exact prune of feat-a gave\n%s", out)
	}
}

// A new dated archive opens with its heading; an existing one gets the block
// after one blank line, and a block it already holds is not appended twice
// (MEMORY-019 AC1, AC4).
func TestAppendArchive(t *testing.T) {
	block := "### thread: feat-b (writer: claude)\n\n**Next action:** None.<br>\n[full thread](threads/feat-b.md)"
	got, changed := AppendArchive("", "2026-11-14", block)
	if want := "# Superseded handoffs 2026-11-14\n\n" + block + "\n"; got != want || !changed {
		t.Errorf("new archive is\n%q\nwant\n%q", got, want)
	}
	other := "### thread: feat-z\n\n**Next action:** None.<br>"
	got2, changed := AppendArchive(got+"\n\n", "2026-11-14", other)
	if want := "# Superseded handoffs 2026-11-14\n\n" + block + "\n\n" + other + "\n"; got2 != want || !changed {
		t.Errorf("appended archive is\n%q\nwant\n%q", got2, want)
	}
	if again, changed := AppendArchive(got2, "2026-11-14", block); changed || again != got2 {
		t.Errorf("a block already archived was appended again:\n%s", again)
	}
}

// The writer archiving its own thread leaves another agent's fork of it in place.
func TestArchiveThreadByItsWriterKeepsAnotherAgentsFork(t *testing.T) {
	withFork, err := StoreThreadAs(archiveMemory, "MEMORY.md", "feat-a", "pi", "**Next action:** Fork work.")
	if err != nil {
		t.Fatal(err)
	}
	res, err := ArchiveThreadAs(withFork.Content, "MEMORY.md", "feat-a", "claude", finishedBody)
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "feat-a" || !strings.Contains(res.Content, "### thread: feat-a+pi") || strings.Contains(res.Content, "### thread: feat-a (writer") {
		t.Errorf("archiving feat-a gave key %q and\n%s", res.Key, res.Content)
	}
}
