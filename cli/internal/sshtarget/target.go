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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
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
	SSHDConfig string // the main config, which must include DropIn's dir
	DropIn     string
	// RelativeInclude says sshd reads only a relative Include, resolved
	// against SSHDConfig's directory: Win32-OpenSSH prefixes any other form
	// with that directory and matches nothing (#2763).
	RelativeInclude bool
	HostKey         string // the ed25519 host public key
	AuthorizedKeys  string
	// AdminKeys says AuthorizedKeys is a root-owned file that only
	// administrators may read: Windows' administrators_authorized_keys,
	// which sshd reads for every member of Administrators instead of the
	// account's own ~/.ssh/authorized_keys.
	AdminKeys    bool
	ClientKey    string // the private key the probe offers
	LogindDropIn string // Linux: the lid policy (Logind)
	// Linux, UFW: its switch, its input policy, the rule files, and the
	// witness of the rule files the last apply read as root.
	UFWConf         string
	UFWDefaults     string
	UFWRules        string
	UFWRules6       string
	FirewallWitness string
	PFAnchor        string // the pf anchor file on macOS
	PFPlist         string // the LaunchDaemon that loads it at boot
	PFToken         string // the pf enable references the step holds
	// Record is what the step found before it first changed each resource
	// (Record); "" keeps none.
	Record string
}

// DefaultPaths are the locations on macOS and Linux.
func DefaultPaths(home string) Paths {
	return Paths{
		SSHDConfig:      "/etc/ssh/sshd_config",
		DropIn:          "/etc/ssh/sshd_config.d/01-workstation.conf",
		HostKey:         "/etc/ssh/ssh_host_ed25519_key.pub",
		AuthorizedKeys:  filepath.Join(home, ".ssh", "authorized_keys"),
		ClientKey:       filepath.Join(home, ".ssh", "id_ed25519"),
		LogindDropIn:    "/etc/systemd/logind.conf.d/50-workstation.conf",
		UFWConf:         "/etc/ufw/ufw.conf",
		UFWDefaults:     "/etc/default/ufw",
		UFWRules:        "/etc/ufw/user.rules",
		UFWRules6:       "/etc/ufw/user6.rules",
		FirewallWitness: "/var/lib/dotf/workstation-firewall",
		PFAnchor:        "/etc/pf.anchors/dotf-ssh",
		PFPlist:         "/Library/LaunchDaemons/" + pfLabel + ".plist",
		PFToken:         "/var/lib/dotf/pf-token",
		Record:          "/var/lib/dotf/workstation-target.json",
	}
}

// WindowsPaths are the locations of the OpenSSH.Server capability under
// programData (%ProgramData%). The owner is an administrator, since only an
// administrator can run the apply, so sshd reads the owner's key from
// administrators_authorized_keys and never from home.
func WindowsPaths(programData, home string) Paths {
	ssh := winJoin(programData, "ssh")
	return Paths{
		SSHDConfig:      winJoin(ssh, "sshd_config"),
		DropIn:          winJoin(ssh, "sshd_config.d", "01-workstation.conf"),
		RelativeInclude: true,
		HostKey:         winJoin(ssh, "ssh_host_ed25519_key.pub"),
		AuthorizedKeys:  winJoin(ssh, "administrators_authorized_keys"),
		AdminKeys:       true,
		ClientKey:       winJoin(home, ".ssh", "id_ed25519"),
		Record:          winJoin(programData, "dotf", "workstation-target.json"),
	}
}

// PathsFor are the locations on goos for the account whose home is home.
func PathsFor(goos, home string) Paths {
	if goos == "darwin" {
		p := DefaultPaths(home)
		// macOS keeps a daemon's state under /var/db; there is no /var/lib.
		p.Record = "/var/db/dotf/workstation-target.json"
		p.PFToken = "/var/db/dotf/pf-token"
		return p
	}
	if goos != "windows" {
		return DefaultPaths(home)
	}
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return WindowsPaths(pd, home)
}

// winJoin joins with Windows' separator whatever the host, so the paths are
// the same when a test builds them on macOS.
func winJoin(elem ...string) string {
	return strings.Join(elem, `\`)
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
	t := Target{User: user, Paths: paths, Run: run, Addrs: addrs,
		Session: os.Getenv("SSH_CONNECTION"), Sudoed: os.Getenv("SUDO_USER") != ""}
	var err error
	if t.Service, err = ServiceFor(goos, run, paths); err != nil {
		return t, err
	}
	t.Power = PowerFor(goos, run, paths)
	t.Firewall, t.FirewallNote = FirewallFor(goos, paths, run, addrs, t.Session, t.Sudoed)
	t.MeshAtBoot = MeshAtBootFor(goos, run)
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
	if runtime.GOOS == "windows" {
		return windowsAccount(u.Username, os.Getenv("COMPUTERNAME"))
	}
	return u.Username, nil
}

// windowsAccount is the name AllowUsers matches a local Windows account by:
// Windows reports MACHINE\name, sshd matches the bare name, and Win32-OpenSSH
// documents account names in its Allow and Deny directives as lower case. A
// domain account would need the domain in the pattern, which the template
// does not render, so it is refused rather than written as a line that
// matches nobody.
func windowsAccount(username, computer string) (string, error) {
	domain, name, found := strings.Cut(username, `\`)
	if !found {
		return strings.ToLower(username), nil
	}
	if !strings.EqualFold(domain, computer) {
		return "", fmt.Errorf("%s is a domain account; the SSH target opens sshd to local accounts only", username)
	}
	return strings.ToLower(name), nil
}

// Service turns sshd on and off on one OS.
type Service interface {
	Enabled() (bool, error)   // read without root
	HostKeys(root Root) error // creates the missing host keys; inert, never undone
	Enable(root Root) error   // starts it
	Disable(root Root) error  // the undo of Enable
	Reload(root Root) error   // makes a running sshd read a changed drop-in
	Validate(root Root) error // `sshd -t` on the installed config
	// Effective is `sshd -T` on the installed config: the values sshd would
	// run with. Validate proves the file parses, Effective that it was read.
	Effective(root Root) ([]byte, error)
}

// Target is one machine's SSH posture.
type Target struct {
	User     string
	Key      []byte // the authorized public key, one line
	Template []byte // the drop-in with {user} and {mesh} placeholders
	Paths    Paths
	Service  Service
	Power    Power // nil where the target declares no power policy yet
	// Firewall is nil where the target manages none, and FirewallNote then
	// says why, so a plan never reads converged in silence.
	Firewall     Firewall
	FirewallNote string
	// MeshAtBoot is MeshAtBootFor's reader; nil where the target reads none.
	MeshAtBoot func() string
	Root       Root
	Run        Runner
	Addrs      func() ([]net.IP, error)
	Dotf       string           // the running build, written into the record
	Session    string           // $SSH_CONNECTION of the running process; "" when local
	Sudoed     bool             // dotf itself runs under sudo, which strips $SSH_CONNECTION
	Now        func() time.Time // nil is time.Now
}

// Plan is what differs from the declared state, and whether an apply could
// proceed.
type Plan struct {
	Pending []string // resource names, in apply order
	Notes   []string // what the target does not manage here, and why
	// MeshAtBoot says why the node may stay off the mesh after a reboot
	// nobody attends; "" when the mesh service starts at boot.
	MeshAtBoot string
	Mesh       net.IP
	LAN        net.IP // nil when the node has no LAN address to test from
	dropIn     []byte
	key        []byte
	enabled    bool
	prevDrop   []byte // nil when absent
	prevKeys   []byte // nil when absent
	record     []byte // the record to write, when resRecord is pending
	prevRecord []byte // nil when absent
}

// Converged reports whether nothing differs.
func (p Plan) Converged() bool { return len(p.Pending) == 0 }

const (
	resKey      = "authorized key"
	resHostKeys = "host keys"
	resDropIn   = "sshd drop-in"
	resService  = "sshd service"
	resFirewall = "firewall"
	resPower    = "power policy"
	resRecord   = "state record"
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
	if t.MeshAtBoot != nil {
		p.MeshAtBoot = t.MeshAtBoot()
	}
	p.key = bytes.TrimSpace(t.Key)
	if len(p.key) == 0 {
		return p, errors.New("no public key to authorize")
	}
	if p.prevKeys, err = readOptional(t.Paths.AuthorizedKeys); err != nil {
		return p, err
	}
	// The firewall goes first: it closes the door before sshd's listener
	// opens, and a firewall that refuses (a rule of the owner's opens port 22
	// off the mesh) refuses before any other resource has changed.
	if t.Firewall != nil {
		differs, err := t.Firewall.Differs()
		if err != nil {
			return p, err
		}
		if differs {
			p.Pending = append(p.Pending, resFirewall)
		}
	} else if t.FirewallNote != "" {
		p.Notes = append(p.Notes, t.FirewallNote)
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
	return p, t.inspectRecord(&p)
}

// inspectRecord makes the record pending, last, when it lacks an entry for a
// resource: it is written from what this plan found, before any change.
func (t Target) inspectRecord(p *Plan) error {
	if t.Paths.Record == "" {
		return nil
	}
	r, err := readRecord(t.Paths.Record)
	if err != nil {
		return err
	}
	if !r.fill(*p, func(res string) bool { return slices.Contains(p.Pending, res) }) {
		return nil
	}
	if p.prevRecord, err = readOptional(t.Paths.Record); err != nil {
		return err
	}
	now := time.Now
	if t.Now != nil {
		now = t.Now
	}
	if r.Dotf == "" {
		r.Dotf = t.Dotf
	}
	r.Version, r.Written = RecordVersion, now().UTC()
	if p.record, err = json.MarshalIndent(r, "", "  "); err != nil {
		return err
	}
	p.record = append(p.record, '\n')
	p.Pending = append(p.Pending, resRecord)
	return nil
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
		// A firewall that armed a commit-confirm timer keeps what it loaded
		// only once the whole proof, sshd's included, has passed.
		if c, ok := t.Firewall.(interface{ Commit(Root) error }); ok {
			err = c.Commit(t.Root)
		}
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
			if err := t.proveEffective(p.dropIn); err != nil {
				return err
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
		case resFirewall:
			u, err := t.Firewall.Apply(t.Root)
			if u != nil {
				*undo = append(*undo, u)
			}
			if err != nil {
				return err
			}
		case resPower:
			u, err := t.Power.Apply(t.Root)
			if u != nil {
				*undo = append(*undo, u)
			}
			if err != nil {
				return err
			}
		case resRecord:
			if err := writeRecord(t.Root, t.Paths.Record, p.record); err != nil {
				return err
			}
			*undo = append(*undo, func() error { return t.restoreRoot(t.Paths.Record, p.prevRecord) })
		}
	}
	return nil
}

// proveEffective reads sshd's dump right after the drop-in is installed and
// fails when it does not carry the drop-in's values. It runs in the apply, where
// root is ready, and not in Prove: Prove also runs on a converged plan and in
// the post-condition of a headless converge, which has no root to ask for. The
// dump does not depend on the connecting address (the drop-in has no Match
// block, measured on OpenSSH 10.3p1), so one dump covers the mesh and the LAN.
func (t Target) proveEffective(dropIn []byte) error {
	out, err := t.Service.Effective(t.Root)
	if err != nil {
		return fmt.Errorf("read sshd's effective config: %w", err)
	}
	return checkEffective(dropIn, out, t.Paths.DropIn)
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
	if t.Firewall != nil {
		if err := t.Firewall.Prove(); err != nil {
			return err
		}
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
	mesh, lan = splitAddrs(ips)
	if mesh == nil {
		return nil, nil, ErrNoMesh
	}
	return mesh, lan, nil
}

// splitAddrs picks the node's first mesh address and first private LAN one.
func splitAddrs(ips []net.IP) (mesh, lan net.IP) {
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
	return mesh, lan
}

// checkInclude refuses a main config that does not read the drop-in dir, and
// one where sshd reads a directive of the drop-in before the drop-in itself.
// sshd keeps the first value it reads for a keyword, so `PasswordAuthentication
// yes` above the Include, or in a drop-in that sorts before ours, silently wins
// and the drop-in only looks applied. Reading these files needs no root, so the
// plan and the probe both see a config that undoes the drop-in.
func (t Target) checkInclude() error {
	conf, err := os.ReadFile(t.Paths.SSHDConfig)
	if errors.Is(err, os.ErrNotExist) {
		// No config usually means no OpenSSH server. Its service probe names
		// the fix where it knows one, as Windows' does with the capability.
		if _, serr := t.Service.Enabled(); serr != nil {
			return fmt.Errorf("no %s: %w", t.Paths.SSHDConfig, serr)
		}
		return fmt.Errorf("no %s, though the sshd service is installed: on Windows sshd writes it at its first start (Start-Service sshd); elsewhere restore it from the OpenSSH server package", t.Paths.SSHDConfig)
	}
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
			if resolved, ok := t.includes(pattern, dir); ok {
				if len(first) > 0 {
					return shadowed(t.Paths.SSHDConfig+" above its Include", first)
				}
				return t.checkEarlierDropIns(resolved, ours)
			}
		}
	}
	if t.Paths.RelativeInclude {
		rel, err := filepath.Rel(filepath.Dir(t.Paths.SSHDConfig), dir)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s does not include the drop-in dir in the form sshd reads, `Include %s/*.conf`; an absolute Include is ignored: the drop-in would never be read",
			t.Paths.SSHDConfig, filepath.ToSlash(rel))
	}
	return fmt.Errorf("%s does not include %s/: the drop-in would never be read", t.Paths.SSHDConfig, dir)
}

// includes resolves an Include pattern as sshd does, a relative one against
// the main config's directory, and says whether it reaches the drop-in dir.
// Where sshd reads only a relative Include, any other form reaches nothing.
func (t Target) includes(pattern, dir string) (string, bool) {
	slash := func(p string) string { return strings.ReplaceAll(p, `\`, "/") }
	absolute := filepath.IsAbs(pattern)
	if t.Paths.RelativeInclude {
		p := slash(pattern)
		absolute = strings.HasPrefix(p, "/") || strings.HasPrefix(strings.ToUpper(p), "__PROGRAMDATA__") ||
			(len(p) > 1 && p[1] == ':')
		if absolute {
			return "", false
		}
	}
	resolved := pattern
	if !absolute {
		resolved = filepath.Join(filepath.Dir(t.Paths.SSHDConfig), pattern)
	}
	return resolved, strings.HasPrefix(slash(resolved), slash(dir)+"/")
}

// checkEarlierDropIns reads the drop-ins the Include reaches before ours:
// glob order is lexical, the order sshd reads them in.
func (t Target) checkEarlierDropIns(pattern string, ours map[string]bool) error {
	matches, err := globAll(pattern)
	if err != nil {
		return fmt.Errorf("read the drop-ins the Include %s reaches: %w", pattern, err)
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

// globAll is filepath.Glob without its blind spot (#2812). Glob drops I/O
// errors, so an unreadable directory anywhere on the pattern's path read as
// holding no drop-in that shadows ours, and the check passed without looking.
// Here a missing path matches nothing and any other error is returned. Each
// directory is read in name order, so the matches keep Glob's lexical order.
func globAll(pattern string) ([]string, error) {
	if !strings.ContainsAny(pattern, `*?[`) {
		if _, err := os.Lstat(pattern); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		return []string{pattern}, nil
	}
	dir, file := filepath.Split(pattern)
	dirs, err := globAll(filepath.Clean(dir))
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			ok, err := filepath.Match(file, e.Name())
			if err != nil {
				return nil, err
			}
			if ok {
				matches = append(matches, filepath.Join(d, e.Name()))
			}
		}
	}
	return matches, nil
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

// fileRoot is a Root that writes root-owned files itself, where the OS has no
// `install`: Elevated on Windows.
type fileRoot interface {
	Install(path string, data []byte) error
	InstallAdminOnly(path string, data []byte) error
	Remove(path string) error
}

// adminFiles is the root that may write AuthorizedKeys when only
// administrators may: one that writes files itself and sets their ACL.
func (t Target) adminFiles() (fileRoot, error) {
	fr, ok := t.Root.(fileRoot)
	if !ok {
		return nil, fmt.Errorf("%s is readable by administrators only, and this root cannot set that ACL", t.Paths.AuthorizedKeys)
	}
	return fr, nil
}

func (t Target) writeKeys(data []byte) error {
	if t.Paths.AdminKeys {
		fr, err := t.adminFiles()
		if err != nil {
			return err
		}
		return fr.InstallAdminOnly(t.Paths.AuthorizedKeys, data)
	}
	if err := os.MkdirAll(filepath.Dir(t.Paths.AuthorizedKeys), 0o700); err != nil {
		return err
	}
	return os.WriteFile(t.Paths.AuthorizedKeys, data, 0o600)
}

func (t Target) restoreKeys(prev []byte) error {
	if prev == nil && t.Paths.AdminKeys {
		fr, err := t.adminFiles()
		if err != nil {
			return err
		}
		return fr.Remove(t.Paths.AuthorizedKeys)
	}
	if prev == nil {
		return os.Remove(t.Paths.AuthorizedKeys)
	}
	return t.writeKeys(prev)
}

func (t Target) installRoot(path string, data []byte) error { return installRoot(t.Root, path, data) }

func (t Target) restoreRoot(path string, prev []byte) error { return restoreRoot(t.Root, path, prev) }

// installRoot writes data to a root-owned path through a file of the user's,
// or, for a root that writes files itself, through that root.
func installRoot(root Root, path string, data []byte) error {
	if fr, ok := root.(fileRoot); ok {
		return fr.Install(path, data)
	}
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
	if fr, ok := root.(fileRoot); ok {
		return fr.Remove(path)
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
