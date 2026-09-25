package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1651: --dry-run must show the migration of the un-threaded block, and the
// command names it on stderr, because nothing else tells the operator that a
// block they did not write moved.
func TestMemHandoffWriteDryRunShowsTheLegacyMigration(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	before := "# M\n\n## Session Handoff\n> Updated: 2026-09-05  \n**Next action:** merge #412.\n\n### thread: master@msi\n\nlive\n"
	if err := os.WriteFile(memory, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "feat-x", "--dry-run"})
	cmd.SetIn(bytes.NewBufferString("**Next action:** new work.\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("handoff-write --dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "### thread: legacy-2026-09-05") {
		t.Errorf("--dry-run does not show the migration:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "legacy-2026-09-05") {
		t.Errorf("stderr does not name the migrated thread:\n%s", errOut.String())
	}
	if after, _ := os.ReadFile(memory); string(after) != before {
		t.Error("--dry-run wrote the file")
	}
}
