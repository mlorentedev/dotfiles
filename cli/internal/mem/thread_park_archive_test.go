package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const parkMemory = `# M

## Session Handoff

### thread: feat-a (writer: claude)

> Updated: 2026-11-10<br>
**Next action:** None.<br>
[full thread](threads/feat-a.md)

### thread: feat-a+pi (writer: pi)

> Updated: 2026-11-10<br>
**Next action:** None.<br>
[full thread](threads/feat-a+pi.md)

### thread: feat-a+agy (writer: agy)

> Updated: 2026-11-10<br>
**Awaiting Manu:** 1, see the thread<br>
[full thread](threads/feat-a+agy.md)

### thread: feat-c (writer: pi)

> Updated: 2026-11-11<br>
**Next action:** Keep going.<br>
[full thread](threads/feat-c.md)
`

// parkVault lays out a project's memory directory as a park finds it.
func parkVault(t *testing.T) (projectDir, memDir string) {
	t.Helper()
	projectDir = filepath.Join(t.TempDir(), "10_projects", "dotfiles")
	memDir = filepath.Join(projectDir, "memory")
	files := map[string]string{
		"MEMORY.md":               parkMemory,
		"threads/feat-a.md":       "### thread: feat-a (writer: claude)\n\n**Next action:** None.\n",
		"threads/feat-a+pi.md":    "### thread: feat-a+pi (writer: pi)\n\n**Next action:** None.\n",
		"threads/feat-a+agy.md":   "### thread: feat-a+agy (writer: agy)\n\n**Awaiting Manu:**\n- Pick A or B on #1\n",
		"threads/feat-c.md":       "### thread: feat-c (writer: pi)\n\n**Next action:** Keep going.\n",
		"archive/feedback_old.md": "an archived memory, not a thread\n",
	}
	for name, body := range files {
		p := filepath.Join(memDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return projectDir, memDir
}

// A park appends the lane's thread and its forks to the dated archive, drops
// them from MEMORY.md and moves their files to archive/threads/, where the
// entries' unchanged pointers resolve from the archive; a fork still awaiting
// Manu stays in MEMORY.md and threads/ and is named (MEMORY-020 AC1-AC3).
func TestArchiveParkedThreadUsesTheDatedArchive(t *testing.T) {
	projectDir, memDir := parkVault(t)
	day := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

	// A park passes the branch; its thread key is the branch made file-safe.
	got, err := ArchiveParkedThread(projectDir, "feat/a", day)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Archived, ",") != "feat-a,feat-a+pi" || strings.Join(got.Kept, ",") != "feat-a+agy" {
		t.Errorf("archived %v kept %v, want [feat-a feat-a+pi] and [feat-a+agy]", got.Archived, got.Kept)
	}
	// Each file is its own thread's, so none is reported shared (MEMORY-021 review).
	if len(got.Shared) != 0 {
		t.Errorf("shared %v, want none: every moved file was its own thread's", got.Shared)
	}
	wantPath := filepath.Join(memDir, "archive", "superseded-handoffs-2026-11-14.md")
	if got.Path != wantPath {
		t.Errorf("archive path %q, want %q", got.Path, wantPath)
	}

	archive, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	// Byte-identical to the blocks MEMORY.md held, in order (AC2).
	wantArchive := "# Superseded handoffs 2026-11-14\n\n" +
		"### thread: feat-a (writer: claude)\n\n> Updated: 2026-11-10<br>\n**Next action:** None.<br>\n[full thread](threads/feat-a.md)\n\n" +
		"### thread: feat-a+pi (writer: pi)\n\n> Updated: 2026-11-10<br>\n**Next action:** None.<br>\n[full thread](threads/feat-a+pi.md)\n"
	if string(archive) != wantArchive {
		t.Errorf("archive is\n%q\nwant\n%q", archive, wantArchive)
	}

	memory, err := os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"### thread: feat-a (writer", "### thread: feat-a+pi"} {
		if strings.Contains(string(memory), gone) {
			t.Errorf("MEMORY.md still holds %q:\n%s", gone, memory)
		}
	}
	for _, kept := range []string{"### thread: feat-a+agy", "### thread: feat-c"} {
		if !strings.Contains(string(memory), kept) {
			t.Errorf("MEMORY.md lost %q:\n%s", kept, memory)
		}
	}

	for _, name := range []string{"feat-a.md", "feat-a+pi.md"} {
		if _, err := os.Stat(filepath.Join(memDir, "archive", "threads", name)); err != nil {
			t.Errorf("archived thread file %s is not under archive/threads/: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(memDir, "threads", name)); !os.IsNotExist(err) {
			t.Errorf("archived thread file %s is still in threads/", name)
		}
	}
	for _, name := range []string{"feat-a+agy.md", "feat-c.md"} {
		if _, err := os.Stat(filepath.Join(memDir, "threads", name)); err != nil {
			t.Errorf("live thread file %s left threads/: %v", name, err)
		}
	}
	for _, name := range []string{"feat-a.md", "feat-a+pi.md", "feat-a+agy.md"} {
		if _, err := os.Stat(filepath.Join(memDir, "archive", name)); !os.IsNotExist(err) {
			t.Errorf("a park wrote archive/%s, the shape MEMORY-020 retires", name)
		}
	}

	// A second park of the same lane changes nothing.
	again, err := ArchiveParkedThread(projectDir, "feat-a", day)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Archived) != 0 {
		t.Errorf("a repeated park archived %v", again.Archived)
	}
	if after, _ := os.ReadFile(wantPath); string(after) != string(archive) {
		t.Errorf("a repeated park changed the archive:\n%s", after)
	}
}

// main, master and an empty project archive nothing and are not errors.
func TestArchiveParkedThreadIgnoresTheDefaultBranch(t *testing.T) {
	projectDir, memDir := parkVault(t)
	for _, branch := range []string{"main", "master", ""} {
		got, err := ArchiveParkedThread(projectDir, branch, time.Now())
		if err != nil || len(got.Archived) != 0 {
			t.Errorf("branch %q: archived %v, err %v", branch, got.Archived, err)
		}
	}
	if _, err := ArchiveParkedThread("", "feat-a", time.Now()); err != nil {
		t.Errorf("no project: %v", err)
	}
	if memory, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md")); string(memory) != parkMemory {
		t.Error("MEMORY.md changed")
	}
}

// A project with no MEMORY.md has nothing to archive.
func TestArchiveParkedThreadWithoutMemory(t *testing.T) {
	got, err := ArchiveParkedThread(t.TempDir(), "feat-a", time.Now())
	if err != nil || len(got.Archived) != 0 {
		t.Errorf("archived %v, err %v", got.Archived, err)
	}
}

// A park of feat-x leaves threads/feat-x.md in place when its heading says it
// is feat/x's: the file is another live thread's body (MEMORY-020 review F1).
func TestArchiveParkedThreadLeavesAFileAnotherKeyHolds(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "p")
	memDir := filepath.Join(projectDir, "memory")
	if err := os.MkdirAll(filepath.Join(memDir, "threads"), 0o755); err != nil {
		t.Fatal(err)
	}
	memory := "# M\n\n## Session Handoff\n\n### thread: feat-x (writer: claude)\n\n**Next action:** None.<br>\n\n### thread: feat/x (writer: pi)\n\n**Next action:** Keep going.<br>\n[full thread](threads/feat-x.md)\n"
	held := "### thread: feat/x (writer: pi)\n\n**Next action:** Keep going.\n"
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(memory), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "threads", "feat-x.md"), []byte(held), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ArchiveParkedThread(projectDir, "feat-x", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Archived, ",") != "feat-x" {
		t.Errorf("archived %v, want feat-x", got.Archived)
	}
	// The archived entry's pointer names feat/x's file, so the park says so (#3521).
	if strings.Join(got.Shared, ",") != "feat-x" {
		t.Errorf("shared %v, want feat-x: its archived pointer names another thread's file", got.Shared)
	}
	if body, err := os.ReadFile(filepath.Join(memDir, "threads", "feat-x.md")); err != nil || string(body) != held {
		t.Errorf("feat/x's thread file was moved or changed: %q (%v)", body, err)
	}
	if index, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md")); !strings.Contains(string(index), "### thread: feat/x") {
		t.Errorf("feat/x's entry left MEMORY.md:\n%s", index)
	}
}

// A move that fails after the entries are archived says which half happened
// for every thread, and does not stop at the first: the entry left MEMORY.md,
// the file did not leave threads/ (MEMORY-020 review 2, PR review).
func TestArchiveParkedThreadNamesAFailedMove(t *testing.T) {
	projectDir, memDir := parkVault(t)
	// A file where archive/threads/ goes: no move can create its directory.
	if err := os.WriteFile(filepath.Join(memDir, "archive", "threads"), []byte("in the way\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ArchiveParkedThread(projectDir, "feat-a", time.Now())
	if err == nil {
		t.Fatal("a failed move was not reported")
	}
	for _, key := range []string{"feat-a", "feat-a+pi"} {
		if !strings.Contains(err.Error(), `thread "`+key+`": its entry is archived, but its file stays in threads/`) {
			t.Errorf("error %q does not name %s's file staying in threads/", err, key)
		}
	}
	if len(got.Archived) != 0 {
		t.Errorf("archived %v, want none: no file left threads/", got.Archived)
	}
	for _, name := range []string{"feat-a.md", "feat-a+pi.md"} {
		if _, err := os.Stat(filepath.Join(memDir, "threads", name)); err != nil {
			t.Errorf("%s left threads/: %v", name, err)
		}
	}
}
