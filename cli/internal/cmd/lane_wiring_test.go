package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// laneWiringFixture is a real repository with a merged, expired-lease worktree
// whose pi session wrote a moment ago (live), and a second one with no session
// (gone). The forge calls both merged. It skips where the fixture cannot mean
// what it says: pi keys a session by a POSIX path, and sweep reaps nothing on a
// platform with no process scan.
func laneWiringFixture(t *testing.T) (repo, live, gone string, forge stubForge) {
	t.Helper()
	repo = worktreeTestRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git", "info", "exclude"), []byte(worktree.MetadataFileName+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, name := range []string{"live", "gone"} {
		wt := filepath.Join(root, name)
		c := exec.Command("git", "worktree", "add", "-q", "-b", "fix/"+name, wt)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git worktree add: %v\n%s", err, out)
		}
		if err := worktree.SaveMetadata(wt, worktree.Metadata{
			Issue: 7 + i, CreatedAt: now.Add(-72 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour), ReapOK: true,
		}); err != nil {
			t.Fatal(err)
		}
		if name == "live" {
			live = wt
		} else {
			gone = wt
		}
	}
	if !strings.HasPrefix(live, "/") {
		t.Skip("pi keys a session by a POSIX path")
	}
	if _, scans := worktree.ProcessInside(gone); !scans {
		t.Skip("no process scan on this platform: sweep reaps nothing")
	}
	key := "--" + strings.ReplaceAll(strings.TrimPrefix(live, "/"), "/", "-") + "--"
	dir := filepath.Join(home, ".pi", "agent", "sessions", key)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	merged := lane.PRInfo{Number: 1, State: lane.PRMerged}
	return repo, live, gone, stubForge{"fix/live": merged, "fix/gone": merged}
}

// realLaneVerbs is the only place the sweep gets its oracle and its session
// stores. The oracle test builds its own runner, so it cannot see either being
// dropped here (CLI-127 review round 2, F1): a missing forge panics, and
// missing session stores reap a worktree whose session can still be resumed.
func TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions(t *testing.T) {
	_, live, gone, forge := laneWiringFixture(t)
	verbs := realLaneVerbs(&cobra.Command{}, lane.Sources{Now: time.Now(), Forge: forge})
	reaped, err := verbs.Sweep()
	if err != nil {
		t.Fatal(err)
	}
	if reaped != 1 {
		t.Errorf("reaped=%d, want 1: only the worktree with no session", reaped)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("sweep removed a worktree whose session can be resumed: %v", err)
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Errorf("sweep left the merged worktree with no session (stat err: %v)", err)
	}
}

// The command a ambiguous row prints is the row's whole promise, so it runs
// here, through the real done, and the worktree it names must be gone after
// (CLI-127 review round 2, F2). done refuses a recent session whatever --force
// says; before the fix the printed command exited non-zero and removed nothing.
func TestPlanResolverForALiveWorktreeRunsThroughDone(t *testing.T) {
	_, live, _, _ := laneWiringFixture(t)
	r := lane.Row{Branch: "fix/live", Path: live, Phase: lane.PhaseLanded, Live: lane.LiveYes}
	acts := lane.Plan([]lane.Row{r}, nil, time.Now())
	if len(acts) != 1 || acts[0].Resolve == "" {
		t.Fatalf("plan = %+v, want one action carrying a command", acts)
	}
	fields := strings.Fields(acts[0].Resolve)
	if len(fields) < 4 || fields[0] != "dotf" || fields[1] != "worktree" || fields[2] != "done" {
		t.Fatalf("command = %q, want a dotf worktree done", acts[0].Resolve)
	}
	if out, err := executeWorktree(t, fields[2:]...); err != nil {
		t.Fatalf("%s: %v\n%s", acts[0].Resolve, err, out)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Errorf("the printed command left the worktree (stat err: %v)", err)
	}
}
