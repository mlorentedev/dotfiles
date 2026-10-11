package converge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
)

// envGenerateEnv is a checkout with a one-variable contract and a deploy dir
// that already exists, as records-mirror leaves it.
func envGenerateEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"env-contract.json": `{"env_vars":[{"name":"VAULT_PATH","default":{"linux":"$HOME/vault"}}]}`,
	})
	e := Env{RepoRoot: repo, Home: t.TempDir(), DeployDir: filepath.Join(t.TempDir(), ".dotfiles"), GOOS: "darwin"}
	if err := os.MkdirAll(e.DeployDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestEnvGenerate_PlanWritesNothingApplyConvergesAndRerunIsANoOp(t *testing.T) {
	e := envGenerateEnv(t)
	r := envGenerate{}
	out := env.DefaultOutput(e.GOOS, e.DeployDir)

	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes != 1 {
		t.Fatalf("plan = %+v, %v; want 1 change", plan, err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("a plan wrote %s", out)
	}

	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 {
		t.Fatalf("apply = %+v, %v; want 1 change", res, err)
	}
	if err := r.Probe(e); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	// The renderer substitutes $HOME textually and keeps the contract's "/".
	if b, _ := os.ReadFile(out); !strings.Contains(string(b), e.Home+"/vault") {
		t.Fatalf("%s does not carry the contract's VAULT_PATH:\n%s", out, b)
	}

	if again, err := r.Reconcile(e, false); err != nil || again.Changes != 0 {
		t.Fatalf("rerun = %+v, %v; want 0 changes", again, err)
	}
}

// TestEnvGenerate_RepairsAStaleFile is the gap this step closes: the rc files
// render the path file only when it is missing, so a file left behind by an
// older contract stayed stale until someone ran `dotf env generate` by hand.
func TestEnvGenerate_RepairsAStaleFile(t *testing.T) {
	e := envGenerateEnv(t)
	out := env.DefaultOutput(e.GOOS, e.DeployDir)
	if err := os.WriteFile(out, []byte("export VAULT_PATH=/old/vault\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := envGenerate{}
	if err := r.Probe(e); err == nil {
		t.Fatal("probe passed on a stale file")
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 {
		t.Fatalf("apply = %+v, %v; want 1 change", res, err)
	}
	if err := r.Probe(e); err != nil {
		t.Fatalf("probe after repair: %v", err)
	}
}

func TestEnvGenerate_MissingContractFails(t *testing.T) {
	e := envGenerateEnv(t)
	if err := os.Remove(filepath.Join(e.RepoRoot, "env-contract.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := (envGenerate{}).Reconcile(e, true); err == nil {
		t.Fatal("a checkout with no env contract converged")
	}
}
