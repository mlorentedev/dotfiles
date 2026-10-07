package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// harnessEnv is a checkout whose manifest declares three instruction files:
// claude and opencode unconditionally, copilot only when copilot is on PATH.
func harnessEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"harness/manifest.json": `{"agents":{"presence":[
			{"agent":"claude","file":".claude/CLAUDE.md","source":"ai/claude/CLAUDE.md"},
			{"agent":"opencode","file":".config/opencode/AGENTS.md","source":"AGENTS.md"},
			{"agent":"copilot","file":".copilot/copilot-instructions.md","source":"ai/copilot/ci.md","requires_command":"copilot"}
		]}}`,
		"ai/claude/CLAUDE.md": "# CLAUDE\n",
		"AGENTS.md":           "# AGENTS\n",
		"ai/copilot/ci.md":    "# COPILOT\n",
	})
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

// deployingRunner stands in for compile-harness.sh --deploy: it copies each
// source and appends a presence region, the shape the real script leaves.
func deployingRunner(t *testing.T) func(Env) error {
	return func(env Env) error {
		for src, dst := range map[string]string{
			"ai/claude/CLAUDE.md": ".claude/CLAUDE.md",
			"AGENTS.md":           ".config/opencode/AGENTS.md",
		} {
			body, err := os.ReadFile(filepath.Join(env.RepoRoot, src))
			if err != nil {
				return err
			}
			region := "\n<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:x) -->\nroster\n<!-- END HARNESS AGENT-PRESENCE -->\n"
			writeFixture(t, env.Home, map[string]string{dst: string(body) + region})
		}
		return nil
	}
}

func noCommands(string) bool { return false }

func TestRecordsHarness_PlanCountsMissingInstructionFilesAndWritesNothing(t *testing.T) {
	env := harnessEnv(t)
	ran := false
	r := recordsHarness{run: func(Env) error { ran = true; return nil }, has: noCommands}

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 2 {
		t.Errorf("want claude and opencode to deploy (copilot is not installed), got %d: %s", plan.Changes, plan.Detail)
	}
	if ran {
		t.Error("a plan ran compile-harness.sh")
	}
	if entries, _ := os.ReadDir(env.Home); len(entries) != 0 {
		t.Errorf("a plan wrote into HOME: %v", entries)
	}
}

func TestRecordsHarness_ApplyDeploysAndASecondPlanIgnoresThePresenceRegion(t *testing.T) {
	env := harnessEnv(t)
	r := recordsHarness{run: deployingRunner(t), has: noCommands}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after a deploy: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %d (%v): %s", again.Changes, err, again.Detail)
	}
}

func TestRecordsHarness_ProbeFailsWhenTheDeployLeftAFileOut(t *testing.T) {
	env := harnessEnv(t)
	r := recordsHarness{run: func(Env) error { return nil }, has: noCommands}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	err := r.Probe(env)
	if err == nil || !strings.Contains(err.Error(), ".claude/CLAUDE.md") {
		t.Fatalf("want the probe to name the missing CLAUDE.md, got %v", err)
	}
}

func TestRecordsHarness_ARunnerFailureFailsTheApply(t *testing.T) {
	env := harnessEnv(t)
	r := recordsHarness{run: func(Env) error { return errors.New("compile-harness.sh exited 2") }, has: noCommands}

	if _, err := r.Reconcile(env, false); err == nil || !strings.Contains(err.Error(), "exited 2") {
		t.Fatalf("want the runner's error, got %v", err)
	}
}

func TestRecordsHarness_ARequiredCommandOnPathBringsItsFileIn(t *testing.T) {
	env := harnessEnv(t)
	r := recordsHarness{run: func(Env) error { return nil }, has: func(c string) bool { return c == "copilot" }}

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 3 {
		t.Errorf("with copilot installed, want 3 files to deploy, got %d", plan.Changes)
	}
}

func TestRecordsHarness_AnApplyWithNoRunnerFailsLoudly(t *testing.T) {
	env := harnessEnv(t)

	if _, err := (recordsHarness{has: noCommands}).Reconcile(env, false); err == nil {
		t.Fatal("an apply with no deploy runner reported success")
	}
}

// Skills are not planned yet, so an apply must run the deploy even when every
// instruction file is current; skipping it would leave a changed skill record
// undeployed behind a clean report.
func TestRecordsHarness_ApplyRunsTheDeployEvenWhenInstructionsAreCurrent(t *testing.T) {
	env := harnessEnv(t)
	if _, err := (recordsHarness{run: deployingRunner(t), has: noCommands}).Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	runs := 0
	r := recordsHarness{run: func(Env) error { runs++; return nil }, has: noCommands}

	res, err := r.Reconcile(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changes != 0 || runs != 1 {
		t.Errorf("want 0 instruction changes and one deploy run, got %d changes and %d runs", res.Changes, runs)
	}
}
