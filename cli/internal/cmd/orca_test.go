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
