package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// #2873: a branch the forge could not be asked about reads "unknown", not
// "-", and the table says why.
func TestWorktreeTableNamesAnUnansweredForge(t *testing.T) {
	var out bytes.Buffer
	err := printWorktreeTable(&out, []worktree.Info{{Path: "/wt/x", Branch: "feat/x",
		State: worktree.StateUnadopted, StateReason: "no dotf lease",
		PRMergedUnknown: "no forge backend for gitea.example"}})
	if err != nil {
		t.Fatal(err)
	}
	row := strings.Split(strings.TrimSpace(out.String()), "\n")[1]
	for _, want := range []string{"unknown", "gitea.example", "no dotf lease"} {
		if !strings.Contains(row, want) {
			t.Errorf("row %q does not contain %q", row, want)
		}
	}
}
