package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// lookPathFor builds the LookPath seam: the named binaries resolve, everything
// else reports "not found". The check keys three distinct outcomes off this
// (no systemctl, no hive, both present), so each test states its host exactly.
func lookPathFor(present ...string) func(string) (string, error) {
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

// unitSeam fakes `systemctl show -p ExecStart -p Environment` in the format it
// REALLY prints. ExecStart is not a bare command but a record:
//
//	ExecStart={ path=… ; argv[]=… ; ignore_errors=no ; start_time=[n/a] ; … }
//
// The first version of these tests fed a plain command string, so all seven
// passed against a format systemd never emits. Captured verbatim from
// `systemctl --user show hive.service` on msi, 2026-08-24 — the same defect
// lesson 230 in this very PR is about, caught by the reviewer rather than by me.
//
// An empty execStart reproduces an unknown unit, where systemctl prints no
// ExecStart line at all.
func unitSeam(execStart, environment string) func(string, ...string) (string, error) {
	var out string
	if execStart != "" {
		bin, _, _ := strings.Cut(execStart, " ")
		out = "ExecStart={ path=" + bin + " ; argv[]=" + execStart +
			" ; ignore_errors=no ; start_time=[n/a] ; stop_time=[n/a] ; pid=0 ; code=(null) ; status=0/0 }\n"
	}
	out += "Environment=" + environment + "\n"
	return func(_ string, _ ...string) (string, error) { return out, nil }
}

// failingSeam makes the systemctl shell-out fail, which is a DIFFERENT state
// from an empty reading: an unanswered question must never be reported as a
// clean bill of health.
func failingSeam(err error) func(string, ...string) (string, error) {
	return func(_ string, _ ...string) (string, error) { return "", err }
}

// The shape a correctly deployed drop-in produces, used wherever a test needs
// "everything is fine except the one thing under test".
const (
	goodExecStart = "/home/u/.local/bin/dotf secrets run --only NAN_API_KEY -- /home/u/.local/bin/hive serve"
	goodEnv       = "HIVE_WORKER_BASE_URL=https://api.nan.builders/v1"
)

// The record format, pinned against output captured verbatim from this machine
// rather than against what the fake produces — otherwise the fake and the parser
// can agree with each other forever while both disagree with systemd.
func TestExecArgv_ParsesTheRecordSystemdActuallyPrints(t *testing.T) {
	const captured = `{ path=/home/manu/.local/bin/hive ; argv[]=/home/manu/.local/bin/hive serve ; ignore_errors=no ; start_time=[n/a] ; stop_time=[n/a] ; pid=0 ; code=(null) ; status=0/0 }`

	if got, want := execArgv(captured), "/home/manu/.local/bin/hive serve"; got != want {
		t.Errorf("execArgv = %q, want %q", got, want)
	}
	// A record whose argv is fine must not be satisfied by the trailing status
	// fields, nor truncated by them.
	if strings.Contains(execArgv(captured), "ignore_errors") {
		t.Error("execArgv leaked a trailing record field into the command line")
	}
	// Plain values pass through: `--value` output and the empty string for an
	// unknown unit both take that path.
	if got := execArgv("/u/bin/dotf secrets run -- /u/bin/hive serve"); got != "/u/bin/dotf secrets run -- /u/bin/hive serve" {
		t.Errorf("a plain command must pass through unchanged, got %q", got)
	}
	if got := execArgv(""); got != "" {
		t.Errorf("empty must stay empty, got %q", got)
	}
}

// The criterion's case, and the one measured on this machine: the daemon is
// supervised and running, every probe says hive is present, and its ExecStart
// injects no credential — so it can serve nothing. This must FAIL, not warn:
// AC9's wording is "fails rather than passing quietly", because the state went
// unnoticed for an unknown length of time precisely by being quiet.
func TestHiveBackendCanServe_NoCredentialInjectionFails(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam("/home/u/.local/bin/hive serve", goodEnv),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if !strings.Contains(got, "can serve nothing") {
		t.Fatalf("expected the probes-present-serves-nothing failure, got: %s", got)
	}
	if !strings.Contains(got, "setup-linux.sh") {
		t.Errorf("the failure must name the remediation, got: %s", got)
	}
}

// A drop-in that ran `dotf secrets run` without scoping to NAN_API_KEY leaves
// the worker exactly as unreachable as no injection at all, so the check must
// not be satisfied by the mechanism alone.
func TestHiveBackendCanServe_WrongVariableStillFails(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath: lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam(
			"/home/u/.local/bin/dotf secrets run --only OPENROUTER_API_KEY -- /home/u/.local/bin/hive serve", goodEnv),
	}

	checkHiveBackendCanServe(sys, rep)

	if got := buf.String(); !strings.Contains(got, "can serve nothing") {
		t.Fatalf("a run that injects the wrong variable must still fail, got: %s", got)
	}
}

// The worker contract has two halves and a daemon holding one serves exactly as
// little as one holding neither. This is not a hypothetical: on 2026-08-24 the
// daemon's own worker_status reported `Configured: no — set
// HIVE_WORKER_BASE_URL`, and the first draft of this check would have passed a
// unit in that state because it only looked for the credential.
func TestHiveBackendCanServe_MissingBaseURLFails(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam(goodExecStart, ""),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if !strings.Contains(got, "can serve nothing") {
		t.Fatalf("a credential with no base URL must still fail, got: %s", got)
	}
	if !strings.Contains(got, "HIVE_WORKER_BASE_URL") {
		t.Errorf("the failure must name the missing half, got: %s", got)
	}
}

// Both halves missing must report both, not stop at the first: an operator who
// fixes only what the message named would restart into the same red.
func TestHiveBackendCanServe_BothHalvesMissingAreBothNamed(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam("/home/u/.local/bin/hive serve", ""),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if !strings.Contains(got, "API key") || !strings.Contains(got, "HIVE_WORKER_BASE_URL") {
		t.Errorf("both missing halves must be named, got: %s", got)
	}
}

func TestHiveBackendCanServe_CredentialInjectedPasses(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam(goodExecStart, goodEnv),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if strings.Contains(got, "can serve nothing") {
		t.Fatalf("a correctly injected daemon must not fail, got: %s", got)
	}
	if !strings.Contains(got, "can reach its pool") {
		t.Errorf("expected the pass line, got: %s", got)
	}
}

// `systemctl show` on an unknown unit SUCCEEDS with an empty value rather than
// erroring. Reading that empty string as "no credential injection" would fail
// every machine that runs hive without the daemon — a false red on a supported
// configuration, which is how a check earns the reputation that gets it muted.
func TestHiveBackendCanServe_UnitNotInstalledIsNotAFailure(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam("", ""),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if strings.Contains(got, "can serve nothing") {
		t.Fatalf("an absent unit is not the probes-present failure, got: %s", got)
	}
	if !strings.Contains(got, "no supervised daemon is declared") {
		t.Errorf("expected the unit-absent info line, got: %s", got)
	}
}

// A machine that never had hive should not carry a line about it in every run.
func TestHiveBackendCanServe_HiveAbsentIsSilent(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl"),
		CommandOutput: unitSeam("", ""),
	}

	checkHiveBackendCanServe(sys, rep)

	if strings.Contains(buf.String(), "can serve nothing") {
		t.Fatalf("no hive means no claim to falsify, got: %s", buf.String())
	}
}

func TestHiveBackendCanServe_WindowsSkips(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		GOOS:          "windows",
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: unitSeam("/home/u/.local/bin/hive serve", goodEnv),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if strings.Contains(got, "can serve nothing") {
		t.Fatalf("Windows supervises via a Scheduled Task; this check must not verdict there, got: %s", got)
	}
	if !strings.Contains(got, "Scheduled Task") {
		t.Errorf("expected the Windows skip line, got: %s", got)
	}
}

// An unreadable systemctl is an unanswered question, not a clean bill of
// health — the same fail-shape distinction dotf pr triage-queue draws between
// "nothing pending" and "could not be determined".
func TestHiveBackendCanServe_UnreadableUnitWarnsRatherThanPasses(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	sys := &System{
		LookPath:      lookPathFor("systemctl", "hive"),
		CommandOutput: failingSeam(errors.New("dbus unavailable")),
	}

	checkHiveBackendCanServe(sys, rep)

	got := buf.String()
	if !strings.Contains(got, "dbus unavailable") {
		t.Fatalf("expected the underlying error surfaced, got: %s", got)
	}
	if strings.Contains(got, "can reach its pool") {
		t.Errorf("an unreadable unit must never report the pass line, got: %s", got)
	}
}
