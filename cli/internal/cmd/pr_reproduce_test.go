package cmd

import (
	"strings"
	"testing"
)

// The row is what a human pastes into ## Review triage, so it names the
// state, the exit and the commit. The exit status says whether a verdict
// exists at all: could-not-run is exit 1 although it is not an error of
// dotf's own.
func TestPrReproduce_PrintsTheRowAndExitsOnTheVerdict(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	writeCommit(t, repo, "a.txt", "one\n")
	head := gitIn(t, repo, "rev-parse", "--short", "HEAD")
	t.Chdir(repo)

	for _, tc := range []struct {
		name    string
		command []string
		state   string
		wantErr bool
	}{
		{"refuted", []string{"git", "status"}, "did-not-reproduce", false},
		{"reproduced", []string{"git", "rev-parse", "--verify", "-q", "refs/heads/nope"}, "reproduced", false},
		{"no verdict", []string{"tool017-no-such-binary"}, "could-not-run", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := execute(t, append([]string{"pr", "reproduce", "--"}, tc.command...)...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			for _, want := range []string{tc.state, head, strings.Join(tc.command, " ")} {
				if !strings.Contains(out, want) {
					t.Errorf("row %q does not name %q", out, want)
				}
			}
		})
	}
}

func TestPrReproduce_RefusesWithoutACommand(t *testing.T) {
	if _, _, err := execute(t, "pr", "reproduce"); err == nil {
		t.Fatal("a reproduction without a check must be refused")
	}
}
