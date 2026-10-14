package sshtarget

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Firewall closes port 22 to every network but the mesh. Differs reads
// without root; Apply returns the undo of what it changed, even when it fails
// part way; Prove reads the state back without root.
type Firewall interface {
	Differs() (bool, error)
	Apply(root Root) (undo func() error, err error)
	Prove() error
}

// MeshInterface is the interface the mesh arrives on. sshd's AllowUsers
// already limits who may log in from the mesh range; the firewall limits where
// a connection may come from at all, so a LAN peer never reaches sshd.
const MeshInterface = "tailscale0"

// FirewallFor is the firewall the target manages on goos, with a note naming
// why there is none: a plan says so rather than reading converged.
func FirewallFor(goos string, run Runner, paths Paths) (Firewall, string) {
	if goos == "windows" {
		return WindowsFirewall{Run: run}, ""
	}
	if goos != "linux" {
		return nil, "firewall not managed on " + goos + " yet: sshd's AllowUsers is the only fence"
	}
	if _, err := os.Stat(paths.UFWConf); err != nil {
		return nil, "no ufw on this machine (" + paths.UFWConf + "): sshd's AllowUsers is the only fence"
	}
	return UFW{
		Conf:     paths.UFWConf,
		Defaults: paths.UFWDefaults,
		Rules:    paths.UFWRules,
		Rules6:   paths.UFWRules6,
		Witness:  paths.FirewallWitness,
	}, ""
}

// UFW is the firewall on Linux. Its rules are readable by root only, so a plan
// without root reads what it can (ufw.conf, the input policy) and a witness:
// the size and mtime of the rule files the last apply read as root. Any rule
// written since moves them, so the plan is pending again and the next apply
// reads the rules as root before it accepts them.
type UFW struct {
	Conf     string // ENABLED=yes|no
	Defaults string // DEFAULT_INPUT_POLICY
	Rules    string // user.rules
	Rules6   string // user6.rules; absent when ufw runs without IPv6
	Witness  string
}

// The mesh rule, and its delete. The comment is ufw's own record of why the
// rule exists, read by whoever runs `ufw status`.
var (
	ufwAllowMesh  = []string{"ufw", "allow", "in", "on", MeshInterface, "to", "any", "port", "22", "proto", "tcp", "comment", "ssh: mesh only"}
	ufwDeleteMesh = []string{"ufw", "delete", "allow", "in", "on", MeshInterface, "to", "any", "port", "22", "proto", "tcp"}
)

func (u UFW) Differs() (bool, error) {
	enabled, policy, err := u.state()
	if err != nil {
		return false, err
	}
	if !enabled || !closedPolicy(policy) {
		return true, nil
	}
	want, err := u.witness()
	if err != nil {
		return false, err
	}
	got, err := readOptional(u.Witness)
	if err != nil {
		return false, err
	}
	return string(got) != want, nil
}

// Apply refuses before it changes anything when a rule opens port 22 off the
// mesh: the rule is the owner's, and deleting it is not this step's call. It
// then changes the machine in the order that keeps a session over the mesh
// alive: the mesh rule, the input policy, and only then ufw itself.
func (u UFW) Apply(root Root) (func() error, error) {
	var undos []func() error
	undo := func() error {
		var errs []error
		for i := len(undos) - 1; i >= 0; i-- {
			errs = append(errs, undos[i]())
		}
		return errors.Join(errs...)
	}
	rules, err := u.readRules(root)
	if err != nil {
		return undo, err
	}
	var exposing []string
	for _, r := range rules {
		exposing = append(exposing, exposingRules(r)...)
	}
	if len(exposing) > 0 {
		return undo, fmt.Errorf("ufw has rules that open port 22 off %s; remove them (sudo ufw status numbered, then sudo ufw delete N) or narrow them to %s, then converge again:\n  %s",
			MeshInterface, MeshInterface, strings.Join(exposing, "\n  "))
	}
	enabled, policy, err := u.state()
	if err != nil {
		return undo, err
	}
	prevWitness, err := readOptional(u.Witness)
	if err != nil {
		return undo, err
	}

	missing := 0
	for _, r := range rules {
		if !hasMeshRule(r) {
			missing++
		}
	}
	if missing > 0 {
		if err := asRoot(root, ufwAllowMesh...); err != nil {
			return undo, err
		}
		// ufw adds the rule to both families at once, and deletes it from both.
		// Where one family had it already, the delete would take the owner's
		// copy too, so the added half stays: it opens nothing off the mesh.
		if missing == len(rules) {
			undos = append(undos, func() error { return asRoot(root, ufwDeleteMesh...) })
		}
	}
	if !closedPolicy(policy) {
		if err := asRoot(root, "ufw", "default", "deny", "incoming"); err != nil {
			return undo, err
		}
		prev := ufwPolicyVerb(policy)
		undos = append(undos, func() error { return asRoot(root, "ufw", "default", prev, "incoming") })
	}
	if !enabled {
		if err := asRoot(root, "ufw", "--force", "enable"); err != nil {
			return undo, err
		}
		undos = append(undos, func() error { return asRoot(root, "ufw", "disable") })
	}

	if err := ufwActive(root); err != nil {
		return undo, err
	}
	want, err := u.witness()
	if err != nil {
		return undo, err
	}
	if string(prevWitness) != want {
		if err := asRoot(root, "install", "-d", "-m", "0755", filepath.Dir(u.Witness)); err != nil {
			return undo, err
		}
		if err := installRoot(root, u.Witness, []byte(want)); err != nil {
			return undo, err
		}
		undos = append(undos, func() error { return restoreRoot(root, u.Witness, prevWitness) })
	}
	return undo, nil
}

// Prove reads back, without root, what the apply left: ufw on, the input
// policy closed, and the rule files as the apply read them.
func (u UFW) Prove() error {
	differs, err := u.Differs()
	if err != nil {
		return err
	}
	if differs {
		return fmt.Errorf("ufw does not read as converged after the apply (%s, %s, %s)", u.Conf, u.Defaults, u.Witness)
	}
	return nil
}

// ufwActive asks ufw, as root, what the kernel now holds: the files say what
// ufw will load, its status says what it loaded.
func ufwActive(root Root) error {
	out, err := root.Run("ufw", "status", "verbose")
	if err != nil {
		return fmt.Errorf("ufw status verbose: %w: %s", err, strings.TrimSpace(string(out)))
	}
	s := string(out)
	switch {
	case !strings.Contains(s, "Status: active"):
		return fmt.Errorf("ufw is not active after the apply:\n%s", s)
	case !strings.Contains(s, "deny (incoming)") && !strings.Contains(s, "reject (incoming)"):
		return fmt.Errorf("ufw does not refuse incoming by default after the apply:\n%s", s)
	case !strings.Contains(s, "22/tcp on "+MeshInterface):
		return fmt.Errorf("ufw lists no port 22 rule on %s after the apply:\n%s", MeshInterface, s)
	}
	return nil
}

// readRules reads each rule file that exists, as root: ufw makes them 0640.
func (u UFW) readRules(root Root) ([]string, error) {
	var rules []string
	for _, p := range []string{u.Rules, u.Rules6} {
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			continue
		}
		out, err := root.Run("cat", p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w: %s", p, err, strings.TrimSpace(string(out)))
		}
		rules = append(rules, string(out))
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("no ufw rule file at %s", u.Rules)
	}
	return rules, nil
}

// state reads ufw's switch and its input policy, both world-readable.
func (u UFW) state() (enabled bool, policy string, err error) {
	conf, err := os.ReadFile(u.Conf)
	if err != nil {
		return false, "", err
	}
	defaults, err := os.ReadFile(u.Defaults)
	if err != nil {
		return false, "", err
	}
	return shellValue(conf, "ENABLED") == "yes", shellValue(defaults, "DEFAULT_INPUT_POLICY"), nil
}

// witness is the size and mtime of each rule file, which stat reads without
// root: /etc/ufw is world-searchable, only the files are not world-readable.
func (u UFW) witness() (string, error) {
	var b strings.Builder
	for _, p := range []string{u.Rules, u.Rules6} {
		fi, err := os.Stat(p)
		switch {
		case errors.Is(err, os.ErrNotExist):
			fmt.Fprintf(&b, "%s absent\n", p)
		case err != nil:
			return "", err
		default:
			fmt.Fprintf(&b, "%s %d %d\n", p, fi.Size(), fi.ModTime().UnixNano())
		}
	}
	return b.String(), nil
}

// closedPolicy reports whether an input policy refuses what no rule allows.
// REJECT is the owner's choice as much as DROP, so it is kept.
func closedPolicy(p string) bool { return p == "DROP" || p == "REJECT" }

// ufwPolicyVerb is the `ufw default` verb that sets policy p back.
func ufwPolicyVerb(p string) string {
	switch p {
	case "DROP":
		return "deny"
	case "REJECT":
		return "reject"
	}
	return "allow"
}

// shellValue reads KEY=value from a shell-style file, quotes stripped.
func shellValue(data []byte, key string) string {
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), key+"="); ok {
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

// ufwTuple is one rule as ufw records it in user.rules:
//
//	### tuple ### <action> <proto> <dport> <dst> <sport> <src> [<dapp> <sapp>] <direction>[_<iface>] [comment=<hex>]
type ufwTuple struct {
	text      string // the tuple as written, for naming it
	action    string // allow, deny, reject or limit, without a _log suffix
	proto     string
	dport     string
	direction string // in or out
	iface     string // empty when the rule names no interface
}

func ufwTuples(rules string) []ufwTuple {
	var ts []ufwTuple
	for _, line := range strings.Split(rules, "\n") {
		text, ok := strings.CutPrefix(line, "### tuple ### ")
		if !ok {
			continue
		}
		f := strings.Fields(text)
		if len(f) > 0 && strings.HasPrefix(f[len(f)-1], "comment=") {
			f = f[:len(f)-1]
		}
		// A route rule filters forwarded traffic, never what reaches sshd.
		if len(f) != 7 && len(f) != 9 || strings.HasPrefix(f[0], "route:") {
			continue
		}
		dir, iface, _ := strings.Cut(f[len(f)-1], "_")
		action, _, _ := strings.Cut(f[0], "_")
		ts = append(ts, ufwTuple{text: strings.TrimSpace(text), action: action, proto: f[1], dport: f[2], direction: dir, iface: iface})
	}
	return ts
}

// opens22 reports whether t lets a TCP connection to port 22 in.
func (t ufwTuple) opens22() bool {
	return (t.action == "allow" || t.action == "limit") && t.direction == "in" &&
		(t.proto == "tcp" || t.proto == "any") && portCovers(t.dport, 22)
}

// exposingRules are the rules that open port 22 on any interface but the
// mesh's. A rule from the mesh's address range without the interface is one:
// the declared rule is the interface, and a range is not one.
func exposingRules(rules string) []string {
	var out []string
	for _, t := range ufwTuples(rules) {
		if t.opens22() && t.iface != MeshInterface {
			out = append(out, t.text)
		}
	}
	return out
}

// hasMeshRule reports whether a rule already opens port 22 on the mesh.
func hasMeshRule(rules string) bool {
	for _, t := range ufwTuples(rules) {
		if t.opens22() && t.iface == MeshInterface {
			return true
		}
	}
	return false
}

// portCovers reports whether a ufw port spec (any, a port, a lo:hi range, or a
// comma list of those) includes port.
func portCovers(spec string, port int) bool {
	if spec == "any" {
		return true
	}
	for _, p := range strings.Split(spec, ",") {
		lo, hi, isRange := strings.Cut(p, ":")
		if !isRange {
			hi = lo
		}
		l, err1 := strconv.Atoi(lo)
		h, err2 := strconv.Atoi(hi)
		if err1 == nil && err2 == nil && l <= port && port <= h {
			return true
		}
	}
	return false
}
