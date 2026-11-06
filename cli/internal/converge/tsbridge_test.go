package converge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

func TestTsBridgeClientRoleCreatesAndRetiresGitDropIn(t *testing.T) {
	home := t.TempDir()
	e := Env{Home: home}
	r := tsBridgeClient{}
	path := tsBridgeGitConfig(home)

	if res, err := r.Reconcile(e, false); err != nil || res.Skip == "" {
		t.Fatalf("opted out = %+v, %v", res, err)
	}
	if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleTSBridgeClient, true); err != nil {
		t.Fatal(err)
	}
	plan, err := r.Reconcile(e, true)
	if err != nil || plan.Changes != 1 {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("plan wrote the drop-in")
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 {
		t.Fatalf("apply = %+v, %v", res, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `[http "https://gitea.kubelab.live"]`) ||
		!strings.Contains(text, "socks5h://127.0.0.1:1081") ||
		strings.Contains(text, "github.com") {
		t.Fatalf("drop-in = %s", raw)
	}
	if err := os.WriteFile(path, []byte(tsBridgeManagedHeader+"[http]\n\tproxy = wrong\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 {
		t.Fatalf("managed drift repair = %+v, %v", res, err)
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 0 {
		t.Fatalf("second apply = %+v, %v", res, err)
	}

	if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleTSBridgeClient, false); err != nil {
		t.Fatal(err)
	}
	if res, err := r.Reconcile(e, false); err != nil || res.Changes != 1 {
		t.Fatalf("retire = %+v, %v", res, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("role removal kept the drop-in")
	}
}

func TestTsBridgeClientRefusesToOverwriteForeignDropIn(t *testing.T) {
	home := t.TempDir()
	if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleTSBridgeClient, true); err != nil {
		t.Fatal(err)
	}
	path := tsBridgeGitConfig(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const foreign = "[http]\n\tproxy = http://foreign\n"
	if err := os.WriteFile(path, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := (tsBridgeClient{}).Reconcile(Env{Home: home}, false); err == nil {
		t.Fatal("foreign drop-in was accepted")
	}
	if raw, _ := os.ReadFile(path); string(raw) != foreign {
		t.Fatal("foreign drop-in was overwritten")
	}
}

func TestDotfilesGitConfigIncludesOptionalTsBridgeDropIn(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(repo, "git", "dotfiles.gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "path = ~/.config/git/ts-bridge.gitconfig") {
		t.Fatal("dotfiles.gitconfig does not include the role-owned drop-in")
	}
}

func TestDotfilesGitConfigWorksWhenOptionalTsBridgeDropInIsMissing(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	home := t.TempDir()
	cmd := exec.Command("git", "config", "--file", filepath.Join(repo, "git", "dotfiles.gitconfig"),
		"--includes", "--get", "user.name")
	cmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "Manu" {
		t.Fatalf("git config with missing optional include = %q, %v", out, err)
	}
}
