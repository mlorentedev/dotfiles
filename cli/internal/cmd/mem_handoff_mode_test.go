package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// handoff-write replaces MEMORY.md through a temp file, and os.CreateTemp makes
// that file 0600, so every write narrowed the file it replaced: on 2026-09-25, 9
// of the vault's 21 MEMORY.md files were 0600, exactly the ones this command had
// written, while the rest kept the umask's 0664. The file is not ours to
// re-permission, the same rule harness bind follows for a settings file.
func TestMemHandoffWriteKeepsTheFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(memory, 0o664); err != nil {
		t.Fatal(err)
	}
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "feat-x"})
	cmd.SetIn(bytes.NewBufferString("**Next action:** x.\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(memory)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o664 {
		t.Errorf("MEMORY.md is %o after the write, want the 664 it had", got)
	}
}
