package converge

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// recordsEnv is a checkout with a harness tree and one manifest target, and a
// deploy dir that does not exist yet: a fresh machine.
func recordsEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"harness/manifest.json": `{"targets":[{"file":"AGENTS.md"}]}`,
		"AGENTS.md":             "# AGENTS\n",
	})
	return Env{RepoRoot: repo, DeployDir: filepath.Join(t.TempDir(), ".dotfiles"), GOOS: "darwin"}
}

func TestRecordsMirror_PlanWritesNothingApplyConvergesAndRerunIsANoOp(t *testing.T) {
	env := recordsEnv(t)
	r := recordsMirror{}

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 2 {
		t.Errorf("plan on a fresh machine: want 2 files to write, got %d (%s)", plan.Changes, plan.Detail)
	}
	if _, err := os.Stat(env.DeployDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the plan created the deploy dir: %v", err)
	}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %d, err %v", again.Changes, err)
	}
}

func TestRecordsMirror_ProbeFailsWhileTheDeployDirDiffers(t *testing.T) {
	env := recordsEnv(t)

	if err := (recordsMirror{}).Probe(env); err == nil {
		t.Fatal("probe passed on a deploy dir that was never mirrored")
	}
}

// A target the manifest declares but the checkout lacks is never reported as
// converged: Mirror and PlanMirror both return ErrMissingTargets, so the plan,
// the apply and the probe all fail and name the gap.
func TestRecordsMirror_ADeclaredTargetTheCheckoutLacksFails(t *testing.T) {
	env := recordsEnv(t)
	if err := os.Remove(filepath.Join(env.RepoRoot, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	r := recordsMirror{}

	if _, err := r.Reconcile(env, true); err == nil || !strings.Contains(err.Error(), "AGENTS.md") {
		t.Errorf("plan: want an error naming AGENTS.md, got %v", err)
	}
	if _, err := r.Reconcile(env, false); err == nil {
		t.Error("apply reported success with a declared target missing")
	}
	if err := r.Probe(env); err == nil {
		t.Error("probe passed with a declared target missing")
	}
}

// A script the checkout deleted is a pending change until it is gone: the plan
// counts and names it, the apply removes it, and the probe and a second plan
// see nothing left (#2266).
func TestRecordsMirror_PrunesALeftoverTheCheckoutDeleted(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	env := recordsEnv(t)
	writeFixture(t, env.RepoRoot, map[string]string{"scripts/live.sh": "live\n", "scripts/old.sh": "retired\n"})
	for _, args := range [][]string{
		{"init", "-q"}, {"add", "-A"}, {"commit", "-q", "-m", "one"},
		{"rm", "-q", "scripts/old.sh"}, {"commit", "-q", "-m", "two"},
	} {
		cmd := exec.Command("git", append([]string{"-C", env.RepoRoot, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	r := recordsMirror{}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(env.DeployDir, "scripts", "old.sh")
	writeFixture(t, env.DeployDir, map[string]string{"scripts/old.sh": "retired\n"})

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 1 || !strings.Contains(plan.Detail, "scripts/old.sh") {
		t.Errorf("plan: want 1 change naming scripts/old.sh, got %d (%s)", plan.Changes, plan.Detail)
	}
	if err := r.Probe(env); err == nil {
		t.Error("probe passed with a leftover in the deploy dir")
	}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the apply left %s: %v", stale, err)
	}
	if err := r.Probe(env); err != nil {
		t.Errorf("probe after apply: %v", err)
	}
	if again, err := r.Reconcile(env, true); err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %d, err %v", again.Changes, err)
	}
}

// The converge detail carries the same warning as the command: a git failure
// inside a checkout mirrors ignored files, and the report says so.
func TestRecordsMirror_NamesAGitFailureThatMirrorsIgnoredFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	env := recordsEnv(t)
	writeFixture(t, env.RepoRoot, map[string]string{".git": "gitdir: " + filepath.Join(env.RepoRoot, "nowhere") + "\n"})

	res, err := recordsMirror{}.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Detail, "so they were mirrored") {
		t.Errorf("the detail must name the git failure: %s", res.Detail)
	}
}
