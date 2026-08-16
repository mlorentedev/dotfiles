package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// lineContaining returns the first report line mentioning substr, so a case can
// assert the status tag of the line it cares about. The DR section emits two
// independent lines (escrow, drill) and a bare Contains over the whole buffer
// would happily match the drill's WARN while the escrow line said something
// else entirely.
func lineContaining(out, substr string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

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

// The escrow severity rule (#997), one case per branch.
//
// Asserted on the STATUS TAG rather than on prose: the tag is the contract
// (it decides doctor's exit code), the wording is not. Remediation text is
// pinned only for the FAIL case, where the message is the deliverable -- an
// operator who cannot act on it has been told nothing.
func TestDR_EscrowSeverityFollowsExposure(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		name     string
		bwBacked int
		regErr   error
		wantTag  string
		wantFail int
	}{
		{
			// Every secret still has its age copy on disk, so an absent escrow
			// costs nothing. Staying quiet here is what keeps the FAIL credible.
			name: "no exposure: absence costs nothing", bwBacked: 0, wantTag: "[SKIP]",
		},
		{
			// The defect #997 exists for: `migrate` drops the age: pointer (#971),
			// so for these the remote account is the only copy in existence.
			name: "exposed: the only copy is remote", bwBacked: 28, wantTag: "[FAIL]", wantFail: 1,
		},
		{
			// Exposure unknown. Must not claim "nothing depends on it" (it was
			// never checked) and must not go red on a healthy machine -- doctor
			// run from any other git repo lands here via env.RepoDir's walk-up.
			name: "registry unreadable: exposure unknown", regErr: errors.New("open registry.yaml: no such file"), wantTag: "[WARN]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DotfilesDir: t.TempDir()}
			var buf bytes.Buffer
			rep := capture(&buf)

			checkDisasterRecovery(drSysExposed(now, tt.bwBacked, tt.regErr), cfg, rep)

			out := buf.String()
			escrowLine := lineContaining(out, "DR escrow")
			if !strings.Contains(escrowLine, tt.wantTag) {
				t.Errorf("want %s on the escrow line, got: %s", tt.wantTag, escrowLine)
			}
			if rep.Failures() != tt.wantFail {
				t.Errorf("want %d failure(s), got %d:\n%s", tt.wantFail, rep.Failures(), out)
			}
			if tt.wantTag != "[FAIL]" {
				return
			}
			// The count is the whole argument: "no backup" is abstract, "28
			// secrets exist only on a remote server" is what makes someone act.
			if !strings.Contains(escrowLine, "28") {
				t.Errorf("the FAIL must name how many secrets have no local copy, got: %s", escrowLine)
			}
			// A bare `dotf secrets backup` fails with "Vault is locked": its
			// export path has no bw serve endpoint (verified against the shipped
			// @bitwarden/cli bundle -- the serve router has no export route), so
			// it needs a CLI session. Permanent, not pending #993, which is why
			// the message names it plainly and must not cite that issue.
			if !strings.Contains(escrowLine, "BW_SESSION") {
				t.Errorf("the FAIL must name the invocation that actually creates an escrow, got: %s", escrowLine)
			}
			if strings.Contains(escrowLine, "#993") {
				t.Errorf("the remedy is permanent, not blocked on #993; citing it dates the message: %s", escrowLine)
			}
		})
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

	checkDisasterRecovery(drSysExposed(now, 28, nil), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("a stale escrow is degraded, not absent; want WARN not FAIL:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "DR escrow is 120 days old") {
		t.Errorf("want the escrow staleness warning, got: %s", buf.String())
	}
}

// A stat error is not proof of absence. Under exposure the absent branch FAILs
// with "these N secrets have no local copy" -- a claim the check has not earned
// when the escrow may be sitting there behind a permission error. Asserting the
// unverified is the exact failure #997 is about, so it must not appear in its fix.
func TestDR_UnreadableEscrowWarnsRatherThanClaimingAbsence(t *testing.T) {
	if runtime.GOOS == "windows" {
		// chmod(0) does not deny the owner on Windows, so the parent stays
		// readable and this exercises nothing. Gated rather than reshaped: the
		// alternatives (a file as the parent dir) map to path-not-found there
		// and would silently test the ABSENT branch instead of this one.
		t.Skip("directory permissions do not gate stat on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions, so the escrow stays readable")
	}
	dir := t.TempDir()
	cfg := &Config{DotfilesDir: dir}
	drDir := filepath.Join(dir, "sensitive", "dr")
	touchAt(t, filepath.Join(drDir, "bitwarden-export.age"), time.Now())
	if err := os.Chmod(drDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(drDir, 0o755) }) // else TempDir cleanup fails

	var buf bytes.Buffer
	rep := capture(&buf)

	checkDisasterRecovery(drSysExposed(time.Now(), 28, nil), cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("an unreadable escrow is not a proven-absent one; want WARN not FAIL:\n%s", buf.String())
	}
	line := lineContaining(buf.String(), "DR escrow")
	if !strings.Contains(line, "[WARN]") || !strings.Contains(line, "cannot inspect") {
		t.Errorf("want the inspection WARN, got: %s", line)
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
