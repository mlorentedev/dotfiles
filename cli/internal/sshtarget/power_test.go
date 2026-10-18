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
// and what logind answers for the lid.
type laptop struct {
	t         *testing.T
	dropIn    string
	lid       string // logind's live HandleLidSwitchExternalPower
	hupReads  bool   // whether SIGHUP makes logind read the drop-in
	canReload bool   // what systemd says of systemd-logind's CanReload
	rootCalls []string
}

func newLaptop(t *testing.T) *laptop {
	return &laptop{t: t, dropIn: filepath.Join(t.TempDir(), "logind.conf.d", "50-workstation.conf"),
		lid: "suspend", hupReads: true, canReload: true}
}

func (l *laptop) user(argv ...string) ([]byte, error) {
	switch {
	case argv[0] == "systemctl" && argv[1] == "show":
		if l.canReload {
			return []byte("yes\n"), nil
		}
		return []byte("no\n"), nil
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

// The undo puts back the previous drop-in, read again by logind.
func TestLogindUndoRestoresThePreviousPolicy(t *testing.T) {
	l := newLaptop(t)
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
}

// A mask holds on battery too, so the policy never masks a sleep target: a
// closed lid on battery keeps the distribution's default, suspend.
func TestLogindLeavesBatteryAlone(t *testing.T) {
	l := newLaptop(t)
	if _, err := l.logind().Apply(l); err != nil {
		t.Fatal(err)
	}
	for _, c := range l.rootCalls {
		if strings.Contains(c, "mask") {
			t.Errorf("the apply ran %q; a mask stops suspend on battery as well", c)
		}
	}
	if strings.Contains(LogindPolicy, "HandleLidSwitch=") {
		t.Error("the drop-in sets HandleLidSwitch, the battery case")
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

func TestPowerForDeclaresAPolicyPerOS(t *testing.T) {
	if _, ok := PowerFor("linux", nil, Paths{}).(Logind); !ok {
		t.Error("linux declares the logind policy")
	}
	if _, ok := PowerFor("darwin", nil, Paths{}).(Pmset); !ok {
		t.Error("darwin declares the pmset profile")
	}
	if _, ok := PowerFor("windows", nil, Paths{}).(Powercfg); !ok {
		t.Error("windows declares the powercfg profile")
	}
}

// A logind that cannot reload is never sent SIGHUP, which would kill it. The
// drop-in waits for logind's next start, and the proof does not fail a run
// for what only a restart, which ends the graphical session, could show.
func TestLogindThatCannotReloadIsNeverSignalled(t *testing.T) {
	l := newLaptop(t)
	l.canReload = false
	p := l.logind()
	undo, err := p.Apply(l)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Prove(); err != nil {
		t.Fatalf("Prove = %v; nothing is provable before logind's next start", err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	for _, c := range l.rootCalls {
		if strings.Contains(c, "kill") {
			t.Errorf("ran %q on a logind that cannot reload; SIGHUP would kill it", c)
		}
	}
}
