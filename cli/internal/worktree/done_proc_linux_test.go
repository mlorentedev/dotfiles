//go:build linux

package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParentPIDSurvivesACommandNameWithParensAndSpaces(t *testing.T) {
	proc := t.TempDir()
	stat := "1234 (a) b (c) d) S 987 1234 1234 0 -1 4194560 0 0 0 0\n"
	if err := os.WriteFile(filepath.Join(proc, "stat"), []byte(stat), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := parentPID(proc); got != 987 {
		t.Fatalf("parentPID = %d, want 987", got)
	}
	if got := parentPID(filepath.Join(proc, "missing")); got != 0 {
		t.Fatalf("an unreadable stat must stop the walk, got %d", got)
	}
}
