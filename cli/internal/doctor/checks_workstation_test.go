package doctor

import (
	"bytes"
	"errors"
	"net"
	"path/filepath"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/sshtarget"
)

const workstationTemplate = "AllowUsers {user}@{mesh}\nPasswordAuthentication no\n"

// The doctor check (#2356, PLAT-001b PR 7d) runs the converge step's plan with
// the step's own gates, one row per branch: neither darwin nor linux → no section; not opted
// in → SKIP; off the mesh → SKIP; a config that shadows the drop-in → WARN with
// Inspect's own message; anything pending → WARN naming the converge command;
// converged → PASS.
func TestCheckWorkstationTarget_ByStatus(t *testing.T) {
	mesh := []net.IP{net.ParseIP("100.64.0.15"), net.ParseIP("10.0.0.130")}
	cases := []struct {
		name    string
		goos    string
		optIn   bool
		addrs   []net.IP
		state   func(t *testing.T, p sshtarget.Paths, user string)
		want    Status
		needle  string
		section bool
	}{
		{name: "windows → no section", goos: "windows", optIn: true, addrs: mesh, section: false},
		{name: "linux, nothing applied → WARN naming the converge command", goos: "linux", optIn: true, addrs: mesh, want: StatusWarn, needle: "dotf converge --only workstation-target", section: true},
		{name: "linux, converged → PASS", goos: "linux", optIn: true, addrs: mesh, want: StatusPass, needle: "as declared for the mesh address 100.64.0.15", section: true,
			state: func(t *testing.T, p sshtarget.Paths, user string) {
				writeFile(t, p.AuthorizedKeys, "ssh-ed25519 AAAAOwner manu@msi\n")
				writeFile(t, p.HostKey, "ssh-ed25519 AAAAHost root@msi\n")
				writeFile(t, p.LogindDropIn, sshtarget.LogindPolicy)
				writeFile(t, p.DropIn, "AllowUsers "+user+"@"+sshtarget.MeshCIDR+"\nPasswordAuthentication no\n")
			}},
		{name: "linux without ufw → PASS that names the gap", goos: "linux", optIn: true, addrs: mesh, want: StatusPass, needle: "no ufw on this machine", section: true,
			state: func(t *testing.T, p sshtarget.Paths, user string) {
				writeConverged(t, p, user)
			}},
		{name: "linux, ufw installed and off → WARN naming the firewall", goos: "linux", optIn: true, addrs: mesh, want: StatusWarn, needle: "differs from the declared state: firewall", section: true,
			state: func(t *testing.T, p sshtarget.Paths, user string) {
				writeConverged(t, p, user)
				writeFile(t, p.UFWConf, "ENABLED=no\n")
				writeFile(t, p.UFWDefaults, "DEFAULT_INPUT_POLICY=\"DROP\"\n")
				writeFile(t, p.UFWRules, "")
			}},
		{name: "not opted in → SKIP", goos: "darwin", addrs: mesh, want: StatusSkip, needle: "dotf env role add ssh-target", section: true},
		{name: "off the mesh → SKIP", goos: "darwin", optIn: true, addrs: []net.IP{net.ParseIP("10.0.0.130")}, want: StatusSkip, needle: "join the mesh", section: true},
		{name: "nothing applied → WARN naming the converge command", goos: "darwin", optIn: true, addrs: mesh, want: StatusWarn, needle: "dotf converge --only workstation-target", section: true},
		{name: "a directive read before the drop-in → WARN", goos: "darwin", optIn: true, addrs: mesh, want: StatusWarn, needle: "sets PasswordAuthentication before the drop-in", section: true,
			state: func(t *testing.T, p sshtarget.Paths, _ string) {
				writeFile(t, p.SSHDConfig, "PasswordAuthentication yes\nInclude "+filepath.Dir(p.DropIn)+"/*\n")
			}},
		{name: "converged → PASS", goos: "darwin", optIn: true, addrs: mesh, want: StatusPass, needle: "as declared for the mesh address 100.64.0.15", section: true,
			state: func(t *testing.T, p sshtarget.Paths, user string) {
				writeFile(t, p.AuthorizedKeys, "ssh-ed25519 AAAAOwner manu@msi\n")
				writeFile(t, p.HostKey, "ssh-ed25519 AAAAHost root@macmini\n")
				writeFile(t, p.DropIn, "AllowUsers "+user+"@"+sshtarget.MeshCIDR+"\nPasswordAuthentication no\n")
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, repo, sys := t.TempDir(), t.TempDir(), t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
			writeFile(t, filepath.Join(repo, sshtarget.KeyFile), "ssh-ed25519 AAAAOwner manu@msi\n")
			writeFile(t, filepath.Join(repo, sshtarget.TemplateFile), workstationTemplate)
			paths := sshtarget.Paths{
				SSHDConfig:      filepath.Join(sys, "sshd_config"),
				DropIn:          filepath.Join(sys, "sshd_config.d", "01-workstation.conf"),
				HostKey:         filepath.Join(sys, "ssh_host_ed25519_key.pub"),
				AuthorizedKeys:  filepath.Join(home, ".ssh", "authorized_keys"),
				ClientKey:       filepath.Join(home, ".ssh", "id_ed25519"),
				LogindDropIn:    filepath.Join(sys, "logind.conf.d", "50-workstation.conf"),
				UFWConf:         filepath.Join(sys, "ufw", "ufw.conf"),
				UFWDefaults:     filepath.Join(sys, "default", "ufw"),
				UFWRules:        filepath.Join(sys, "ufw", "user.rules"),
				UFWRules6:       filepath.Join(sys, "ufw", "user6.rules"),
				FirewallWitness: filepath.Join(sys, "dotf", "workstation-firewall"),
			}
			writeFile(t, paths.SSHDConfig, "Include "+filepath.Dir(paths.DropIn)+"/*\n")
			loaded := false
			if tc.state != nil {
				u, err := sshtarget.CurrentUser()
				if err != nil {
					t.Fatal(err)
				}
				tc.state(t, paths, u)
				loaded = tc.want == StatusPass
			}
			if tc.optIn {
				if _, err := envpkg.SetMachineRole(envpkg.MachinePath(home), envpkg.RoleSSHTarget, true); err != nil {
					t.Fatal(err)
				}
			}
			s := newSys(map[string]string{"HOME": home, "DOTFILES_REPO_DIR": repo}, nil, nil)
			s.GOOS = tc.goos
			s.SSHTargetPaths = func(string) sshtarget.Paths { return paths }
			s.MeshAddrs = func() ([]net.IP, error) { return tc.addrs, nil }
			s.CommandOutput = func(name string, args ...string) (string, error) {
				switch {
				case name == "launchctl" && args[0] == "print":
				case name == "systemctl" && args[0] == "list-unit-files":
					return "ssh.service enabled enabled\nssh.socket enabled enabled\n", nil
				case name == "systemctl" && (args[0] == "is-enabled" || args[0] == "is-active"):
				default:
					t.Fatalf("doctor ran %s %v; it must only read", name, args)
				}
				if loaded {
					return "", nil
				}
				return "", errors.New("exit status 113")
			}
			var buf bytes.Buffer
			checkWorkstationTarget(s, &Config{}, capture(&buf))

			if !tc.section {
				if buf.Len() != 0 {
					t.Fatalf("want no section on %s, got:\n%s", tc.goos, buf.String())
				}
				return
			}
			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}
}

// writeConverged leaves the Linux SSH target as a converge leaves it, the
// firewall aside.
func writeConverged(t *testing.T, p sshtarget.Paths, user string) {
	t.Helper()
	writeFile(t, p.AuthorizedKeys, "ssh-ed25519 AAAAOwner manu@msi\n")
	writeFile(t, p.HostKey, "ssh-ed25519 AAAAHost root@msi\n")
	writeFile(t, p.LogindDropIn, sshtarget.LogindPolicy)
	writeFile(t, p.DropIn, "AllowUsers "+user+"@"+sshtarget.MeshCIDR+"\nPasswordAuthentication no\n")
}
