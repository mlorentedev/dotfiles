package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

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
	sys.CommandOutput = func(name string, args ...string) (string, error) {
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
