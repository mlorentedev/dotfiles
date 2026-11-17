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

// HARNESS-142: a review worktree is detached, so its BRANCH cell would be
// empty; the table names it as a review worktree instead of a bare blank that
// reads as someone's abandoned checkout.
func TestWorktreeTableLabelsAReviewWorktree(t *testing.T) {
	var out bytes.Buffer
	err := printWorktreeTable(&out, []worktree.Info{{Path: "/wt/r", Head: "4ed8cff5fb93", IsDetached: true,
		Metadata: &worktree.Metadata{Kind: worktree.KindReview}, State: worktree.StateActive}})
	if err != nil {
		t.Fatal(err)
	}
	row := strings.Split(strings.TrimSpace(out.String()), "\n")[1]
	if !strings.Contains(row, "(review 4ed8cff)") {
		t.Errorf("row %q does not label the review worktree and its commit", row)
	}
}
