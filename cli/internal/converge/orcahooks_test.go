package converge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/orca"
)

const (
	// orca.json as Orca generates it: a PowerShell hook with a 5 s timeout.
	orcaConfigSlow  = `{"hooks":{"sessionStart":[{"type":"command","powershell":"& copilot-hook.ps1","timeoutSec":5}]}}`
	orcaScriptSlow  = "$body = $input | Out-String\nInvoke-WebRequest -Uri $url -Method Post -Body $body -ContentType 'application/json' | Out-Null\n"
	orcaScriptOdd   = "$r = Invoke-WebRequest $url -Body $b\n" // not at the line start: the swap does not know it
	orcaScriptTuned = "$body = $input | Out-String\n"
)

func writeOrcaFiles(t *testing.T, home, config, script string) (string, string) {
	t.Helper()
	cfg, scr := orca.HookPaths(home)
	for path, body := range map[string]string{cfg: config, scr: script} {
		if body == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return cfg, scr
}

func newOrcaHooks() orcaHooks {
	return orcaHooks{now: func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC) }}
}

func TestOrcaHooks_WithoutOrcaIsSkipped(t *testing.T) {
	res, err := newOrcaHooks().Reconcile(Env{Home: t.TempDir()}, false)
	if err != nil || res.Skip == "" {
		t.Fatalf("no Orca files: res=%+v err=%v, want a skip", res, err)
	}
}

func TestOrcaHooks_PlansAppliesProbesAndIsInSyncAfter(t *testing.T) {
	home := t.TempDir()
	cfg, scr := writeOrcaFiles(t, home, orcaConfigSlow, orcaScriptSlow)
	r, env := newOrcaHooks(), Env{Home: home}

	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || !strings.Contains(plan.Detail, "to tune") {
		t.Fatalf("plan: res=%+v err=%v, want 2 changes to tune", plan, err)
	}
	if got, _ := os.ReadFile(cfg); string(got) != orcaConfigSlow {
		t.Fatalf("a plan wrote orca.json:\n%s", got)
	}
	if err := r.Probe(env); err == nil {
		t.Fatal("the probe passed on untuned hooks")
	}

	apply, err := r.Reconcile(env, false)
	if err != nil || apply.Changes != 2 || !strings.Contains(apply.Detail, "tuned") {
		t.Fatalf("apply: res=%+v err=%v", apply, err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	if got, _ := os.ReadFile(scr); strings.Contains(string(got), "Invoke-WebRequest") {
		t.Fatalf("script not tuned:\n%s", got)
	}

	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Fatalf("second plan: res=%+v err=%v, want 0 changes", again, err)
	}
}

// A script whose POST line the swap does not know is left as it is and named:
// it is not a change the step can make, so it neither counts as one nor fails
// the probe, or every run would plan it and then fail.
func TestOrcaHooks_AnUnrecognisedScriptIsNamedNotPlanned(t *testing.T) {
	home := t.TempDir()
	_, scr := writeOrcaFiles(t, home, "", orcaScriptOdd)
	r, env := newOrcaHooks(), Env{Home: home}

	for _, dry := range []bool{true, false} {
		res, err := r.Reconcile(env, dry)
		if err != nil || res.Changes != 0 || !strings.Contains(res.Detail, "review it by hand") {
			t.Fatalf("dry=%v: res=%+v err=%v, want 0 changes and the script named", dry, res, err)
		}
	}
	if got, _ := os.ReadFile(scr); string(got) != orcaScriptOdd {
		t.Fatalf("an unrecognised script was rewritten:\n%s", got)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe on an unrecognised script: %v", err)
	}
}

func TestOrcaHooks_TunedHooksAreInSync(t *testing.T) {
	home := t.TempDir()
	writeOrcaFiles(t, home, strings.Replace(orcaConfigSlow, `"timeoutSec":5`, `"timeoutSec":30`, 1), orcaScriptTuned)
	res, err := newOrcaHooks().Reconcile(Env{Home: home}, true)
	if err != nil || res.Changes != 0 || res.Skip != "" {
		t.Fatalf("res=%+v err=%v, want in sync", res, err)
	}
}

// orca-hooks follows configs-deploy, which installs the Copilot settings the
// hooks sit beside, as the bare `dotf deploy` tunes them after its configs.
func TestRegistry_OrcaHooksFollowsConfigsDeploy(t *testing.T) {
	pos := map[string]int{}
	for i, r := range Registry(Options{}) {
		pos[r.Name()] = i
	}
	c, ok1 := pos["configs-deploy"]
	o, ok2 := pos["orca-hooks"]
	if !ok1 || !ok2 || o != c+1 {
		t.Fatalf("orca-hooks at %d (present %v), configs-deploy at %d: want it right after", o, ok2, c)
	}
}
