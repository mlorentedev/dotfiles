package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configsEnv is a checkout whose ai/deploy.json declares one entry for every
// OS, one that requires a command the machine lacks, and one for another OS,
// and an empty HOME.
func configsEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"ai/deploy.json": `{"version": 4, "configs": [
		  {"name": "agent", "src": "ai/agent.json", "dst": "{HOME}/.agent/settings.json", "mode": "0644"},
		  {"name": "absent", "src": "ai/absent.json", "dst": "{HOME}/.absent/settings.json", "mode": "0644", "requires": "absent-tool"},
		  {"name": "winonly", "src": "ai/absent.json", "dst": "{HOME}/.win/settings.json", "mode": "0644", "platforms": ["windows"]}
		]}`,
		"ai/agent.json":  `{"model":"pinned"}`,
		"ai/absent.json": `{}`,
	})
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

func newConfigsDeploy() configsDeploy {
	return configsDeploy{
		render:  func(string) error { return nil },
		resolve: func(string) string { return "" },
		has:     func(string) bool { return false },
	}
}

func TestConfigsDeploy_PlanWritesNothingApplyConvergesAndRerunIsANoOp(t *testing.T) {
	env := configsEnv(t)
	r := newConfigsDeploy()

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 1 || !strings.Contains(plan.Detail, "agent") {
		t.Errorf("plan: want 1 change naming agent, got %d (%s)", plan.Changes, plan.Detail)
	}
	// An absent command and another OS are different reasons: the first is a
	// config that applies here and did not land, so it is named, not counted.
	for _, want := range []string{"absent (absent-tool not installed)", "1 not for this machine"} {
		if !strings.Contains(plan.Detail, want) {
			t.Errorf("plan detail %q does not say %q", plan.Detail, want)
		}
	}
	if _, err := os.Stat(filepath.Join(env.Home, ".agent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the plan wrote under HOME: %v", err)
	}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.Home, ".absent")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an entry whose command is absent was deployed: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %d (%s), err %v", again.Changes, again.Detail, err)
	}
}

// The case the reconciler exists for: a template change merged to main (here
// the pinned model) reaches the machine without a manual `dotf deploy`.
func TestConfigsDeploy_ATemplateChangeIsPlannedAndFailsTheProbeUntilApplied(t *testing.T) {
	env := configsEnv(t)
	r := newConfigsDeploy()
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, env.RepoRoot, map[string]string{"ai/agent.json": `{"model":"next"}`})

	if err := r.Probe(env); err == nil || !strings.Contains(err.Error(), "agent") {
		t.Errorf("probe on a stale config: want an error naming agent, got %v", err)
	}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(env.Home, ".agent", "settings.json"))
	if err != nil || !strings.Contains(string(got), "next") {
		t.Errorf("the template change did not land: %q, %v", got, err)
	}
}

func TestConfigsDeploy_CopilotRetiredSettingsPlanApplyAndSecondPlan(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	copilotHome := filepath.Join(home, ".copilot")
	writeFixture(t, repo, map[string]string{
		"ai/deploy.json": `{"version": 6, "configs": [{
		  "name": "copilot-settings",
		  "src": "ai/copilot/settings.json",
		  "dst": "{COPILOT_HOME}/settings.json",
		  "mode": "0644",
		  "strategy": "merge",
		  "requires": "copilot"
		}]}`,
		"ai/copilot/settings.json":         `{"model":"gpt-5.6-sol"}`,
		"ai/copilot/retired-settings.json": `{"version":1,"retired_paths":["telemetry.disabled"]}`,
	})
	writeFixture(t, home, map[string]string{
		".copilot/settings.json": `{"telemetry":{"disabled":true},"model":"gpt-5.6-sol","keep":true}`,
	})
	env := Env{RepoRoot: repo, Home: home, GOOS: "windows"}
	r := configsDeploy{
		render: func(string) error { return nil },
		resolve: func(name string) string {
			if name == "COPILOT_HOME" {
				return copilotHome
			}
			return ""
		},
		has: func(name string) bool { return name == "copilot" },
	}

	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 1 {
		t.Fatalf("plan = %+v, err=%v", plan, err)
	}
	if raw, _ := os.ReadFile(filepath.Join(copilotHome, "settings.json")); !strings.Contains(string(raw), "telemetry") {
		t.Fatal("plan rewrote settings")
	}

	applied, err := r.Reconcile(env, false)
	if err != nil || applied.Changes != 1 {
		t.Fatalf("apply = %+v, err=%v", applied, err)
	}
	raw, err := os.ReadFile(filepath.Join(copilotHome, "settings.json"))
	if err != nil || strings.Contains(string(raw), "telemetry") || !strings.Contains(string(raw), `"keep": true`) {
		t.Fatalf("settings = %s, err=%v", raw, err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Fatalf("second plan = %+v, err=%v", again, err)
	}
}

// A registry built without the secrets renderer would install {env:VAR}
// placeholders over resolved values; it fails instead of passing quietly.
func TestConfigsDeploy_WithoutARendererFailsLoudly(t *testing.T) {
	r := newConfigsDeploy()
	r.render = nil
	if _, err := r.Reconcile(configsEnv(t), false); err == nil {
		t.Error("an apply with no renderer wired passed")
	}
}
