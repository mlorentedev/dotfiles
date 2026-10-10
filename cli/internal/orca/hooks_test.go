package orca

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The Pester cases of scripts/orca-hook-tune.ps1, ported with the script
// (CLI-062, #1338): what Orca generates, what the repair leaves behind.

const orcaJSON5s = `{
  "version": 1,
  "hooks": {
    "SessionStart": [{"type": "command", "powershell": "x", "timeoutSec": 5}],
    "PreToolUse":   [{"type": "command", "powershell": "y", "timeoutSec": 5}]
  }
}
`

const copilotHookIWR = "param()\r\n$body = '{}'\r\n    Invoke-WebRequest -Uri ('http://127.0.0.1:' + $env:ORCA_AGENT_HOOK_PORT + '/hook/copilot') -Method POST -Body $body | Out-Null\r\nexit 0\r\n"

func fixedNow() time.Time { return time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC) }

func hookFixture(t *testing.T, cfg, scr string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	c, s := filepath.Join(dir, "orca.json"), filepath.Join(dir, "copilot-hook.ps1")
	if cfg != "" {
		if err := os.WriteFile(c, []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if scr != "" {
		if err := os.WriteFile(s, []byte(scr), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return c, s
}

// "skips cleanly when neither file exists"
func TestTuneHooks_NothingToDoWithoutOrca(t *testing.T) {
	c, s := hookFixture(t, "", "")
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Nothing() || rep.Changed != 0 {
		t.Fatalf("both files absent must be nothing to do: %+v", rep)
	}
}

// "bumps orca.json timeout and rewrites copilot-hook.ps1 to HttpWebRequest"
// + "writes a timestamped backup before changing a file"
func TestTuneHooks_RepairsBothFilesWithBackups(t *testing.T) {
	c, s := hookFixture(t, orcaJSON5s, copilotHookIWR)
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Changed != 2 || rep.Drift() {
		t.Fatalf("both files must be repaired: %+v", rep)
	}
	cfg, _ := os.ReadFile(c)
	if strings.Contains(string(cfg), `"timeoutSec": 5`) || strings.Count(string(cfg), `"timeoutSec": 30`) != 2 {
		t.Fatalf("every timeout below the floor must be raised:\n%s", cfg)
	}
	scr, _ := os.ReadFile(s)
	if strings.Contains(string(scr), "Invoke-WebRequest") || !strings.Contains(string(scr), "[System.Net.HttpWebRequest]::Create($uri)") {
		t.Fatalf("the POST must be swapped:\n%s", scr)
	}
	if !strings.Contains(string(scr), "    $req.Method = 'POST'") {
		t.Fatalf("the replacement must keep the original line's indentation:\n%s", scr)
	}
	for _, bak := range []string{c + ".bak.20260829-120000", s + ".bak.20260829-120000"} {
		if _, err := os.Stat(bak); err != nil {
			t.Errorf("backup missing: %s", bak)
		}
	}
	if len(rep.Backups) != 2 {
		t.Errorf("two backups must be reported, got %v", rep.Backups)
	}
}

// "is idempotent: a second run changes nothing"
func TestTuneHooks_SecondRunChangesNothing(t *testing.T) {
	c, s := hookFixture(t, orcaJSON5s, copilotHookIWR)
	if _, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow); err != nil {
		t.Fatal(err)
	}
	later := func() time.Time { return fixedNow().Add(time.Hour) }
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, later)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Changed != 0 || rep.Drift() || len(rep.Backups) != 0 {
		t.Fatalf("second run must change nothing: %+v", rep)
	}
	if _, err := os.Stat(c + ".bak.20260829-130000"); err == nil {
		t.Fatal("no backup may be written when nothing changes")
	}
}

// "leaves an already-generous timeout untouched"
func TestTuneHooks_LeavesAGenerousTimeout(t *testing.T) {
	cfg := strings.ReplaceAll(orcaJSON5s, `"timeoutSec": 5`, `"timeoutSec": 45`)
	c, s := hookFixture(t, cfg, "")
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Changed != 0 || rep.ConfigDrift {
		t.Fatalf("45 is above the floor: %+v", rep)
	}
	got, _ := os.ReadFile(c)
	if string(got) != cfg {
		t.Fatal("an untouched file must be byte-identical")
	}
}

// "Check mode exits 1 on drift and 0 once clean"
func TestTuneHooks_CheckReportsAndWritesNothing(t *testing.T) {
	c, s := hookFixture(t, orcaJSON5s, copilotHookIWR)
	rep, err := TuneHooks(c, s, DefaultHookTimeout, true, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ConfigDrift || !rep.ScriptDrift || rep.Changed != 0 {
		t.Fatalf("check must report both drifts and write nothing: %+v", rep)
	}
	if got, _ := os.ReadFile(c); string(got) != orcaJSON5s {
		t.Fatal("check must not write the config")
	}
	if _, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow); err != nil {
		t.Fatal(err)
	}
	rep, err = TuneHooks(c, s, DefaultHookTimeout, true, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Drift() {
		t.Fatalf("check must be clean after the repair: %+v", rep)
	}
}

// An Invoke-WebRequest the swap does not recognise is reported, not guessed.
func TestTuneHooks_UnrecognisedPostIsLeftAlone(t *testing.T) {
	scr := "param()\r\n$r = Invoke-WebRequest -Uri 'http://x' # not at line start as a statement\r\n"
	c, s := hookFixture(t, "", scr)
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ScriptUnrecognised || rep.Changed != 0 {
		t.Fatalf("an unknown POST shape must be reported and left: %+v", rep)
	}
	if got, _ := os.ReadFile(s); string(got) != scr {
		t.Fatal("an unrecognised script must be byte-identical")
	}
}

// TestTuneScript_WritesTheRetiredScriptsBlockByteForByte pins the whole
// HttpWebRequest block, not a line of it. The block was copied from
// scripts/orca-hook-tune.ps1, and a box that script already tuned must read as
// clean; a regressed line here would still pass the Invoke-WebRequest drift
// check and ship a broken hook. The expected text is the retired script's
// block (its lines 148-161 at 4718e46^), each line carrying the captured
// indentation, in the file's own CRLF endings.
func TestTuneScript_WritesTheRetiredScriptsBlockByteForByte(t *testing.T) {
	const want = "param()\r\n$body = '{}'\r\n" +
		"    $uri = 'http://127.0.0.1:' + $env:ORCA_AGENT_HOOK_PORT + '/hook/copilot'\r\n" +
		"    $req = [System.Net.HttpWebRequest]::Create($uri)\r\n" +
		"    $req.Method = 'POST'\r\n" +
		"    $req.ContentType = 'application/json'\r\n" +
		"    $req.Headers.Add('X-Orca-Agent-Hook-Token', $env:ORCA_AGENT_HOOK_TOKEN)\r\n" +
		"    $req.Timeout = 2000\r\n" +
		"    $req.ReadWriteTimeout = 2000\r\n" +
		"    $reqBytes = [System.Text.Encoding]::UTF8.GetBytes($body)\r\n" +
		"    $req.ContentLength = $reqBytes.Length\r\n" +
		"    $reqStream = $req.GetRequestStream()\r\n" +
		"    $reqStream.Write($reqBytes, 0, $reqBytes.Length)\r\n" +
		"    $reqStream.Close()\r\n" +
		"    $resp = $req.GetResponse()\r\n" +
		"    $resp.Close()\r\n" +
		"exit 0\r\n"
	got, ok := TuneScript([]byte(copilotHookIWR))
	if !ok {
		t.Fatal("the fixture's POST line must be recognised")
	}
	if string(got) != want {
		t.Fatalf("tuned script differs from the retired script's block:\n got %q\nwant %q", got, want)
	}
}

// #1748: the staging file is unique to each write. A fixed `<file>.tmp` would be
// shared by two tuners running at once (setup and doctor --fix), so one could
// rename the other's half-written content into place. A directory squatting on
// the old fixed name stands in for that other writer: it makes a write to the
// fixed path fail, and a unique name never touches it.
func TestTuneHooks_StagesEachWriteInItsOwnTempFile(t *testing.T) {
	c, s := hookFixture(t, orcaJSON5s, copilotHookIWR)
	before, err := os.Stat(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{c, s} {
		if err := os.Mkdir(p+".tmp", 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatalf("a tuner must not depend on a fixed staging path: %v", err)
	}
	if rep.Changed != 2 {
		t.Fatalf("both files must be repaired: %+v", rep)
	}
	entries, err := os.ReadDir(filepath.Dir(c))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 6 { // two hooks, two backups, two squatters
		t.Fatalf("a staging file was left behind: %v", names)
	}
	info, err := os.Stat(c)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's mode is whatever the umask left, so compare with it rather
	// than with a literal.
	if info.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("the tuned file must keep its %v, got %v", before.Mode().Perm(), info.Mode().Perm())
	}
}

// The tuned file keeps the mode it had, whatever CreateTemp gave the staging
// file and whatever the umask: a hook an operator tightened stays tight, and a
// readable one stays readable.
func TestTuneHooks_KeepsEachFilesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	for _, mode := range []os.FileMode{0o600, 0o640, 0o644} {
		c, s := hookFixture(t, orcaJSON5s, copilotHookIWR)
		for _, p := range []string{c, s} {
			if err := os.Chmod(p, mode); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{c, s} {
			info, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != mode {
				t.Errorf("%s: mode %v, want %v", filepath.Base(p), info.Mode().Perm(), mode)
			}
		}
	}
}

// What Orca generates on Linux and macOS: bash hooks running copilot-hook.sh,
// with the same 5 s timeout (measured on msi, 2026-10-10, #2252).
const orcaJSONBash5s = `{
  "version": 1,
  "hooks": {
    "SessionStart": [{"type": "command", "bash": "x", "timeoutSec": 5}],
    "PreToolUse":   [{"type": "command", "bash": "y", "timeoutSec": 5}]
  }
}
`

func TestTimeoutFloorApplies(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          bool
	}{
		{"powershell hooks", orcaJSON5s, true},
		{"bash hooks", orcaJSONBash5s, false},
		{"one powershell entry among bash", `{"hooks":{"A":[{"bash":"x","timeoutSec":5}],"B":[{"powershell":"y","timeoutSec":5}]}}`, true},
		{"unparseable is left to the caller", `{"hooks": [`, true},
	} {
		if got := TimeoutFloorApplies([]byte(tc.content)); got != tc.want {
			t.Errorf("%s: TimeoutFloorApplies = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The floor is the PowerShell hook's (lesson 111). A bash hook's 5 s is left
// alone: no drift, no write, no backup, so a converge on Linux stops tuning a
// file the next Orca start reverts (#2252).
func TestTuneHooks_LeavesBashHookTimeoutsAlone(t *testing.T) {
	c, s := hookFixture(t, orcaJSONBash5s, "")
	rep, err := TuneHooks(c, s, DefaultHookTimeout, false, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if rep.TimeoutFloorApplies || rep.Drift() || rep.Changed != 0 || len(rep.Backups) != 0 {
		t.Fatalf("a bash hook must not be tuned: %+v", rep)
	}
	got, err := os.ReadFile(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != orcaJSONBash5s {
		t.Fatalf("orca.json was rewritten:\n%s", got)
	}
}
