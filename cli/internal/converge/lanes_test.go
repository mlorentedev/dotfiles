package converge

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

func laneGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// laneRepo is a repository whose origin's main carries the marker when
// committed is set, and whose working tree alone carries it when it is not.
func laneRepo(t *testing.T, name string, committed bool) string {
	t.Helper()
	root := t.TempDir()
	origin, repo := filepath.Join(root, "origin.git"), filepath.Join(root, name)
	for _, d := range []string{origin, repo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	laneGit(t, origin, "init", "--bare", "-b", "main")
	laneGit(t, repo, "init", "-b", "main")
	laneGit(t, repo, "remote", "add", "origin", origin)
	writeFixture(t, repo, map[string]string{"README.md": "hello\n"})
	laneGit(t, repo, "add", "README.md")
	laneGit(t, repo, "commit", "-m", "initial")
	laneGit(t, repo, "push", "-u", "origin", "main")
	if _, err := lane.WriteLanesMarker(repo); err != nil {
		t.Fatal(err)
	}
	if committed {
		laneGit(t, repo, "add", lane.LanesMarkerFile)
		laneGit(t, repo, "-c", "core.hooksPath="+t.TempDir(), "commit", "-m", "declare lanes")
		laneGit(t, repo, "push", "origin", "main")
	}
	return repo
}

func lanesStep(wt string) lanes {
	return lanes{
		resolve: func(k string) string {
			if k == "WORKTREE_ROOT" {
				return wt
			}
			return ""
		},
		has: func(string) bool { return true },
	}
}

func recordOf(t *testing.T, repo string) string {
	t.Helper()
	return filepath.Join(laneGit(t, repo, "rev-parse", "--path-format=absolute", "--git-common-dir"), lane.ActivationFile)
}

func hasRecord(t *testing.T, repo string) bool {
	t.Helper()
	_, err := os.Stat(recordOf(t, repo))
	return err == nil
}

// From zero: a plan names the change and makes none; the apply makes it, the
// probe passes, and the next run reports nothing to do.
func TestLanes_ActivatesTheCommittedMarkerOnce(t *testing.T) {
	repo, wt := laneRepo(t, "dotfiles", true), t.TempDir()
	env, step := Env{RepoRoot: repo, DeployDir: t.TempDir(), GOOS: runtime.GOOS}, lanesStep(wt)

	plan, err := step.Reconcile(env, true)
	if err != nil || plan.Changes == 0 || hasRecord(t, repo) {
		t.Fatalf("plan = %+v, %v, record=%v; want changes and no record", plan, err, hasRecord(t, repo))
	}
	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != plan.Changes || !hasRecord(t, repo) {
		t.Fatalf("apply = %+v, %v; want %d changes and a record", res, err, plan.Changes)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
	if again, err := step.Reconcile(env, false); err != nil || again.Changes != 0 {
		t.Errorf("second run = %+v, %v; want changes=0", again, err)
	}
}

// A marker that exists only in a working tree activates nothing, and says why.
func TestLanes_WorkingTreeMarkerActivatesNothing(t *testing.T) {
	repo, wt := laneRepo(t, "dotfiles", false), t.TempDir()
	env, step := Env{RepoRoot: repo, DeployDir: t.TempDir(), GOOS: runtime.GOOS}, lanesStep(wt)

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 0 || hasRecord(t, repo) {
		t.Fatalf("Reconcile = %+v, %v, record=%v; want no change, no record", res, err, hasRecord(t, repo))
	}
	if !strings.Contains(res.Detail, lane.LanesMarkerFile) {
		t.Errorf("detail %q does not name %s", res.Detail, lane.LanesMarkerFile)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe of a repository that does not run lanes: %v", err)
	}
}

// The step acts on RepoRoot and on nothing else: DeployDir is never read, and
// never activated, even when it is a repository that carries the marker.
func TestLanes_NeverActivatesTheDeployDir(t *testing.T) {
	deploy := laneRepo(t, "deployed", true)
	repo, wt := laneRepo(t, "dotfiles", false), t.TempDir()
	env, step := Env{RepoRoot: repo, DeployDir: deploy, GOOS: runtime.GOOS}, lanesStep(wt)

	if _, err := step.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if hasRecord(t, deploy) {
		t.Errorf("the deploy dir was activated")
	}
	if entries, _ := os.ReadDir(wt); len(entries) != 0 {
		t.Errorf("lanes were created for a repository that does not run them: %v", entries)
	}
}

// RepoRoot and DeployDir resolving to one physical path is refused, however
// they are spelled: through a symlink, and from a linked worktree of the
// deploy clone.
func TestLanes_RefusesWhenRepoRootIsTheDeployDir(t *testing.T) {
	repo := laneRepo(t, "dotfiles", true)
	link := filepath.Join(t.TempDir(), "deploy-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	laneGit(t, repo, "worktree", "add", "--detach", linked)

	for name, env := range map[string]Env{
		"same path":       {RepoRoot: repo, DeployDir: repo},
		"symlink":         {RepoRoot: repo, DeployDir: link},
		"linked worktree": {RepoRoot: linked, DeployDir: repo},
	} {
		t.Run(name, func(t *testing.T) {
			env.GOOS = runtime.GOOS
			step := lanesStep(t.TempDir())
			res, err := step.Reconcile(env, false)
			if err != nil || res.Changes != 0 || res.Skip == "" {
				t.Fatalf("Reconcile = %+v, %v; want a skip that says why", res, err)
			}
			if hasRecord(t, repo) {
				t.Errorf("the deploy clone was activated")
			}
		})
	}
}

func TestLanes_SkipsWithoutAWorktreeRoot(t *testing.T) {
	repo := laneRepo(t, "dotfiles", true)
	step := lanesStep("")
	res, err := step.Reconcile(Env{RepoRoot: repo, DeployDir: t.TempDir(), GOOS: runtime.GOOS}, false)
	if err != nil || res.Skip == "" || hasRecord(t, repo) {
		t.Errorf("Reconcile = %+v, %v, record=%v; want a skip", res, err, hasRecord(t, repo))
	}
}

// The probe holds the post-condition: a repository that runs lanes and has
// none of them is not converged.
func TestLanes_ProbeFailsWhileActivationIsMissing(t *testing.T) {
	repo := laneRepo(t, "dotfiles", true)
	env := Env{RepoRoot: repo, DeployDir: t.TempDir(), GOOS: runtime.GOOS}
	if err := lanesStep(t.TempDir()).Probe(env); err == nil {
		t.Errorf("probe passed with the lanes and the record missing")
	}
}

func TestRegistry_HasTheLanesStep(t *testing.T) {
	for _, r := range Registry(Options{}) {
		if r.Name() == "lanes" {
			return
		}
	}
	t.Errorf("no lanes step in the registry")
}

// A lane path something else occupies is one machine's collision: the step
// reports it, with the repair, as a skip, and the steps after it still run.
func TestLanes_SkipsWithTheRepairWhenALanePathIsOccupied(t *testing.T) {
	repo, wt := laneRepo(t, "dotfiles", true), t.TempDir()
	if err := os.MkdirAll(filepath.Join(wt, "dotfiles", lane.LaneFastTrack), 0o755); err != nil {
		t.Fatal(err)
	}
	env, step := Env{RepoRoot: repo, DeployDir: t.TempDir(), GOOS: runtime.GOOS}, lanesStep(wt)

	for _, dry := range []bool{true, false} {
		res, err := step.Reconcile(env, dry)
		if err != nil || res.Skip == "" || !strings.Contains(res.Skip, "move it aside") {
			t.Fatalf("dryRun=%v: Reconcile = %+v, %v; want a skip that carries the repair", dry, res, err)
		}
	}
	if hasRecord(t, repo) {
		t.Errorf("the repository was activated despite the occupied lane path")
	}
}
