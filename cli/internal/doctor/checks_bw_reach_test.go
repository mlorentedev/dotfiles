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
	t.Chdir(root)

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
	t.Chdir(root)

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
	_, _, err := sys.CommandOutputBounded(150*time.Millisecond, "sleep", "10")
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
	out, _, err := realSystem().CommandOutputBounded(10*time.Second, "echo", "alive")
	if err != nil {
		t.Fatalf("a fast command must not error: %v", err)
	}
	if !strings.Contains(out, "alive") {
		t.Fatalf("output must pass through, got %q", out)
	}
}

// The seam must keep the two streams APART. Merging them is what let one line of
// bw startup chatter defeat the whole reach check: `bw` contracts JSON on stdout
// and diagnostics on stderr, and its first invocation on a fresh machine prints
// `Could not find data file, "…/data.json"; creating it instead.` to stderr.
// Asserted against realSystem(), because the fake cannot prove a property of the
// production closure.
func TestCommandOutputBounded_KeepsStreamsSeparate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell redirection")
	}
	stdout, stderr, err := realSystem().CommandOutputBounded(
		10*time.Second, "sh", "-c", "echo to-stdout; echo to-stderr >&2")
	if err != nil {
		t.Fatalf("command must succeed: %v", err)
	}
	if !strings.Contains(stdout, "to-stdout") || strings.Contains(stdout, "to-stderr") {
		t.Fatalf("stdout must carry only stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "to-stderr") || strings.Contains(stderr, "to-stdout") {
		t.Fatalf("stderr must carry only stderr, got %q", stderr)
	}
}

// An unreadable registry must surface as an error, not as a silent zero — a
// silent zero is indistinguishable from "nothing migrated" and would downgrade
// severity exactly when the check can no longer tell.
func TestBWBackedSecrets_MissingRegistryErrors(t *testing.T) {
	t.Setenv("DOTFILES_REPO_DIR", t.TempDir())
	t.Chdir(t.TempDir())
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

// Every other test in this file calls checkBitwardenReach directly, so deleting
// its registration in Run() (doctor.go) left the entire cli/ suite green — 13
// packages ok, exit 0. A Run() refactor or a bad merge resolution could drop the
// whole reach section and CI would not notice: the same class as #898 (a check
// never observed failing is not evidence) that this spec exists to fix, one
// level up from round 1's surviving producer mutant.
//
// The section header is emitted before the has("bw") skip, so this holds whether
// or not bw is installed on the machine running the tests.
func TestRun_RegistersTheBitwardenReachSection(t *testing.T) {
	home := t.TempDir()
	sys := newSys(nil, nil, nil)
	sys.Getenv = func(k string) string {
		if k == "HOME" || k == "USERPROFILE" {
			return home
		}
		return ""
	}

	var out bytes.Buffer
	if _, err := Run(Options{Out: &out, System: sys, StartDir: home, Verbose: true}); err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	if !strings.Contains(out.String(), "Bitwarden reach") {
		t.Fatal("full-mode doctor must run the Bitwarden reach section — it is unregistered in Run()")
	}
}

// bw prints diagnostics on stderr while contracting JSON on stdout. Reading a
// MERGED stream made one line of chatter fail the json.Unmarshal, and the check
// then returned early — skipping all three tiers and reporting only "no
// parseable JSON". The state is not exotic: it is bw's first invocation on a
// machine, i.e. exactly what `dotf doctor` triggers at the end of setup-linux.sh
// on a freshly provisioned box. Post-migration it also silently downgrades the
// AC1 FAIL to a WARN, defeating the escalation the severity policy exists for.
func TestBWReach_ToleratesCLIChatterOnStderr(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, []string{"bw"}, nil)
	sys.BWBackedSecrets = func() (int, error) { return 3, nil }
	sys.CommandOutputBounded = func(_ time.Duration, _ string, _ ...string) (string, string, error) {
		return bwStatusJSON("unauthenticated", ""),
			`Could not find data file, "/home/u/.config/Bitwarden CLI/data.json"; creating it instead.`,
			nil
	}
	checkBitwardenReach(sys, rep)

	out := buf.String()
	if strings.Contains(out, "no parseable JSON") {
		t.Fatalf("stderr chatter must not defeat the stdout parse; got:\n%s", out)
	}
	// The tier must still reach its verdict: exposure is 3, so this is a FAIL.
	if rep.Failures() != 1 {
		t.Fatalf("tier 1 must still evaluate and FAIL at exposure 3; got %d\n%s", rep.Failures(), out)
	}
	if !strings.Contains(out, "bw login") {
		t.Fatalf("the FAIL must still name the recovery verb; got:\n%s", out)
	}
}

// Severity follows exposure on the sync tier too, not only on the
// unauthenticated one. With every registry entry still on age, an unreachable
// vault breaks nothing — and a flat FAIL would exit doctor 1 merely because the
// machine is offline or the 45s deadline fired. doctor's own precedent for an
// unreachable remote is a WARN (checks_pat.go, api.github.com).
func TestBWReach_SyncFailureIsAdvisoryAtZeroExposure(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, []string{"bw"}, nil)
	sys.BWBackedSecrets = func() (int, error) { return 0, nil }
	sys.CommandOutputBounded = func(_ time.Duration, _ string, args ...string) (string, string, error) {
		if len(args) > 0 && args[0] == "status" {
			return bwStatusJSON("unlocked", bwSyncFresh), "", nil
		}
		return "", "", errors.New("bw timed out after 45s")
	}
	checkBitwardenReach(sys, rep)

	out := buf.String()
	if rep.Failures() != 0 {
		t.Fatalf("nothing resolves through bw yet, so an unreachable vault must not FAIL; got %d\n%s",
			rep.Failures(), out)
	}
	if !strings.Contains(out, "no backend:bw secret depends on it yet") {
		t.Fatalf("the WARN must say why it is advisory; got:\n%s", out)
	}
	// The reason still has to reach the operator, or the WARN is unactionable.
	if !strings.Contains(out, "timed out") {
		t.Fatalf("the WARN must carry the underlying failure; got:\n%s", out)
	}
}

// The failing direction: unlocked but the server rejects the token — the exact
// shape of the incident, which `bw list` (local cache) would have passed.
func TestBWReach_UnlockedButSyncFailsIsAFail(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(nil, []string{"bw"}, nil)
	sys.BWBackedSecrets = func() (int, error) { return 2, nil }
	sys.CommandOutputBounded = func(_ time.Duration, name string, args ...string) (string, string, error) {
		if len(args) > 0 && args[0] == "status" {
			return bwStatusJSON("unlocked", bwSyncFresh), "", nil
		}
		return "", "invalid_grant\nnode:internal/process/promises:322", errors.New("exit status 1")
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

// runBWReachWithEnv is runBWReach with an injected environment and a
// CommandOutputBounded that can fail `bw sync` on demand — the flag's blast
// radius is the property under test, so every branch after `unauthenticated`
// has to be reachable with the flag on.
func runBWReachWithEnv(t *testing.T, env map[string]string, status, lastSync string, live int, syncErr error) (string, *Report) {
	t.Helper()
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := newSys(env, []string{"bw"}, nil)
	sys.BWBackedSecrets = func() (int, error) { return live, nil }
	sys.CommandOutputBounded = func(_ time.Duration, _ string, args ...string) (string, string, error) {
		if len(args) > 0 && args[0] == "status" {
			return bwStatusJSON(status, lastSync), "", nil
		}
		if syncErr != nil {
			return "", "invalid_grant", syncErr
		}
		return "Syncing complete.", "", nil
	}
	checkBitwardenReach(sys, rep)
	rep.Summary()
	return buf.String(), rep
}

// TEST-005 (#1313): the CI runner has no Bitwarden identity by design, and the
// gate used to allow-list the resulting FAIL — a permanent condition on a list
// whose contract is "runner-only failures with a fix pending". A DECLARED flag
// turns that one state into a SKIP that says why. The table pins the blast
// radius: only the unauthenticated branch reads the flag; a locked vault, an
// unlocked one and a failing sync report exactly as they do without it. The
// last row pins the other half of "declared": CI-shaped variables without the
// flag change nothing.
func TestBWReach_DeclaredNoIdentity(t *testing.T) {
	declared := map[string]string{noIdentityEnv: "1"}
	cases := []struct {
		name     string
		env      map[string]string // nil → the flag is declared
		status   string
		lastSync string
		live     int
		syncErr  error
		wantFail int
		wantWarn int
		wantSub  string
		notSub   string
	}{
		{
			name:   "unauthenticated at exposure 28 → SKIP naming the declaration, no FAIL",
			status: "unauthenticated", live: 28,
			wantFail: 0, wantWarn: 0,
			wantSub: "no Bitwarden identity on this runner (declared via " + noIdentityEnv + ")",
			notSub:  "bw login",
		},
		{
			name:   "unauthenticated at exposure 0 → SKIP, not the advisory WARN",
			status: "unauthenticated", live: 0,
			wantFail: 0, wantWarn: 0,
			wantSub: "reach verified on real boxes only",
			notSub:  "no backend:bw secret depends on it yet",
		},
		{
			name:   "locked vault → unchanged: sync age evaluated, token not exercised",
			status: "locked", lastSync: bwSyncFresh, live: 28,
			wantFail: 0, wantWarn: 0,
			wantSub: "token not exercised",
			notSub:  "declared",
		},
		{
			name:   "locked vault with a stale sync → the BUG-074 WARN still fires",
			status: "locked", lastSync: bwSyncStale, live: 28,
			wantFail: 0, wantWarn: 1,
			wantSub: "BUG-074",
			notSub:  "declared",
		},
		{
			name:   "unlocked vault, sync fails at exposure 2 → still exactly one FAIL",
			status: "unlocked", lastSync: bwSyncFresh, live: 2, syncErr: errors.New("exit status 1"),
			wantFail: 1, wantWarn: 0,
			wantSub: "invalid_grant",
			notSub:  "declared",
		},
		{
			name:   "unlocked vault, sync succeeds → reach verified as before",
			status: "unlocked", lastSync: bwSyncFresh, live: 2,
			wantFail: 0, wantWarn: 0,
			wantSub: "reach verified (authenticated sync round-trip)",
			notSub:  "declared",
		},
		{
			// Declared, never sniffed: CI-shaped variables without the flag leave
			// an unauthenticated vault at exposure FAILing, as on a real box.
			name:   "CI=true and GITHUB_ACTIONS=true without the flag → still FAILs, nothing declared",
			env:    map[string]string{"CI": "true", "GITHUB_ACTIONS": "true"},
			status: "unauthenticated", live: 28,
			wantFail: 1, wantWarn: 0,
			wantSub: "bw login",
			notSub:  "declared",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := tc.env
			if env == nil {
				env = declared
			}
			out, rep := runBWReachWithEnv(t, env, tc.status, tc.lastSync, tc.live, tc.syncErr)
			if rep.Failures() != tc.wantFail {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFail, out)
			}
			if rep.Warnings() != tc.wantWarn {
				t.Fatalf("warnings = %d, want %d\n%s", rep.Warnings(), tc.wantWarn, out)
			}
			if !strings.Contains(out, tc.wantSub) {
				t.Fatalf("output missing %q\n%s", tc.wantSub, out)
			}
			if strings.Contains(out, tc.notSub) {
				t.Fatalf("output must not contain %q\n%s", tc.notSub, out)
			}
		})
	}
}
