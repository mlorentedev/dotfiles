package converge

import (
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// catalogFixture is a packages.json with one entry of each kind the step must
// tell apart: mise (a release binary), hive (waits on uv, which the mise sync
// installs), bw (waits on npm, which nothing installs) and gh (apt, behind a
// sudo password).
const catalogFixture = `{"tools": [
  {"name": "mise", "version": "2026.9.13", "source": {"type": "github-release", "repo": "jdx/mise", "asset": {"darwin/arm64": "mise-macos-arm64"}, "checksums": "SHASUMS256.txt"}},
  {"name": "hive", "version": "4.2.2", "source": {"type": "uv-tool", "package": "hive-vault"}},
  {"name": "bw", "version": "2026.9.0", "source": {"type": "npm", "package": "@bitwarden/cli"}},
  {"name": "gh", "source": {"type": "system", "apt": "gh"}}
]}`

// machine is the fake world: the commands on PATH and the log of what ran,
// shared by the fake catalog and the fake mise so a test can assert order.
type machine struct {
	path    map[string]bool
	present map[string]bool
	fail    map[string]error
	log     []string
}

func (m *machine) has(name string) bool { return m.path[name] }

// fakeCatalog classifies an entry the way tools.Installer does, from the
// machine's state, and installs by marking it present (mise lands on PATH).
type fakeCatalog struct{ m *machine }

func (c fakeCatalog) Plan(t tools.Tool) tools.Plan {
	p := tools.Plan{Name: t.Name}
	manager := map[string]string{"uv-tool": "uv", "npm": "npm"}[t.Source.Type]
	switch {
	case c.m.present[t.Name]:
		p.Action = tools.PlanSkip
	case t.Source.Type == "system":
		p.Action, p.Note = tools.PlanNeedsSudo, "run: sudo apt-get install -y gh"
	case manager != "" && !c.m.path[manager]:
		p.Action, p.Note = tools.PlanMissingManager, "waits on "+manager
	default:
		p.Action = tools.PlanInstall
	}
	return p
}

func (c fakeCatalog) Install(t tools.Tool) (tools.Result, error) {
	if c.Plan(t).Action != tools.PlanInstall {
		return tools.Skipped, nil
	}
	if err := c.m.fail[t.Name]; err != nil {
		return tools.Skipped, err
	}
	c.m.present[t.Name], c.m.path[t.Name] = true, true
	c.m.log = append(c.m.log, "install "+t.Name)
	return tools.Installed, nil
}

// freshMachine has neither mise nor uv nor npm; `mise install` brings uv.
func freshMachine(t *testing.T) (toolsSync, Env, *machine) {
	t.Helper()
	env := toolsEnv(t)
	writeFixture(t, env.RepoRoot, map[string]string{"packages.json": catalogFixture})
	m := &machine{path: map[string]bool{}, present: map[string]bool{}, fail: map[string]error{}}
	run, stdout, _ := fakeMise()
	logged := func(name string, args ...string) ([]byte, error) {
		if name == "mise" && args[0] == "install" {
			m.log = append(m.log, "mise install")
			m.path["uv"] = true
		}
		return run(name, args...)
	}
	r := toolsSync{run: logged, stdout: stdout, has: m.has, getenv: func(string) string { return "" }, catalog: fakeCatalog{m}}
	return r, env, m
}

// One apply converges a fresh machine: install places mise, the sync brings
// uv, and the second pass installs what waited on it. What converge cannot
// change (no npm, a sudo password) is reported, and the probe accepts it.
func TestToolsCatalog_OneApplyConvergesAFreshMachine(t *testing.T) {
	r, env, m := freshMachine(t)

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"catalog to install: mise", "hive (uv)", "bw (npm)", "gh (run: sudo apt-get install -y gh)", "mise is not on PATH"} {
		if !strings.Contains(plan.Detail, want) {
			t.Errorf("plan detail lacks %q: %s", want, plan.Detail)
		}
	}
	if len(m.log) != 0 {
		t.Fatalf("a plan ran %v", m.log)
	}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(m.log, ", "); got != "install mise, mise install, install hive" {
		t.Errorf("apply order: %s; want install, sync, install", got)
	}
	if err := r.Probe(env); err != nil {
		t.Errorf("probe after one apply: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Errorf("second plan: %+v, %v; want 0 changes", again, err)
	}
}

// A failed install does not stop the others, fails the apply naming it, and
// stays a failure in the probe until it lands.
func TestToolsCatalog_AFailedInstallFailsTheStepAndTheProbe(t *testing.T) {
	r, env, m := freshMachine(t)
	m.path["uv"] = true
	m.fail["mise"] = errors.New("mise: checksum mismatch")

	res, err := r.Reconcile(env, false)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("apply: want the install error, got %v", err)
	}
	if !m.present["hive"] || res.Changes != 1 {
		t.Errorf("hive present=%v, changes=%d: a failure must not stop the other entries", m.present["hive"], res.Changes)
	}
	if err := r.Probe(env); err == nil || !strings.Contains(err.Error(), "mise") {
		t.Errorf("probe: want mise still to install, got %v", err)
	}
}

// An entry Install refuses fails the plan, as it fails `dotf tools install
// --dry-run`: a converge plan must not promise what the apply refuses.
func TestToolsCatalog_ARefusedEntryFailsThePlan(t *testing.T) {
	r, env, _ := freshMachine(t)
	r.catalog = refusing{r.catalog}
	if _, err := r.Reconcile(env, true); err == nil || !strings.Contains(err.Error(), "no checksums") {
		t.Errorf("plan: want the refusal, got %v", err)
	}
}

type refusing struct{ CatalogInstaller }

func (refusing) Plan(t tools.Tool) tools.Plan {
	return tools.Plan{Name: t.Name, Action: tools.PlanRefused, Note: "no checksums file declared"}
}
