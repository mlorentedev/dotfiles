package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// drSys builds a System for the DR check with NO bw-backed secrets, i.e. the
// pre-migration machine where an absent escrow costs nothing. Severity is keyed
// to that count (#997), so most tests need to state it; this is the neutral one.
func drSys(now time.Time) *System { return drSysExposed(now, 0, nil) }

// drSysExposed states the exposure explicitly: how many registry entries resolve
// through Bitwarden, and whether the registry could be read at all. Those two
// are the only inputs that move this check's severity.
func drSysExposed(now time.Time, bwBacked int, regErr error) *System {
	return &System{
		Getenv:          func(string) string { return "" },
		LookPath:        func(n string) (string, error) { return "/usr/bin/" + n, nil },
		Now:             func() time.Time { return now },
		BWBackedSecrets: func() (int, error) { return bwBacked, regErr },
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

// With nothing resolving through Bitwarden, an absent escrow costs nothing --
// every secret still has its age copy on disk. SKIP, because a red doctor for a
// harmless condition is how operators are trained to ignore red doctors.
func TestDR_AbsentEscrowNoExposure_Skips(t *testing.T) {
	cfg := &Config{DotfilesDir: t.TempDir()}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSysExposed(time.Now(), 0, nil), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("absent escrow with no bw-backed secrets must not FAIL, got %d:\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "no DR escrow") {
		t.Errorf("want the escrow SKIP, got: %s", buf.String())
	}
}

// The defect this issue exists for (#997). Once a secret resolves through
// Bitwarden, `migrate` has dropped its age: pointer (#971) -- so the remote
// account is the ONLY copy, and an absent escrow is a single point of total
// loss. Reported for months as "nothing to check here".
func TestDR_AbsentEscrowWithExposure_Fails(t *testing.T) {
	cfg := &Config{DotfilesDir: t.TempDir()}
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSysExposed(time.Now(), 26, nil), cfg, rep)

	if rep.Failures() != 1 {
		t.Fatalf("absent escrow with bw-backed secrets must FAIL, got %d failure(s):\n%s", rep.Failures(), buf.String())
	}
	out := buf.String()
	// The count is the whole argument: "no backup" is abstract, "26 secrets
	// exist only on a remote server" is what makes an operator act.
	if !strings.Contains(out, "26") {
		t.Errorf("the FAIL must name how many secrets have no local copy, got: %s", out)
	}
	// The remedy must be the invocation that actually works. A bare
	// `dotf secrets backup` fails with "Vault is locked": its export path has no
	// bw serve endpoint (verified against the shipped @bitwarden/cli bundle --
	// the serve router has no export route), so it needs a CLI session. This is
	// permanent, not pending #993, which is why the message names it plainly.
	if !strings.Contains(out, "BW_SESSION") {
		t.Errorf("the FAIL must name the invocation that actually creates an escrow, got: %s", out)
	}
	if strings.Contains(out, "#993") {
		t.Errorf("the remedy is permanent, not blocked on #993; citing it dates the message: %s", out)
	}
}

// A stale escrow stays a WARN even under exposure: a 120-day-old copy is a
// degraded backup, not an absent one, and the two must not read alike.
func TestDR_StaleEscrowWithExposure_StillWarns(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DotfilesDir: dir}
	now := time.Now()
	touchAt(t, filepath.Join(dir, "sensitive", "dr", "bitwarden-export.age"), now.Add(-120*24*time.Hour))
	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSysExposed(now, 26, nil), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("a stale escrow is degraded, not absent; want WARN not FAIL:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "DR escrow is 120 days old") {
		t.Errorf("want the escrow staleness warning, got: %s", buf.String())
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
