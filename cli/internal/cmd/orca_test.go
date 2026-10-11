package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func orcaHookPair(t *testing.T, script string) (cfg, scr string) {
	t.Helper()
	dir := t.TempDir()
	cfg, scr = filepath.Join(dir, "orca.json"), filepath.Join(dir, "copilot-hook.ps1")
	if err := os.WriteFile(cfg, []byte(`{"hooks":{"x":{"timeoutSec":30}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scr, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	return cfg, scr
}

// An unrecognised POST line is reported and left alone. The run must not then
// also claim the hooks are in sync: the two lines would contradict each other.
func TestRunOrcaTuneHooksUnrecognisedIsNotInSync(t *testing.T) {
	cfg, scr := orcaHookPair(t, "$r = Invoke-WebRequest -Uri $u -Method POST\n")
	var out bytes.Buffer
	if err := runOrcaTuneHooks(&out, cfg, scr, 30, false); err != nil {
		t.Fatalf("apply exits 0 on an unrecognised line, as the retired script did: %v", err)
	}
	if !strings.Contains(out.String(), "unrecognised") || strings.Contains(out.String(), "in sync") {
		t.Fatalf("want the unrecognised line and no in-sync claim:\n%s", out.String())
	}
	if err := runOrcaTuneHooks(&out, cfg, scr, 30, true); err == nil {
		t.Fatal("--check must exit non-zero while the script still uses Invoke-WebRequest")
	}
}

func TestRunOrcaTuneHooksInSyncWhenNothingDrifts(t *testing.T) {
	cfg, scr := orcaHookPair(t, "$req = [System.Net.HttpWebRequest]::Create($uri)\n")
	var out bytes.Buffer
	if err := runOrcaTuneHooks(&out, cfg, scr, 30, false); err != nil || !strings.Contains(out.String(), "in sync") {
		t.Fatalf("a tuned pair is in sync: err=%v\n%s", err, out.String())
	}
}

// --check names each file's state on its own line: drift where it drifts, ok
// where it does not, so the operator sees which of the two Orca regenerated.
func TestRunOrcaTuneHooksCheckNamesEachFilesState(t *testing.T) {
	cfg, scr := orcaHookPair(t, "$req = [System.Net.HttpWebRequest]::Create($uri)\n")
	var out bytes.Buffer
	if err := runOrcaTuneHooks(&out, cfg, scr, 60, true); err == nil {
		t.Fatal("--check must exit non-zero while a timeout is below the floor")
	}
	if !strings.Contains(out.String(), "drift: "+cfg+" has a hook timeoutSec < 60") ||
		!strings.Contains(out.String(), "ok: copilot-hook.ps1 uses HttpWebRequest") {
		t.Fatalf("want config drift and script ok:\n%s", out.String())
	}

	cfg, scr = orcaHookPair(t, "Invoke-WebRequest -Uri $u -Method POST\n")
	out.Reset()
	if err := runOrcaTuneHooks(&out, cfg, scr, 30, true); err == nil {
		t.Fatal("--check must exit non-zero while the script still uses Invoke-WebRequest")
	}
	if !strings.Contains(out.String(), "ok: orca.json hook timeouts >= 30") ||
		!strings.Contains(out.String(), "drift: "+scr+" still uses Invoke-WebRequest") {
		t.Fatalf("want config ok and script drift:\n%s", out.String())
	}
}

// On Linux and macOS Orca registers bash hooks, which the timeout floor does
// not cover: --check passes and says why, rather than reporting drift no
// converge can hold (#2252).
func TestRunOrcaTuneHooksCheckPassesABashHook(t *testing.T) {
	dir := t.TempDir()
	cfg, scr := filepath.Join(dir, "orca.json"), filepath.Join(dir, "copilot-hook.ps1")
	if err := os.WriteFile(cfg, []byte(`{"hooks":{"PreToolUse":[{"type":"command","bash":"x","timeoutSec":5}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runOrcaTuneHooks(&out, cfg, scr, 30, true); err != nil {
		t.Fatalf("--check must pass a bash hook: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "the timeoutSec floor does not apply") {
		t.Fatalf("want the reason named:\n%s", out.String())
	}
}
