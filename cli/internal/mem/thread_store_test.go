package mem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const storedBody = `> Updated: 2026-10-21<br>
**Last task:** Landed the store.<br>
**Decisions:** The thread file is the source.<br>
**Next action:** Write slice 3.<br>
**Verify:** ` + "`go test ./internal/mem`" + `<br>
**Awaiting Manu:**
- A or B on #2841
- whether lane-status-ff may go
Journal: sessions/2026-10-21-dotfiles-claude-feat-x.md`

func TestStoreThreadWritesTheBodyToItsFileAndADerivedEntry(t *testing.T) {
	memory := filepath.Join("vault", "memory", "MEMORY.md")
	res, err := StoreThreadAs("# M\n\n## Session Handoff\n", memory, "feat-x", "claude", storedBody)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("vault", "memory", "threads", "feat-x.md"); res.File != want {
		t.Errorf("thread file is %q, want %q", res.File, want)
	}
	if want := "### thread: feat-x (writer: claude)\n\n" + storedBody + "\n"; res.FileBody != want {
		t.Errorf("thread file holds\n%s\nwant the heading and the body as sent:\n%s", res.FileBody, want)
	}
	for _, want := range []string{
		"### thread: feat-x (writer: claude)",
		"> Updated: 2026-10-21<br>",
		"**Next action:** Write slice 3.<br>",
		"**Awaiting Manu:** 2, see the thread<br>",
		"Journal: sessions/2026-10-21-dotfiles-claude-feat-x.md",
		"[full thread](threads/feat-x.md)",
	} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("the entry lacks %q:\n%s", want, res.Content)
		}
	}
	for _, gone := range []string{"Landed the store", "The thread file is the source", "go test"} {
		if strings.Contains(res.Content, gone) {
			t.Errorf("the entry carries %q, which belongs only in the thread file", gone)
		}
	}
}

// The entry quotes the first line of the Next action and cuts it, so a long one
// cannot rebuild the block the store exists to shrink.
func TestTheDerivedEntryCutsALongNextAction(t *testing.T) {
	long := strings.Repeat("word ", 100)
	res, err := StoreThreadAs("## Session Handoff\n", "MEMORY.md", "feat-x", "", "**Next action:** "+long+"\nsecond line")
	if err != nil {
		t.Fatal(err)
	}
	entry := strings.TrimPrefix(res.Content, "## Session Handoff\n")
	if len(entry) > 350 {
		t.Errorf("the entry is %d bytes, want at most 350:\n%s", len(entry), entry)
	}
	if !strings.Contains(entry, "…<br>") || strings.Contains(entry, "second line") {
		t.Errorf("the Next action was not cut to its first line with an ellipsis:\n%s", entry)
	}
}

// Every quoted field is capped, not only the Next action: a long Updated or
// Journal line must not rebuild the block the entry replaces. With every field
// at its cap, the entry is at most 450 B.
func TestTheDerivedEntryCapsEveryQuotedField(t *testing.T) {
	long := strings.Repeat("x", 600)
	body := "> Updated: " + long + "<br>\n**Next action:** " + long + "<br>\n**Awaiting Manu:**\n- a\n- b\nJournal: sessions/" + long + ".md"
	res, err := StoreThreadAs("## Session Handoff\n", "MEMORY.md", "feat-x", "claude", body)
	if err != nil {
		t.Fatal(err)
	}
	entry := strings.TrimPrefix(res.Content, "## Session Handoff\n")
	if len(entry) > 450 {
		t.Errorf("the entry is %d bytes, want at most 450:\n%s", len(entry), entry)
	}
}

// The store refuses a key with whitespace, as the inline writer does: a
// heading is matched by its first word, so such a key would append a copy on
// every write.
func TestStoreRefusesAKeyWithWhitespace(t *testing.T) {
	if _, err := StoreThreadAs("## Session Handoff\n", "MEMORY.md", "a b", "claude", "**Next action:** x"); err == nil {
		t.Fatal("a key with whitespace was accepted")
	}
}

// AC4: the shape of dotfiles' MEMORY.md as measured on 2026-10-20 (12,010 bytes
// before the handoff section, eight threads of the sizes below, 23,423 bytes in
// all). Inline, the Verify and Awaiting Manu fields push it past the budget;
// stored, it ends far under it.
func TestAMemoryShapedLikeDotfilesStaysUnderBudgetWithTheNewFields(t *testing.T) {
	sizes := []int{2634, 1282, 1375, 1177, 1054, 1286, 1282, 1303}
	prefix := "# Project Memory\n\n" + strings.Repeat("- an index line that points at a memory file.\n", 12010/47) + "\n## Session Handoff\n"

	inline, stored := prefix, prefix
	for i, size := range sizes {
		key := fmt.Sprintf("feat-%d", i)
		body := syntheticThread(size) + "\n**Verify:** `" + strings.Repeat("gh pr view 1 ", 20) + "`<br>\n" +
			"**Awaiting Manu:**\n" + strings.Repeat("- a decision only the owner takes, with its issue #1234\n", 5)
		var err error
		if inline, _, err = WriteThread(inline, key, body); err != nil {
			t.Fatal(err)
		}
		res, err := StoreThreadAs(stored, "MEMORY.md", key, "claude", body)
		if err != nil {
			t.Fatal(err)
		}
		stored = res.Content
	}
	if len(inline) <= MemoryByteBudget {
		t.Fatalf("the inline fixture is %d bytes, not past the %d budget: the test proves nothing", len(inline), MemoryByteBudget)
	}
	if len(stored) > MemoryByteBudget {
		t.Errorf("stored, the file is %d bytes, past the %d budget", len(stored), MemoryByteBudget)
	}
}

// syntheticThread is a handoff body of about size bytes in the canonical shape.
func syntheticThread(size int) string {
	head := "> Updated: 2026-10-20<br>\n**Next action:** After the release, run the acceptance on the lab box and record it on the issue.<br>\n" +
		"Journal: sessions/2026-10-20-dotfiles-claude-feat-x.md\n**Last task:** "
	return head + strings.Repeat("x", max(size-len(head), 1)) + "<br>"
}

// A thread written by a release from before the store is an inline block. The
// next write moves it to the new shape, and every other thread keeps its bytes.
func TestStoreThreadMovesAnInlineBlockAndLeavesTheOthersByteIdentical(t *testing.T) {
	doc := "# M\n\n## Session Handoff\n\n### thread: feat-a\n\n**Last task:** a's work.\n**Next action:** a's step.\n\n" +
		"### thread: feat-b\n\n**Last task:** b's work.\n**Next action:** b's step.\n"
	res, err := StoreThreadAs(doc, "MEMORY.md", "feat-a", "", "**Next action:** a's new step.")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Content, "a's work") || !strings.Contains(res.Content, "[full thread](threads/feat-a.md)") {
		t.Errorf("feat-a is still inline:\n%s", res.Content)
	}
	b := "### thread: feat-b\n\n**Last task:** b's work.\n**Next action:** b's step.\n"
	if !strings.HasSuffix(res.Content, b) {
		t.Errorf("feat-b changed:\n%s", res.Content)
	}
}

// A fork keeps the other agent's entry and stores this agent's thread under the
// forked key, file included (#1690).
func TestStoreThreadForksAroundAnotherAgentsEntry(t *testing.T) {
	res, err := StoreThreadAs(memoryStampedByPi, "MEMORY.md", "master@msi", "claude", "**Next action:** mine.")
	if err != nil {
		t.Fatal(err)
	}
	if res.Key != "master@msi+claude" || res.Kept != "pi" || filepath.Base(res.File) != "master@msi+claude.md" {
		t.Errorf("wrote %q to %q (kept %q), want the fork master@msi+claude", res.Key, res.File, res.Kept)
	}
	if !strings.Contains(res.Content, "pi's next step.") {
		t.Errorf("pi's entry was replaced:\n%s", res.Content)
	}
}

// An explicit --thread is taken as given, so the key decides a path. A slash is
// mapped as a branch is, and anything that could leave threads/ is refused.
func TestThreadFileNames(t *testing.T) {
	for key, want := range map[string]string{
		"feat-x":                   "feat-x.md",
		"master@msi+claude":        "master@msi+claude.md",
		"fix/ux355-init-headscale": "fix-ux355-init-headscale.md",
		"release/1.2@macmini":      "release-1.2@macmini.md",
		"..":                       "",
		".hidden":                  "",
		"a*b":                      "",
		"feat/../../../etc/passwd": "feat-..-..-..-etc-passwd.md",
	} {
		got, err := threadFileName(key)
		if want == "" {
			if err == nil {
				t.Errorf("%q: named %q, want a refusal", key, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%q: got %q (%v), want %q", key, got, err, want)
		}
	}
}

// fix/x and fix-x name one file. The file's own heading says whose it is, so the
// second key is refused instead of replacing the first one's thread.
func TestCheckThreadFileRefusesAFileAnotherKeyHolds(t *testing.T) {
	held := []byte("### thread: fix/x (writer: pi)\n\n**Next action:** pi's.\n")
	if err := CheckThreadFile(held, "fix-x"); err == nil || !strings.Contains(err.Error(), "fix/x") {
		t.Errorf("fix-x over fix/x's file: %v, want a refusal naming fix/x", err)
	}
	if err := CheckThreadFile(append([]byte("\n\n"), held...), "fix-x"); err == nil {
		t.Error("a leading blank line hid the heading, and fix-x took fix/x's file")
	}
	if err := CheckThreadFile(held, "fix/x"); err != nil {
		t.Errorf("the key's own file: %v", err)
	}
	if err := CheckThreadFile(nil, "fix-x"); err != nil {
		t.Errorf("no file yet: %v", err)
	}
}

// A pointer in a body would turn an inline block into one that reads as stored,
// so the next read would follow it instead of the body: structure, refused.
func TestStoreThreadRefusesAPointerInTheBody(t *testing.T) {
	_, err := StoreThreadAs("## Session Handoff\n", "MEMORY.md", "feat-x", "", "**Next action:** x.\n[full thread](threads/feat-y.md)")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("got %v, want a refusal naming line 2", err)
	}
}

func TestThreadTextFollowsThePointer(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "threads"), 0o755); err != nil {
		t.Fatal(err)
	}
	full := "### thread: feat-x\n\n**Last task:** everything.\n"
	if err := os.WriteFile(filepath.Join(dir, "threads", "feat-x.md"), []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := "### thread: feat-x\n\n**Next action:** y.<br>\n[full thread](threads/feat-x.md)"
	if got := threadText(entry, dir); got != strings.TrimRight(full, "\n") {
		t.Errorf("a stored entry read as\n%s\nwant its file", got)
	}
	inline := "### thread: feat-x\n\n**Last task:** inline.\n"
	if got := threadText(inline, dir); got != inline {
		t.Errorf("an inline block read as %q", got)
	}
	// A missing file leaves the entry: it is still a handoff, the way an
	// unreadable MEMORY.md is a no-op rather than an error at session end.
	missing := strings.ReplaceAll(entry, "feat-x.md", "feat-gone.md")
	if got := threadText(missing, dir); got != missing {
		t.Errorf("a pointer to no file read as %q", got)
	}
}

func TestCheckMemoryBudget(t *testing.T) {
	over := "## Session Handoff\n\n### thread: new\n\n> Updated: 2026-10-21\n\n### thread: old\n\n> Updated: 2026-09-01\n\n### thread: undated\n\nx\n" +
		strings.Repeat("y", MemoryByteBudget)
	err := CheckMemoryBudget("", over)
	if err == nil {
		t.Fatal("a write that crosses the budget passed")
	}
	msg := err.Error()
	if o, n := strings.Index(msg, "  old (2026-09-01)"), strings.Index(msg, "  new (2026-10-21)"); o < 0 || n < 0 || o > n {
		t.Errorf("the refusal does not name the dated threads oldest first:\n%s", msg)
	}
	// A write that shrinks or holds a file already past the budget must pass, or
	// no file over it could ever be brought back under.
	if err := CheckMemoryBudget(over+"z", over); err != nil {
		t.Errorf("a shrinking write was refused: %v", err)
	}
	if err := CheckMemoryBudget("", "small"); err != nil {
		t.Errorf("a small write was refused: %v", err)
	}
}

// #1477, measured on the real file: an undated thread is not "the oldest". The
// pre-thread block and some live threads carry no Updated line, and the first
// refusal named them first. Undated threads are listed apart and never
// suggested, a Journal date stands in for a missing Updated line, and each
// suggestion says how to archive that thread: a stored one by its file, an
// inline one by its block.
func TestTheBudgetRefusalSuggestsOnlyDatedThreadsAndHowToArchiveEach(t *testing.T) {
	over := "## Session Handoff\n\n### thread: legacy-undated\n\n- a live parked item\n\n" +
		"### thread: by-journal\n\n**Next action:** x<br>\nJournal: sessions/2026-08-02-dotfiles-claude-by-journal.md\n\n" +
		"### thread: stored\n\n> Updated: 2026-09-01<br>\n[full thread](threads/stored.md)\n\n" +
		"### thread: fresh\n\n> Updated: 2026-10-21<br>\n" + strings.Repeat("y", MemoryByteBudget)
	err := CheckMemoryBudget("", over)
	if err == nil {
		t.Fatal("a write that crosses the budget passed")
	}
	msg := err.Error()
	for _, want := range []string{
		"\n  by-journal (2026-08-02): move its block from MEMORY.md to archive/",
		"\n  stored (2026-09-01): move threads/stored.md to archive/ and delete its entry from MEMORY.md",
		"\nUndated, so not suggested: legacy-undated",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal lacks %q:\n%s", want, msg)
		}
	}
	if strings.Index(msg, "by-journal (") > strings.Index(msg, "stored (") {
		t.Errorf("the dated threads are not oldest first:\n%s", msg)
	}

	none := "## Session Handoff\n\n### thread: a\n\nx\n" + strings.Repeat("y", MemoryByteBudget)
	if err := CheckMemoryBudget("", none); err == nil || !strings.Contains(err.Error(), "No thread carries a date") {
		t.Errorf("with no dated thread the refusal must say so and suggest none: %v", err)
	}
}
