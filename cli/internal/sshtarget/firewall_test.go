package sshtarget

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The fixtures in testdata/ufw are user.rules files ufw 0.36.2 wrote in an
// ubuntu:24.04 container: empty (a fresh install), ours (only the mesh rule)
// and mixed (the mesh rule plus one rule of each shape that opens or does not
// open port 22). They are ufw's own output, so a parser that reads them reads
// what a real machine holds.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "ufw", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExposingRulesNamesEveryRuleThatOpensPort22OffTheMesh(t *testing.T) {
	got := exposingRules(fixture(t, "mixed.rules"))
	want := []string{
		"allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in",
		"allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 OpenSSH - in",
		"allow any any 0.0.0.0/0 any 192.168.1.0/24 in",
		"allow tcp 20:30 0.0.0.0/0 any 0.0.0.0/0 in",
		"allow tcp 22,80,443 0.0.0.0/0 any 0.0.0.0/0 in",
		"allow any 22 0.0.0.0/0 any 0.0.0.0/0 in_eth0",
		// From the mesh range but on any interface: refused too, because the
		// declared rule is the interface, and a range is not one.
		"allow tcp 22 0.0.0.0/0 any 100.64.0.0/10 in",
		"allow any 22 0.0.0.0/0 any 0.0.0.0/0 in",
	}
	if !slices.Equal(got, want) {
		t.Errorf("exposingRules:\n got  %q\n want %q", got, want)
	}
	if got := exposingRules(fixture(t, "ours.rules")); len(got) != 0 {
		t.Errorf("the mesh rule alone is reported as exposing: %q", got)
	}
}

func TestHasMeshRuleReadsBothAddressFamilies(t *testing.T) {
	for name, want := range map[string]bool{"ours.rules": true, "ours6.rules": true, "mixed.rules": true, "mixed6.rules": true, "empty.rules": false} {
		if got := hasMeshRule(fixture(t, name)); got != want {
			t.Errorf("hasMeshRule(%s) = %v, want %v", name, got, want)
		}
	}
}

// host is a fake Linux machine with UFW: its config files in a temp tree,
// and a root that runs the ufw commands the way ufw changes those files.
type host struct {
	t         *testing.T
	u         UFW
	rootCalls []string
	failOn    string // a root command prefix that fails
	// statusOverride, when set, is what `ufw status verbose` prints.
	statusOverride string
}

func newHost(t *testing.T, enabled bool, policy, rules, rules6 string) *host {
	d := t.TempDir()
	h := &host{t: t, u: UFW{
		Conf:     filepath.Join(d, "etc", "ufw", "ufw.conf"),
		Defaults: filepath.Join(d, "etc", "default", "ufw"),
		Rules:    filepath.Join(d, "etc", "ufw", "user.rules"),
		Rules6:   filepath.Join(d, "etc", "ufw", "user6.rules"),
		Witness:  filepath.Join(d, "var", "lib", "dotf", "workstation-firewall"),
	}}
	h.write(h.u.Conf, "# /etc/ufw/ufw.conf\nENABLED="+map[bool]string{true: "yes", false: "no"}[enabled]+"\nLOGLEVEL=low\n")
	h.write(h.u.Defaults, "IPV6=yes\nDEFAULT_INPUT_POLICY=\""+policy+"\"\nDEFAULT_OUTPUT_POLICY=\"ACCEPT\"\n")
	h.write(h.u.Rules, rules)
	if rules6 != "" {
		h.write(h.u.Rules6, rules6)
	}
	return h
}

func (h *host) write(path, data string) {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *host) read(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

func (h *host) Ready() error { return nil }

const (
	meshTuple4 = "### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in_tailscale0 comment=7373683a206d657368206f6e6c79\n"
	meshTuple6 = "### tuple ### allow tcp 22 ::/0 any ::/0 in_tailscale0 comment=7373683a206d657368206f6e6c79\n"
)

func (h *host) Run(argv ...string) ([]byte, error) {
	call := strings.Join(argv, " ")
	h.rootCalls = append(h.rootCalls, call)
	if h.failOn != "" && strings.HasPrefix(call, h.failOn) {
		return []byte("ERROR: simulated"), errors.New("exit status 1")
	}
	switch {
	case argv[0] == "cat":
		return os.ReadFile(argv[1])
	case argv[0] == "install" && argv[1] == "-d":
		return nil, os.MkdirAll(argv[len(argv)-1], 0o755)
	case argv[0] == "install":
		data, err := os.ReadFile(argv[len(argv)-2])
		if err != nil {
			return nil, err
		}
		return nil, os.WriteFile(argv[len(argv)-1], data, 0o644)
	case argv[0] == "rm":
		if err := os.Remove(argv[len(argv)-1]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	case call == "ufw allow in on tailscale0 to any port 22 proto tcp comment ssh: mesh only":
		if !strings.Contains(h.read(h.u.Rules), "in_tailscale0") {
			h.write(h.u.Rules, h.read(h.u.Rules)+meshTuple4)
		}
		if !strings.Contains(h.read(h.u.Rules6), "in_tailscale0") {
			h.write(h.u.Rules6, h.read(h.u.Rules6)+meshTuple6)
		}
	case call == "ufw delete allow in on tailscale0 to any port 22 proto tcp":
		h.write(h.u.Rules, strings.ReplaceAll(h.read(h.u.Rules), meshTuple4, ""))
		h.write(h.u.Rules6, strings.ReplaceAll(h.read(h.u.Rules6), meshTuple6, ""))
	case len(argv) == 4 && argv[0] == "ufw" && argv[1] == "default" && argv[3] == "incoming":
		v := map[string]string{"deny": "DROP", "allow": "ACCEPT", "reject": "REJECT"}[argv[2]]
		h.write(h.u.Defaults, "IPV6=yes\nDEFAULT_INPUT_POLICY=\""+v+"\"\nDEFAULT_OUTPUT_POLICY=\"ACCEPT\"\n")
	case call == "ufw status verbose":
		return []byte(h.status()), nil
	case call == "ufw --force enable":
		h.write(h.u.Conf, strings.Replace(h.read(h.u.Conf), "ENABLED=no", "ENABLED=yes", 1))
	case call == "ufw disable":
		h.write(h.u.Conf, strings.Replace(h.read(h.u.Conf), "ENABLED=yes", "ENABLED=no", 1))
	default:
		h.t.Fatalf("unexpected root command %q", call)
	}
	return nil, nil
}

// status renders `ufw status verbose` from the files, in ufw 0.36.2's format.
func (h *host) status() string {
	if h.statusOverride != "" {
		return h.statusOverride
	}
	if !strings.Contains(h.read(h.u.Conf), "ENABLED=yes") {
		return "Status: inactive\n"
	}
	policy := map[string]string{"DROP": "deny", "ACCEPT": "allow", "REJECT": "reject"}[shellValue([]byte(h.read(h.u.Defaults)), "DEFAULT_INPUT_POLICY")]
	s := "Status: active\nLogging: on (low)\nDefault: " + policy + " (incoming), allow (outgoing), deny (routed)\nNew profiles: skip\n\n" +
		"To                         Action      From\n--                         ------      ----\n"
	if strings.Contains(h.read(h.u.Rules), "in_tailscale0") {
		s += "22/tcp on tailscale0       ALLOW IN    Anywhere                   # ssh: mesh only\n"
	}
	return s
}

// A fresh install: UFW off, input policy DROP, no rules. Apply adds the mesh
// rule before it enables the firewall, so an apply run over the mesh never
// cuts its own session, and the second Differs reads the machine converged.
func TestUFWConvergesAFreshInstallInTheOrderThatKeepsTheMeshSession(t *testing.T) {
	h := newHost(t, false, "DROP", fixture(t, "empty.rules"), fixture(t, "empty.rules"))
	if d, err := h.u.Differs(); err != nil || !d {
		t.Fatalf("Differs() = %v, %v on a fresh install, want true", d, err)
	}
	if _, err := h.u.Apply(h); err != nil {
		t.Fatal(err)
	}
	allow := slices.Index(h.rootCalls, "ufw allow in on tailscale0 to any port 22 proto tcp comment ssh: mesh only")
	enable := slices.Index(h.rootCalls, "ufw --force enable")
	if allow < 0 || enable < 0 || allow > enable {
		t.Errorf("want the mesh rule added before ufw is enabled, root ran %q", h.rootCalls)
	}
	if err := h.u.Prove(); err != nil {
		t.Fatalf("Prove after Apply: %v", err)
	}
	if d, err := h.u.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v after Apply, want false: a second converge must change nothing", d, err)
	}
}

func TestUFWSetsTheInputPolicyToDenyAndItsUndoRestoresIt(t *testing.T) {
	h := newHost(t, true, "ACCEPT", fixture(t, "ours.rules"), fixture(t, "ours6.rules"))
	undo, err := h.u.Apply(h)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.read(h.u.Defaults), `DEFAULT_INPUT_POLICY="DROP"`) {
		t.Fatalf("input policy not DROP after Apply:\n%s", h.read(h.u.Defaults))
	}
	if slices.Contains(h.rootCalls, "ufw --force enable") {
		t.Errorf("enabled an already enabled ufw: %q", h.rootCalls)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.read(h.u.Defaults), `DEFAULT_INPUT_POLICY="ACCEPT"`) {
		t.Errorf("undo left the policy at:\n%s", h.read(h.u.Defaults))
	}
	if _, err := os.Stat(h.u.Witness); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("undo left the witness behind: %v", err)
	}
}

// A rule that opens port 22 off the mesh is the owner's, so the step neither
// deletes it nor runs beside it: it refuses before changing anything, and
// names the rule.
func TestUFWRefusesARuleThatOpensPort22OffTheMesh(t *testing.T) {
	h := newHost(t, false, "DROP", fixture(t, "mixed.rules"), fixture(t, "mixed6.rules"))
	_, err := h.u.Apply(h)
	if err == nil || !strings.Contains(err.Error(), "allow tcp 22,80,443 0.0.0.0/0 any 0.0.0.0/0 in") ||
		!strings.Contains(err.Error(), "allow any any ::/0 any") && !strings.Contains(err.Error(), "allow tcp 22 ::/0 any ::/0 in") {
		t.Fatalf("want a refusal naming the v4 and v6 rules, got %v", err)
	}
	for _, c := range h.rootCalls {
		if !strings.HasPrefix(c, "cat ") {
			t.Errorf("changed the machine before refusing: %q", h.rootCalls)
			break
		}
	}
}

// The witness is what lets a plan without root see a rule added later: any
// write to user.rules moves its size or mtime away from what the apply that
// read the rules as root recorded.
func TestUFWDiffersWhenTheRulesChangeAfterAnApply(t *testing.T) {
	h := newHost(t, false, "DROP", fixture(t, "empty.rules"), fixture(t, "empty.rules"))
	if _, err := h.u.Apply(h); err != nil {
		t.Fatal(err)
	}
	h.write(h.u.Rules, h.read(h.u.Rules)+"### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in\n")
	if d, err := h.u.Differs(); err != nil || !d {
		t.Fatalf("Differs() = %v, %v after a rule was added, want true", d, err)
	}
	if _, err := h.u.Apply(h); err == nil {
		t.Error("the re-apply accepted the added off-mesh rule")
	}
}

func TestUFWUndoesEveryChangeWhenEnableFails(t *testing.T) {
	h := newHost(t, false, "ACCEPT", fixture(t, "empty.rules"), fixture(t, "empty.rules"))
	h.failOn = "ufw --force enable"
	undo, err := h.u.Apply(h)
	if err == nil {
		t.Fatal("Apply succeeded with enable failing")
	}
	if undo == nil {
		t.Fatal("no undo returned for a partial apply")
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h.read(h.u.Rules), "in_tailscale0") || !strings.Contains(h.read(h.u.Defaults), `"ACCEPT"`) {
		t.Errorf("undo left changes: rules\n%s\ndefaults\n%s", h.read(h.u.Rules), h.read(h.u.Defaults))
	}
}

// A mesh rule the machine already had is not this run's to remove.
func TestUFWUndoKeepsAMeshRuleThatWasThereBefore(t *testing.T) {
	h := newHost(t, false, "DROP", fixture(t, "ours.rules"), fixture(t, "ours6.rules"))
	h.failOn = "ufw --force enable"
	undo, _ := h.u.Apply(h)
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(h.rootCalls, "ufw delete allow in on tailscale0 to any port 22 proto tcp") {
		t.Errorf("undo deleted a rule the run did not add: %q", h.rootCalls)
	}
}

func TestFirewallForManagesUFWOnlyWhereItIsInstalled(t *testing.T) {
	d := t.TempDir()
	paths := Paths{UFWConf: filepath.Join(d, "ufw.conf")}
	if f, note := FirewallFor("linux", nil, paths); f != nil || !strings.Contains(note, "ufw") {
		t.Errorf("no ufw.conf: FirewallFor = %v, %q; want nil and a note naming ufw", f, note)
	}
	if err := os.WriteFile(paths.UFWConf, []byte("ENABLED=no\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if f, note := FirewallFor("linux", nil, paths); f == nil || note != "" {
		t.Errorf("with ufw.conf: FirewallFor = %v, %q; want UFW and no note", f, note)
	}
	if f, note := FirewallFor("darwin", nil, paths); f != nil || note == "" {
		t.Errorf("darwin: FirewallFor = %v, %q; want nil and a note", f, note)
	}
	if f, note := FirewallFor("windows", nil, paths); !isType[WindowsFirewall](f) || note != "" {
		t.Errorf("windows: FirewallFor = %v, %q; want WindowsFirewall and no note", f, note)
	}
}

// The files can say one thing and the kernel another (an iptables backend
// that refused the load). The apply asks ufw what it loaded, and undoes the
// run when the answer is not the declared state.
func TestUFWUndoesTheRunWhenUFWDoesNotReportTheDeclaredState(t *testing.T) {
	h := newHost(t, false, "ACCEPT", fixture(t, "empty.rules"), fixture(t, "empty.rules"))
	h.statusOverride = "Status: inactive\n"
	undo, err := h.u.Apply(h)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("Apply = %v, want the status check to fail it", err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h.read(h.u.Conf), "ENABLED=yes") || !strings.Contains(h.read(h.u.Defaults), `"ACCEPT"`) {
		t.Errorf("undo left changes:\n%s\n%s", h.read(h.u.Conf), h.read(h.u.Defaults))
	}
}
