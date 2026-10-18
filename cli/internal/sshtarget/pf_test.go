package sshtarget

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var pfMesh = net.ParseIP("100.64.0.15")

// pfMac is a fake macOS for the pf firewall: the files on disk, what pf holds,
// what launchd has loaded, and every command run as root, in order.
type pfMac struct {
	t           *testing.T
	anchor      string
	plist       string
	loaded      bool // pf holds our anchor's rules
	enabled     bool
	jobLoaded   bool
	failLoad    bool // `pfctl -E -a … -f` succeeds but pf holds nothing
	noToken     bool // `pfctl -E` prints no token
	token       string
	boot        string          // kern.bootsessionuuid
	next        int             // the token the next `pfctl -E` takes
	held        map[string]bool // pf's enable references, by token
	printErr    error           // what `launchctl print` fails with, when not nil
	killErr     error           // what killing the timer fails with, when not nil
	timerKilled bool
	rootCalls   []string
}

func newPFMac(t *testing.T) *pfMac {
	d := t.TempDir()
	return &pfMac{t: t, anchor: filepath.Join(d, "dotf-ssh"), plist: filepath.Join(d, "pf-ssh.plist"),
		token: filepath.Join(d, "db", "pf-token"), boot: "BOOT-1", next: 1001, held: map[string]bool{}}
}

func (m *pfMac) pf(session string) *PF {
	return m.pfAs(session, false)
}

func (m *pfMac) pfAs(session string, sudoed bool) *PF {
	return &PF{Anchor: m.anchor, Plist: m.plist, Token: m.token, Run: m.user, Session: session, Sudoed: sudoed,
		Mesh: func() (net.IP, error) { return pfMesh, nil }}
}

func (m *pfMac) user(argv ...string) ([]byte, error) {
	if slices.Equal(argv, []string{sysctl, "-n", "kern.bootsessionuuid"}) {
		return []byte(m.boot + "\n"), nil
	}
	if slices.Equal(argv, []string{"launchctl", "print", pfLaunchd}) {
		if m.printErr != nil {
			return []byte("Bad request."), m.printErr
		}
		if m.jobLoaded {
			return nil, nil
		}
		return []byte("Could not find service"), exitErr(113)
	}
	m.t.Fatalf("unexpected user command %v", argv)
	return nil, nil
}

func (m *pfMac) Ready() error { return nil }

func (m *pfMac) Run(argv ...string) ([]byte, error) {
	call := strings.Join(argv, " ")
	m.rootCalls = append(m.rootCalls, call)
	switch {
	case argv[0] == "install" && argv[1] == "-d":
		return nil, os.MkdirAll(argv[len(argv)-1], 0o755)
	case argv[0] == "install":
		data, err := os.ReadFile(argv[len(argv)-2])
		if err != nil {
			return nil, err
		}
		return nil, os.WriteFile(argv[len(argv)-1], data, 0o644)
	case argv[0] == "rm":
		return nil, os.Remove(argv[len(argv)-1])
	case argv[0] == "/bin/sh":
		return []byte("4242\n"), nil
	case call == "kill 4242":
		if m.killErr != nil {
			return []byte("kill: 4242: No such process"), m.killErr
		}
		m.timerKilled = true
	case call == pfctl+" -E -a "+PFAnchor+" -f "+m.anchor:
		m.enabled, m.loaded = true, !m.failLoad
		tok := strconv.Itoa(m.next)
		m.next++
		m.held[tok] = true
		if m.noToken {
			return []byte("pf enabled\n"), nil
		}
		return []byte("pf enabled\nToken : " + tok + "\n"), nil
	case len(argv) == 3 && argv[0] == pfctl && argv[1] == "-X":
		if !m.held[argv[2]] {
			return []byte("pfctl: DIOCSTOPREF: Invalid argument"), exitErr(1)
		}
		delete(m.held, argv[2])
		m.enabled = len(m.held) > 0
	case call == pfctl+" -a "+PFAnchor+" -F all":
		m.loaded = false
	case call == pfctl+" -s info":
		if m.enabled {
			return []byte("Status: Enabled for 0 days 00:00:01\n"), nil
		}
		return []byte("Status: Disabled\n"), nil
	case call == pfctl+" -a "+PFAnchor+" -s rules":
		if m.loaded {
			return []byte("pass in quick on lo0 proto tcp from any to any port = 22 flags S/SA keep state\n" +
				"pass in quick inet proto tcp from any to 100.64.0.15 port = 22 flags S/SA keep state\n" +
				"block drop in quick proto tcp from any to any port = 22\n"), nil
		}
		return nil, nil
	case call == "launchctl bootstrap system "+m.plist:
		m.jobLoaded = true
	case call == "launchctl bootout "+pfLaunchd:
		m.jobLoaded = false
	case call == "launchctl enable "+pfLaunchd, call == "launchctl disable "+pfLaunchd:
	default:
		m.t.Fatalf("unexpected root command %q", call)
	}
	return nil, nil
}

func (m *pfMac) index(call string) int {
	return slices.IndexFunc(m.rootCalls, func(c string) bool { return strings.HasPrefix(c, call) })
}

func TestPFLoadsTheAnchorBehindATimerAndCommitDisarmsIt(t *testing.T) {
	m := newPFMac(t)
	f := m.pf("")
	if d, err := f.Differs(); err != nil || !d {
		t.Fatalf("Differs() = %v, %v on a fresh Mac, want true", d, err)
	}
	if _, err := f.Apply(m); err != nil {
		t.Fatal(err)
	}
	arm, load := m.index("/bin/sh -c nohup"), m.index(pfctl+" -E -a")
	if arm < 0 || load < 0 || arm > load {
		t.Fatalf("the timer must be armed before the anchor loads: %q", m.rootCalls)
	}
	if !m.loaded || !m.enabled || !m.jobLoaded {
		t.Fatalf("after Apply: anchor loaded %v, pf enabled %v, job loaded %v", m.loaded, m.enabled, m.jobLoaded)
	}
	if m.timerKilled {
		t.Fatal("Apply disarmed the timer; only Commit may, once the whole proof passed")
	}
	if err := f.Prove(); err != nil {
		t.Fatalf("Prove after Apply: %v", err)
	}
	if err := f.Commit(m); err != nil || !m.timerKilled {
		t.Fatalf("Commit = %v, timer killed %v", err, m.timerKilled)
	}
	if d, err := f.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v after Apply, want false: a second run must change nothing", d, err)
	}
}

// The anchor matches the node's own mesh address as the destination: a LAN
// host can spoof a source, not where its packet arrives.
func TestPFRulesMatchTheDestinationAndBlockOnlyPort22(t *testing.T) {
	r := PFRules(pfMesh)
	if strings.Contains(r, "from 100.64") || !strings.Contains(r, "to 100.64.0.15 port 22") {
		t.Errorf("the pass rule must match the mesh address as destination:\n%s", r)
	}
	for _, line := range strings.Split(strings.TrimSpace(r), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "port 22") {
			t.Errorf("rule %q reaches beyond port 22", line)
		}
	}
	if !strings.Contains(r, "pass in quick on lo0") {
		t.Errorf("loopback must stay open, the proof connects through it:\n%s", r)
	}
}

// `-a` keeps Apple's main ruleset: a plain `pfctl -E -f` would replace it.
func TestPFLaunchDaemonLoadsOnlyItsAnchor(t *testing.T) {
	p := PFLaunchDaemon("/etc/pf.anchors/dotf-ssh")
	if !strings.Contains(p, "<string>-a</string>\n\t\t<string>"+PFAnchor+"</string>") || !strings.Contains(p, "<key>RunAtLoad</key>") {
		t.Errorf("the job must load the anchor with -a at boot:\n%s", p)
	}
}

// An apply over an ssh session that arrived on the LAN would cut it: refused
// before anything changes.
func TestPFRefusesASessionTheAnchorWouldCut(t *testing.T) {
	m := newPFMac(t)
	_, err := m.pf("10.0.0.20 51234 10.0.0.130 22").Apply(m)
	if err == nil || !strings.Contains(err.Error(), "arrived on 10.0.0.130") {
		t.Fatalf("Apply = %v, want a refusal naming the LAN address", err)
	}
	if len(m.rootCalls) != 0 {
		t.Errorf("the refusal ran %q as root", m.rootCalls)
	}
	for _, session := range []string{"", "100.64.0.9 51234 100.64.0.15 22", "::1 51234 ::1 22"} {
		m := newPFMac(t)
		if _, err := m.pf(session).Apply(m); err != nil {
			t.Errorf("session %q: Apply = %v, want it to proceed", session, err)
		}
	}
}

// A session the guard cannot read is refused, never waved through: a server
// address that does not parse, a variable of the wrong shape, and a dotf run
// under sudo, which strips $SSH_CONNECTION.
func TestPFRefusesASessionItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		session string
		sudoed  bool
		want    string
	}{
		{"10.0.0.20 51234 fe80::1%en0 22", false, "could not parse"},
		{"10.0.0.20 51234", false, "could not parse"},
		{"", true, "under sudo"},
	} {
		m := newPFMac(t)
		_, err := m.pfAs(tc.session, tc.sudoed).Apply(m)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("session %q, sudo %v: Apply = %v, want a refusal saying %q", tc.session, tc.sudoed, err, tc.want)
		}
		if len(m.rootCalls) != 0 {
			t.Errorf("session %q, sudo %v: the refusal ran %q as root", tc.session, tc.sudoed, m.rootCalls)
		}
	}
	m := newPFMac(t)
	if _, err := m.pfAs("100.64.0.9 51234 100.64.0.15 22", true).Apply(m); err != nil {
		t.Errorf("a sudo that kept $SSH_CONNECTION: Apply = %v, want it to proceed", err)
	}
}

// A launchd that cannot be read is an error, never a job that is not loaded:
// read as unloaded, a converged Mac would plan pf forever, and Apply's undo
// would boot out a job that was loaded (#2731).
func TestPFStopsWhenLaunchdCannotBeRead(t *testing.T) {
	m := newPFMac(t)
	f := m.pf("")
	if _, err := f.Apply(m); err != nil {
		t.Fatal(err)
	}
	if err := f.Commit(m); err != nil {
		t.Fatal(err)
	}
	m.printErr, m.rootCalls = exitErr(5), nil
	if d, err := f.Differs(); err == nil || !strings.Contains(err.Error(), pfLaunchd) {
		t.Errorf("Differs() = %v, %v; want an error naming %s", d, err, pfLaunchd)
	}
	if _, err := f.Apply(m); err == nil || !strings.Contains(err.Error(), pfLaunchd) {
		t.Errorf("Apply = %v; want an error naming %s", err, pfLaunchd)
	}
	if len(m.rootCalls) != 0 {
		t.Errorf("a failed read ran %q as root", m.rootCalls)
	}
}

// Commit keeps the anchor only when it stopped a timer that had not fired. The
// timer flushes the anchor first, so a kill that fails, or an anchor gone once
// the kill succeeded, means the proof outlived the timer and port 22 is
// unfenced: the run did not converge, and Target undoes it.
func TestPFCommitFailsWhenTheTimerFiredFirst(t *testing.T) {
	for name, fire := range map[string]func(m *pfMac){
		"the timer is gone": func(m *pfMac) { m.killErr, m.loaded, m.jobLoaded = exitErr(1), false, false },
		"it was flushing":   func(m *pfMac) { m.loaded = false },
		"sudo refused kill": func(m *pfMac) { m.killErr = exitErr(1) },
	} {
		m := newPFMac(t)
		f := m.pf("")
		if _, err := f.Apply(m); err != nil {
			t.Fatal(err)
		}
		fire(m)
		if err := f.Commit(m); err == nil || !strings.Contains(err.Error(), "before the proof finished") {
			t.Errorf("%s: Commit = %v, want the fired timer reported", name, err)
		}
	}
}

// A load pf does not take is undone: our anchor flushed, the files put back,
// the timer disarmed, and pf never switched off.
func TestPFFailedLoadFlushesOnlyItsAnchor(t *testing.T) {
	m := newPFMac(t)
	undo, err := m.pf("").Apply(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	checkPFUndone(t, m)

	m = newPFMac(t)
	m.failLoad = true
	undo, err = m.pf("").Apply(m)
	if err == nil || !strings.Contains(err.Error(), "does not hold the port 22 block") {
		t.Fatalf("Apply = %v, want the load reported", err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	checkPFUndone(t, m)
}

func checkPFUndone(t *testing.T, m *pfMac) {
	t.Helper()
	for _, c := range m.rootCalls {
		if strings.Contains(c, "pfctl -d") || strings.HasSuffix(c, " -F all") && !strings.Contains(c, "-a "+PFAnchor) {
			t.Errorf("the undo ran %q, which reaches beyond our anchor", c)
		}
	}
	if m.loaded || m.jobLoaded || !m.timerKilled || len(m.held) != 0 {
		t.Errorf("after the undo: anchor loaded %v, job loaded %v, timer killed %v, references held %v", m.loaded, m.jobLoaded, m.timerKilled, m.held)
	}
	for _, p := range []string{m.anchor, m.plist, m.token} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s survived the undo: %v", p, err)
		}
	}
}

// The timer undoes what a killed run could not: it flushes the anchor and
// keeps the job from loading it again at the next boot.
func TestPFTimerFlushesTheAnchorAndDisablesTheJob(t *testing.T) {
	m := newPFMac(t)
	if _, err := m.pf("").Apply(m); err != nil {
		t.Fatal(err)
	}
	script := m.rootCalls[m.index("/bin/sh -c nohup")]
	for _, want := range []string{"sleep 120", pfctl + " -a " + PFAnchor + " -F all", "launchctl disable " + pfLaunchd} {
		if !strings.Contains(script, want) {
			t.Errorf("the timer %q does not run %q", script, want)
		}
	}
	if strings.Contains(script, "pfctl -d") {
		t.Errorf("the timer switches pf off: %q", script)
	}
	timer := pfTimer(m.token)
	for _, want := range []string{"kern.bootsessionuuid", pfctl + ` -X "$t"`, "< " + m.token, "rm -f " + m.token} {
		if !strings.Contains(timer, want) {
			t.Errorf("the timer %q does not run %q", timer, want)
		}
	}
	if runtime.GOOS == "windows" {
		return
	}
	if out, err := exec.Command("/bin/sh", "-n", "-c", timer).CombinedOutput(); err != nil {
		t.Errorf("the timer does not parse: %v: %s", err, out)
	}
}

// Apply records the reference its `pfctl -E` takes, with the boot it was taken
// in, before the proof: the undo, the timer and the retirement release it from
// that file (#2755). A second apply in the same boot releases the first one
// once it commits, so the step holds one reference, not one per run.
func TestPFRecordsAndReleasesItsEnableReference(t *testing.T) {
	m := newPFMac(t)
	f := m.pf("")
	if _, err := f.Apply(m); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, m.token); got != "1001 BOOT-1\n" {
		t.Fatalf("token file after Apply = %q, want the reference and its boot", got)
	}
	rec := slices.IndexFunc(m.rootCalls, func(c string) bool { return strings.HasPrefix(c, "install -m") && strings.HasSuffix(c, " "+m.token) })
	if load, proof := m.index(pfctl+" -E"), m.index(pfctl+" -s info"); rec < load || rec > proof {
		t.Fatalf("the token must be recorded right after -E, before the proof: %q", m.rootCalls)
	}
	if err := f.Commit(m); err != nil {
		t.Fatal(err)
	}

	f = m.pf("")
	if _, err := f.Apply(m); err != nil {
		t.Fatal(err)
	}
	if !m.held["1001"] || !m.held["1002"] {
		t.Fatalf("before Commit both references must stay held, for the undo: %v", m.held)
	}
	if err := f.Commit(m); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, m.token); got != "1002 BOOT-1\n" || m.held["1001"] || !m.held["1002"] {
		t.Errorf("after the second Commit: token file %q, references held %v; want only 1002", got, m.held)
	}
}

// A token from an earlier boot names nothing of ours: pf's references end at
// reboot, and the same number can be another holder's now. It is dropped from
// the file and never released.
func TestPFNeverReleasesATokenFromAnEarlierBoot(t *testing.T) {
	m := newPFMac(t)
	writeTokenFile(t, m.token, "1001 BOOT-0\n")
	m.held["1001"], m.next = true, 1002 // another holder's, in this boot
	f := m.pf("")
	if _, err := f.Apply(m); err != nil {
		t.Fatal(err)
	}
	if err := f.Commit(m); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, m.token); got != "1002 BOOT-1\n" || !m.held["1001"] {
		t.Errorf("token file %q, references held %v; want 1001 untouched and only 1002 recorded", got, m.held)
	}
}

// The undo of a re-apply releases the reference it took and puts the file back,
// so the reference the earlier run took stays held and recorded.
func TestPFUndoKeepsTheEarlierRunsReference(t *testing.T) {
	m := newPFMac(t)
	writeTokenFile(t, m.token, "1001 BOOT-1\n")
	m.held["1001"], m.next = true, 1002
	undo, err := m.pf("").Apply(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, m.token); got != "1001 BOOT-1\n" || !m.held["1001"] || m.held["1002"] {
		t.Errorf("after the undo: token file %q, references held %v; want only 1001", got, m.held)
	}
}

// A `pfctl -E` whose token cannot be read took a reference nothing can release:
// the apply fails, its undo flushes the anchor, and the error says what is left.
func TestPFFailsWhenTheTokenCannotBeRead(t *testing.T) {
	m := newPFMac(t)
	m.noToken = true
	undo, err := m.pf("").Apply(m)
	if err == nil || !strings.Contains(err.Error(), "until the next reboot") {
		t.Fatalf("Apply = %v, want the unreleasable reference reported", err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if m.loaded {
		t.Error("the undo left the anchor loaded")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeTokenFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The timer's release runs for real against stubs: it releases this boot's
// references, never an earlier boot's, and removes the file.
func TestPFTimerReleasesOnlyThisBootsReferences(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the timer is a POSIX sh script; it runs on macOS only")
	}
	d := t.TempDir()
	log := filepath.Join(d, "calls")
	stub := func(name, body string) string {
		p := filepath.Join(d, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	token := filepath.Join(d, "pf-token")
	writeTokenFile(t, token, "1000 BOOT-0\n1001 BOOT-1\n1002 BOOT-1\n")
	script := strings.NewReplacer(
		"sleep 120", "sleep 0",
		pfctl, stub("pfctl", `echo "pfctl $*" >> `+log),
		sysctl, stub("sysctl", "echo BOOT-1"),
	).Replace(pfTimer(token))
	stub("launchctl", `echo "launchctl $*" >> `+log)
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+d+":/usr/bin:/bin")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("timer: %v: %s", err, out)
	}
	want := "pfctl -a " + PFAnchor + " -F all\npfctl -X 1001\npfctl -X 1002\n" +
		"launchctl bootout " + pfLaunchd + "\nlaunchctl disable " + pfLaunchd + "\n"
	if got := readFile(t, log); got != want {
		t.Errorf("the timer ran\n%s\nwant\n%s", got, want)
	}
	if _, err := os.Stat(token); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the token file survived the timer: %v", err)
	}
}
