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
		if name == "mise" && args[0] == "ls" {
			return []byte("{}"), nil
		}
		if strings.HasPrefix(name, "/bin/") {
			return run(name, args...) // a tool's banner is on its stdout
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
// lands already has its instructions (ADR-045 decision 4). env-generate sits
// between the mirror, which creates the deploy dir it writes into, and every
// install, so the path file is current before any agent lands (#2013 P6). The configs come
// after the tools, which answer each entry's `requires`, and before git-config,
// whose probe needs the dotfiles.gitconfig include target that a config entry
// deploys. git-lfs follows git-config: both write the global config, and
// tools installs the git-lfs it runs (#2201). pi-settings sits between tools, which installs pi, and
// configs-deploy, whose pi-compaction entry merges into the file it seeds
// (#1948). shell-completions follows configs-deploy, which deploys the .zshrc
// that reads what it writes, and tools, which installs herdr. records-bind follows configs-deploy, which seeds the settings files
// the hooks merge into. env-persist closes the configs step: the env files are configs too
// (ADR-045 decision 4, step 5). self-deploy follows it: the trigger is a
// user-session agent like env-persist's, and it names the dotf the tools layer
// installed (OPS-053). workstation-target follows: it reads the
// deployed public key's source in the checkout and changes how the machine
// faces the network, so it comes after everything a user session needs.
// agent-extensions comes after tools, whose PATH reaches the claude mise
// installs, and last among the native steps, so a plugin that cannot install
// stops nothing else (#2387). The setup script runs last, after every native
// step (PLAT-001b PR 5).
func TestRegistry_ToolsRunAfterRecords(t *testing.T) {
	var names []string
	for _, r := range Registry(Options{}) {
		names = append(names, r.Name())
	}
	if got := strings.Join(names, ","); got != "checkout,records-mirror,env-generate,records-harness,tools,pwsh-modules,pi-settings,configs-deploy,shell-completions,records-bind,git-config,git-lfs,git-hooks,env-persist,self-deploy,workstation-target,agent-extensions,legacy-setup" {
		t.Errorf("registry order: %s", got)
	}
}

// A CLI marked "# mise: latest" that is installed and behind its newest
// release is one change, named as an upgrade; applying it upgrades through
// mise, and the probe then passes.
func TestToolsSync_AnOutdatedLatestToolIsAnUpgrade(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{"versions.conf": "# mise: latest\nJQ_VERSION=1.8.2\n"})
	env := Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
	have, upgrades := "1.8.2", 0
	run := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "install":
			return nil, nil
		case name == "mise" && args[0] == "upgrade" && strings.Join(args[1:], " ") == "jq":
			have, upgrades = "1.8.3", upgrades+1
			return nil, nil
		case name == "/bin/jq":
			return []byte("jq-" + have), nil
		}
		return nil, errors.New("unexpected: " + name + " " + strings.Join(args, " "))
	}
	stdout := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "which":
			return []byte("/bin/jq\n"), nil
		case name == "mise" && args[0] == "outdated":
			if have == "1.8.3" {
				return []byte("{}"), nil
			}
			return []byte(`{"jq": {"requested": "latest", "current": "1.8.2", "latest": "1.8.3"}}`), nil
		case name == "mise" && args[0] == "ls":
			return []byte("{}"), nil
		case name == "/bin/jq":
			return run(name, args...)
		}
		return nil, errors.New("unexpected: " + name + " " + strings.Join(args, " "))
	}
	r := toolsSync{run: run, stdout: stdout, has: func(string) bool { return true }, getenv: func(string) string { return "" }}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	have, upgrades = "1.8.2", 0 // upstream releases 1.8.3 after the first converge
	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 1 || !strings.Contains(plan.Detail, "to upgrade to their newest release: jq") {
		t.Fatalf("plan: want one upgrade naming jq, got %+v, %v", plan, err)
	}
	if _, err := r.Reconcile(env, false); err != nil || upgrades != 1 {
		t.Fatalf("apply: want one upgrade, got %d, %v", upgrades, err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after the upgrade: %v", err)
	}
}

// #2526: an upgrade upstream has not caught up with (a release still inside
// mise's minimum_release_age) neither fails the apply nor the probe. The step
// names the tool as behind, so the report says why it is not current.
func TestToolsSync_ALatestToolUpstreamHasNotReleasedIsBehindNotFailed(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{"versions.conf": "# mise: latest\nJQ_VERSION=1.8.2\n"})
	env := Env{RepoRoot: repo, Home: t.TempDir(), GOOS: "darwin"}
	upgrades, outdates := 0, 0
	run := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "install":
			return nil, nil
		case name == "mise" && args[0] == "upgrade":
			upgrades++ // exits 0 and installs nothing newer
			return nil, nil
		case name == "/bin/jq":
			return []byte("jq-1.8.2"), nil
		}
		return nil, errors.New("unexpected: " + name + " " + strings.Join(args, " "))
	}
	stdout := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "which":
			return []byte("/bin/jq\n"), nil
		case name == "mise" && args[0] == "outdated":
			outdates++
			return []byte(`{"jq": {"requested": "latest", "current": "1.8.2", "latest": "1.8.3"}}`), nil
		case name == "mise" && args[0] == "ls":
			return []byte("{}"), nil
		case name == "/bin/jq":
			return run(name, args...)
		}
		return nil, errors.New("unexpected: " + name + " " + strings.Join(args, " "))
	}
	r := toolsSync{run: run, stdout: stdout, has: func(string) bool { return true }, getenv: func(string) string { return "" }}
	res, err := r.Reconcile(env, false)
	if err != nil || upgrades != 1 {
		t.Fatalf("apply: want one upgrade and no error, got %d, %v", upgrades, err)
	}
	if !strings.Contains(res.Detail, "still behind their newest release, retried next run: jq") || strings.Contains(res.Detail, "upgraded") {
		t.Errorf("detail must name jq as behind, not as upgraded: %q", res.Detail)
	}
	asked := outdates
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe: a release upstream is not this machine's drift: %v", err)
	}
	if outdates != asked {
		t.Errorf("the probe must not ask mise outdated, asked %d more time(s)", outdates-asked)
	}
}

// #2375: an install no pin asks for is drift the plan names, the probe
// reports and an apply removes, once: the second plan has nothing to prune.
func TestToolsSync_PlansReportsAndPrunesAnOrphanedInstall(t *testing.T) {
	env := toolsEnv(t)
	run, stdout, _ := fakeMise()
	orphan, prunes := true, 0
	pruning := func(name string, args ...string) ([]byte, error) {
		if name == "mise" && args[0] == "prune" {
			if strings.Join(args[1:], " ") != "--yes --tools powershell" {
				return nil, errors.New("pruned the wrong set: " + strings.Join(args, " "))
			}
			orphan, prunes = false, prunes+1
			return nil, nil
		}
		return run(name, args...)
	}
	listing := func(name string, args ...string) ([]byte, error) {
		if name == "mise" && args[0] == "ls" && orphan {
			return []byte(`{"powershell": [{"version": "7.5.11", "installed": true, "active": false}]}`), nil
		}
		return stdout(name, args...)
	}
	r := toolsSync{run: pruning, stdout: listing, has: func(string) bool { return true }, getenv: func(string) string { return "" }}

	plan, err := r.Reconcile(env, true)
	if err != nil || !strings.Contains(plan.Detail, "to prune, asked for by no pin: powershell@7.5.11") || prunes != 0 {
		t.Fatalf("plan: want the orphan named and nothing pruned, got %+v (%d prunes), %v", plan, prunes, err)
	}
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil || prunes != 1 {
		t.Fatalf("probe after apply: %v (%d prunes)", err, prunes)
	}
	orphan = true // a tool installed by hand after the converge
	if err := r.Probe(env); err == nil || !strings.Contains(err.Error(), "asked for by no pin: powershell@7.5.11") {
		t.Errorf("probe with an orphan: want it named, got %v", err)
	}
	orphan = false
	if again, err := r.Reconcile(env, true); err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %+v, %v", again, err)
	}
}
