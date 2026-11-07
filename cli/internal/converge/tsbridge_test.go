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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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

func TestTsBridgeClientWithoutRoleRefusesForeignIncludedDropIn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	home := t.TempDir()
	path := tsBridgeGitConfig(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const foreign = "[http]\n\tproxy = http://foreign\n"
	if err := os.WriteFile(path, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := (tsBridgeClient{}).Reconcile(Env{Home: home}, false)
	if err == nil || !strings.Contains(err.Error(), "Git includes this path regardless of role state") {
		t.Fatalf("role-absent foreign drop-in error = %v", err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != foreign {
		t.Fatal("foreign drop-in was changed")
	}
}

func TestTsBridgeClientProbeByState(t *testing.T) {
	cases := []struct {
		name, file    string
		role, wantErr bool
	}{
		{name: "opted out and missing", file: "missing"},
		{name: "opted in and missing", file: "missing", role: true, wantErr: true},
		{name: "opted in and current", file: "current", role: true},
		{name: "opted out and managed", file: "current", wantErr: true},
		{name: "foreign file", file: "foreign", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			home := t.TempDir()
			if tc.role {
				if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleTSBridgeClient, true); err != nil {
					t.Fatal(err)
				}
			}
			if tc.file != "missing" {
				path := tsBridgeGitConfig(home)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				body := tsBridgeGitContent
				if tc.file == "foreign" {
					body = []byte("[http]\n\tproxy = http://foreign\n")
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := (tsBridgeClient{}).Probe(Env{Home: home}); (err != nil) != tc.wantErr {
				t.Fatalf("Probe() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDotfilesGitConfigIncludesOptionalTsBridgeDropIn(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(repo, "git", "dotfiles.gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "path = ts-bridge.gitconfig") {
		t.Fatal("dotfiles.gitconfig does not include the role-owned drop-in")
	}
}

func TestDotfilesGitConfigResolvesTsBridgeDropInBesideParent(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	dir := t.TempDir()
	parent := filepath.Join(dir, "dotfiles.gitconfig")
	raw, err := os.ReadFile(filepath.Join(repo, "git", "dotfiles.gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ts-bridge.gitconfig"), tsBridgeGitContent, 0o600); err != nil {
		t.Fatal(err)
	}
	otherHome := t.TempDir()
	cmd := exec.Command("git", "config", "--file", parent, "--includes",
		"--get-urlmatch", "http.proxy", "https://gitea.kubelab.live")
	cmd.Env = append(os.Environ(), "HOME="+otherHome, "USERPROFILE="+otherHome)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "socks5h://127.0.0.1:1081" {
		t.Fatalf("relative bridge include = %q, %v", out, err)
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
