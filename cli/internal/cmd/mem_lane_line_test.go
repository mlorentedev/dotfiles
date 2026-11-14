package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneFixture is a repository whose activation record is written (or not) and
// which has one lane on feat/x. It writes the two records by hand: the line
// reads them, and how Activate makes them is lane's own tests' business.
type laneFixture struct{ main, lane string }

func newLaneFixture(t *testing.T, active bool) laneFixture {
	t.Helper()
	root := t.TempDir()
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	f := laneFixture{main: filepath.Join(root, "repo"), lane: filepath.Join(root, "feat-lane-primary")}
	if err := os.MkdirAll(f.main, 0o755); err != nil {
		t.Fatal(err)
	}
	git(f.main, "init", "-b", "main")
	git(f.main, "-c", "core.hooksPath="+t.TempDir(), "commit", "--allow-empty", "-m", "init")
	git(f.main, "worktree", "add", "-b", "feat/x", f.lane)
	common := git(f.main, "rev-parse", "--path-format=absolute", "--git-common-dir")
	admin := git(f.lane, "rev-parse", "--path-format=absolute", "--git-dir")
	if err := os.WriteFile(filepath.Join(admin, lane.LaneIdentityFile), []byte("feat-lane-primary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if active {
		if err := os.WriteFile(filepath.Join(common, lane.ActivationFile), []byte("active\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

const (
	inLaneLine  = "lane `feat-lane-primary` ACTIVE on `feat/x`"
	outLaneLine = "this repository runs lanes: start with `dotf lane switch <branch>`"
)

func sessionBrief(t *testing.T, cwd string) string {
	t.Helper()
	t.Setenv("SESSION_BRIEF_CWD", cwd)
	cmd := newMemCmd()
	cmd.SetArgs([]string{"session-start", "--format", "stdout"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// The three states a session can be in, as `mem session-start` and `mem resume`
// print them (HARNESS-195 AC3). Inactive prints nothing at all, not an empty line.
func TestMemLaneLine_GoldenForTheThreeStates(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n\n### thread: feat-x (writer: pi)\n\n**Next action:** n.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	active, inactive := newLaneFixture(t, true), newLaneFixture(t, false)
	cases := []struct {
		name, cwd, line string
	}{
		{"in a lane", active.lane, inLaneLine},
		{"outside the lanes of an active repository", active.main, outLaneLine},
		{"a repository that runs no lanes", inactive.lane, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The brief also carries the vault's health, which this test does
			// not control: the lane line is what it pins, first and once.
			assertLaneLine(t, "session-start", sessionBrief(t, c.cwd), c.line)
			t.Chdir(c.cwd)
			got := runMemResumeT(t, "--memory", memory, "--thread", "feat-x", "--agent", "pi", "--format", "context")
			assertLaneLine(t, "resume", got, c.line)
		})
	}
}

// assertLaneLine pins the line as the output's first and only one, or, for
// want == "", that no lane line is there at all.
func assertLaneLine(t *testing.T, who, out, want string) {
	t.Helper()
	first, _, _ := strings.Cut(out, "\n")
	mentions := strings.Count(out, "lane `") + strings.Count(out, "runs lanes")
	switch {
	case want == "" && mentions != 0:
		t.Errorf("%s printed a lane line for a repository without lanes:\n%s", who, out)
	case want != "" && (first != want || mentions != 1):
		t.Errorf("%s: first line %q with %d lane mentions, want %q once:\n%s", who, first, mentions, want, out)
	}
}
