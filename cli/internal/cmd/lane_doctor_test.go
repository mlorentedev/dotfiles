package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

const coherentMarker = `{"version":1,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["copilot-worktrees"]}`

// laneDoctorRepo is a repository with its lanes provisioned when init is
// true, and with marker written over .dotf/lanes.json when it is not empty.
func laneDoctorRepo(t *testing.T, init bool, marker string) string {
	t.Helper()
	repo := worktreeTestRepo(t)
	if init {
		if _, err := lane.InitLanes(repo, os.Getenv("WORKTREE_ROOT")); err != nil {
			t.Fatal(err)
		}
		// Init no longer writes the marker; a coherent repository has it.
		if marker == "" {
			marker = coherentMarker
		}
	}
	if marker != "" {
		path := filepath.Join(repo, filepath.FromSlash(lane.LanesMarkerFile))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(marker), 0o644); err != nil {
			t.Fatal(err)
		}
		// A repository this machine runs lanes for carries the activation record.
		common := filepath.Join(repo, ".git") // the main checkout
		if err := os.WriteFile(filepath.Join(common, lane.ActivationFile), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// lane doctor is silent on a coherent repository and says one thing when the
// repository is not lane-managed or its lanes are not initialised here.
func TestLaneDoctor_ExitsZeroWithAtMostOneLine(t *testing.T) {
	cases := []struct {
		name   string
		init   bool
		marker string
		lines  int
		want   string
	}{
		{"coherent repository: no output", true, "", 0, ""},
		{"not lane-managed: one line", false, "", 1, "not lane-managed"},
		{"marker present, lanes not initialised here: one line", false, coherentMarker, 1, "not initialised"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(laneDoctorRepo(t, tc.init, tc.marker))
			stdout, stderr, err := execute(t, "lane", "doctor")
			if err != nil || stderr != "" || strings.Count(stdout, "\n") != tc.lines || !strings.Contains(stdout, tc.want) {
				t.Fatalf("want exit 0, %d line(s) saying %q, got err=%v stdout=%q stderr=%q", tc.lines, tc.want, err, stdout, stderr)
			}
		})
	}
}

// A finding exits 1 with the ERROR/FIX pairs on stdout; cobra's Error: line,
// which would only repeat their count, is silenced.
func TestLaneDoctor_AFindingExitsOneWithoutCobrasErrorLine(t *testing.T) {
	t.Chdir(laneDoctorRepo(t, true, strings.Replace(coherentMarker, `"version":1`, `"version":7`, 1)))
	stdout, stderr, err := execute(t, "lane", "doctor")
	if ExitCode(err) != 1 {
		t.Fatalf("want exit 1, got %d (%v)", ExitCode(err), err)
	}
	if !strings.Contains(stdout, "ERROR: version: is 7, expected 1") || !strings.Contains(stdout, "\nFIX: ") {
		t.Errorf("stdout = %q, want the ERROR/FIX pair", stdout)
	}
	if strings.Contains(stderr, "Error:") {
		t.Errorf("cobra's Error: line repeats the findings: %q", stderr)
	}
}

// A leaseless detached worktree is a warning, not a finding: it names the
// recovery and does not change the exit code, and it is reported whether or not
// the lanes are initialised here (#3436).
func TestLaneDoctor_WarnsOfALeaselessDetachedWorktreeWithoutFailing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		init   bool
		marker string
	}{
		{"lanes initialised", true, ""},
		{"lanes not initialised", false, coherentMarker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := laneDoctorRepo(t, tc.init, tc.marker)
			wt := filepath.Join(t.TempDir(), "scratch")
			if out, err := exec.Command("git", "-C", repo, "worktree", "add", "-q", "--detach", wt, "HEAD").CombinedOutput(); err != nil {
				t.Fatalf("git worktree add: %v\n%s", err, out)
			}
			t.Chdir(repo)
			stdout, stderr, err := execute(t, "lane", "doctor")
			if err != nil || stderr != "" {
				t.Fatalf("a warning must not fail the command: err=%v stderr=%q", err, stderr)
			}
			for _, want := range []string{"WARN: ", "scratch", "dotf worktree adopt"} {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout lacks %q:\n%s", want, stdout)
				}
			}
		})
	}
}

// A deployed guard that lags the repository's copy is a warning with its fix,
// and the exit status stays what it was (AC7 of HARNESS-195).
func TestLaneDoctor_WarnsOfADriftedDeployedGuardWithoutFailing(t *testing.T) {
	repo := laneDoctorRepo(t, true, coherentMarker)
	deploy := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	if out, err := exec.Command("git", "-C", repo, "config", "core.hooksPath", filepath.Join(deploy, "git-hooks")).CombinedOutput(); err != nil {
		t.Fatalf("git config: %v\n%s", err, out)
	}
	for root, content := range map[string]string{repo: "new\n", deploy: "old\n"} {
		p := filepath.Join(root, "git-hooks", filepath.FromSlash(lane.GuardRelPath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repo)
	stdout, stderr, err := execute(t, "lane", "doctor")
	if err != nil || stderr != "" {
		t.Fatalf("a warning must not fail the command: err=%v stderr=%q", err, stderr)
	}
	for _, want := range []string{"WARN: ", "lane-guard.sh", "dotf hooks install"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout)
		}
	}
}

// A failure to audit is printed by runLaneDoctor with its own "lane doctor:"
// prefix, so the errors beneath it must not carry one: the line read
// "lane doctor: lane doctor: reading ...".
func TestLaneDoctor_AFailureToAuditCarriesItsPrefixOnce(t *testing.T) {
	repo := laneDoctorRepo(t, false, coherentMarker)
	// A directory where the marker should be: present, and not readable as one.
	marker := filepath.Join(repo, filepath.FromSlash(lane.LanesMarkerFile))
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(marker, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	_, stderr, err := execute(t, "lane", "doctor")
	if got := ExitCode(err); got != 2 {
		t.Fatalf("exit code = %d, want 2 (the audit could not run): %v", got, err)
	}
	if n := strings.Count(stderr, "lane doctor:"); n != 1 {
		t.Errorf("stderr carries the prefix %d times, want once:\n%s", n, stderr)
	}
}

// Doctor and guard agree on what is managed (#3437): a repository whose working
// tree carries the marker but which this machine never activated is one the guard
// does not enforce, so doctor reports it as such and audits nothing, even a
// marker that would be a finding.
func TestLaneDoctor_AMarkerWithoutTheActivationRecordIsNotManaged(t *testing.T) {
	repo := laneDoctorRepo(t, true, strings.Replace(coherentMarker, `"version":1`, `"version":7`, 1))
	common := filepath.Join(repo, ".git") // the main checkout
	if err := os.Remove(filepath.Join(common, lane.ActivationFile)); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	stdout, stderr, err := execute(t, "lane", "doctor")
	if err != nil || stderr != "" || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("want exit 0 and one line, got err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
	for _, want := range []string{"not lane-managed", "activation record", "dotf converge --only lanes"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q: %q", want, stdout)
		}
	}
}
