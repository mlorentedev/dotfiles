package converge

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
  {"name": "gh", "source": {"type": "system", "apt": "gh"}},
  {"name": "veracrypt", "source": {"type": "system", "cask": "veracrypt", "needs_sudo": true}}
]}`

// machine is the fake world: the commands on PATH and the log of what ran,
// shared by the fake catalog and the fake mise so a test can assert order.
type machine struct {
	path    map[string]bool
	present map[string]bool
	fail    map[string]error
	offPath map[string]bool // installed where this process's PATH does not reach
	goos    string          // "" installs every entry, as on an OS each supports
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
	case c.m.goos != "" && !t.SupportsOS(c.m.goos):
		p.Action, p.Note = tools.PlanUnsupported, "not installed on "+c.m.goos+" by this catalog"
	case t.Source.Type == "system":
		p.Action, p.Manager, p.Package = tools.PlanNeedsSudo, "apt", t.Source.Apt
		if t.Source.Cask != "" {
			p.Manager, p.Package = "brew-cask", t.Source.Cask
		}
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
	c.m.present[t.Name], c.m.path[t.Name] = true, !c.m.offPath[t.Name]
	c.m.log = append(c.m.log, "install "+t.Name)
	return tools.Installed, nil
}

// freshMachine has neither mise nor uv nor npm; `mise install` brings uv.
func freshMachine(t *testing.T) (toolsSync, Env, *machine) {
	t.Helper()
	env := toolsEnv(t)
	writeFixture(t, env.RepoRoot, map[string]string{
		"packages.json": catalogFixture,
		"versions.conf": "# mise: cli\nJQ_VERSION=1.8.2\n# mise: cli\nUV_VERSION=0.12.18\n",
	})
	m := &machine{path: map[string]bool{}, present: map[string]bool{}, fail: map[string]error{}, offPath: map[string]bool{}}
	synced := false
	run := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "install":
			m.log = append(m.log, "mise install")
			synced, m.path["uv"] = true, !m.offPath["uv"]
			return nil, nil
		case name == "/bin/jq":
			return []byte("jq-1.8.2"), nil
		case name == "/bin/uv":
			return []byte("uv 0.12.18"), nil
		}
		return nil, errors.New("unexpected: " + name)
	}
	stdout := func(name string, args ...string) ([]byte, error) {
		if synced && name == "mise" && args[0] == "which" {
			return []byte("/bin/" + args[len(args)-1] + "\n"), nil
		}
		if name == "mise" && args[0] == "ls" {
			return []byte("{}"), nil
		}
		if strings.HasPrefix(name, "/bin/") {
			return run(name, args...) // a tool's banner is on its stdout
		}
		return nil, errors.New("not installed")
	}
	r := toolsSync{run: run, stdout: stdout, has: m.has, getenv: func(string) string { return "" }, catalog: fakeCatalog{m}}
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
	for _, want := range []string{"catalog to install: mise", "hive (uv)", "bw (npm)", "needs sudo: gh, veracrypt (run once: sudo apt-get update && sudo apt-get install -y --no-remove gh && brew install --cask veracrypt)", "mise is not on PATH"} {
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
	if n := strings.Count(err.Error(), "checksum mismatch"); n != 1 {
		t.Errorf("the failure was attempted or reported %d times, want once: %v", n, err)
	}
	if err := r.Probe(env); err == nil || !strings.Contains(err.Error(), "mise") {
		t.Errorf("probe: want mise still to install, got %v", err)
	}
}

// A manager this machine's own tools install, placed where this process's PATH
// does not reach, is not a wait: a fresh machine whose PATH lacks ~/.local/bin
// or the mise shims would otherwise report OK with the sync never run and the
// uv tools never installed, and a second run would report 0 changes.
func TestToolsCatalog_AManagerItInstalledButCannotReachFailsTheProbe(t *testing.T) {
	for _, tc := range []struct{ off, want string }{
		{"mise", "mise"},
		{"uv", "uv (for hive)"},
	} {
		t.Run(tc.off, func(t *testing.T) {
			r, env, m := freshMachine(t)
			m.offPath[tc.off] = true
			if _, err := r.Reconcile(env, false); err != nil {
				t.Fatal(err)
			}
			err := r.Probe(env)
			if err == nil || !strings.Contains(err.Error(), "not on PATH: "+tc.want) {
				t.Errorf("probe: want %q named as unreachable, got %v", tc.want, err)
			}
		})
	}
}

// On an OS whose catalog does not install mise (Windows), the mise pins are
// not this machine's tools: uv, which a uv-tool entry waits on, arrives from
// elsewhere later in the run (#2754). The wait is reported, the probe holds,
// and the skip names no command that would skip too.
func TestToolsCatalog_AManagerOnlyMiseProvidesIsAWaitWhereMiseIsNotInstalled(t *testing.T) {
	r, env, m := freshMachine(t)
	env.GOOS, m.goos = "windows", "windows"
	writeFixture(t, env.RepoRoot, map[string]string{"packages.json": strings.Replace(catalogFixture,
		`"checksums": "SHASUMS256.txt"}`, `"checksums": "SHASUMS256.txt", "platforms": ["linux", "darwin"]}`, 1)})

	res, err := r.Reconcile(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Detail, "waiting on a manager: hive (uv)") {
		t.Errorf("apply detail lacks the wait on uv: %s", res.Detail)
	}
	if strings.Contains(res.Detail, "dotf tools install mise") {
		t.Errorf("the detail names a command that skips mise on this OS: %s", res.Detail)
	}
	if err := r.Probe(env); err != nil {
		t.Errorf("probe: want the wait on uv accepted, got %v", err)
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

// The step reaches what the tool layer places whatever PATH its caller had:
// the hourly `dotf update` runs with a minimal one, and before this every run
// there skipped the sync and, with the catalog wired, failed the probe on a
// mise sitting in ~/.local/bin. The caller's entries stay first, and a run
// adds each directory once.
func TestTools_ReachesWhereTheToolLayerPlacesWhateverTheCallersPath(t *testing.T) {
	bins := t.TempDir()
	mise := filepath.Join(bins, "mise")
	if runtime.GOOS == "windows" {
		mise += ".exe"
	}
	if err := os.WriteFile(mise, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // a fake binary must be executable
		t.Fatal(err)
	}
	caller := t.TempDir()
	r := toolsSync{has: onPath, bins: []string{bins, caller, bins}}
	for _, call := range []struct {
		name string
		run  func() error
	}{
		{"reconcile", func() error { _, err := r.Reconcile(toolsEnv(t), true); return err }},
		{"probe", func() error { return r.Probe(toolsEnv(t)) }},
	} {
		t.Run(call.name, func(t *testing.T) {
			t.Setenv("PATH", caller)
			if onPath("mise") {
				t.Fatal("precondition: mise must not be on the caller's PATH")
			}
			_ = call.run()
			want := caller + string(os.PathListSeparator) + bins
			if got := os.Getenv("PATH"); got != want {
				t.Errorf("PATH = %q, want %q", got, want)
			}
			if !onPath("mise") {
				t.Error("mise in the tool layer's dir is not reachable after the step")
			}
		})
	}
}

func TestRegistry_HandsTheToolLayerDirsToTheToolsStep(t *testing.T) {
	for _, rec := range Registry(Options{ToolsBinDirs: []string{"/shims", "/bin"}}) {
		if ts, ok := rec.(toolsSync); ok {
			if strings.Join(ts.bins, "|") != "/shims|/bin" {
				t.Errorf("tools step bins = %v", ts.bins)
			}
			return
		}
	}
	t.Fatal("the registry has no tools step")
}

// Every entry that waits on a sudo password is in ONE command, not one per
// package: a fresh Linux box has up to nine apt entries (#2308).
func TestWalkCatalog_NeedsSudoIsOneCommandForEveryPackage(t *testing.T) {
	m := &machine{path: map[string]bool{}, present: map[string]bool{}, fail: map[string]error{}, offPath: map[string]bool{}}
	entries := []tools.Tool{
		{Name: "gh", Source: tools.Source{Type: "system", Apt: "gh"}},
		{Name: "docker", Source: tools.Source{Type: "system", Apt: "docker.io"}},
		{Name: "parallel", Source: tools.Source{Type: "system", Apt: "parallel"}},
	}
	got := walkCatalog(fakeCatalog{m}, entries, true).detail(false)
	want := "needs sudo: gh, docker, parallel (run once: sudo apt-get update && sudo apt-get install -y --no-remove gh docker.io parallel)"
	if got != want {
		t.Errorf("detail = %q\nwant     %q", got, want)
	}
	if strings.Count(got, "apt-get install") != 1 {
		t.Errorf("want one command, got %q", got)
	}
}

// BUG-134: a cask whose installer asks for a sudo password stopped the whole
// converge at the tools step, so nothing after it was deployed. With the real
// Installer in the step and sudo refusing, the cask is reported as waiting on a
// person and the step after it still runs.
func TestRun_ACaskThatNeedsSudoDoesNotStopTheStepsAfterTools(t *testing.T) {
	r, env, _ := freshMachine(t)
	writeFixture(t, env.RepoRoot, map[string]string{
		"packages.json": `{"tools": [{"name": "veracrypt", "source": {"type": "system", "cask": "veracrypt", "needs_sudo": true}}]}`,
	})
	var ran [][]string
	r.catalog = &tools.Installer{
		GOOS: "darwin", GOARCH: "arm64", Out: &strings.Builder{},
		HasCommand: func(string) bool { return true },
		Query: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("exit status 1") // sudo wants a password; the cask is not installed
		},
		Run: func(name string, args ...string) error {
			ran = append(ran, append([]string{name}, args...))
			return errors.New("sudo: a terminal is required to read the password")
		},
	}
	env.GOOS = "darwin"
	next := &fake{name: "configs-deploy", changes: 1}

	rep, err := Run([]Reconciler{r, next}, env, false)
	if err != nil {
		t.Fatalf("converge failed: %v", err)
	}
	if !next.applied {
		t.Error("the step after tools never ran")
	}
	if len(ran) != 0 {
		t.Errorf("ran %v; a cask that cannot install unattended is not attempted", ran)
	}
	if want := "needs sudo: veracrypt (run once: brew install --cask veracrypt)"; !strings.Contains(rep.Entries[0].Detail, want) {
		t.Errorf("tools detail %q lacks %q", rep.Entries[0].Detail, want)
	}
}

// Every uv-tool entry in the shipped catalog waits on a uv this OS's own tool
// layer provides, through the mise pins or a catalog entry. Windows had none,
// so its uv came from legacy-setup, the last step, and `tools` never
// converged in one run (#2754).
func TestShippedCatalog_EveryOSProvidesTheUVItsUVToolsNeed(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	entries, err := loadCatalog(Env{RepoRoot: repo})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo, "versions.conf"))
	if err != nil {
		t.Fatal(err)
	}
	pins, err := tools.ParseMisePins(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		has := provided(entries, pins.Tools, goos, false)
		for _, e := range entries {
			if e.Source.Type == "uv-tool" && e.SupportsOS(goos) && !has["uv"] {
				t.Errorf("%s: %s is a uv-tool entry, and nothing on %s provides uv", goos, e.Name, goos)
			}
		}
	}
}
