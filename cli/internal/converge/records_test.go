package converge

import (
	"errors"
	"os"
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
