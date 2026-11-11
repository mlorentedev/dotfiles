package converge

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

// upgradeCatalog is a packages.json with one entry of each kind the step must
// tell apart: system entries on every manager, a cask that needs sudo, and two
// entries that are not system packages and so are never its targets.
const upgradeCatalog = `{"tools": [
  {"name": "mise", "version": "2026.9.13", "source": {"type": "github-release", "repo": "jdx/mise", "asset": {"darwin/arm64": "mise-macos-arm64"}, "checksums": "SHASUMS256.txt"}},
  {"name": "hive", "version": "4.2.2", "source": {"type": "uv-tool", "package": "hive-vault"}},
  {"name": "git", "source": {"type": "system", "apt": "git", "brew": "git", "winget": "Git.Git"}},
  {"name": "wget", "source": {"type": "system", "apt": "wget", "brew": "wget"}},
  {"name": "eza", "source": {"type": "system", "apt": "eza", "winget": "eza-community.eza"}},
  {"name": "firefox", "source": {"type": "system", "cask": "firefox"}},
  {"name": "veracrypt", "source": {"type": "system", "cask": "veracrypt", "needs_sudo": true}}
]}`

// pkgFake is the fake machine behind the runner: what each manager reports
// outdated, and a log of every command. An upgrade clears the package from the
// manager's outdated list, so a second listing is what the real manager would
// answer, and a step that upgrades but leaves it listed fails its probe.
type pkgFake struct {
	log      []string
	outdated map[string][]string // manager (brew, brew-cask, apt, winget) -> names behind
	pinned   map[string]bool     // brew formulae the manager lists but will not upgrade
	sudoOK   bool
	fail     map[string]error // a command, as joined argv, that fails
}

func newPkgFake(outdated map[string][]string) *pkgFake {
	return &pkgFake{outdated: outdated, pinned: map[string]bool{}, fail: map[string]error{}}
}

func (f *pkgFake) clear(manager string, names ...string) {
	f.outdated[manager] = slices.DeleteFunc(f.outdated[manager], func(n string) bool {
		return slices.ContainsFunc(names, func(c string) bool { return strings.EqualFold(c, n) })
	})
}

func (f *pkgFake) run(name string, args ...string) ([]byte, error) {
	argv := strings.Join(append([]string{name}, args...), " ")
	f.log = append(f.log, argv)
	if err := f.fail[argv]; err != nil {
		return nil, err
	}
	switch {
	case argv == "sudo -n true":
		if !f.sudoOK {
			return nil, errors.New("sudo: a password is required")
		}
	case argv == "brew outdated --json=v2 --formula":
		return f.brewJSON("formulae", "brew"), nil
	case argv == "brew outdated --json=v2 --cask":
		return f.brewJSON("casks", "brew-cask"), nil
	case strings.HasPrefix(argv, "brew upgrade --cask "):
		f.clear("brew-cask", args[2:]...)
	case strings.HasPrefix(argv, "brew upgrade "):
		f.clear("brew", slices.DeleteFunc(slices.Clone(args[1:]), func(n string) bool { return f.pinned[n] })...)
	case argv == "apt list --upgradable":
		return f.aptList(), nil
	case strings.HasPrefix(argv, "sudo -n apt-get install --only-upgrade -y "):
		f.clear("apt", args[5:]...)
	case strings.HasPrefix(argv, "apt-get install --only-upgrade -y "):
		f.clear("apt", args[3:]...)
	case argv == "winget upgrade --accept-source-agreements":
		return f.wingetTable(), nil
	case strings.HasPrefix(argv, "winget upgrade --id "):
		f.clear("winget", args[2])
	case slices.Contains([]string{
		"brew update", "brew cleanup", "brew autoremove",
		"sudo -n apt-get update", "apt-get update", "sudo -n apt-get autoremove -y", "apt-get autoremove -y",
		"winget source update",
	}, argv):
	default:
		return nil, errors.New("unexpected command: " + argv)
	}
	return nil, nil
}

// brewJSON answers `brew outdated --json=v2`, with the fields brew prints.
func (f *pkgFake) brewJSON(key, manager string) []byte {
	type entry struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
		Pinned            bool     `json:"pinned"`
	}
	entries := []entry{}
	for _, n := range f.outdated[manager] {
		entries = append(entries, entry{n, []string{"1.0"}, "2.0", f.pinned[n]})
	}
	out, _ := json.Marshal(map[string]any{"formulae": []entry{}, "casks": []entry{}, key: entries})
	return out
}

func (f *pkgFake) aptList() []byte {
	lines := []string{"Listing..."}
	for _, n := range f.outdated["apt"] {
		lines = append(lines, n+"/jammy-updates 2.0 amd64 [upgradable from: 1.0]")
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func (f *pkgFake) wingetTable() []byte {
	lines := []string{
		"Name           Id                 Version Available Source",
		"-----------------------------------------------------------",
	}
	for _, id := range f.outdated["winget"] {
		lines = append(lines, "Some App       "+id+"   1.0     2.0       winget")
	}
	lines = append(lines, "Other App      Other.App          1.0     2.0       winget")
	return []byte(strings.Join(lines, "\r\n") + "\r\n")
}

// ran lists the commands that were run and that start with prefix.
func (f *pkgFake) ran(prefix string) []string {
	var out []string
	for _, argv := range f.log {
		if strings.HasPrefix(argv, prefix) {
			out = append(out, argv)
		}
	}
	return out
}

func (f *pkgFake) index(argv string) int { return slices.Index(f.log, argv) }

func upgradeEnv(t *testing.T, goos string) Env {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{"packages.json": upgradeCatalog})
	return Env{RepoRoot: repo, Home: t.TempDir(), GOOS: goos}
}

func upgradeStep(f *pkgFake) systemUpgrade {
	return systemUpgrade{
		run:  f.run,
		has:  func(string) bool { return true },
		root: func() bool { return false },
		arch: "amd64",
	}
}

// AC1: a plan lists the outdated catalog formulae and casks, only those, and
// reports the refresh as info. It cannot know what is outdated without it.
func TestSystemUpgradeDarwinPlanListsOnlyOutdatedCatalogEntries(t *testing.T) {
	env := upgradeEnv(t, "darwin")
	f := newPkgFake(map[string][]string{
		"brew":      {"git", "jq"}, // jq is installed by hand: outdated, not in the catalog
		"brew-cask": {"firefox", "slack"},
	})
	res, err := upgradeStep(f).Reconcile(env, true)
	if err != nil || res.Skip != "" {
		t.Fatalf("Reconcile = %+v, %v", res, err)
	}
	if res.Changes != 2 {
		t.Errorf("Changes = %d, want 2 (git, firefox); detail %q", res.Changes, res.Detail)
	}
	for _, want := range []string{"git", "firefox", "brew update"} {
		if !strings.Contains(res.Detail, want) {
			t.Errorf("detail %q does not mention %q", res.Detail, want)
		}
	}
	for _, not := range []string{"jq", "slack", "wget"} {
		if strings.Contains(res.Detail, not) {
			t.Errorf("detail %q names %q, which is not an outdated catalog entry", res.Detail, not)
		}
	}
	if f.index("brew update") < 0 {
		t.Error("a plan must refresh the index: it cannot know what is outdated otherwise")
	}
	if got := f.ran("brew upgrade"); len(got) != 0 {
		t.Errorf("a plan upgraded: %v", got)
	}
	if got := f.ran("brew cleanup"); len(got) != 0 {
		t.Errorf("a plan cleaned up: %v", got)
	}
}

// AC2: an apply upgrades them, cleans up after the upgrade, and a second run
// has nothing left to do and runs neither the upgrade nor the housekeeping.
func TestSystemUpgradeDarwinApplyUpgradesCleansAndConverges(t *testing.T) {
	env := upgradeEnv(t, "darwin")
	f := newPkgFake(map[string][]string{"brew": {"git", "wget"}, "brew-cask": {"firefox"}})
	step := upgradeStep(f)

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 3 {
		t.Fatalf("first apply = %+v, %v; want 3 changes", res, err)
	}
	for _, want := range []string{"brew upgrade git wget", "brew upgrade --cask firefox", "brew cleanup", "brew autoremove"} {
		if f.index(want) < 0 {
			t.Errorf("did not run %q; ran %v", want, f.log)
		}
	}
	if f.index("brew cleanup") < f.index("brew upgrade git wget") || f.index("brew autoremove") < f.index("brew upgrade --cask firefox") {
		t.Errorf("housekeeping ran before the upgrades: %v", f.log)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("Probe after the apply = %v", err)
	}

	f.log = nil
	res, err = step.Reconcile(env, false)
	if err != nil || res.Changes != 0 {
		t.Fatalf("second apply = %+v, %v; want changed=0", res, err)
	}
	if got := f.ran("brew upgrade"); len(got) != 0 {
		t.Errorf("second run upgraded: %v", got)
	}
	if f.index("brew cleanup") >= 0 || f.index("brew autoremove") >= 0 {
		t.Errorf("housekeeping runs only after an apply that upgraded something: %v", f.log)
	}
	if f.index("brew update") < 0 {
		t.Error("the refresh still runs when nothing is outdated")
	}
}

// A pinned formula is listed by brew and never moved by it: it is not a change,
// or the probe could never pass.
func TestSystemUpgradeDarwinApplyIgnoresPinnedFormula(t *testing.T) {
	env := upgradeEnv(t, "darwin")
	f := newPkgFake(map[string][]string{"brew": {"git"}})
	f.pinned["git"] = true
	step := upgradeStep(f)
	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 0 || len(f.ran("brew upgrade")) != 0 {
		t.Fatalf("Reconcile = %+v, %v; ran %v", res, err, f.log)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("Probe = %v", err)
	}
}

// A cask marked needs_sudo escalates inside brew, which has no terminal on a
// daily run: without sudo -n it is left out and named, and the others go on.
func TestSystemUpgradeDarwinApplySkipsASudoCaskWithoutSudo(t *testing.T) {
	for _, sudoOK := range []bool{false, true} {
		env := upgradeEnv(t, "darwin")
		f := newPkgFake(map[string][]string{"brew-cask": {"veracrypt", "firefox"}})
		f.sudoOK = sudoOK
		step := upgradeStep(f)
		res, err := step.Reconcile(env, false)
		if err != nil {
			t.Fatalf("sudoOK=%v: %v", sudoOK, err)
		}
		if sudoOK {
			if res.Changes != 2 || f.index("brew upgrade --cask veracrypt firefox") < 0 && f.index("brew upgrade --cask firefox veracrypt") < 0 {
				t.Errorf("with sudo: %+v, ran %v", res, f.log)
			}
			continue
		}
		if res.Changes != 1 || f.index("brew upgrade --cask firefox") < 0 {
			t.Errorf("without sudo: %+v, ran %v", res, f.log)
		}
		if !strings.Contains(res.Detail, "veracrypt") || !strings.Contains(res.Detail, "needs sudo") {
			t.Errorf("detail %q does not name the cask left for sudo", res.Detail)
		}
		if err := step.Probe(env); err != nil {
			t.Errorf("a cask skipped for privilege must not fail the probe: %v", err)
		}
	}
}

// AC3: a package the catalog does not name is never passed to a manager's
// upgrade, on any OS, even when the manager reports it outdated.
func TestSystemUpgradeOutdatedPackageOutsideCatalogIsNeverUpgraded(t *testing.T) {
	cases := []struct {
		goos     string
		outdated map[string][]string
	}{
		{"darwin", map[string][]string{"brew": {"git", "jq"}, "brew-cask": {"slack"}}},
		{"linux", map[string][]string{"apt": {"git", "curl"}}},
		{"windows", map[string][]string{"winget": {"Git.Git", "Slack.Slack"}}},
	}
	for _, c := range cases {
		t.Run(c.goos, func(t *testing.T) {
			env := upgradeEnv(t, c.goos)
			f := newPkgFake(c.outdated)
			f.sudoOK = true
			res, err := upgradeStep(f).Reconcile(env, false)
			if err != nil || res.Changes != 1 {
				t.Fatalf("Reconcile = %+v, %v; want only git", res, err)
			}
			for _, argv := range f.log {
				for _, outside := range []string{"jq", "slack", "Slack.Slack", "curl", "Other.App"} {
					if strings.Contains(argv, outside) {
						t.Errorf("%q names %s, which the catalog does not", argv, outside)
					}
				}
			}
		})
	}
}

// AC4: without non-interactive sudo, the step skips, names why and what to run,
// and runs no apt command at all (the refresh needs root too).
func TestSystemUpgradeLinuxWithoutSudoSkips(t *testing.T) {
	env := upgradeEnv(t, "linux")
	f := newPkgFake(map[string][]string{"apt": {"git"}})
	res, err := upgradeStep(f).Reconcile(env, false)
	if err != nil {
		t.Fatalf("a missing sudo must not fail the run: %v", err)
	}
	for _, want := range []string{"needs root", "dotf converge --only system-upgrade"} {
		if !strings.Contains(res.Skip, want) {
			t.Errorf("Skip = %q, want it to mention %q", res.Skip, want)
		}
	}
	if res.Changes != 0 {
		t.Errorf("Changes = %d", res.Changes)
	}
	if got := f.ran("sudo -n apt"); len(got) != 0 {
		t.Errorf("ran apt without sudo: %v", got)
	}
	if got := f.ran("apt"); len(got) != 0 {
		t.Errorf("ran apt without sudo: %v", got)
	}
}

// AC4: with sudo, only the catalog's outdated packages are upgraded, then
// autoremove runs; a plan stops before the upgrade.
func TestSystemUpgradeLinuxWithSudoUpgradesCatalogPackages(t *testing.T) {
	env := upgradeEnv(t, "linux")
	f := newPkgFake(map[string][]string{"apt": {"git", "curl", "eza"}})
	f.sudoOK = true
	step := upgradeStep(f)

	plan, err := step.Reconcile(env, true)
	if err != nil || plan.Changes != 2 || len(f.ran("sudo -n apt-get install")) != 0 {
		t.Fatalf("plan = %+v, %v; ran %v", plan, err, f.log)
	}
	if f.index("sudo -n apt-get update") < 0 {
		t.Error("a plan refreshes the index")
	}

	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 2 {
		t.Fatalf("apply = %+v, %v", res, err)
	}
	if f.index("sudo -n apt-get install --only-upgrade -y git eza") < 0 {
		t.Errorf("ran %v", f.log)
	}
	if f.index("sudo -n apt-get autoremove -y") < f.index("sudo -n apt-get install --only-upgrade -y git eza") || f.index("sudo -n apt-get autoremove -y") < 0 {
		t.Errorf("autoremove must follow the upgrade: %v", f.log)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("Probe = %v", err)
	}
}

// As root there is no sudo to ask for: the commands run bare, as the install
// path's asRoot does.
func TestSystemUpgradeLinuxAsRootRunsWithoutSudo(t *testing.T) {
	env := upgradeEnv(t, "linux")
	f := newPkgFake(map[string][]string{"apt": {"git"}})
	step := upgradeStep(f)
	step.root = func() bool { return true }
	res, err := step.Reconcile(env, false)
	if err != nil || res.Changes != 1 || res.Skip != "" {
		t.Fatalf("Reconcile = %+v, %v", res, err)
	}
	if f.index("apt-get install --only-upgrade -y git") < 0 || len(f.ran("sudo")) != 0 {
		t.Errorf("ran %v", f.log)
	}
}

// AC5: one winget upgrade per outdated catalog id, ids matched without regard
// to case; a failing id does not stop the next, fails the step, and the probe
// names it.
func TestSystemUpgradeWindowsUpgradesPerIdAndIsolatesAFailure(t *testing.T) {
	env := upgradeEnv(t, "windows")
	f := newPkgFake(map[string][]string{"winget": {"GIT.git", "EZA-Community.Eza"}})
	failing := "winget upgrade --id Git.Git -e --silent --accept-source-agreements --accept-package-agreements"
	f.fail[failing] = errors.New("exit status 0x8a150056: elevation required")
	step := upgradeStep(f)

	res, err := step.Reconcile(env, false)
	if err == nil || !strings.Contains(err.Error(), "Git.Git") {
		t.Fatalf("err = %v, want one naming Git.Git", err)
	}
	if res.Changes != 1 {
		t.Errorf("Changes = %d, want 1: only eza was upgraded", res.Changes)
	}
	if f.index(failing) < 0 || f.index("winget upgrade --id eza-community.eza -e --silent --accept-source-agreements --accept-package-agreements") < 0 {
		t.Errorf("each id gets its own upgrade, whatever happened to the first: %v", f.log)
	}
	if err := step.Probe(env); err == nil || !strings.Contains(err.Error(), "Git.Git") {
		t.Errorf("Probe = %v, want it to name Git.Git", err)
	}

	delete(f.fail, failing)
	if res, err := step.Reconcile(env, false); err != nil || res.Changes != 1 {
		t.Fatalf("retry = %+v, %v", res, err)
	}
	if err := step.Probe(env); err != nil {
		t.Errorf("Probe after the retry = %v", err)
	}
}

func TestSystemUpgradeWindowsPlanUpgradesNothing(t *testing.T) {
	env := upgradeEnv(t, "windows")
	f := newPkgFake(map[string][]string{"winget": {"Git.Git"}})
	res, err := upgradeStep(f).Reconcile(env, true)
	if err != nil || res.Changes != 1 || len(f.ran("winget upgrade --id")) != 0 {
		t.Fatalf("plan = %+v, %v; ran %v", res, err, f.log)
	}
	if f.index("winget source update") < 0 {
		t.Error("a plan refreshes the source")
	}
}

// AC6: a refresh that fails is a skip carrying the manager's error, never a
// failure and never an upgrade from an index that could not be updated.
func TestSystemUpgradeRefreshFailureSkipsWithTheManagersError(t *testing.T) {
	cases := []struct {
		goos, refresh, manager string
	}{
		{"darwin", "brew update", "brew"},
		{"linux", "sudo -n apt-get update", "apt"},
		{"windows", "winget source update", "winget"},
	}
	for _, c := range cases {
		t.Run(c.goos, func(t *testing.T) {
			env := upgradeEnv(t, c.goos)
			f := newPkgFake(map[string][]string{"brew": {"git"}, "apt": {"git"}, "winget": {"Git.Git"}})
			f.sudoOK = true
			f.fail[c.refresh] = errors.New("could not resolve host")
			res, err := upgradeStep(f).Reconcile(env, false)
			if err != nil {
				t.Fatalf("a failed refresh must not fail the run: %v", err)
			}
			for _, want := range []string{c.manager + " refresh failed", "could not resolve host"} {
				if !strings.Contains(res.Skip, want) {
					t.Errorf("Skip = %q, want it to mention %q", res.Skip, want)
				}
			}
			for _, argv := range f.log {
				if strings.Contains(argv, "upgrade") && !strings.HasSuffix(argv, "--upgradable") || strings.Contains(argv, "autoremove") {
					t.Errorf("ran %q after the refresh failed", argv)
				}
			}
		})
	}
}

// A machine without the manager (no brew yet, a Linux without apt-get) is a
// skip that says so; so is a registry whose caller wired no runner.
func TestSystemUpgradeSkipsWithoutAManagerOrARunner(t *testing.T) {
	env := upgradeEnv(t, "darwin")
	f := newPkgFake(map[string][]string{"brew": {"git"}})
	step := upgradeStep(f)
	step.has = func(string) bool { return false }
	if res, err := step.Reconcile(env, false); err != nil || !strings.Contains(res.Skip, "brew") || len(f.log) != 0 {
		t.Errorf("no manager: %+v, %v, ran %v", res, err, f.log)
	}

	unwired := systemUpgrade{has: func(string) bool { return true }}
	if res, err := unwired.Reconcile(env, false); err != nil || res.Skip == "" {
		t.Errorf("no runner: %+v, %v", res, err)
	}
	if err := unwired.Probe(env); err != nil {
		t.Errorf("an unwired step has nothing to probe: %v", err)
	}
}

// A catalog with no system entry for this OS has nothing to refresh either: no
// manager command runs.
func TestSystemUpgradeWithoutCatalogTargetsRunsNothing(t *testing.T) {
	env := upgradeEnv(t, "darwin")
	writeFixture(t, env.RepoRoot, map[string]string{"packages.json": `{"tools": [
  {"name": "hive", "version": "4.2.2", "source": {"type": "uv-tool", "package": "hive-vault"}}]}`})
	f := newPkgFake(map[string][]string{})
	res, err := upgradeStep(f).Reconcile(env, false)
	if err != nil || res.Changes != 0 || res.Skip != "" || len(f.log) != 0 {
		t.Errorf("Reconcile = %+v, %v, ran %v", res, err, f.log)
	}
}

func TestSystemUpgradeNameAndPlatforms(t *testing.T) {
	step := systemUpgrade{}
	if step.Name() != "system-upgrade" || step.Platforms() != nil {
		t.Errorf("Name %q, Platforms %v", step.Name(), step.Platforms())
	}
}
