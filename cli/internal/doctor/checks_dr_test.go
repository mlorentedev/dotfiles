package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func drSys(now time.Time) *System {
	return &System{
		Getenv:   func(string) string { return "" },
		LookPath: func(n string) (string, error) { return "/usr/bin/" + n, nil },
		Now:      func() time.Time { return now },
	}
}

// touchAt writes path and backdates it, so "how long since the drill" is a
// property of the fixture rather than of when the suite happens to run.
func touchAt(t *testing.T, path string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

// A drill that never happened is the state this check exists for. It must not be
// silent, and it must not be a FAIL either -- nothing is broken, it is unproven.
func TestDR_NoDrillRecorded_Warns(t *testing.T) {
	cfg := &Config{DotfilesDir: t.TempDir()}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSys(time.Now()), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("an unproven backup is not a broken one; want WARN not FAIL:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "no recovery drill recorded") {
		t.Errorf("want the never-drilled warning, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "guide-secrets-governance.md") {
		t.Errorf("the warning must name the runbook to run, got: %s", buf.String())
	}
}

func TestDR_RecentDrillPasses(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DotfilesDir: dir}
	now := time.Now()
	touchAt(t, drillMarkerPath(cfg), now.Add(-30*24*time.Hour))
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSys(now), cfg, rep)

	if !strings.Contains(buf.String(), "recovery drill run 30 days ago") {
		t.Errorf("want the drill PASS with its age, got: %s", buf.String())
	}
}

// The decay direction. Without this, a drill done once in 2019 reads the same as
// one done last week -- which is the failure this check exists to prevent.
func TestDR_StaleDrillWarns(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DotfilesDir: dir}
	now := time.Now()
	touchAt(t, drillMarkerPath(cfg), now.Add(-400*24*time.Hour))
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSys(now), cfg, rep)

	if !strings.Contains(buf.String(), "last recovery drill was 400 days ago") {
		t.Errorf("want the staleness warning with its age, got: %s", buf.String())
	}
}

// An absent escrow is a known state (#586 item 5 has not shipped), not a
// regression -- SKIP, so it does not train anyone to ignore this section.
func TestDR_AbsentEscrowSkipsNotFails(t *testing.T) {
	cfg := &Config{DotfilesDir: t.TempDir()}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSys(time.Now()), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("absent escrow must not FAIL, got %d", rep.Failures())
	}
	if !strings.Contains(buf.String(), "no DR escrow") {
		t.Errorf("want the escrow SKIP, got: %s", buf.String())
	}
}

func TestDR_StaleEscrowWarns(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DotfilesDir: dir}
	now := time.Now()
	touchAt(t, filepath.Join(dir, "sensitive", "dr", "bitwarden-export.age"), now.Add(-120*24*time.Hour))
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSys(now), cfg, rep)

	if !strings.Contains(buf.String(), "DR escrow is 120 days old") {
		t.Errorf("want the escrow staleness warning, got: %s", buf.String())
	}
}
