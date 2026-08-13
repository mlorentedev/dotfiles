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

// testRegistry is a minimal registry covering every backend the counter must
// discriminate: two bw entries among an age entry and a floor entry.
const testRegistry = `version: 1
secrets:
  - id: A_AGE
    plane: app
    backend: age
    age: a.key
    expose: { env: A_AGE }
  - id: B_BW
    plane: app
    backend: bw
    bw: { item: b-item, field: api-key }
    expose: { env: B_BW }
  - id: C_BW
    plane: app
    backend: bw
    bw: { item: c-item, field: api-key }
    expose: { env: C_BW }
  - id: D_FLOOR
    plane: floor
    backend: age-offline
    age: d.key
    expose: { file: { var: D_FLOOR, path: "~/.ssh/id_test", mode: "0600" } }
`

// TestBWBackedSecrets_CountsOnlyBWBackend exercises the PRODUCTION counter, not
// the BWBackedSecrets seam.
//
// Every other test in this file injects a fake count, so before this test the
// real predicate had never executed anywhere: not in CI, and not in the live
// smoke either, since every registry entry is still `age` and the branch is
// therefore unreachable on the real machine. Breaking `s.Backend == "bw"` left
// the whole cli/ suite green — the exact mutant the severity seam exists to
// prevent, surviving unnoticed.
func TestBWBackedSecrets_CountsOnlyBWBackend(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "secrets"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secrets", "registry.yaml"), []byte(testRegistry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", root)

	n, err := bwBackedSecrets()
	if err != nil {
		t.Fatalf("counting a valid registry must not error: %v", err)
	}
	// 2 of 4: age and age-offline must not count, or severity would escalate on
	// a fully un-migrated machine.
	if n != 2 {
		t.Fatalf("expected 2 bw-backed entries (B_BW, C_BW), got %d", n)
	}
}

// The zero case is the one that governs severity today — every entry on age must
// yield 0, or doctor FAILs on machines where nothing depends on Bitwarden.
func TestBWBackedSecrets_ZeroWhenNothingMigrated(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "secrets"), 0o750); err != nil {
		t.Fatal(err)
	}
	noBW := `version: 1
secrets:
  - id: A_AGE
    plane: app
    backend: age
    age: a.key
    expose: { env: A_AGE }
`
	if err := os.WriteFile(filepath.Join(root, "secrets", "registry.yaml"), []byte(noBW), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", root)

	n, err := bwBackedSecrets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("no bw-backed entry may be counted, got %d", n)
	}
}

// Exercises the PRODUCTION CommandOutputBounded closure, not the fake: the
// deadline is the whole point of the seam, and a fake that ignores it would
// prove nothing. doctor is the last step of setup-linux.sh, so an unbounded
// subprocess here hangs a bootstrap.
func TestCommandOutputBounded_KillsAnOverrunningCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no portable sleep binary on Windows; the deadline logic itself is OS-independent")
	}
	sys := realSystem()

	start := time.Now()
	_, err := sys.CommandOutputBounded(150*time.Millisecond, "sleep", "10")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a command outliving its deadline must error, not return success")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("the error must name the timeout, got: %v", err)
	}
	// Generous ceiling: this asserts the process was killed rather than waited
	// out, without being flaky on a loaded CI runner.
	if elapsed > 5*time.Second {
		t.Fatalf("deadline did not kill the process; took %s", elapsed)
	}
}

// The same seam must stay transparent for commands that finish in time.
func TestCommandOutputBounded_PassesThroughFastCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no portable echo binary semantics on Windows")
	}
	out, err := realSystem().CommandOutputBounded(10*time.Second, "echo", "alive")
	if err != nil {
		t.Fatalf("a fast command must not error: %v", err)
	}
	if !strings.Contains(out, "alive") {
		t.Fatalf("output must pass through, got %q", out)
	}
}

// An unreadable registry must surface as an error, not as a silent zero — a
// silent zero is indistinguishable from "nothing migrated" and would downgrade
// severity exactly when the check can no longer tell.
func TestBWBackedSecrets_MissingRegistryErrors(t *testing.T) {
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir())
	if _, err := bwBackedSecrets(); err == nil {
		t.Fatal("a missing registry must error, never report zero")
	}
}

// bwStatusJSON builds a `bw status` payload. Extra upstream fields are included
// deliberately: the check must ignore what it does not consume.
func bwStatusJSON(status, lastSync string) string {
	sync := "null"
	if lastSync != "" {
		sync = `"` + lastSync + `"`
	}
	return `{"serverUrl":null,"lastSync":` + sync +
		`,"userEmail":"op@example.test","userId":"abc","status":"` + status + `"}`
}

// runBWReach drives checkBitwardenReach with a fake System. live is the number
// of registry entries on backend: bw — the input the severity policy keys on.
func runBWReach(t *testing.T, onPath []string, cmdOut map[string]string, live int) (string, int) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, onPath, cmdOut)
	sys.BWBackedSecrets = func() (int, error) { return live, nil }
	checkBitwardenReach(sys, rep)
	rep.Summary()
	return buf.String(), rep.Failures()
}

// Timestamps are relative to fixedTestNow (2026-06-17T12:00:00Z).
const (
	bwSyncFresh = "2026-06-10T12:00:00Z" // 7d — inside the window
	bwSyncStale = "2026-05-03T12:00:00Z" // 45d — the BUG-074 incident's own age
)

func TestBWReach_AbsentBinarySkips(t *testing.T) {
	// The FAIL for a missing bw belongs to checkSecretsTooling; counting it here
	// too would report one defect twice.
	out, fails := runBWReach(t, nil, nil, 0)
	if fails != 0 {
		t.Fatalf("absent bw must not FAIL here (tooling owns it); got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "reach unverifiable") {
		t.Fatalf("expected a skip naming the tooling owner; got:\n%s", out)
	}
}

// The regression that motivated BUG-074: a dead session must be REPORTED. Before
// the fix, this state produced a green "bw ... found" and nothing else.
func TestBWReach_UnauthenticatedIsReported(t *testing.T) {
	cmd := map[string]string{"bw status": bwStatusJSON("unauthenticated", "")}
	out, _ := runBWReach(t, []string{"bw"}, cmd, 0)
	if !strings.Contains(out, "unauthenticated") {
		t.Fatalf("a dead session must be surfaced; got:\n%s", out)
	}
	// The wrong verb is the trap: `bw unlock` prompts for the master password and
	// then fails, which reads as a credential problem.
	if !strings.Contains(out, "bw login") || !strings.Contains(out, "not `bw unlock`") {
		t.Fatalf("must name `bw login` and rule out `bw unlock`; got:\n%s", out)
	}
}

// Severity is keyed to exposure, not to a flat policy. Same vault state, both
// directions, so neither branch can rot unobserved.
func TestBWReach_SeverityFollowsExposure(t *testing.T) {
	cmd := map[string]string{"bw status": bwStatusJSON("unauthenticated", "")}

	out, fails := runBWReach(t, []string{"bw"}, cmd, 0)
	if fails != 0 {
		t.Fatalf("no bw-backed secret ⇒ advisory, not FAIL; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "no backend:bw secret depends on it yet") {
		t.Fatalf("expected the advisory rationale; got:\n%s", out)
	}

	out, fails = runBWReach(t, []string{"bw"}, cmd, 3)
	if fails != 1 {
		t.Fatalf("3 bw-backed secrets ⇒ exactly one FAIL; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "3 registry secret(s)") {
		t.Fatalf("the FAIL must quantify the exposure; got:\n%s", out)
	}
}

// The tier that would actually have caught the incident: status says "locked"
// (healthy-looking) while the token has been rotting for 45 days.
func TestBWReach_StaleSyncWarnsWhileStatusLooksHealthy(t *testing.T) {
	cmd := map[string]string{"bw status": bwStatusJSON("locked", bwSyncStale)}
	out, fails := runBWReach(t, []string{"bw"}, cmd, 0)
	if fails != 0 {
		t.Fatalf("staleness is a WARN, not a FAIL; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "45d ago") || !strings.Contains(out, "BUG-074") {
		t.Fatalf("expected a staleness warning citing the age; got:\n%s", out)
	}
}

// Clock skew must not be laundered into freshness. A sync stamped in the future
// means the clock moved; treating it as fresh would hide a genuinely expired
// token behind the skew, and printing "synced -3d ago" is nonsense.
func TestBWReach_FutureSyncIsSkewNotFreshness(t *testing.T) {
	cmd := map[string]string{"bw status": bwStatusJSON("locked", "2026-06-20T12:00:00Z")} // 3d after fixedTestNow
	out, fails := runBWReach(t, []string{"bw"}, cmd, 0)
	if fails != 0 {
		t.Fatalf("skew is a WARN, not a FAIL; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "in the future") || !strings.Contains(out, "clock") {
		t.Fatalf("expected a clock-skew warning; got:\n%s", out)
	}
	if strings.Contains(out, "-") && strings.Contains(out, "d ago") {
		t.Fatalf("must not report a negative age; got:\n%s", out)
	}
}

func TestBWReach_FreshLockedVaultIsCleanButUnproven(t *testing.T) {
	cmd := map[string]string{"bw status": bwStatusJSON("locked", bwSyncFresh)}
	out, fails := runBWReach(t, []string{"bw"}, cmd, 0)
	if fails != 0 {
		t.Fatalf("a fresh locked vault is not a defect; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "synced 7d ago") {
		t.Fatalf("expected the sync age; got:\n%s", out)
	}
	// A locked vault must never read as proof of reach.
	if strings.Contains(out, "reach verified") {
		t.Fatalf("locked vault must not claim verified reach; got:\n%s", out)
	}
	if !strings.Contains(out, "token not exercised") {
		t.Fatalf("expected an explicit not-exercised note; got:\n%s", out)
	}
}

// The passing direction of the mutation test: a live vault proves reach.
func TestBWReach_UnlockedVaultProvesReach(t *testing.T) {
	cmd := map[string]string{
		"bw status": bwStatusJSON("unlocked", bwSyncFresh),
		"bw sync":   "Syncing complete.",
	}
	out, fails := runBWReach(t, []string{"bw"}, cmd, 5)
	if fails != 0 {
		t.Fatalf("a reachable vault must pass; got %d\n%s", fails, out)
	}
	if !strings.Contains(out, "reach verified (authenticated sync round-trip)") {
		t.Fatalf("expected the round-trip PASS; got:\n%s", out)
	}
}

// The failing direction: unlocked but the server rejects the token — the exact
// shape of the incident, which `bw list` (local cache) would have passed.
func TestBWReach_UnlockedButSyncFailsIsAFail(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, []string{"bw"}, nil)
	sys.BWBackedSecrets = func() (int, error) { return 2, nil }
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "status" {
			return bwStatusJSON("unlocked", bwSyncFresh), nil
		}
		return "invalid_grant\nnode:internal/process/promises:322", errors.New("exit status 1")
	}
	checkBitwardenReach(sys, rep)
	rep.Summary()

	out := buf.String()
	if rep.Failures() != 1 {
		t.Fatalf("an unreachable live SSOT must FAIL; got %d\n%s", rep.Failures(), out)
	}
	// The Node stacktrace must collapse to one line, not flood the report.
	if strings.Contains(out, "node:internal") {
		t.Fatalf("multi-line CLI crash must be collapsed; got:\n%s", out)
	}
	if !strings.Contains(out, "invalid_grant") {
		t.Fatalf("the FAIL must carry the CLI's own reason; got:\n%s", out)
	}
}

func TestBWReach_DegradesOnUnusableStatus(t *testing.T) {
	for name, payload := range map[string]string{
		"not json":          "You are not logged in.",
		"unknown status":    bwStatusJSON("banana", bwSyncFresh),
		"never synced":      bwStatusJSON("locked", ""),
		"unparseable stamp": bwStatusJSON("locked", "yesterday"),
	} {
		t.Run(name, func(t *testing.T) {
			cmd := map[string]string{"bw status": payload}
			out, fails := runBWReach(t, []string{"bw"}, cmd, 0)
			// Unusable input is never evidence of health — but it is not a defect
			// of the vault either, so it warns rather than failing.
			if fails != 0 {
				t.Fatalf("unusable status must not FAIL; got %d\n%s", fails, out)
			}
			if strings.Contains(out, "reach verified") {
				t.Fatalf("unusable status must not claim verified reach; got:\n%s", out)
			}
		})
	}
}

// A registry that cannot be read costs the severity signal; the check must say
// so rather than silently treating the vault as unexposed.
func TestBWReach_UnreadableRegistryDegradesSeverity(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, []string{"bw"}, map[string]string{
		"bw status": bwStatusJSON("unauthenticated", ""),
	})
	sys.BWBackedSecrets = func() (int, error) { return 0, errors.New("open registry.yaml: no such file") }
	checkBitwardenReach(sys, rep)
	rep.Summary()

	out := buf.String()
	if !strings.Contains(out, "severity degraded") {
		t.Fatalf("expected an explicit severity-degraded note; got:\n%s", out)
	}
	if rep.Failures() != 0 {
		t.Fatalf("degraded severity must not FAIL; got %d\n%s", rep.Failures(), out)
	}
}
