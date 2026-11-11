package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// lane doctor is silent on a coherent repository, says one thing when the
// repository is not lane-managed or its lanes are not initialised here, and
// prints ERROR/FIX pairs with exit 1 on a finding, never cobra's Error: line.
func TestLaneDoctor_ExitContract(t *testing.T) {
	t.Run("coherent repository: exit 0, no output", func(t *testing.T) {
		repo := worktreeTestRepo(t)
		if _, err := lane.InitLanes(repo, os.Getenv("WORKTREE_ROOT")); err != nil {
			t.Fatal(err)
		}
		t.Chdir(repo)
		stdout, stderr, err := execute(t, "lane", "doctor")
		if err != nil || stdout != "" || stderr != "" {
			t.Fatalf("want silence and exit 0, got err=%v stdout=%q stderr=%q", err, stdout, stderr)
		}
	})
	t.Run("not lane-managed: exit 0, one line", func(t *testing.T) {
		repo := worktreeTestRepo(t)
		t.Chdir(repo)
		stdout, _, err := execute(t, "lane", "doctor")
		if err != nil || strings.Count(stdout, "\n") != 1 || !strings.Contains(stdout, "not lane-managed") {
			t.Fatalf("want one line and exit 0, got err=%v stdout=%q", err, stdout)
		}
	})
	t.Run("a finding: exit 1, ERROR/FIX on stdout, no Error: line", func(t *testing.T) {
		repo := worktreeTestRepo(t)
		if _, err := lane.InitLanes(repo, os.Getenv("WORKTREE_ROOT")); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(repo, filepath.FromSlash(lane.LanesMarkerFile))
		if err := os.WriteFile(marker, []byte(`{"version":7,"canonical_lanes":["feat-lane-primary","feat-lane-secondary","fast-track-lane"],"third_party_roots":["x"]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(repo)
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
	})
}
