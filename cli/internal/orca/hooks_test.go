package orca

import (
	"os"
	"path/filepath"
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
