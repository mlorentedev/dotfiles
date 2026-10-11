package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hooksGit is a global git config holding only core.hooksPath. ignoreSet makes
// the write succeed and change nothing, a wiring that did not take.
type hooksGit struct {
	hooksPath string
	ignoreSet bool
	writes    int
}

func (g *hooksGit) run(name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	switch {
	case cmd == "git config --global --get core.hooksPath":
		if g.hooksPath == "" {
			return nil, gitExit(1)
		}
		return []byte(g.hooksPath + "\n"), nil
	case strings.HasPrefix(cmd, "git config --global core.hooksPath "):
		g.writes++
		if !g.ignoreSet {
			g.hooksPath = args[len(args)-1]
		}
		return nil, nil
	}
	return nil, errors.New("hooksGit: unexpected " + cmd)
}

// hooksEnv is a checkout carrying a GUARD dispatcher and an empty deploy dir.
func hooksEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"git-hooks/pre-commit":               "#!/usr/bin/env bash\n",
		"git-hooks/lib/memory-sink-guard.sh": "#!/usr/bin/env bash\nexit 0\n",
	})
	return Env{RepoRoot: repo, Home: t.TempDir(), DeployDir: t.TempDir(), GOOS: "linux"}
}

func hooksStep(g *hooksGit) gitHooks {
	return gitHooks{run: g.run, has: func(string) bool { return true }}
}

// From zero: a plan names both changes and makes neither; the apply makes
// both, the probe passes, and the next run has nothing to do.
func TestGitHooks_FromZeroDeploysAndWires(t *testing.T) {
	env, g := hooksEnv(t), &hooksGit{}
	step := hooksStep(g)

	plan, err := step.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || !strings.Contains(plan.Detail, "to apply: ") {
		t.Fatalf("plan = %+v, %v; want 2 changes to apply", plan, err)
	}
	if _, err := os.Stat(filepath.Join(env.DeployDir, "git-hooks")); !os.IsNotExist(err) || g.writes != 0 {
		t.Fatalf("the plan changed the machine: mirror stat %v, %d git writes", err, g.writes)
	}

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 2 {
		t.Fatalf("apply = %+v, %v; want 2 changes", res, err)
	}
	if err := step.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	if again, err := step.Reconcile(env, false); err != nil || again.Changes != 0 {
		t.Errorf("second run = %+v, %v; want converged", again, err)
	}
}

// Someone else's hooksPath is kept and named; only the mirror is a change,
// and the probe does not fail a run over what converge will not take.
func TestGitHooks_KeepsAForeignHooksPath(t *testing.T) {
	env := hooksEnv(t)
	foreign := t.TempDir()
	g := &hooksGit{hooksPath: foreign}
	step := hooksStep(g)

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 1 || !strings.Contains(res.Detail, "left as it is: core.hooksPath is "+foreign) {
		t.Fatalf("apply = %+v, %v; want the mirror alone, the foreign path named", res, err)
	}
	if g.hooksPath != foreign || g.writes != 0 {
		t.Errorf("hooksPath = %q after %d writes; want %q untouched", g.hooksPath, g.writes, foreign)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("probe: %v", err)
	}
}

// A wiring that did not take fails the probe: an unset hooksPath is a guard
// that never runs.
func TestGitHooks_ProbeFailsWhileTheHooksPathIsUnset(t *testing.T) {
	env := hooksEnv(t)
	step := hooksStep(&hooksGit{ignoreSet: true})
	if _, err := step.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), "core.hooksPath is still unset") {
		t.Errorf("probe = %v; want the unset hooksPath named", err)
	}
}

func TestGitHooks_SkipsWithoutGit(t *testing.T) {
	step := gitHooks{run: (&hooksGit{}).run, has: func(string) bool { return false }}
	if res, err := step.Reconcile(hooksEnv(t), false); err != nil || res.Skip != "git is not on PATH" {
		t.Errorf("got %+v, %v; want skipped for git", res, err)
	}
}

// A mirror that does not match the checkout fails the probe, wired or not: a
// stale security hook is trusted, which makes it worse than none.
func TestGitHooks_ProbeFailsOnAStaleMirror(t *testing.T) {
	env := hooksEnv(t)
	step := hooksStep(&hooksGit{hooksPath: filepath.Join(env.DeployDir, "git-hooks")})
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), "still differs from the checkout") {
		t.Errorf("probe = %v; want the stale mirror named", err)
	}
}
