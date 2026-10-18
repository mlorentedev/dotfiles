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

// launchdUnknown is `launchctl print`'s failure for a service launchd does not
// know: exit 113, the one failure that reads as "not loaded" (#2731).
type launchdUnknown struct{}

func (launchdUnknown) Error() string { return "exit status 113" }
func (launchdUnknown) ExitCode() int { return 113 }

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
		root: root,
		run: func(argv ...string) ([]byte, error) {
			if strings.Join(argv, " ") == "pmset -g custom" {
				return []byte("AC Power:\n sleep 0\n disksleep 10\n womp 1\n autorestart 0\n powernap 1\n"), nil
			}
			return []byte("Could not find service"), launchdUnknown{}
		},
		addrs: func() ([]net.IP, error) { return []net.IP{net.ParseIP("100.64.0.15")}, nil },
		paths: func(string, string) sshtarget.Paths { return paths },
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
	if res.Changes != 6 || !strings.HasPrefix(res.Detail, "to apply as root: firewall, authorized key, host keys, sshd drop-in, sshd service, power policy") {
		t.Fatalf("plan = %+v", res)
	}
	if !strings.Contains(res.Detail, "only once someone logs in") {
		t.Errorf("the plan does not say the mesh needs a login after a reboot: %q", res.Detail)
	}
}

func TestWorkstationTarget_ApplyWithoutRootIsSkippedWithThePlan(t *testing.T) {
	for why, ready := range map[string]error{
		"no terminal":              sshtarget.ErrNoTerminal,  // sudo with nothing to ask on
		"administrator rights are": sshtarget.ErrNotElevated, // Windows, not elevated
	} {
		r, e := workstationFixture(t, noRoot{t: t, ready: ready})
		if _, err := env.SetMachineRole(env.MachinePath(e.Home), env.RoleSSHTarget, true); err != nil {
			t.Fatal(err)
		}
		res, err := r.Reconcile(e, false)
		if err != nil {
			t.Fatalf("%s: %v", why, err)
		}
		if !strings.Contains(res.Skip, why) || !strings.Contains(res.Skip, "to apply as root") {
			t.Fatalf("Reconcile = %+v; want a skip naming %q and carrying the plan", res, why)
		}
	}
}

// A failed apply names what it planned, never a proof that did not run nor
// changes it kept (#2765: the guest's detail claimed both).
func TestWorkstationTarget_AFailedApplyClaimsNothingItDidNotDo(t *testing.T) {
	r, e := workstationFixture(t, noRoot{t: t, ready: errors.New("sudo: 3 incorrect password attempts")})
	if _, err := env.SetMachineRole(env.MachinePath(e.Home), env.RoleSSHTarget, true); err != nil {
		t.Fatal(err)
	}
	res, err := r.Reconcile(e, false)
	if err == nil {
		t.Fatal("Reconcile succeeded with a root that refused")
	}
	// The runner prints the detail of a step that kept changes, then the
	// error; Apply undid everything, so neither may claim work or a proof.
	line := res.Detail + "; " + err.Error()
	if res.Changes != 0 || strings.Contains(line, "applied") || strings.Contains(line, "accepts the key") {
		t.Fatalf("a failed apply claimed work or a proof: %d changes, %q", res.Changes, line)
	}
	if !strings.Contains(err.Error(), "planned as root: firewall, authorized key") || !strings.Contains(err.Error(), "incorrect password") {
		t.Fatalf("a failed apply does not name what it planned and what failed: %v", err)
	}
}

func TestWorkstationTarget_IsSkippedWithoutRunners(t *testing.T) {
	res, err := workstationTarget{}.Reconcile(Env{GOOS: "darwin"}, true)
	if err != nil || res.Skip == "" {
		t.Fatalf("Reconcile = %+v, %v; want skipped", res, err)
	}
}

func TestWorkstationTarget_ANodeOffTheMeshIsSkipped(t *testing.T) {
	r, e := workstationFixture(t, noRoot{t: t})
	if _, err := env.SetMachineRole(env.MachinePath(e.Home), env.RoleSSHTarget, true); err != nil {
		t.Fatal(err)
	}
	r.addrs = func() ([]net.IP, error) { return []net.IP{net.ParseIP("10.0.0.130")}, nil }
	res, err := r.Reconcile(e, false)
	if err != nil || !strings.Contains(res.Skip, "join the mesh") {
		t.Fatalf("Reconcile = %+v, %v; want a skip naming the mesh", res, err)
	}
}

// retiringFixture is a machine whose role was removed and whose record the
// step left: every entry unknown, so a run removes the record alone.
func retiringFixture(t *testing.T, root sshtarget.Root) (workstationTarget, Env, string) {
	t.Helper()
	r, e := workstationFixture(t, root)
	paths := r.paths("darwin", e.Home)
	paths.Record = filepath.Join(t.TempDir(), "workstation-target.json")
	r.paths = func(string, string) sshtarget.Paths { return paths }
	writeFile(t, paths.Record, `{"version":1,"authorized_key":{"state":"unknown"},"sshd_drop_in":{"state":"unknown"},"sshd_listener":{"state":"unknown"}}`)
	return r, e, paths.Record
}

func TestWorkstationTarget_PlansTheRetirementWithoutRoot(t *testing.T) {
	r, e, _ := retiringFixture(t, noRoot{t: t})
	res, err := r.Reconcile(e, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 1 || !strings.HasPrefix(res.Detail, "to retire as root: state record") || !strings.Contains(res.Detail, "sshd service: left") {
		t.Fatalf("plan = %+v", res)
	}
}

func TestWorkstationTarget_RetirementWithoutRootIsSkippedWithThePlan(t *testing.T) {
	r, e, _ := retiringFixture(t, noRoot{t: t, ready: sshtarget.ErrNoTerminal})
	res, err := r.Reconcile(e, false)
	if err != nil || !strings.Contains(res.Skip, "to retire as root: state record") {
		t.Fatalf("Reconcile = %+v, %v; want a skip carrying the retirement plan", res, err)
	}
}

// Probe holds the retirement to what it planned: a record still there with
// work pending fails it, none passes.
func TestWorkstationTarget_ProbeChecksTheRetirement(t *testing.T) {
	r, e, record := retiringFixture(t, noRoot{t: t})
	if err := r.Probe(e); err == nil || !strings.Contains(err.Error(), "state record") {
		t.Errorf("Probe = %v, want the pending record named", err)
	}
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(e); err != nil {
		t.Errorf("Probe after the record went = %v", err)
	}
}
