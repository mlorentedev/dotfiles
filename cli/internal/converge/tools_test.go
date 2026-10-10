package converge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// toolsEnv is a checkout whose versions.conf marks one CLI for mise.
func toolsEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{"versions.conf": "# mise: cli\nJQ_VERSION=1.8.2\n"})
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
}

// fakeMise installs jq on `mise install` and answers which/--version after.
func fakeMise() (run, stdout tools.Runner, installs *int) {
	installed, n := false, 0
	run = func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "install":
			installed, n = true, n+1
			return nil, nil
		case name == "/bin/jq":
			return []byte("jq-1.8.2"), nil
		}
		return nil, errors.New("unexpected: " + name)
	}
	stdout = func(name string, args ...string) ([]byte, error) {
		if installed && name == "mise" && args[0] == "which" {
			return []byte("/bin/jq\n"), nil
		}
		return nil, errors.New("not installed")
	}
	return run, stdout, &n
}

func miseConfig(env Env) string { return filepath.Join(env.Home, ".config", "mise") }

func TestToolsSync_PlanThenApplyThenNothingToDo(t *testing.T) {
	env := toolsEnv(t)
	run, stdout, installs := fakeMise()
	r := toolsSync{run: run, stdout: stdout, has: func(string) bool { return true }, getenv: func(string) string { return "" }}

	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || !strings.Contains(plan.Detail, "jq") {
		t.Fatalf("plan: want the config and jq, got %+v, %v", plan, err)
	}
	if _, err := os.Stat(miseConfig(env)); !os.IsNotExist(err) {
		t.Fatal("a plan wrote the mise config")
	}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 || *installs != 1 {
		t.Errorf("second plan: want 0 changes after one install, got %+v (%d installs), %v", again, *installs, err)
	}
}

func TestToolsSync_WithoutMiseIsASkipNamingTheRemedy(t *testing.T) {
	env := toolsEnv(t)
	r := toolsSync{has: func(string) bool { return false }, getenv: func(string) string { return "" }}

	res, err := r.Reconcile(env, false)
	if err != nil || !strings.Contains(res.Skip, "dotf tools install mise") {
		t.Fatalf("want a skip naming the remedy, got %+v, %v", res, err)
	}
}

func TestToolsSync_UnwiredRunnersAreASkipNotARealRun(t *testing.T) {
	env := toolsEnv(t)
	res, err := (toolsSync{has: func(string) bool { return true }}).Reconcile(env, false)
	if err != nil || res.Skip == "" {
		t.Fatalf("want a skip when no mise runner is wired, got %+v, %v", res, err)
	}
}

// A reconciler that declares itself not applicable is reported skipped with its
// reason, never passed, and is not probed.
func TestRun_AReconcilerSkipIsReportedAndNotProbed(t *testing.T) {
	s := &skipper{}
	rep, err := Run([]Reconciler{s}, Env{GOOS: "darwin"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if e := rep.Entries[0]; e.Status != StatusSkipped || e.Detail != "mise is not on PATH" || s.probed {
		t.Errorf("got %+v, probed %v", e, s.probed)
	}
}

type skipper struct{ probed bool }

func (*skipper) Name() string        { return "tools" }
func (*skipper) Platforms() []string { return nil }
func (*skipper) Reconcile(Env, bool) (Result, error) {
	return Result{Skip: "mise is not on PATH"}, nil
}
func (s *skipper) Probe(Env) error { s.probed = true; return nil }

// The order is the contract: the checkout first, since every step reads it
// (PLAT-001b PR 4a); then records, so an agent that starts once a tool
// lands already has its instructions (ADR-045 decision 4). The configs come
// after the tools, which answer each entry's `requires`, and before git-config,
// whose probe needs the dotfiles.gitconfig include target that a config entry
// deploys. env-persist closes the configs step: the env files are configs too
// (ADR-045 decision 4, step 5). The setup script runs last, after every native
// step (PLAT-001b PR 5).
func TestRegistry_ToolsRunAfterRecords(t *testing.T) {
	var names []string
	for _, r := range Registry(Options{}) {
		names = append(names, r.Name())
	}
	if got := strings.Join(names, ","); got != "checkout,records-mirror,records-harness,tools,configs-deploy,git-config,env-persist,legacy-setup" {
		t.Errorf("registry order: %s", got)
	}
}
