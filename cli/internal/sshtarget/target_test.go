package sshtarget

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOwner manu@msi"
	hostKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHost root@macmini"
	template = "AllowUsers {user}@{mesh}\nPasswordAuthentication no\n"
)

var (
	meshIP = net.ParseIP("100.64.0.15")
	lanIP  = net.ParseIP("192.168.1.20")
)

// machine is a fake macOS node: files under a temp root, launchd's state, and
// an sshd whose answer to the probe the test sets.
type machine struct {
	t          *testing.T
	paths      Paths
	loaded     bool
	ready      error // what Root.Ready returns
	sshdT      error // what `sshd -t` returns
	accepts    func(addr string) bool
	unreached  string // an address ssh times out on before reaching sshd
	rootCalls  []string
	forbidRoot bool
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	dir := t.TempDir()
	m := &machine{t: t, paths: Paths{
		SSHDConfig:     filepath.Join(dir, "etc/ssh/sshd_config"),
		DropIn:         filepath.Join(dir, "etc/ssh/sshd_config.d/01-workstation.conf"),
		HostKey:        filepath.Join(dir, "etc/ssh/ssh_host_ed25519_key.pub"),
		AuthorizedKeys: filepath.Join(dir, "home/.ssh/authorized_keys"),
		ClientKey:      filepath.Join(dir, "home/.ssh/id_ed25519"),
	}}
	// The sshd that is configured accepts the key from the mesh only, and
	// only once the key is authorized and sshd runs.
	m.accepts = func(addr string) bool {
		keys, _ := os.ReadFile(m.paths.AuthorizedKeys)
		return m.loaded && strings.Contains(string(keys), "AAAAIOwner") && strings.HasPrefix(addr, "100.")
	}
	m.write(m.paths.SSHDConfig, "Include "+filepath.Dir(m.paths.DropIn)+"/*\n")
	if err := os.MkdirAll(filepath.Dir(m.paths.DropIn), 0o755); err != nil {
		t.Fatal(err)
	}
	return m
}

func (m *machine) write(path, data string) {
	m.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) Ready() error {
	if m.forbidRoot {
		m.t.Fatal("asked for root")
	}
	return m.ready
}

func (m *machine) Run(argv ...string) ([]byte, error) {
	if m.forbidRoot {
		m.t.Fatalf("ran as root: %v", argv)
	}
	m.rootCalls = append(m.rootCalls, strings.Join(argv, " "))
	switch argv[0] {
	case "install":
		data, err := os.ReadFile(argv[len(argv)-2])
		if err != nil {
			return nil, err
		}
		return nil, os.WriteFile(argv[len(argv)-1], data, 0o644)
	case "rm":
		return nil, os.Remove(argv[len(argv)-1])
	case "ssh-keygen":
		m.write(m.paths.HostKey, hostKey+"\n")
	case "/usr/sbin/sshd":
		return []byte("bad option"), m.sshdT
	case "launchctl":
		switch argv[1] {
		case "bootstrap":
			m.loaded = true
		case "bootout":
			m.loaded = false
		}
	}
	return nil, nil
}

// user runs launchctl print and the ssh probe as the user.
func (m *machine) user(argv ...string) ([]byte, error) {
	switch argv[0] {
	case "launchctl":
		if m.loaded {
			return nil, nil
		}
		return nil, errors.New("exit status 113")
	case "ssh":
		target := argv[len(argv)-2]
		addr := target[strings.Index(target, "@")+1:]
		if addr == m.unreached {
			return []byte("ssh: connect to host " + addr + " port 22: Operation timed out"), errors.New("exit status 255")
		}
		if m.accepts(addr) {
			return []byte("debug1: Connection established.\ndebug1: Server accepts key: id_ed25519 ED25519\nEnter passphrase"), errors.New("exit status 255")
		}
		return []byte("debug1: Connection established.\nPermission denied (publickey)."), errors.New("exit status 255")
	}
	m.t.Fatalf("unexpected user command %v", argv)
	return nil, nil
}

func (m *machine) target() Target {
	return Target{
		User: "manu", Key: []byte(testKey + "\n"), Template: []byte(template),
		Paths: m.paths, Service: Launchd{Run: m.user}, Root: m, Run: m.user,
		Addrs: func() ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1"), lanIP, meshIP}, nil },
	}
}

func TestInspectPlansEveryResourceWithoutRoot(t *testing.T) {
	m := newMachine(t)
	m.forbidRoot = true
	p, err := m.target().Inspect()
	if err != nil {
		t.Fatal(err)
	}
	want := "authorized key,host keys,sshd drop-in,sshd service"
	if got := strings.Join(p.Pending, ","); got != want {
		t.Fatalf("pending = %s, want %s", got, want)
	}
	if !p.Mesh.Equal(meshIP) || !p.LAN.Equal(lanIP) {
		t.Errorf("addresses = %v, %v; want %v, %v", p.Mesh, p.LAN, meshIP, lanIP)
	}
}

func TestApplyConvergesAndASecondInspectHasNothingToDo(t *testing.T) {
	m := newMachine(t)
	tg := m.target()
	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if err := tg.Apply(p); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	drop, _ := os.ReadFile(m.paths.DropIn)
	if string(drop) != "AllowUsers manu@100.64.0.0/10\nPasswordAuthentication no\n" {
		t.Errorf("drop-in = %q", drop)
	}
	again, err := tg.Inspect()
	if err != nil || !again.Converged() {
		t.Fatalf("second Inspect = %v, %v; want converged", again.Pending, err)
	}
	// Host keys come before validation, and validation before sshd listens.
	order := strings.Join(m.rootCalls, "|")
	if strings.Index(order, "ssh-keygen") > strings.Index(order, "sshd -t") || strings.Index(order, "sshd -t") > strings.Index(order, "bootstrap") {
		t.Errorf("root commands out of order: %s", order)
	}
}

func TestApplyKeepsTheKeysAlreadyAuthorized(t *testing.T) {
	m := newMachine(t)
	m.write(m.paths.AuthorizedKeys, "ssh-ed25519 AAAAOther other@host")
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err != nil {
		t.Fatal(err)
	}
	keys, _ := os.ReadFile(m.paths.AuthorizedKeys)
	if string(keys) != "ssh-ed25519 AAAAOther other@host\n"+testKey+"\n" {
		t.Errorf("authorized_keys = %q", keys)
	}
}

func TestApplyWithoutATerminalChangesNothing(t *testing.T) {
	m := newMachine(t)
	m.ready = ErrNoTerminal
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("Apply = %v, want ErrNoTerminal", err)
	}
	if len(m.rootCalls) > 0 {
		t.Errorf("ran as root: %v", m.rootCalls)
	}
	if _, err := os.Stat(m.paths.AuthorizedKeys); err == nil {
		t.Error("authorized_keys written without root being ready")
	}
}

func TestAFailedProbeUndoesTheRunInReverse(t *testing.T) {
	m := newMachine(t)
	m.accepts = func(string) bool { return false }
	tg := m.target()
	p, _ := tg.Inspect()
	err := tg.Apply(p)
	if err == nil || !strings.Contains(err.Error(), "does not accept the key from the mesh") || !strings.Contains(err.Error(), "undone") {
		t.Fatalf("Apply = %v, want a mesh refusal that was undone", err)
	}
	if m.loaded {
		t.Error("sshd still loaded after the undo")
	}
	if _, err := os.Stat(m.paths.DropIn); err == nil {
		t.Error("drop-in still installed after the undo")
	}
	if _, err := os.Stat(m.paths.AuthorizedKeys); err == nil {
		t.Error("authorized_keys still present after the undo")
	}
	order := strings.Join(m.rootCalls, "|")
	if strings.Index(order, "bootout") > strings.Index(order, "rm -f") {
		t.Errorf("undo not in reverse order (sshd stops before its config goes): %s", order)
	}
}

func TestAnSSHDThatAcceptsTheLANFails(t *testing.T) {
	m := newMachine(t)
	m.accepts = func(string) bool { return m.loaded }
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err == nil || !strings.Contains(err.Error(), "LAN address 192.168.1.20") {
		t.Fatalf("Apply = %v, want the LAN acceptance named", err)
	}
	if m.loaded {
		t.Error("an sshd open to the LAN was left running")
	}
}

func TestALANAddressSSHNeverReachesLeavesTheRefusalUnproven(t *testing.T) {
	m := newMachine(t)
	m.unreached = lanIP.String()
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err == nil || !strings.Contains(err.Error(), "never reached sshd at the LAN address 192.168.1.20") {
		t.Fatalf("Apply = %v, want the unreached LAN address named", err)
	}
	if m.loaded {
		t.Error("sshd left running on an unproven refusal")
	}
}

func TestARejectedDropInRestoresThePreviousOne(t *testing.T) {
	m := newMachine(t)
	m.write(m.paths.DropIn, "# previous\n")
	m.sshdT = errors.New("exit status 255")
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err == nil || !strings.Contains(err.Error(), "sshd rejects the new drop-in") {
		t.Fatalf("Apply = %v, want the validation failure", err)
	}
	if drop, _ := os.ReadFile(m.paths.DropIn); string(drop) != "# previous\n" {
		t.Errorf("drop-in after a rejected apply = %q, want the previous one", drop)
	}
	if m.loaded {
		t.Error("sshd started with a config it rejected")
	}
}

func TestInspectRefusesANodeOutsideTheMesh(t *testing.T) {
	m := newMachine(t)
	tg := m.target()
	tg.Addrs = func() ([]net.IP, error) { return []net.IP{lanIP}, nil }
	if _, err := tg.Inspect(); err == nil || !strings.Contains(err.Error(), "join the mesh") {
		t.Fatalf("Inspect = %v, want a mesh refusal", err)
	}
}

func TestInspectRefusesAConfigThatSkipsTheDropInDir(t *testing.T) {
	m := newMachine(t)
	m.write(m.paths.SSHDConfig, "UsePAM yes\n")
	if _, err := m.target().Inspect(); err == nil || !strings.Contains(err.Error(), "never be read") {
		t.Fatalf("Inspect = %v, want the missing Include named", err)
	}
}

func TestInspectRefusesADirectiveSSHDReadsBeforeTheDropIn(t *testing.T) {
	for name, setup := range map[string]func(m *machine){
		"main config above the Include": func(m *machine) {
			m.write(m.paths.SSHDConfig, "PasswordAuthentication yes\nInclude "+filepath.Dir(m.paths.DropIn)+"/*\n")
		},
		"drop-in that sorts first": func(m *machine) {
			m.write(filepath.Join(filepath.Dir(m.paths.DropIn), "00-local.conf"), "# local\nPasswordAuthentication yes\n")
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			setup(m)
			if _, err := m.target().Inspect(); err == nil || !strings.Contains(err.Error(), "sets PasswordAuthentication before the drop-in") {
				t.Fatalf("Inspect = %v, want the shadowing directive named", err)
			}
		})
	}
}

func TestInspectAllowsWhatTheDropInDoesNotSet(t *testing.T) {
	m := newMachine(t)
	dir := filepath.Dir(m.paths.DropIn)
	m.write(m.paths.SSHDConfig, "Port 22\nInclude "+dir+"/*\nPasswordAuthentication yes\n")
	m.write(filepath.Join(dir, "00-local.conf"), "UsePAM yes\nMatch User other\n\tPasswordAuthentication yes\n")
	m.write(filepath.Join(dir, "100-macos.conf"), "PasswordAuthentication yes\n")
	if _, err := m.target().Inspect(); err != nil {
		t.Fatalf("Inspect = %v; directives sshd reads after the drop-in, or that it does not set, are not ours to refuse", err)
	}
}

func TestHasKeyIgnoresOptionsAndComments(t *testing.T) {
	keys := []byte("from=\"100.64.0.0/10\" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOwner old-comment\n")
	if !hasKey(keys, []byte(testKey)) {
		t.Error("a key with options and another comment was not recognised")
	}
	if hasKey([]byte("ssh-ed25519 AAAAOther x\n"), []byte(testKey)) {
		t.Error("another key was taken for the owner's")
	}
}

// reloadCounter is launchd with its reloads counted, standing for a service
// manager whose running sshd keeps the config it last read (systemd).
type reloadCounter struct {
	Launchd
	reloads *int
}

func (r reloadCounter) Reload(Root) error { *r.reloads++; return nil }

// When sshd already runs, a failed proof must leave it running the previous
// drop-in, not only with the previous file on disk: the undo reloads too.
func TestAFailedProbeReloadsARunningSSHDBackToThePreviousDropIn(t *testing.T) {
	m := newMachine(t)
	m.loaded = true
	m.write(m.paths.HostKey, hostKey+"\n")
	m.accepts = func(string) bool { return false }
	reloads := 0
	tg := m.target()
	tg.Service = reloadCounter{Launchd: Launchd{Run: m.user}, reloads: &reloads}
	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if err := tg.Apply(p); err == nil {
		t.Fatal("Apply succeeded though sshd refused the mesh")
	}
	if reloads != 2 {
		t.Errorf("sshd reloaded %d times, want 2: once for the new drop-in, once for its undo", reloads)
	}
}

// refusingFirewall differs and refuses, as UFW does over a rule of the
// owner's that opens port 22 off the mesh.
type refusingFirewall struct{}

func (refusingFirewall) Differs() (bool, error) { return true, nil }
func (refusingFirewall) Apply(Root) (func() error, error) {
	return func() error { return nil }, errors.New("ufw has rules that open port 22 off tailscale0")
}
func (refusingFirewall) Prove() error { return nil }

func TestAFirewallThatRefusesDoesSoBeforeAnythingChanges(t *testing.T) {
	m := newMachine(t)
	tg := m.target()
	tg.Firewall = refusingFirewall{}
	p, err := tg.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if p.Pending[0] != resFirewall {
		t.Fatalf("pending = %q, want the firewall first", p.Pending)
	}
	if err := tg.Apply(p); err == nil || !strings.Contains(err.Error(), "off tailscale0") {
		t.Fatalf("Apply = %v, want the firewall's refusal", err)
	}
	if len(m.rootCalls) != 0 {
		t.Errorf("root ran before the refusal: %q", m.rootCalls)
	}
}
