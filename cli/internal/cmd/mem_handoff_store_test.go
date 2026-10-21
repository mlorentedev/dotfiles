package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runHandoffWrite runs handoff-write over memory with body on stdin.
func runHandoffWrite(t *testing.T, memory, body string, extra ...string) (string, string, error) {
	t.Helper()
	cmd := newMemCmd()
	cmd.SetArgs(append([]string{"handoff-write", "--memory", memory, "--thread", "feat-x"}, extra...))
	cmd.SetIn(bytes.NewBufferString(body))
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

// MEMORY-008: the body goes to the thread's own file, MEMORY.md gets the entry
// that points at it, and a new thread file takes MEMORY.md's mode.
func TestMemHandoffWriteStoresTheThreadInItsOwnFile(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(memory, 0o664); err != nil {
		t.Fatal(err)
	}
	body := "**Decisions:** kept in the file.\n**Next action:** next.\n"
	if _, _, err := runHandoffWrite(t, memory, body); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(filepath.Dir(memory), "threads", "feat-x.md")
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "kept in the file") {
		t.Errorf("the thread file lacks the body:\n%s", got)
	}
	index, _ := os.ReadFile(memory)
	if !strings.Contains(string(index), "[full thread](threads/feat-x.md)") || strings.Contains(string(index), "kept in the file") {
		t.Errorf("MEMORY.md does not hold the derived entry:\n%s", index)
	}
	if fi, err := os.Stat(file); err == nil && runtime.GOOS != "windows" && fi.Mode().Perm() != 0o664 {
		t.Errorf("the new thread file is %o, want MEMORY.md's 664", fi.Mode().Perm())
	}
	// A second write that changes only what the file holds still writes it.
	if out, _, err := runHandoffWrite(t, memory, strings.Replace(body, "kept", "still kept", 1)); err != nil || strings.Contains(out, "unchanged") {
		t.Fatalf("a change to the file alone read as unchanged: %q, %v", out, err)
	}
	if got, _ := os.ReadFile(file); !strings.Contains(string(got), "still kept") {
		t.Errorf("the thread file was not rewritten:\n%s", got)
	}
}

// AC4 (#1477): a write that takes MEMORY.md past the budget fails, names the
// oldest threads, and leaves both files as they were.
func TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	before := "# M\n" + strings.Repeat("x", 24_800) + "\n## Session Handoff\n\n### thread: old\n\n> Updated: 2026-09-01\n**Next action:** x.\n"
	if err := os.WriteFile(memory, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runHandoffWrite(t, memory, "> Updated: 2026-10-21\n**Next action:** "+strings.Repeat("y", 150)+"\n")
	if err == nil || !strings.Contains(err.Error(), "past the 25000") || !strings.Contains(err.Error(), "old") {
		t.Fatalf("got %v, want a refusal naming the budget and the oldest thread", err)
	}
	if after, _ := os.ReadFile(memory); string(after) != before {
		t.Error("MEMORY.md changed on a refused write")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(memory), "threads", "feat-x.md")); !os.IsNotExist(err) {
		t.Errorf("a refused write left a thread file: %v", err)
	}
}

// A dry run writes neither file and says where the thread would go.
func TestMemHandoffWriteDryRunStoresNothing(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	before := "# M\n\n## Session Handoff\n"
	if err := os.WriteFile(memory, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, err := runHandoffWrite(t, memory, "**Next action:** x.\n", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[full thread](threads/feat-x.md)") || !strings.Contains(errOut, filepath.Join("threads", "feat-x.md")) {
		t.Errorf("stdout %q, stderr %q: want the document and the thread file named", out, errOut)
	}
	if after, _ := os.ReadFile(memory); string(after) != before {
		t.Error("a dry run changed MEMORY.md")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(memory), "threads")); !os.IsNotExist(err) {
		t.Errorf("a dry run created threads/: %v", err)
	}
}
