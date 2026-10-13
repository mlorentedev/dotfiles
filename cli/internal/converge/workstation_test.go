package converge

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
)

// noRoot is a Root that must not be reached, or that has no terminal.
type noRoot struct {
	t     *testing.T
	ready error
}

func (r noRoot) Ready() error {
	if r.ready == nil {
		r.t.Fatal("asked for root")
	}
	return r.ready
}

func (r noRoot) Run(argv ...string) ([]byte, error) {
	r.t.Fatalf("ran as root: %v", argv)
	return nil, nil
}

func workstationFixture(t *testing.T, root sshtarget.Root) (workstationTarget, Env) {
	t.Helper()
	repo, sys := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeFile(t, filepath.Join(repo, "ssh", "id_ed25519.pub"), "ssh-ed25519 AAAAOwner manu@msi\n")
	writeFile(t, filepath.Join(repo, "ssh", "sshd-workstation.conf"), "AllowUsers {user}@{mesh}\n")
	paths := sshtarget.Paths{
		SSHDConfig:     filepath.Join(sys, "sshd_config"),
		DropIn:         filepath.Join(sys, "sshd_config.d", "01-workstation.conf"),
		HostKey:        filepath.Join(sys, "ssh_host_ed25519_key.pub"),
		AuthorizedKeys: filepath.Join(sys, "authorized_keys"),
		ClientKey:      filepath.Join(sys, "id_ed25519"),
	}
	writeFile(t, paths.SSHDConfig, "Include "+filepath.Dir(paths.DropIn)+"/*\n")
	r := workstationTarget{
		root:  root,
		run:   func(...string) ([]byte, error) { return nil, errors.New("exit status 113") },
		addrs: func() ([]net.IP, error) { return []net.IP{net.ParseIP("100.64.0.15")}, nil },
		paths: func(string) sshtarget.Paths { return paths },
		user:  func() (string, error) { return "manu", nil },
	}
	return r, Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkstationTarget_SkipsAMachineThatHasNotOptedIn(t *testing.T) {
	r, e := workstationFixture(t, noRoot{t: t})
	res, err := r.Reconcile(e, false)
	if err != nil || !strings.Contains(res.Skip, "dotf env role add ssh-target") {
		t.Fatalf("Reconcile = %+v, %v; want a skip naming the opt-in", res, err)
	}
}

func TestWorkstationTarget_PlanNeedsNoRoot(t *testing.T) {
	r, e := workstationFixture(t, noRoot{t: t})
	if _, err := env.SetMachineRole(env.MachinePath(e.Home), env.RoleSSHTarget, true); err != nil {
		t.Fatal(err)
	}
	res, err := r.Reconcile(e, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 4 || !strings.HasPrefix(res.Detail, "to apply as root: authorized key, host keys, sshd drop-in, sshd service") {
		t.Fatalf("plan = %+v", res)
	}
}

func TestWorkstationTarget_ApplyWithoutATerminalIsSkippedWithThePlan(t *testing.T) {
	r, e := workstationFixture(t, noRoot{t: t, ready: sshtarget.ErrNoTerminal})
	if _, err := env.SetMachineRole(env.MachinePath(e.Home), env.RoleSSHTarget, true); err != nil {
		t.Fatal(err)
	}
	res, err := r.Reconcile(e, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Skip, "no terminal") || !strings.Contains(res.Skip, "to apply as root") {
		t.Fatalf("Reconcile = %+v; want a skip carrying the plan", res)
	}
}

func TestWorkstationTarget_IsSkippedWithoutRunners(t *testing.T) {
	res, err := workstationTarget{}.Reconcile(Env{GOOS: "darwin"}, true)
	if err != nil || res.Skip == "" {
		t.Fatalf("Reconcile = %+v, %v; want skipped", res, err)
	}
}
