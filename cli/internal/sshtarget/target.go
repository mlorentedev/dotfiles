// Package sshtarget makes a machine accept SSH from the mesh and from nothing
// else (#2356, #2013 A1). converge's workstation-target step drives it on a
// machine that declares the ssh-target role.
//
// Inspect reads only what a user may read, so a plan never needs root. Apply
// asks for root once, changes what differs, and then proves the result by
// consequence: sshd must accept the owner's key from the node's mesh address
// and refuse it from its LAN address. If either fails, every change of the run
// is undone in reverse order, so a failed apply leaves the machine as it was.
package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// MeshCIDR is the tailnet's address range (CGNAT, as headscale allocates it).
const MeshCIDR = "100.64.0.0/10"

// ErrNoTerminal is what Root.Ready returns when root needs a password and
// there is no terminal to ask on. The apply stops before changing anything.
var ErrNoTerminal = errors.New("root is needed and there is no terminal to ask on")

// ErrNoMesh is what Inspect returns on a node with no mesh address. Such a
// node cannot be proven, and with sshd open to the mesh only it is closed, not
// open, so converge reports the step skipped rather than failed.
var ErrNoMesh = errors.New("no address in " + MeshCIDR + ": join the mesh first (tailscale up)")

// Runner runs a command as the current user and returns its combined output.
type Runner func(argv ...string) ([]byte, error)

// Root runs commands as root. Production asks for the password once, on the
// terminal, and runs each command with `sudo -n` (ADR-045 amendment 2026-10-12).
type Root interface {
	Ready() error
	Run(argv ...string) ([]byte, error)
}

// Paths are the files the target reads and writes. Tests point them at a
// temporary tree.
type Paths struct {
	SSHDConfig     string // the main config, which must include DropIn's dir
	DropIn         string
	HostKey        string // the ed25519 host public key
	AuthorizedKeys string
	ClientKey      string // the private key the probe offers
	LogindDropIn   string // Linux: the lid policy (Logind)
}

// DefaultPaths are the locations on macOS and Linux.
func DefaultPaths(home string) Paths {
	return Paths{
		SSHDConfig:     "/etc/ssh/sshd_config",
		DropIn:         "/etc/ssh/sshd_config.d/01-workstation.conf",
		HostKey:        "/etc/ssh/ssh_host_ed25519_key.pub",
		AuthorizedKeys: filepath.Join(home, ".ssh", "authorized_keys"),
		ClientKey:      filepath.Join(home, ".ssh", "id_ed25519"),
		LogindDropIn:   "/etc/systemd/logind.conf.d/50-workstation.conf",
	}
}

// The repo files a target is built from, relative to the checkout.
const (
	KeyFile      = "ssh/id_ed25519.pub"
	TemplateFile = "ssh/sshd-workstation.conf"
)

// New builds the target for user on goos from the checkout at repoRoot: the
// owner's key and the drop-in template, with goos's service manager. Root is
// left for the caller, because only an apply needs it and doctor never applies.
func New(repoRoot, user, goos string, paths Paths, run Runner, addrs func() ([]net.IP, error)) (Target, error) {
	t := Target{User: user, Paths: paths, Run: run, Addrs: addrs}
	var err error
	if t.Service, err = ServiceFor(goos, run); err != nil {
		return t, err
	}
	t.Power = PowerFor(goos, run, paths)
	if t.Key, err = os.ReadFile(filepath.Join(repoRoot, KeyFile)); err != nil {
		return t, err
	}
	t.Template, err = os.ReadFile(filepath.Join(repoRoot, TemplateFile))
	return t, err
}

// CurrentUser is the account sshd is opened to: the one running dotf. Converge
// and doctor both ask here, so they render the same AllowUsers line.
func CurrentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

// Service turns sshd on and off on one OS.
type Service interface {
	Enabled() (bool, error)   // read without root
	HostKeys(root Root) error // creates the missing host keys; inert, never undone
	Enable(root Root) error   // starts it
	Disable(root Root) error  // the undo of Enable
	Reload(root Root) error   // makes a running sshd read a changed drop-in
	Validate(root Root) error // `sshd -t` on the installed config
}

// Target is one machine's SSH posture.
type Target struct {
	User     string
	Key      []byte // the authorized public key, one line
	Template []byte // the drop-in with {user} and {mesh} placeholders
	Paths    Paths
	Service  Service
	Power    Power // nil where the target declares no power policy yet
	Root     Root
	Run      Runner
	Addrs    func() ([]net.IP, error)
}

// Plan is what differs from the declared state, and whether an apply could
// proceed.
type Plan struct {
	Pending  []string // resource names, in apply order
	Mesh     net.IP
	LAN      net.IP // nil when the node has no LAN address to test from
	dropIn   []byte
	key      []byte
	enabled  bool
	prevDrop []byte // nil when absent
	prevKeys []byte // nil when absent
}

// Converged reports whether nothing differs.
func (p Plan) Converged() bool { return len(p.Pending) == 0 }

const (
	resKey      = "authorized key"
	resHostKeys = "host keys"
	resDropIn   = "sshd drop-in"
	resService  = "sshd service"
	resPower    = "power policy"
)

// DropIn renders the drop-in for user.
func (t Target) DropIn() []byte {
	r := strings.NewReplacer("{user}", t.User, "{mesh}", MeshCIDR)
	return []byte(r.Replace(string(t.Template)))
}

// Inspect compares the machine with the declared state. It runs no command
// as root and changes nothing; the plan is its result.
func (t Target) Inspect() (Plan, error) {
	var p Plan
	var err error
	if p.Mesh, p.LAN, err = t.addresses(); err != nil {
		return p, err
	}
	if err := t.checkInclude(); err != nil {
		return p, err
	}
	p.key = bytes.TrimSpace(t.Key)
	if len(p.key) == 0 {
		return p, errors.New("no public key to authorize")
	}
	if p.prevKeys, err = readOptional(t.Paths.AuthorizedKeys); err != nil {
		return p, err
	}
	if !hasKey(p.prevKeys, p.key) {
		p.Pending = append(p.Pending, resKey)
	}
	if _, err := os.Stat(t.Paths.HostKey); errors.Is(err, os.ErrNotExist) {
		// Before the drop-in: `sshd -t` refuses a config with no host key.
		p.Pending = append(p.Pending, resHostKeys)
	} else if err != nil {
		return p, err
	}
	p.dropIn = t.DropIn()
	if p.prevDrop, err = readOptional(t.Paths.DropIn); err != nil {
		return p, err
	}
	if !bytes.Equal(p.prevDrop, p.dropIn) {
		p.Pending = append(p.Pending, resDropIn)
	}
	if p.enabled, err = t.Service.Enabled(); err != nil {
		return p, err
	}
	if !p.enabled {
		p.Pending = append(p.Pending, resService)
	}
	if t.Power != nil {
		differs, err := t.Power.Differs()
		if err != nil {
			return p, err
		}
		if differs {
			p.Pending = append(p.Pending, resPower)
		}
	}
	return p, nil
}

// Apply converges what p says differs, then proves the result. On any
// failure it undoes the run's changes in reverse order and returns the error,
// joined with any undo that failed too.
func (t Target) Apply(p Plan) error {
	if p.Converged() {
		return t.Prove(p)
	}
	if err := t.Root.Ready(); err != nil {
		return err
	}
	var undo []func() error
	err := t.apply(p, &undo)
	if err == nil {
		err = t.Prove(p)
	}
	if err == nil {
		return nil
	}
	for i := len(undo) - 1; i >= 0; i-- {
		if uerr := undo[i](); uerr != nil {
			err = errors.Join(err, fmt.Errorf("undo: %w", uerr))
		}
	}
	return fmt.Errorf("%w (the run's changes were undone)", err)
}

func (t Target) apply(p Plan, undo *[]func() error) error {
	for _, res := range p.Pending {
		switch res {
		case resKey:
			if err := t.writeKeys(append(appendLine(p.prevKeys), append(p.key, '\n')...)); err != nil {
				return err
			}
			*undo = append(*undo, func() error { return t.restoreKeys(p.prevKeys) })
		case resHostKeys:
			if err := t.Service.HostKeys(t.Root); err != nil {
				return err
			}
		case resDropIn:
			if err := t.installRoot(t.Paths.DropIn, p.dropIn); err != nil {
				return err
			}
			*undo = append(*undo, func() error {
				if err := t.restoreRoot(t.Paths.DropIn, p.prevDrop); err != nil || !p.enabled {
					return err
				}
				// A running daemon has read the new drop-in; restoring the
				// file is not enough until it reads the old one back.
				return t.Service.Reload(t.Root)
			})
			if err := t.Service.Validate(t.Root); err != nil {
				return fmt.Errorf("sshd rejects the new drop-in: %w", err)
			}
			if p.enabled {
				if err := t.Service.Reload(t.Root); err != nil {
					return err
				}
			}
		case resService:
			if err := t.Service.Enable(t.Root); err != nil {
				return err
			}
			*undo = append(*undo, func() error { return t.Service.Disable(t.Root) })
		case resPower:
			u, err := t.Power.Apply(t.Root)
			if u != nil {
				*undo = append(*undo, u)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Prove asks sshd, through the mesh address, whether it accepts the owner's
// key, and asks again through the LAN address, where it must not. It reads
// `Server accepts key` from the client's debug output, so it needs neither
// the key's passphrase nor an agent, and never opens a session.
//
// What this proves is the sshd layer: the listener, AllowUsers and the
// authorized key. A connection to the node's own address does not cross its
// firewall, so the firewall is proven from another node (OBS-036).
func (t Target) Prove(p Plan) error {
	if err := t.proveSSH(p); err != nil {
		return err
	}
	if t.Power != nil {
		return t.Power.Prove()
	}
	return nil
}

func (t Target) proveSSH(p Plan) error {
	hosts, err := t.knownHosts(p)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(hosts) }()
	if accepted, _ := t.ask(hosts, p.Mesh); !accepted {
		return fmt.Errorf("sshd does not accept the key from the mesh address %s", p.Mesh)
	}
	if p.LAN == nil {
		return nil
	}
	accepted, reached := t.ask(hosts, p.LAN)
	if accepted {
		return fmt.Errorf("sshd accepts the key from the LAN address %s; it must refuse everything outside %s", p.LAN, MeshCIDR)
	}
	if !reached {
		return fmt.Errorf("ssh never reached sshd at the LAN address %s, so its refusal is unproven", p.LAN)
	}
	return nil
}

// ask offers the owner's key to sshd at addr. reached says the connection got
// to sshd at all, so a refusal can be told apart from a timeout or no route.
func (t Target) ask(hosts string, addr net.IP) (accepted, reached bool) {
	a := addr.String()
	out, err := t.Run("ssh", "-v", "-F", "/dev/null",
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile="+hosts, "-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey", "-o", "ConnectTimeout=10",
		"-i", t.Paths.ClientKey, "-b", a, t.User+"@"+a, "true")
	accepted = err == nil || bytes.Contains(out, []byte("Server accepts key"))
	return accepted, accepted || bytes.Contains(out, []byte("Connection established"))
}

// knownHosts writes a known_hosts file naming the node's own host key for
// both addresses, so the probe checks the host key instead of trusting it.
func (t Target) knownHosts(p Plan) (string, error) {
	key, err := os.ReadFile(t.Paths.HostKey)
	if err != nil {
		return "", fmt.Errorf("read the host key: %w", err)
	}
	fields := strings.Fields(string(key))
	if len(fields) < 2 {
		return "", fmt.Errorf("%s is not a public key", t.Paths.HostKey)
	}
	names := p.Mesh.String()
	if p.LAN != nil {
		names += "," + p.LAN.String()
	}
	f, err := os.CreateTemp("", "workstation-known-hosts-*")
	if err != nil {
		return "", err
	}
	_, werr := fmt.Fprintf(f, "%s %s %s\n", names, fields[0], fields[1])
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(f.Name())
		return "", werr
	}
	return f.Name(), nil
}

// addresses finds the node's mesh address, and an address outside the mesh
// to test the refusal from.
func (t Target) addresses() (mesh, lan net.IP, err error) {
	ips, err := t.Addrs()
	if err != nil {
		return nil, nil, err
	}
	_, cidr, _ := net.ParseCIDR(MeshCIDR)
	for _, ip := range ips {
		switch {
		case ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast():
		case cidr.Contains(ip):
			if mesh == nil {
				mesh = ip
			}
		case ip.IsPrivate() && lan == nil:
			lan = ip
		}
	}
	if mesh == nil {
		return nil, nil, ErrNoMesh
	}
	return mesh, lan, nil
}

// checkInclude refuses a main config that does not read the drop-in dir, and
// one where sshd reads a directive of the drop-in before the drop-in itself.
// sshd keeps the first value it reads for a keyword, so `PasswordAuthentication
// yes` above the Include, or in a drop-in that sorts before ours, silently wins
// and the drop-in only looks applied. Reading these files needs no root, so the
// plan and the probe both see a config that undoes the drop-in.
func (t Target) checkInclude() error {
	conf, err := os.ReadFile(t.Paths.SSHDConfig)
	if err != nil {
		return fmt.Errorf("read %s: %w", t.Paths.SSHDConfig, err)
	}
	ours := map[string]bool{}
	for _, k := range directives(t.DropIn()) {
		ours[strings.ToLower(k)] = true
	}
	dir := filepath.Dir(t.Paths.DropIn)
	var first []string
	for _, line := range strings.Split(string(conf), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if strings.EqualFold(f[0], "Match") {
			break
		}
		if !strings.EqualFold(f[0], "Include") {
			if ours[strings.ToLower(f[0])] {
				first = append(first, f[0])
			}
			continue
		}
		for _, pattern := range f[1:] {
			if strings.HasPrefix(pattern, dir+"/") {
				if len(first) > 0 {
					return shadowed(t.Paths.SSHDConfig+" above its Include", first)
				}
				return t.checkEarlierDropIns(pattern, ours)
			}
		}
	}
	return fmt.Errorf("%s does not include %s/: the drop-in would never be read", t.Paths.SSHDConfig, dir)
}

// checkEarlierDropIns reads the drop-ins the Include reaches before ours:
// glob order is lexical, the order sshd reads them in.
func (t Target) checkEarlierDropIns(pattern string, ours map[string]bool) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("the include pattern %s: %w", pattern, err)
	}
	for _, m := range matches {
		if m >= t.Paths.DropIn {
			break
		}
		data, err := os.ReadFile(m)
		if err != nil {
			return err
		}
		var first []string
		for _, k := range directives(data) {
			if ours[strings.ToLower(k)] {
				first = append(first, k)
			}
		}
		if len(first) > 0 {
			return shadowed(m, first)
		}
	}
	return nil
}

func shadowed(where string, keys []string) error {
	return fmt.Errorf("%s sets %s before the drop-in, and sshd keeps the first value it reads; move it after the drop-in or remove it", where, strings.Join(keys, ", "))
}

// directives lists the keywords a config sets unconditionally: those above
// its first Match block.
func directives(data []byte) []string {
	var keys []string
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if strings.EqualFold(f[0], "Match") {
			break
		}
		keys = append(keys, f[0])
	}
	return keys
}

func (t Target) writeKeys(data []byte) error {
	if err := os.MkdirAll(filepath.Dir(t.Paths.AuthorizedKeys), 0o700); err != nil {
		return err
	}
	return os.WriteFile(t.Paths.AuthorizedKeys, data, 0o600)
}

func (t Target) restoreKeys(prev []byte) error {
	if prev == nil {
		return os.Remove(t.Paths.AuthorizedKeys)
	}
	return t.writeKeys(prev)
}

func (t Target) installRoot(path string, data []byte) error { return installRoot(t.Root, path, data) }

func (t Target) restoreRoot(path string, prev []byte) error { return restoreRoot(t.Root, path, prev) }

// installRoot writes data to a root-owned path through a file of the user's.
func installRoot(root Root, path string, data []byte) error {
	f, err := os.CreateTemp("", "workstation-dropin-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	if out, err := root.Run("install", "-m", "0644", "-o", "0", "-g", "0", f.Name(), path); err != nil {
		return fmt.Errorf("install %s: %w: %s", path, err, bytes.TrimSpace(out))
	}
	return nil
}

func restoreRoot(root Root, path string, prev []byte) error {
	if prev != nil {
		return installRoot(root, path, prev)
	}
	if out, err := root.Run("rm", "-f", path); err != nil {
		return fmt.Errorf("remove %s: %w: %s", path, err, bytes.TrimSpace(out))
	}
	return nil
}

func readOptional(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

// hasKey reports whether keys authorizes key: some line holds its type and
// blob, whatever options precede them.
func hasKey(keys, key []byte) bool {
	f := strings.Fields(string(key))
	if len(f) < 2 {
		return false
	}
	want := f[0] + " " + f[1]
	for _, line := range strings.Split(string(keys), "\n") {
		if strings.Contains(strings.Join(strings.Fields(line), " "), want) {
			return true
		}
	}
	return false
}

// appendLine returns b ending in a newline, unless it is empty.
func appendLine(b []byte) []byte {
	if len(b) == 0 || b[len(b)-1] == '\n' {
		return bytes.Clone(b)
	}
	return append(bytes.Clone(b), '\n')
}

// InterfaceAddrs lists the IP addresses of the node's interfaces.
func InterfaceAddrs() ([]net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			ips = append(ips, n.IP)
		}
	}
	return ips, nil
}
