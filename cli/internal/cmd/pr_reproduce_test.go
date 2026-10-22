package cmd

import (
	"os"
	"path/filepath"
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

// A probe that does not land prints its row and exits 1: no verdict.
func TestPrReproduce_AProbeThatDoesNotLandHasNoVerdict(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	writeCommit(t, repo, "a.txt", "one\n")
	t.Chdir(repo)
	probe := filepath.Join(t.TempDir(), "empty.diff")
	if err := os.WriteFile(probe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := execute(t, "pr", "reproduce", "--probe", probe, "--", "git", "status")
	if err == nil {
		t.Fatal("a probe that did not land must not exit 0")
	}
	if !strings.Contains(out, "did-not-land") || !strings.Contains(out, "empty.diff") {
		t.Errorf("row %q does not name the state and the probe", out)
	}
}

// The row says the check was inverted: "reproduced (exit 0)" reads as a
// contradiction without it.
func TestPrReproduce_NamesAnInvertedCheck(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	writeCommit(t, repo, "a.txt", "one\n")
	t.Chdir(repo)
	out, _, err := execute(t, "pr", "reproduce", "--defect-when-passes", "--", "git", "status")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{"reproduced (exit 0", "defect when the check passes"} {
		if !strings.Contains(out, want) {
			t.Errorf("row %q does not name %q", out, want)
		}
	}
}

// The row is the record a reviewer replays, so it must survive the shell and
// the markdown table it is pasted into (#2926 follow-up: a quoted pattern
// lost its quotes, and a pipe split the cell).
func TestCommandLineCanBeRunAgain(t *testing.T) {
	for _, tc := range []struct {
		argv []string
		want string
	}{
		{[]string{"go", "test", "./cli/x", "-run", "TestY"}, "go test ./cli/x -run TestY"},
		{[]string{"grep", "-q", "a fourth", "f.md"}, "grep -q 'a fourth' f.md"},
		{[]string{"sh", "-c", "gh api x | grep -q '^## K'"}, `sh -c 'gh api x \| grep -q '\''^## K'\'''`},
		{[]string{"echo", ""}, "echo ''"},
		{[]string{"ls", "*.go"}, "ls '*.go'"},
	} {
		if got := commandLine(tc.argv); got != tc.want {
			t.Errorf("commandLine(%q) = %s, want %s", tc.argv, got, tc.want)
		}
	}
}
