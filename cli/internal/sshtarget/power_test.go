package sshtarget

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// laptop is a fake systemd host for the power policy: the drop-in on disk,
// which sleep targets are masked, and what logind answers for the lid.
type laptop struct {
	t         *testing.T
	dropIn    string
	masked    map[string]bool
	lid       string // logind's live HandleLidSwitchExternalPower
	hupReads  bool   // whether SIGHUP makes logind read the drop-in
	rootCalls []string
}

func newLaptop(t *testing.T) *laptop {
	return &laptop{t: t, dropIn: filepath.Join(t.TempDir(), "logind.conf.d", "50-workstation.conf"),
		masked: map[string]bool{}, lid: "suspend", hupReads: true}
}

func (l *laptop) user(argv ...string) ([]byte, error) {
	switch {
	case argv[0] == "systemctl" && argv[1] == "is-enabled":
		if l.masked[argv[2]] {
			return []byte("masked\n"), nil
		}
		return []byte("static\n"), nil
	case argv[0] == "busctl":
		return []byte(`s "` + l.lid + `"` + "\n"), nil
	}
	l.t.Fatalf("unexpected user command %v", argv)
	return nil, nil
}

func (l *laptop) Ready() error { return nil }

func (l *laptop) Run(argv ...string) ([]byte, error) {
	l.rootCalls = append(l.rootCalls, strings.Join(argv, " "))
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
	case slices.Equal(argv[:2], []string{"systemctl", "mask"}):
		for _, u := range argv[2:] {
			l.masked[u] = true
		}
	case slices.Equal(argv[:2], []string{"systemctl", "unmask"}):
		for _, u := range argv[2:] {
			delete(l.masked, u)
		}
	case slices.Equal(argv[:2], []string{"systemctl", "kill"}):
		if l.hupReads {
			l.lid = "suspend"
			if b, err := os.ReadFile(l.dropIn); err == nil && strings.Contains(string(b), "HandleLidSwitchExternalPower=ignore") {
				l.lid = "ignore"
			}
		}
	}
	return nil, nil
}

func (l *laptop) logind() Logind { return Logind{Run: l.user, DropIn: l.dropIn} }

func TestLogindAppliesAndProvesTheLidPolicy(t *testing.T) {
	l := newLaptop(t)
	p := l.logind()
	if d, err := p.Differs(); err != nil || !d {
		t.Fatalf("Differs() = %v, %v on a fresh laptop, want true", d, err)
	}
	if _, err := p.Apply(l); err != nil {
		t.Fatal(err)
	}
	if err := p.Prove(); err != nil {
		t.Fatalf("Prove after Apply: %v", err)
	}
	if d, err := p.Differs(); err != nil || d {
		t.Errorf("Differs() = %v, %v after Apply, want false: a second run must change nothing", d, err)
	}
}

// The undo puts back exactly what the run changed: the previous drop-in, read
// again by logind, and only the targets this run masked.
func TestLogindUndoRestoresOnlyWhatTheRunChanged(t *testing.T) {
	l := newLaptop(t)
	l.masked["hibernate.target"] = true // the owner had masked it already
	undo, err := l.logind().Apply(l)
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.dropIn); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the drop-in survived the undo: %v", err)
	}
	if l.lid != "suspend" {
		t.Errorf("logind answers %q after the undo, want its previous policy", l.lid)
	}
	if want := map[string]bool{"hibernate.target": true}; !mapsEqual(l.masked, want) {
		t.Errorf("masked after undo = %v, want only what was masked before: %v", l.masked, want)
	}
}

// A logind that does not read the drop-in on SIGHUP is reported, naming the
// restart; the apply never restarts it, because that ends the sessions.
func TestLogindProofFailsWhenLogindKeepsTheOldPolicy(t *testing.T) {
	l := newLaptop(t)
	l.hupReads = false
	p := l.logind()
	if _, err := p.Apply(l); err != nil {
		t.Fatal(err)
	}
	err := p.Prove()
	if err == nil || !strings.Contains(err.Error(), "systemctl restart systemd-logind") {
		t.Fatalf("Prove = %v, want it to name the restart", err)
	}
	for _, c := range l.rootCalls {
		if strings.Contains(c, "restart") {
			t.Errorf("the apply ran %q; restarting logind ends the graphical session", c)
		}
	}
}

func TestLogindProofFailsWhileASleepTargetIsUnmasked(t *testing.T) {
	l := newLaptop(t)
	p := l.logind()
	if _, err := p.Apply(l); err != nil {
		t.Fatal(err)
	}
	delete(l.masked, "suspend.target")
	if err := p.Prove(); err == nil || !strings.Contains(err.Error(), "suspend.target") {
		t.Fatalf("Prove = %v, want it to name suspend.target", err)
	}
}

func TestPowerForDeclaresAPolicyOnLinuxOnly(t *testing.T) {
	if _, ok := PowerFor("linux", nil, Paths{}).(Logind); !ok {
		t.Error("linux declares the logind policy")
	}
	if p := PowerFor("darwin", nil, Paths{}); p != nil {
		t.Errorf("darwin → %T; pmset is not declared yet", p)
	}
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
