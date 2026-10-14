package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// runLogFixture is convergeFixture whose harness deploy runs a real child
// process, the way uv, npm, winget and the setup script do: the child inherits
// dotf's stdout and prints what it was given.
func runLogFixture(t *testing.T, childSays string) (repo, state string, console *[]*os.File) {
	t.Helper()
	repo, home := convergeFixture(t)
	state = filepath.Join(home, "state")
	t.Setenv("XDG_STATE_HOME", state)
	seen := []*os.File{}
	base := convergeOptions
	convergeOptions = func() converge.Options {
		o := base()
		o.RunHarnessDeploy = func(converge.Env) error {
			seen = append(seen, consoleOut())
			c := exec.Command(helperArgv()[0], helperArgv()[1:]...) //nolint:gosec // the test binary
			c.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO="+childSays, "EXIT_CODE=0")
			c.Stdout, c.Stderr = os.Stdout, os.Stderr
			return c.Run()
		}
		return o
	}
	return repo, filepath.Join(state, "dotfiles"), &seen
}

func runTranscripts(t *testing.T, state string) []string {
	t.Helper()
	logs, err := filepath.Glob(filepath.Join(state, "converge", "runs", "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	return logs
}

// The transcript holds what a child printed, with its credentials masked, and
// the report the run ended with: the two things a failed install on a box
// nobody can reach needs to be diagnosed after the terminal is gone.
func TestConverge_ApplyKeepsARedactedTranscript(t *testing.T) {
	const envSecret = "env-held-secret-value-42"
	t.Setenv("GITHUB_TOKEN", envSecret)
	gh := "ghp_" + strings.Repeat("Q7w8", 9)
	repo, state, _ := runLogFixture(t, "child: pushing with "+gh+" and "+envSecret+"\n")

	stdout, _, err := execute(t, "converge", "--repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	logs := runTranscripts(t, state)
	if len(logs) != 1 {
		t.Fatalf("want one transcript, got %v", logs)
	}
	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "child: pushing with [REDACTED]") {
		t.Errorf("the child's output is missing or unmasked:\n%s", got)
	}
	if !strings.Contains(got, "[REDACTED:GITHUB_TOKEN]") {
		t.Errorf("a secret-named variable's value should be masked by name:\n%s", got)
	}
	for _, leak := range []string{gh, envSecret} {
		if strings.Contains(got, leak) {
			t.Errorf("the transcript holds %q", leak)
		}
	}
	if !strings.Contains(got, "converge on ") || !strings.Contains(got, " changed, ") {
		t.Errorf("the transcript should end with the run's report:\n%s", got)
	}
	if _, err := os.Stat(strings.TrimSuffix(logs[0], ".log") + ".json"); err != nil {
		t.Errorf("the run's report is not beside its transcript: %v", err)
	}
	if !strings.Contains(stdout, "log: "+logs[0]) {
		t.Errorf("the run should say where its transcript is:\n%s", stdout)
	}
}

// While the run is captured, a terminal check still sees the real console: a
// sudo prompt or the identity walk must not decide "no terminal" because dotf
// itself is now writing to a pipe. And the swap is undone when the run ends.
func TestConverge_TerminalChecksSeeTheRealConsole(t *testing.T) {
	before := os.Stdout
	repo, _, seen := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if len(*seen) == 0 || (*seen)[0] != before {
		t.Errorf("during the run consoleOut() was %v, want the original stdout %v", *seen, before)
	}
	if os.Stdout != before {
		t.Errorf("stdout was not restored after the run")
	}
}

// A plan changes nothing and keeps nothing.
func TestConvergePlan_KeepsNoTranscript(t *testing.T) {
	repo, state, _ := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--plan", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if logs := runTranscripts(t, state); len(logs) != 0 {
		t.Errorf("a plan wrote a transcript: %v", logs)
	}
}

// An --only run is a run: it gets a transcript, and leaves last.json, the
// latest full run, alone.
func TestConvergeOnly_KeepsATranscriptButNotLastJSON(t *testing.T) {
	repo, state, _ := runLogFixture(t, "hello\n")
	if _, _, err := execute(t, "converge", "--only", "records-mirror", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	if logs := runTranscripts(t, state); len(logs) != 1 {
		t.Errorf("an --only run should keep one transcript, got %v", logs)
	}
	if _, err := os.Stat(filepath.Join(state, "converge", "last.json")); !os.IsNotExist(err) {
		t.Errorf("an --only run wrote last.json: %v", err)
	}
}
