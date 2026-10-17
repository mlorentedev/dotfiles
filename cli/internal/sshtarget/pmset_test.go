package sshtarget

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// macmini is the macmini's `pmset -g custom` as read on 2026-10-17, before the
// profile: autorestart and disksleep differ, everything else already matches.
var macmini = map[string]string{
	"Sleep On Power Button": "1", "autorestartatconnect": "0", "lowpowermode": "0",
	"standby": "0", "ttyskeepawake": "1", "powernap": "1", "displaysleep": "15",
	"womp": "1", "networkoversleep": "0", "sleep": "0", "tcpkeepalive": "1",
	"autorestart": "0", "disksleep": "10",
}

// fakeMac answers `pmset -g custom` from its AC values and applies `pmset -c`
// run as root. battery, when set, is a Battery Power section printed first.
type fakeMac struct {
	t         *testing.T
	ac        map[string]string
	battery   map[string]string
	failSet   bool // `pmset -c` sets its first pair, then fails
	rootCalls [][]string
}

func newFakeMac(t *testing.T, ac map[string]string) *fakeMac {
	m := &fakeMac{t: t, ac: map[string]string{}}
	for k, v := range ac {
		m.ac[k] = v
	}
	return m
}

func section(name string, vals map[string]string) string {
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(name + ":\n")
	for _, k := range keys {
		fmt.Fprintf(&b, " %-20s %s\n", k, vals[k])
	}
	return b.String()
}

func (m *fakeMac) user(argv ...string) ([]byte, error) {
	if !slices.Equal(argv, []string{"pmset", "-g", "custom"}) {
		m.t.Fatalf("unexpected user command %v", argv)
	}
	out := ""
	if m.battery != nil {
		out += section("Battery Power", m.battery)
	}
	return []byte(out + section("AC Power", m.ac)), nil
}

func (m *fakeMac) Ready() error { return nil }

func (m *fakeMac) Run(argv ...string) ([]byte, error) {
	m.rootCalls = append(m.rootCalls, argv)
	if len(argv) < 4 || argv[0] != "pmset" || argv[1] != "-c" || len(argv)%2 != 0 {
		m.t.Fatalf("unexpected root command %v", argv)
	}
	for i := 2; i < len(argv); i += 2 {
		m.ac[argv[i]] = argv[i+1]
		if m.failSet {
			m.failSet = false
			return []byte("pmset: write failed"), errors.New("exit status 1")
		}
	}
	return nil, nil
}

func TestPmsetSetsOnlyWhatDiffersOnACAndProvesIt(t *testing.T) {
	m := newFakeMac(t, macmini)
	p := Pmset{Run: m.user}
	if d, err := p.Differs(); err != nil || !d {
		t.Fatalf("Differs() = %v, %v on the macmini, want true", d, err)
	}
	if _, err := p.Apply(m); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"pmset", "-c", "disksleep", "0", "autorestart", "1"}}
	if !slices.EqualFunc(m.rootCalls, want, slices.Equal[[]string]) {
		t.Errorf("root ran %v, want one `pmset -c` with only the two differing keys %v", m.rootCalls, want)
	}
	if err := p.Prove(); err != nil {
		t.Fatalf("Prove after Apply: %v", err)
	}
	if d, err := p.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v after Apply, want false: a second run must change nothing", d, err)
	}
	if m.ac["displaysleep"] != "15" {
		t.Errorf("displaysleep = %s, want it left at 15", m.ac["displaysleep"])
	}
}

func TestPmsetConvergedRunsNothingAsRoot(t *testing.T) {
	ac := map[string]string{}
	for _, s := range PmsetProfile {
		ac[s.Key] = s.Value
	}
	m := newFakeMac(t, ac)
	if _, err := (Pmset{Run: m.user}).Apply(m); err != nil {
		t.Fatal(err)
	}
	if len(m.rootCalls) != 0 {
		t.Errorf("a converged Mac ran %v as root", m.rootCalls)
	}
}

// The undo sets back exactly the values read before the apply.
func TestPmsetUndoRestoresThePreviousValues(t *testing.T) {
	m := newFakeMac(t, macmini)
	undo, err := (Pmset{Run: m.user}).Apply(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	for k, v := range macmini {
		if m.ac[k] != v {
			t.Errorf("%s = %s after the undo, want %s", k, m.ac[k], v)
		}
	}
}

// pmset writes key by key, so a failure part way returns an undo that puts
// back whatever it reached.
func TestPmsetFailedApplyReturnsItsUndo(t *testing.T) {
	m := newFakeMac(t, macmini)
	m.failSet = true
	undo, err := (Pmset{Run: m.user}).Apply(m)
	if err == nil {
		t.Fatal("Apply succeeded on a failing pmset")
	}
	if m.ac["disksleep"] != "0" {
		t.Fatalf("the fake did not set the first pair before failing")
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if m.ac["disksleep"] != "10" {
		t.Errorf("disksleep = %s after the undo, want 10", m.ac["disksleep"])
	}
}

// Battery is never read or set: a laptop that never sleeps on battery drains
// to a hard power-off. A key differing on battery only leaves the plan clean.
func TestPmsetLeavesBatteryAlone(t *testing.T) {
	ac := map[string]string{}
	for _, s := range PmsetProfile {
		ac[s.Key] = s.Value
	}
	m := newFakeMac(t, ac)
	m.battery = map[string]string{"sleep": "1", "disksleep": "10", "autorestart": "0"}
	p := Pmset{Run: m.user}
	if d, err := p.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v with only battery differing, want false", d, err)
	}
	for _, c := range m.rootCalls {
		if slices.Contains(c, "-b") || slices.Contains(c, "-a") {
			t.Errorf("root ran %v, which reaches battery", c)
		}
	}
}

// A key the hardware does not report on AC is not this machine's to set.
func TestPmsetSkipsAKeyTheHardwareDoesNotReport(t *testing.T) {
	ac := map[string]string{"sleep": "0", "disksleep": "0", "autorestart": "1", "powernap": "1"}
	m := newFakeMac(t, ac) // no womp: a Mac without Ethernet
	p := Pmset{Run: m.user}
	if d, err := p.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v without womp, want false", d, err)
	}
	if err := p.Prove(); err != nil {
		t.Errorf("Prove without womp: %v", err)
	}
}

func TestPmsetProofNamesWhatStillDiffers(t *testing.T) {
	m := newFakeMac(t, macmini)
	err := (Pmset{Run: m.user}).Prove()
	if err == nil || !strings.Contains(err.Error(), "autorestart 0 (want 1)") || !strings.Contains(err.Error(), "disksleep 10 (want 0)") {
		t.Fatalf("Prove = %v, want it to name autorestart and disksleep", err)
	}
}

func TestPmsetRefusesOutputWithoutAnACSection(t *testing.T) {
	p := Pmset{Run: func(...string) ([]byte, error) { return []byte("Battery Power:\n sleep 1\n"), nil }}
	if _, err := p.Differs(); err == nil || !strings.Contains(err.Error(), "no AC Power section") {
		t.Fatalf("Differs = %v, want it to refuse output with no AC section", err)
	}
}
