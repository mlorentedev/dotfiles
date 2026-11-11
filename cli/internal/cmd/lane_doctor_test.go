package cmd

import (
	"os"
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
	}
	if marker != "" {
		path := filepath.Join(repo, filepath.FromSlash(lane.LanesMarkerFile))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(marker), 0o644); err != nil {
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
