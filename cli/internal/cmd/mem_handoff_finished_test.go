package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const finishedMemory = "# M\n\n## Session Handoff\n\n### thread: feat-a\n\n**Next action:** Keep going.<br>\n"

// finishedSetup writes MEMORY.md and pins the date the archive is named by.
func finishedSetup(t *testing.T) (memory, archive string) {
	t.Helper()
	saved := now
	now = func() time.Time { return time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = saved })
	memory = filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte(finishedMemory), 0o644); err != nil {
		t.Fatal(err)
	}
	return memory, filepath.Join(filepath.Dir(memory), "archive", "superseded-handoffs-2026-11-14.md")
}

// MEMORY-019 AC1: --finished archives the entry, writes the thread file and
// leaves every other thread in place.
func TestHandoffWriteFinishedArchivesTheThread(t *testing.T) {
	memory, archive := finishedSetup(t)
	_, errOut, err := runHandoffWrite(t, memory, "**Next action:** None for this thread.\n", "--finished")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, `archived   thread "feat-x" to archive/superseded-handoffs-2026-11-14.md`) {
		t.Errorf("stderr does not report the move:\n%s", errOut)
	}
	if index, _ := os.ReadFile(memory); string(index) != finishedMemory {
		t.Errorf("MEMORY.md is\n%s\nwant it as it was:\n%s", index, finishedMemory)
	}
	got, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Superseded handoffs 2026-11-14\n\n### thread: feat-x\n\n**Next action:** None for this thread.<br>\n[full thread](threads/feat-x.md)\n"
	if string(got) != want {
		t.Errorf("archive holds\n%q\nwant\n%q", got, want)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(memory), "threads", "feat-x.md")); err != nil {
		t.Errorf("the thread file was not written: %v", err)
	}
}

// MEMORY-019 AC2: a thread awaiting Manu is refused with nothing written, and
// a "None until" thread written without the flag stays where it is.
func TestHandoffWriteFinishedRefusesAwaitingAndNoneIsNotAMarker(t *testing.T) {
	memory, archive := finishedSetup(t)
	body := "**Next action:** None.\n**Awaiting Manu:**\n- A or B\n"
	if _, _, err := runHandoffWrite(t, memory, body, "--finished"); err == nil || !strings.Contains(err.Error(), "Awaiting Manu") {
		t.Fatalf("err = %v, want the Awaiting Manu refusal", err)
	}
	if index, _ := os.ReadFile(memory); string(index) != finishedMemory {
		t.Errorf("a refused write changed MEMORY.md:\n%s", index)
	}
	for _, p := range []string{archive, filepath.Join(filepath.Dir(memory), "threads", "feat-x.md")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("a refused write left %s", p)
		}
	}
	if _, _, err := runHandoffWrite(t, memory, "**Next action:** None until the release, then roll out.\n"); err != nil {
		t.Fatal(err)
	}
	if index, _ := os.ReadFile(memory); !strings.Contains(string(index), "### thread: feat-x") {
		t.Errorf("a thread written without --finished left MEMORY.md:\n%s", index)
	}
	if _, err := os.Stat(archive); !os.IsNotExist(err) {
		t.Error("a thread written without --finished was archived")
	}
}

// MEMORY-019 AC4: a repeated finished write appends nothing, and a dry run
// writes none of the three files but names the archive.
func TestHandoffWriteFinishedIsIdempotentAndDryRuns(t *testing.T) {
	memory, archive := finishedSetup(t)
	body := "**Next action:** None for this thread.\n"
	out, errOut, err := runHandoffWrite(t, memory, body, "--finished", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut, archive) || !strings.Contains(out, "## Session Handoff") {
		t.Errorf("dry run: stdout %q, stderr %q; want MEMORY.md on stdout and the archive named", out, errOut)
	}
	if _, err := os.Stat(filepath.Dir(archive)); !os.IsNotExist(err) {
		t.Error("a dry run created the archive directory")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(memory), "threads")); !os.IsNotExist(err) {
		t.Error("a dry run wrote the thread file")
	}
	if _, _, err := runHandoffWrite(t, memory, body, "--finished"); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(archive)
	out, _, err = runHandoffWrite(t, memory, body, "--finished")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unchanged") {
		t.Errorf("a repeated finished write did not say unchanged: %q", out)
	}
	if again, _ := os.ReadFile(archive); string(again) != string(first) {
		t.Errorf("a repeated finished write changed the archive:\n%s", again)
	}
}
