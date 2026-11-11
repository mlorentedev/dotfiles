package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

const versionsFixture = `# versions.conf
GO_VERSION=1.26.0
# mise: cli
AGE_VERSION=1.3.1
# A note about shellcheck.
# mise: cli
SHELLCHECK_VERSION=0.11.0
PI_VERSION=0.87.1
# mise: cli
GOLANGCI_LINT_VERSION=2.12.2
`

func TestParseMiseTools_ReadsOnlyMarkedPinsAndDerivesTheName(t *testing.T) {
	got, err := ParseMiseTools([]byte(versionsFixture))
	if err != nil {
		t.Fatal(err)
	}
	want := []MiseTool{{"age", "1.3.1"}, {"golangci-lint", "2.12.2"}, {"shellcheck", "0.11.0"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tool %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseMiseTools_RejectsAMarkerThatMarksNoPin(t *testing.T) {
	for name, body := range map[string]string{
		"marker before a comment": "# mise: cli\n# stray\nAGE_VERSION=1.3.1\n",
		"marker before a non-pin": "# mise: cli\nDOTFILES_DIR=x\n",
		"marker at end of file":   "AGE_VERSION=1.3.1\n# mise: cli\n",
		"marker before no value":  "# mise: cli\nAGE_VERSION=\n",
	} {
		if _, err := ParseMiseTools([]byte(body)); err == nil {
			t.Errorf("%s: want an error, got none", name)
		}
	}
}

func TestRenderMiseConfig_IsSortedPinnedAndNamesItsSource(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}, {"golangci-lint", "2.12.2"}}, ""))
	for _, want := range []string{"dotf tools sync", "versions.conf", "[tools]\n", "age = \"1.3.1\"\n", "golangci-lint = \"2.12.2\"\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered config lacks %q:\n%s", want, out)
		}
	}
}

func TestRenderMiseConfig_WindowsBatsUsesRunnableNpmBackend(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the Windows backend is exercised on the Windows runner")
	}
	out := string(RenderMiseConfig([]MiseTool{{"bats", "1.13.0"}, {"jq", "1.8.2"}}, ""))
	for _, want := range []string{`"npm:bats" = "1.13.0"`, `jq = "1.8.2"`} {
		if !strings.Contains(out, want) {
			t.Errorf("Windows mise config lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "npm.package_manager") || strings.Contains(out, "[settings.npm]") {
		t.Errorf("bats must not change the package manager of unrelated npm tools:\n%s", out)
	}
	if strings.Contains(out, "\nbats = ") {
		t.Errorf("the aqua bats recipe excludes Windows:\n%s", out)
	}
}

func TestRenderMiseConfig_WindowsDirenvNamesTheReleaseBinaryAsExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the Windows direnv release lacks an .exe suffix")
	}
	out := string(RenderMiseConfig([]MiseTool{{"direnv", "2.37.1"}, {"jq", "1.8.2"}}, ""))
	if !strings.Contains(out, `"github:direnv/direnv" = { version = "2.37.1", bin = "direnv.exe" }`) {
		t.Errorf("Windows mise config does not name the runnable direnv binary:\n%s", out)
	}
	if !strings.Contains(out, `jq = "1.8.2"`) {
		t.Errorf("unrelated pins must use the registry:\n%s", out)
	}
	if strings.Contains(out, "\ndirenv = ") {
		t.Errorf("the aqua recipe installs direnv without .exe on Windows:\n%s", out)
	}
}

// #2438: a catalog npm CLI moves under mise, so one owner holds it on every OS.
// Aqua has no recipe for these four, so each short name is aliased to mise's npm
// backend, and every other reader (mise which, ls, bin-paths, the doctor) keeps
// asking for copilot. Scoped packages (@github/copilot) and a name that is not
// the package (opencode-ai) are why this is a table and not a rule.
func TestRenderMiseConfig_RoutesNpmCLIsToTheNpmBackend(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{
		{"bw", "2026.5.0"}, {"copilot", "1.0.81"}, {"jq", "1.8.2"}, {"opencode", "1.16.2"}, {"yarn", "1.22.22"},
	}, ""))
	for _, want := range []string{
		"bw = \"2026.5.0\"\n", "copilot = \"1.0.81\"\n", "opencode = \"1.16.2\"\n", "yarn = \"1.22.22\"\n",
		"bw = \"npm:@bitwarden/cli\"\n", "copilot = \"npm:@github/copilot\"\n",
		"opencode = \"npm:opencode-ai\"\n", "yarn = \"npm:yarn\"\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "jq = \"npm:") || strings.Count(out, "[tool_alias]") != 1 {
		t.Errorf("only the npm CLIs are aliased, under one section:\n%s", out)
	}
	if out := string(RenderMiseConfig([]MiseTool{{"jq", "1.8.2"}}, "")); strings.Contains(out, "npm:") {
		t.Errorf("no npm alias without an npm CLI:\n%s", out)
	}
}

// An alias section holds both kinds: herdr's attested backend and the npm CLIs.
func TestRenderMiseConfig_NpmAliasesShareTheSectionWithHerdr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no herdr alias on Windows")
	}
	out := string(RenderMiseConfig([]MiseTool{{"herdr", "0.9.3"}, {"yarn", "1.22.22"}}, ""))
	if strings.Count(out, "[tool_alias]") != 1 || !strings.Contains(out, "herdr = \"github:herdrdev/herdr\"") || !strings.Contains(out, "yarn = \"npm:yarn\"") {
		t.Errorf("one [tool_alias] with both:\n%s", out)
	}
}

// The npm backend runs the package on Node, and mise does not order Node for a
// tool whose pin it cannot see: refuse before writing a config nothing can fill.
func TestSync_RefusesAnNpmCLIWithoutPinnedNode(t *testing.T) {
	s := MiseSync{ConfigDir: t.TempDir(), Run: func(string, ...string) ([]byte, error) {
		return nil, errors.New("not installed")
	}}
	_, err := s.Apply([]MiseTool{{"copilot", "1.0.81"}})
	if err == nil || !strings.Contains(err.Error(), "node") || !strings.Contains(err.Error(), "copilot") {
		t.Fatalf("want the missing Node pin and the tool named, got %v", err)
	}
	if _, err := os.Stat(s.ConfigPath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused sync must not write an unfulfillable config: %v", err)
	}
}

func TestNpmInstallTarget(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"install", "--force", "npm:bats"}, "bats"},
		{[]string{"install", "--force", "copilot"}, "copilot"},
		{[]string{"install", "--force", "yarn"}, "yarn"},
		{[]string{"install", "--force", "jq"}, ""},
		{[]string{"install", "--force", "npm:prettier"}, ""},
		{[]string{"install", "node"}, ""},
		{[]string{"install"}, ""},
		{[]string{"which", "copilot"}, ""},
	} {
		if got := npmInstallTarget("mise", tc.args...); got != tc.want {
			t.Errorf("npmInstallTarget(%q) = %q, want %q", tc.args, got, tc.want)
		}
	}
	if got := npmInstallTarget("jq", "install", "--force", "yarn"); got != "" {
		t.Errorf("only mise installs: %q", got)
	}
}

// #2013 H3: aqua's registry entry for herdr checks the release digest only,
// while mise's github backend checks it and the GitHub artifact attestations
// upstream publishes for the linux and macOS assets. A [tool_alias] routes
// the short name there, so every other reader (mise which, where, bin-paths,
// the doctor and verify-setup.bats) keeps asking for herdr. The Windows zip
// carries no attestation, and mise refuses an asset without one, so Windows
// renders no alias.
func TestRenderMiseConfig_RoutesHerdrToTheBackendThatVerifiesAttestations(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"herdr", "0.9.3"}, {"jq", "1.8.2"}}, ""))
	alias := "\n[tool_alias]\nherdr = \"github:herdrdev/herdr\"\n"
	if runtime.GOOS == "windows" {
		if strings.Contains(out, "[tool_alias]") {
			t.Errorf("the Windows herdr zip has no attestation; no alias there:\n%s", out)
		}
		return
	}
	if !strings.Contains(out, alias) || !strings.Contains(out, "herdr = \"0.9.3\"\n") {
		t.Errorf("want herdr pinned by its short name and aliased to the github backend:\n%s", out)
	}
	if strings.Count(out, "[tool_alias]") != 1 || strings.Contains(out, "jq = \"github:") {
		t.Errorf("only herdr is aliased:\n%s", out)
	}
	if out := string(RenderMiseConfig([]MiseTool{{"jq", "1.8.2"}}, "")); strings.Contains(out, "[tool_alias]") {
		t.Errorf("no alias section without herdr:\n%s", out)
	}
}

// mise itself is pinned in packages.json and installed by `dotf tools install`.
// Its own update notice tells the user to run `mise self-update`, which would
// move it past that pin, so the rendered config turns the notice off.
func TestRenderMiseConfig_TurnsOffMiseOwnUpdateNotice(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}}, ""))
	if !strings.Contains(out, "\n[settings]\ndisable_update_warning = true\n") {
		t.Errorf("want disable_update_warning = true under [settings]:\n%s", out)
	}
	// The pins stay in [tools]: a table header ends the one before it.
	if strings.Index(out, "age = ") > strings.Index(out, "[settings]") {
		t.Errorf("the pins must come before [settings]:\n%s", out)
	}
}

// mise's default 30m budget for one download failed a from-zero converge on
// argocd, the largest pinned asset, over a slow link (#3181).
func TestRenderMiseConfig_SizesTheDownloadBudgetToTheLargestAsset(t *testing.T) {
	for _, py := range []string{"", "/cfg/py.txt"} {
		out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}}, py))
		var settings struct {
			Settings map[string]any `toml:"settings"`
		}
		if err := toml.Unmarshal([]byte(out), &settings); err != nil {
			t.Fatalf("rendered config does not parse: %v\n%s", err, out)
		}
		if got := settings.Settings["http_download_timeout"]; got != "2h" {
			t.Errorf("http_download_timeout = %v, want \"2h\":\n%s", got, out)
		}
	}
}

// A tool mise keeps current must not also update itself: Claude Code's own
// updater would replace the binary mise installed, outside the pin's floor and
// outside converge's report (#2013 D9). The env lives in the config mise owns,
// so it reaches every claude mise runs (activate and shims alike) and no other.
func TestRenderMiseConfig_TurnsOffTheOwnUpdaterOfAToolMiseManages(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}, {"claude", "2.1.296"}}, "", "claude"))
	if !strings.Contains(out, "\n[env]\nDISABLE_AUTOUPDATER = \"1\"\n") {
		t.Errorf("want DISABLE_AUTOUPDATER = \"1\" under [env] when mise manages claude:\n%s", out)
	}
	if strings.Index(out, "claude = ") > strings.Index(out, "[env]") {
		t.Errorf("the pins must come before [env]:\n%s", out)
	}
	without := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}}, ""))
	if strings.Contains(without, "[env]") || strings.Contains(without, "DISABLE_AUTOUPDATER") {
		t.Errorf("a config that does not manage claude must leave its updater alone:\n%s", without)
	}
}

// pi has no updater of its own, but its startup check tells the user to run
// `pi update`, which would install a second pi outside mise. The check is off
// wherever mise manages pi, and both settings land in one [env] table.
func TestRenderMiseConfig_SilencesPisUpdateCheck(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"claude", "2.1.296"}, {"pi", "1.2.0"}}, "", "claude", "pi"))
	if !strings.Contains(out, "\n[env]\nDISABLE_AUTOUPDATER = \"1\"\nPI_SKIP_VERSION_CHECK = \"1\"\n") {
		t.Errorf("want both settings under one [env]:\n%s", out)
	}
}

// gh has no updater either, but its notice names the channel that installed
// it before mise did (`brew upgrade gh`), a second gh outside the pin (#2136).
func TestRenderMiseConfig_SilencesGhsUpdateNotice(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"gh", "2.102.0"}}, ""))
	if !strings.Contains(out, "\n[env]\nGH_NO_UPDATE_NOTIFIER = \"1\"\n") {
		t.Errorf("want GH_NO_UPDATE_NOTIFIER = \"1\" under [env] when mise manages gh:\n%s", out)
	}
}

// fakeMise answers `mise which <tool>` for the tools in installed, and
// `<path> --version` with the version installed records; `mise install` marks
// every pending tool installed at its pin.
type fakeMise struct {
	installed map[string]string // tool -> version
	pending   []MiseTool
	installs  int
	// newest is each tool's newest release. `mise outdated` reports a tool
	// only when the config at cfg requests it as "latest", as mise does: an
	// exact pin is never outdated. `mise upgrade` installs the newest.
	newest   map[string]string
	cfg      string
	outdates int
	upgrades [][]string
	// prunable is each tool's installed versions no tracked config requires,
	// as `mise ls --prunable --json` reports them; `mise prune` removes the
	// tools it names.
	prunable map[string][]string
	// keptByConfig is installs the config at cfg still asks for: prunable
	// only once a write drops the tool from it, as mise counts a tracked
	// config's requests (#2677).
	keptByConfig map[string][]string
	listPrune    int
	prunes       [][]string
	listFails    bool // `mise ls --prunable` exits non-zero
	// listFailsFrom, when > 0, makes the listing exit non-zero from that
	// call on: the first plan sees the orphans, the check after the prune
	// cannot.
	listFailsFrom int
	// listFailsAt, when > 0, makes only that call of the listing fail.
	listFailsAt int
	// calls counts every mise command, refused or not.
	calls int
}

func (f *fakeMise) run(name string, args ...string) ([]byte, error) {
	if name == "mise" {
		f.calls++
		// Real mise refuses every command over a config at cfg that does not
		// parse (measured: `ls`, `outdated`, `ls --prunable` all exit 1).
		if body, err := os.ReadFile(f.cfg); err == nil && toml.Unmarshal(body, &map[string]any{}) != nil {
			return []byte("mise ERROR Error loading settings file " + f.cfg + ": TOML parse error"), errors.New("exit status 1")
		}
	}
	switch {
	case name == "mise" && len(args) >= 2 && args[0] == "outdated" && args[1] == "--json":
		f.outdates++
		body, _ := os.ReadFile(f.cfg)
		report := map[string]map[string]string{}
		for _, tool := range args[2:] {
			if strings.Contains(string(body), tool+` = "latest"`) && f.newest[tool] != "" && f.newest[tool] != f.installed[tool] {
				report[tool] = map[string]string{"requested": "latest", "current": f.installed[tool], "latest": f.newest[tool]}
			}
		}
		return json.Marshal(report)
	case name == "mise" && len(args) >= 2 && args[0] == "upgrade":
		f.upgrades = append(f.upgrades, args[1:])
		for _, tool := range args[1:] {
			f.installed[tool] = f.newest[tool]
		}
		return nil, nil
	case name == "mise" && len(args) == 3 && args[0] == "ls" && args[1] == "--prunable" && args[2] == "--json":
		f.listPrune++
		if f.listFails || (f.listFailsFrom > 0 && f.listPrune >= f.listFailsFrom) || f.listPrune == f.listFailsAt {
			return []byte("mise ERROR failed to read a tracked config"), errors.New("exit status 1")
		}
		report := map[string][]map[string]any{}
		for tool, versions := range f.prunable {
			for _, v := range versions {
				report[tool] = append(report[tool], map[string]any{"version": v, "installed": true, "active": false})
			}
		}
		body, _ := os.ReadFile(f.cfg)
		for tool, versions := range f.keptByConfig {
			if strings.Contains(string(body), `"`+tool+`"`) {
				continue
			}
			for _, v := range versions {
				report[tool] = append(report[tool], map[string]any{"version": v, "installed": true, "active": false})
			}
		}
		return json.Marshal(report)
	case name == "mise" && len(args) >= 3 && args[0] == "prune" && args[1] == "--yes" && args[2] == "--tools":
		f.prunes = append(f.prunes, args[3:])
		for _, tool := range args[3:] {
			delete(f.prunable, tool)
			delete(f.keptByConfig, tool)
		}
		return nil, nil
	case name == "mise" && len(args) == 2 && args[0] == "which":
		if _, ok := f.installed[args[1]]; ok {
			return []byte("/mise/installs/" + args[1] + "\n"), nil
		}
		return nil, errors.New("not installed")
	case name == "mise" && len(args) == 1 && args[0] == "install":
		f.installs++
		for _, t := range f.pending {
			f.installed[t.Name] = t.Version
		}
		return nil, nil
	case strings.HasPrefix(name, "/mise/installs/") && len(args) == 1 && args[0] == "--version":
		return []byte(f.installed[strings.TrimPrefix(name, "/mise/installs/")] + "\n"), nil
	}
	return nil, errors.New("unexpected call: " + name + " " + strings.Join(args, " "))
}

func TestSync_RendersInstallsProbesAndIsIdempotent(t *testing.T) {
	cfgDir := t.TempDir()
	tools := []MiseTool{{"age", "1.3.1"}, {"jq", "1.8.2"}}
	f := &fakeMise{installed: map[string]string{}, pending: tools}
	s := MiseSync{ConfigDir: cfgDir, Run: f.run}

	plan, err := s.Plan(tools)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ConfigChanged || len(plan.Missing) != 2 {
		t.Fatalf("fresh plan: want the config to change and 2 tools missing, got %+v", plan)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "conf.d")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a plan wrote the config")
	}

	if _, err := s.Apply(tools); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(cfgDir, "conf.d", "dotfiles.toml")); err != nil || !strings.Contains(string(body), `jq = "1.8.2"`) {
		t.Fatalf("rendered config: %q, %v", body, err)
	}

	again, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if again.ConfigChanged || len(again.Missing) != 0 || f.installs != 1 {
		t.Errorf("second run: want nothing to do and no second install, got %+v after %d installs", again, f.installs)
	}
}

func TestSync_WindowsBatsRequiresARunnableCmdShim(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows has a distinct launcher for Bash scripts")
	}
	for _, tc := range []struct {
		name    string
		cmdShim bool
		missing bool
	}{
		{"npm generated launcher", true, false},
		{"aube package without launcher", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func(name string, args ...string) ([]byte, error) {
				switch {
				case name == "mise" && len(args) == 2 && args[0] == "which" && args[1] == "bats":
					return []byte(`C:\mise\bats`), nil
				case name == "mise" && len(args) == 2 && args[0] == "which" && args[1] == "bats.cmd" && tc.cmdShim:
					return []byte(`C:\mise\bats.cmd`), nil
				case name == `C:\mise\bats.cmd` && len(args) == 1 && args[0] == "--version":
					return []byte("Bats 1.13.0"), nil
				}
				return nil, errors.New("unavailable on Windows")
			}
			plan, err := (MiseSync{ConfigDir: t.TempDir(), Run: run}).Plan([]MiseTool{{"bats", "1.13.0"}})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(plan.Missing) == 1; got != tc.missing {
				t.Errorf("missing bats = %v, want %v; plan: %+v", got, tc.missing, plan)
			}
		})
	}
}

func TestSync_WindowsInstallsBatsWithNodeBeforeTheRest(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows bats requires npm to create a .cmd launcher")
	}
	pins := []MiseTool{{"node", "24.11.1"}, {"bats", "1.13.0"}, {"jq", "1.8.2"}}
	installed := map[string]bool{}
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		if name == "mise" {
			switch {
			case len(args) == 2 && args[0] == "which":
				key := strings.TrimSuffix(args[1], ".cmd")
				if installed[key] {
					return []byte(key), nil
				}
				return nil, errors.New("not installed")
			case len(args) >= 1 && args[0] == "install":
				calls = append(calls, strings.Join(args, " "))
				if len(args) == 2 && args[1] == "node" {
					installed["node"] = true
				} else if len(args) == 3 && args[1] == "--force" && args[2] == "npm:bats" {
					if !installed["node"] {
						return nil, errors.New("bats installed before node")
					}
					installed["bats"] = true
				} else if len(args) == 1 {
					installed["jq"] = true
				} else {
					return nil, errors.New("unexpected install order")
				}
				return nil, nil
			}
		}
		if installed[name] && len(args) == 1 && args[0] == "--version" {
			for _, pin := range pins {
				if pin.Name == name {
					return []byte(pin.Version), nil
				}
			}
		}
		return nil, errors.New("unexpected command")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: run}
	if _, err := s.Apply(pins); err != nil {
		t.Fatal(err)
	}
	second, err := s.Apply(pins)
	if err != nil || second.Pending() {
		t.Fatalf("second sync: %+v, %v", second, err)
	}
	if got := strings.Join(calls, ", "); got != "install node, install --force npm:bats, install" {
		t.Errorf("install sequence = %q, want scoped bats before the other pins", got)
	}
}

// #2438: the npm CLIs get the same treatment as bats on Windows: Node first,
// then each through the npm installer, then the bare install for the rest.
func TestSync_WindowsInstallsNpmCLIsWithNodeFirst(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows needs npm to create the .cmd launchers")
	}
	pins := []MiseTool{{"copilot", "1.0.81"}, {"jq", "1.8.2"}, {"node", "24.11.1"}, {"yarn", "1.22.22"}}
	installed := map[string]bool{}
	var calls []string
	run := func(name string, args ...string) ([]byte, error) {
		if name == "mise" {
			switch {
			case len(args) == 2 && args[0] == "which":
				if installed[args[1]] {
					return []byte(args[1]), nil
				}
				return nil, errors.New("not installed")
			case len(args) >= 1 && args[0] == "install":
				calls = append(calls, strings.Join(args, " "))
				switch {
				case len(args) == 2 && args[1] == "node":
					installed["node"] = true
				case len(args) == 3 && args[1] == "--force" && (args[2] == "copilot" || args[2] == "yarn"):
					if !installed["node"] {
						return nil, errors.New(args[2] + " installed before node")
					}
					installed[args[2]] = true
				case len(args) == 1:
					installed["jq"] = true
				default:
					return nil, errors.New("unexpected install")
				}
				return nil, nil
			}
		}
		if installed[name] && len(args) == 1 && args[0] == "--version" {
			for _, pin := range pins {
				if pin.Name == name {
					return []byte(pin.Version), nil
				}
			}
		}
		return nil, errors.New("unexpected command")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: run}
	if _, err := s.Apply(pins); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls, ", "); got != "install node, install --force copilot, install --force yarn, install" {
		t.Errorf("install sequence = %q", got)
	}
}

func TestSync_WindowsRepairsAnAubeBatsWithoutLauncher(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("mise's aube reports bats installed but creates no Windows launcher")
	}
	tools := []MiseTool{{"bats", "1.13.0"}, {"node", "24.11.1"}}
	reinstalled := false
	run := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && len(args) == 2 && args[0] == "which" && args[1] == "node":
			return []byte("node.exe"), nil
		case name == "mise" && len(args) == 2 && args[0] == "which" && args[1] == "bats.cmd" && reinstalled:
			return []byte("bats.cmd"), nil
		case name == "mise" && len(args) == 2 && args[0] == "install" && args[1] == "node":
			return nil, nil
		case name == "mise" && len(args) == 3 && args[0] == "install" && args[1] == "--force" && args[2] == "npm:bats":
			reinstalled = true
			return nil, nil
		case name == "mise" && len(args) == 1 && args[0] == "install":
			return nil, nil
		case name == "node.exe" && len(args) == 1 && args[0] == "--version":
			return []byte("v24.11.1"), nil
		case name == "bats.cmd" && len(args) == 1 && args[0] == "--version":
			return []byte("Bats 1.13.0"), nil
		}
		return nil, errors.New("not found")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: run}
	if _, err := s.Apply(tools); err != nil || !reinstalled {
		t.Errorf("broken bats should be replaced by npm: reinstalled=%v, err=%v", reinstalled, err)
	}
}

func TestSync_WindowsRefusesBatsWithoutPinnedNode(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the Windows npm installer needs mise's Node from zero")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: func(string, ...string) ([]byte, error) {
		return nil, errors.New("not installed")
	}}
	if _, err := s.Apply([]MiseTool{{"bats", "1.13.0"}}); err == nil || !strings.Contains(err.Error(), "node") {
		t.Fatalf("want missing Node pin named, got %v", err)
	}
	if _, err := os.Stat(s.ConfigPath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused sync must not write an unfulfillable config: %v", err)
	}
}

func TestHomeRunners_WindowsScopesNpmToBatsInstall(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows bats needs npm, other mise npm tools keep their installer")
	}
	if got := miseInstallEnv("mise", "install", "--force", "npm:bats"); got != "MISE_NPM_PACKAGE_MANAGER=npm" {
		t.Errorf("bats install env = %q", got)
	}
	if got := miseInstallEnv("mise", "install", "--force", "copilot"); got != "MISE_NPM_PACKAGE_MANAGER=npm" {
		t.Errorf("copilot install env = %q", got)
	}
	for _, args := range [][]string{{"mise", "install"}, {"mise", "install", "node"}, {"mise", "which", "bats.cmd"}, {"mise", "install", "npm:prettier"}} {
		if got := miseInstallEnv(args[0], args[1:]...); got != "" {
			t.Errorf("%v unexpectedly changes the npm installer: %q", args, got)
		}
	}
}

func TestSync_FailsNamingAToolThatDoesNotRunAtItsPin(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}, {"jq", "1.8.2"}}
	// mise install "succeeds", but jq answers below its pin.
	f := &fakeMise{installed: map[string]string{}, pending: []MiseTool{{"age", "1.3.1"}, {"jq", "1.7.1"}}}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run}

	_, err := s.Apply(tools)
	if err == nil || !strings.Contains(err.Error(), "jq") || strings.Contains(err.Error(), "age") {
		t.Fatalf("want an error naming only jq, got %v", err)
	}
}

func TestSync_LeavesTheHandWrittenConfigAlone(t *testing.T) {
	cfgDir := t.TempDir()
	hand := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(hand, []byte("[tools]\ngo = \"1.26.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{}, pending: tools}

	if _, err := (MiseSync{ConfigDir: cfgDir, Run: f.run}).Apply(tools); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(hand); string(body) != "[tools]\ngo = \"1.26.0\"\n" {
		t.Errorf("sync touched config.toml: %q", body)
	}
}

func TestMiseConfigDir_FollowsMiseOwnResolution(t *testing.T) {
	home := filepath.FromSlash("/home/u")
	cases := []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"MISE_CONFIG_DIR": "/x/mise"}, "/x/mise"},
		{map[string]string{"XDG_CONFIG_HOME": "/xdg"}, filepath.Join("/xdg", "mise")},
		{map[string]string{}, filepath.Join(home, ".config", "mise")},
	}
	for _, tc := range cases {
		got := MiseConfigDir(home, func(k string) string { return tc.env[k] })
		if got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.env, got, tc.want)
		}
	}
}

func TestMiseShimsDir_FollowsMiseOwnResolution(t *testing.T) {
	home := filepath.FromSlash("/home/u")
	local := filepath.FromSlash("/profiles/u/Local")
	cases := []struct {
		goos string
		env  map[string]string
		want string
	}{
		{"linux", map[string]string{"MISE_DATA_DIR": "/d/mise", "XDG_DATA_HOME": "/xdg"}, filepath.Join("/d/mise", "shims")},
		{"linux", map[string]string{"XDG_DATA_HOME": "/xdg"}, filepath.Join("/xdg", "mise", "shims")},
		{"linux", map[string]string{}, filepath.Join(home, ".local", "share", "mise", "shims")},
		{"darwin", map[string]string{"LOCALAPPDATA": local}, filepath.Join(home, ".local", "share", "mise", "shims")},
		// mise's Windows default is %LOCALAPPDATA%\mise, not the XDG one.
		{"windows", map[string]string{"LOCALAPPDATA": local}, filepath.Join(local, "mise", "shims")},
		{"windows", map[string]string{}, filepath.Join(home, "AppData", "Local", "mise", "shims")},
		{"windows", map[string]string{"XDG_DATA_HOME": "/xdg", "LOCALAPPDATA": local}, filepath.Join("/xdg", "mise", "shims")},
	}
	for _, tc := range cases {
		if got := MiseShimsDir(home, tc.goos, func(k string) string { return tc.env[k] }); got != tc.want {
			t.Errorf("%s %v: got %q, want %q", tc.goos, tc.env, got, tc.want)
		}
	}
}

func TestParseMiseTools_RejectsAFileWithNoMarkers(t *testing.T) {
	if _, err := ParseMiseTools([]byte("GO_VERSION=1.26.0\nAGE_VERSION=1.3.1\n")); err == nil || !strings.Contains(err.Error(), "nothing to sync") {
		t.Fatalf("want an error for a versions.conf with no markers, got %v", err)
	}
}

// mise prints warnings on stderr; only stdout is the path.
func TestSync_ReadsTheToolPathFromStdoutOnly(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	merged := func(name string, args ...string) ([]byte, error) {
		if name == "mise" && args[0] == "which" {
			return []byte("mise WARN  config not trusted\n/bin/age\n"), nil
		}
		if name == "/bin/age" {
			return []byte("v1.3.1"), nil
		}
		return nil, nil
	}
	stdout := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && args[0] == "which":
			return []byte("/bin/age\n"), nil
		case name == "/bin/age":
			return []byte("v1.3.1"), nil
		}
		return nil, errors.New("unexpected")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: merged, Stdout: stdout}

	p, err := s.Plan(tools)
	if err != nil || len(p.Missing) != 0 {
		t.Fatalf("want age at its pin through the stdout runner, got %+v, %v", p, err)
	}
}

// A tool is read the way ProbeRunner reads it (BUG-142, the BUG-141 class):
// stdout first, so a warning that reaches the merged stream before the banner
// never wins; a failed run only by its stdout, so ansible-lint's crash with
// ansible-core's 2.21.5 on stderr is not at its pin.
func TestSync_ProbesStdoutFirstAndAFailedRunOnlyThere(t *testing.T) {
	for name, tc := range map[string]struct {
		merged, stdout string
		fails, missing bool
	}{
		"a crash naming another package's version": {merged: "UNSUPPORTED: ansible-core 2.21.5\n", fails: true, missing: true},
		"a version, then an unrelated failure":     {merged: "age v1.3.1\nerror: x\n", stdout: "age v1.3.1\n", fails: true},
		"a warning on stderr before the banner":    {merged: "urllib3 needs OpenSSL 1.1.1+\nage v1.3.1\n", stdout: "age v1.3.1\n"},
		"a version only on stderr, exit 0":         {merged: "age v1.3.1\n"},
	} {
		t.Run(name, func(t *testing.T) {
			var fail error
			if tc.fails {
				fail = errors.New("exit status 3")
			}
			merged := func(name string, args ...string) ([]byte, error) {
				if name == "/bin/age" {
					return []byte(tc.merged), fail
				}
				return nil, nil
			}
			stdout := func(name string, args ...string) ([]byte, error) {
				switch name {
				case "mise":
					return []byte("/bin/age\n"), nil
				case "/bin/age":
					return []byte(tc.stdout), fail
				}
				return nil, errors.New("unexpected")
			}
			s := MiseSync{ConfigDir: t.TempDir(), Run: merged, Stdout: stdout}
			p, err := s.Plan([]MiseTool{{"age", "1.3.1"}})
			if err != nil || (len(p.Missing) == 1) != tc.missing {
				t.Fatalf("missing = %v, want %v (%v)", p.Missing, tc.missing, err)
			}
		})
	}
}

// A comment that reads like the marker but is not it would otherwise be an
// ordinary comment: the pin under it never installs while sync reports success.
func TestParseMiseTools_RejectsANearMissMarker(t *testing.T) {
	for _, marker := range []string{"#mise: cli", "# MISE: cli", "# mise:cli"} {
		body := "# mise: cli\nAGE_VERSION=1.3.1\n" + marker + "\nJQ_VERSION=1.8.2\n"
		_, err := ParseMiseTools([]byte(body))
		if err == nil || !strings.Contains(err.Error(), "line 3") {
			t.Errorf("%q: want an error naming line 3, got %v", marker, err)
		}
	}
}

func TestParseMisePins_ReadsBothMarkersSorted(t *testing.T) {
	body := "# mise: cli\nPYTHON_VERSION=3.12.6\n# mise: python-package\nPYYAML_VERSION=6.0.3\n# mise: cli\nAGE_VERSION=1.3.1\n# mise: python-package\nATTRS_VERSION=25.1.0\n"
	p, err := ParseMisePins([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := names(p.Tools); got != "age, python" {
		t.Errorf("tools: got %q", got)
	}
	if got := names(p.PythonPackages); got != "attrs, pyyaml" {
		t.Errorf("python packages: got %q", got)
	}
}

func TestParseMisePins_RejectsAPackageNothingWouldInstall(t *testing.T) {
	body := "# mise: cli\nAGE_VERSION=1.3.1\n# mise: python-package\nPYYAML_VERSION=6.0.3\n"
	_, err := ParseMisePins([]byte(body))
	if err == nil || !strings.Contains(err.Error(), "pyyaml") || !strings.Contains(err.Error(), "PYTHON_VERSION") {
		t.Fatalf("want an error naming pyyaml and PYTHON_VERSION, got %v", err)
	}
}

func TestParseMisePins_NearMissNamesBothMarkers(t *testing.T) {
	body := "# mise: cli\nPYTHON_VERSION=3.12.6\n# mise: python-packages\nPYYAML_VERSION=6.0.3\n"
	_, err := ParseMisePins([]byte(body))
	if err == nil || !strings.Contains(err.Error(), MiseMarker) || !strings.Contains(err.Error(), MisePythonPackageMarker) {
		t.Fatalf("want an error naming both markers, got %v", err)
	}
}

func TestRenderMiseConfig_NamesThePackagesFileOnlyWhenThereIsOne(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"python", "3.12.6"}}, "/cfg/conf.d/dotfiles-python-packages.txt"))
	if !strings.Contains(out, "\n[settings]\ndisable_update_warning = true\npython.default_packages_file = \"/cfg/conf.d/dotfiles-python-packages.txt\"\n") {
		t.Errorf("want python.default_packages_file under [settings]:\n%s", out)
	}
	if strings.Contains(string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}}, "")), "default_packages_file") {
		t.Error("a config with no python packages must not name a packages file")
	}
}

// fakePython extends fakeMise with a python that answers the distribution
// probe and `-m pip install`. When mise installs python, it installs the
// packages the packages file lists at that moment, as mise does.
type fakePython struct {
	fakeMise
	pkgFile  string
	dists    map[string]string // distribution -> installed version
	pips     [][]string
	pipFails bool
	pipNoop  bool
}

func (f *fakePython) run(name string, args ...string) ([]byte, error) {
	const python = "/mise/installs/python"
	switch {
	case name == "mise" && len(args) == 1 && args[0] == "install":
		_, had := f.installed["python"]
		out, err := f.fakeMise.run(name, args...)
		if _, has := f.installed["python"]; has && !had {
			body, _ := os.ReadFile(f.pkgFile)
			for _, l := range strings.Split(string(body), "\n") {
				if n, v, ok := strings.Cut(l, "=="); ok && !strings.HasPrefix(l, "#") {
					f.dists[n] = v
				}
			}
		}
		return out, err
	case name == python && len(args) == 3 && args[0] == "-c":
		if v, ok := f.dists[args[2]]; ok {
			return []byte(v + "\n"), nil
		}
		return []byte("PackageNotFoundError: " + args[2]), errors.New("exit status 1")
	case name == python && len(args) > 2 && args[0] == "-m" && args[1] == "pip":
		f.pips = append(f.pips, args)
		if f.pipFails {
			return []byte("ERROR: no network"), errors.New("exit status 1")
		}
		if !f.pipNoop {
			for _, a := range args {
				if n, v, ok := strings.Cut(a, "=="); ok {
					f.dists[n] = v
				}
			}
		}
		return nil, nil
	}
	return f.fakeMise.run(name, args...)
}

func newFakePython(t *testing.T, installed map[string]string) (*fakePython, MiseSync, []MiseTool) {
	t.Helper()
	tools := []MiseTool{{"python", "3.12.6"}}
	f := &fakePython{fakeMise: fakeMise{installed: installed, pending: tools}, dists: map[string]string{}}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, PythonPackages: []MiseTool{{"pyyaml", "6.0.3"}}}
	f.pkgFile = s.PythonPackagesPath()
	return f, s, tools
}

// A fresh machine: the packages file exists before `mise install`, so mise
// installs the packages with the Python and no pip run is needed.
func TestSync_WritesThePackagesFileBeforeMiseInstallsPython(t *testing.T) {
	f, s, tools := newFakePython(t, map[string]string{})
	plan, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.MissingPackages) != 1 || f.dists["pyyaml"] != "6.0.3" || len(f.pips) != 0 {
		t.Fatalf("want pyyaml installed by mise and no pip run, got plan %+v, dists %v, pips %v", plan, f.dists, f.pips)
	}
	cfg, err := os.ReadFile(s.ConfigPath())
	if err != nil || !strings.Contains(string(cfg), "python.default_packages_file") {
		t.Fatalf("config must name the packages file: %q, %v", cfg, err)
	}
}

// A Python installed before the package was declared: mise will not install
// into it again, so the sync installs the package with pip, once.
func TestSync_PipInstallsIntoAPythonAlreadyAtItsPin(t *testing.T) {
	f, s, tools := newFakePython(t, map[string]string{"python": "3.12.6"})
	if _, err := s.Apply(tools); err != nil {
		t.Fatal(err)
	}
	if len(f.pips) != 1 || !strings.Contains(strings.Join(f.pips[0], " "), "pyyaml==6.0.3") {
		t.Fatalf("want one pip install of pyyaml==6.0.3, got %v", f.pips)
	}
	again, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if again.ConfigChanged || len(again.MissingPackages) != 0 || len(f.pips) != 1 || f.installs != 1 {
		t.Errorf("second run: want nothing to do, got %+v after %d pips and %d installs", again, len(f.pips), f.installs)
	}
}

func TestSync_ReportsAFailedPipInstall(t *testing.T) {
	f, s, tools := newFakePython(t, map[string]string{"python": "3.12.6"})
	f.pipFails = true
	if _, err := s.Apply(tools); err == nil || !strings.Contains(err.Error(), "pip install") || !strings.Contains(err.Error(), "no network") {
		t.Fatalf("want the pip error with its output, got %v", err)
	}
}

// pip exiting 0 is not the proof: the package must import at its pin after.
func TestSync_FailsWhenThePackageStillDoesNotImport(t *testing.T) {
	f, s, tools := newFakePython(t, map[string]string{"python": "3.12.6"})
	f.pipNoop = true
	if _, err := s.Apply(tools); err == nil || !strings.Contains(err.Error(), "not importable") || !strings.Contains(err.Error(), "pyyaml") {
		t.Fatalf("want a post-condition error naming pyyaml, got %v", err)
	}
}

// Only bats needs Git Bash, so only bats gets it: a sync that prepended Git's
// bin to every child changed what mise and every other probe resolved (#2346).
func TestGitBashScope_OnlyTheBatsInstallAndProbe(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"mise", []string{"install", "--force", "npm:bats"}, true},
		{`C:\Users\m\AppData\Local\mise\installs\npm-bats\1.13.0\bats.cmd`, []string{"--version"}, true},
		{"BATS.CMD", []string{"--version"}, true},
		{"mise", []string{"install"}, false},
		{"mise", []string{"install", "node"}, false},
		{"mise", []string{"which", "bats.cmd"}, false},
		{"mise", []string{"install", "--force", "npm:prettier"}, false},
		{`C:\tools\jq.exe`, []string{"--version"}, false},
	} {
		if got := gitBashScope(tc.name, tc.args...); got != tc.want {
			t.Errorf("gitBashScope(%q, %q) = %v, want %v", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestHomeRunners_WindowsGivesGitBashToBatsAlone(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash is the Windows bats runtime")
	}
	root := t.TempDir()
	gitCmd := filepath.Join(root, "cmd", "git.cmd")
	bash := filepath.Join(root, "bin", "bash.exe")
	for _, dir := range []string{filepath.Dir(gitCmd), filepath.Dir(bash)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	execPath := filepath.Join(root, "mingw64", "libexec", "git-core")
	if err := os.WriteFile(gitCmd, []byte("@echo "+execPath+"\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bash, []byte("test fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A stand-in for mise's bats.cmd: it reports the first bash it can see.
	batsCmd := filepath.Join(t.TempDir(), "bats.cmd")
	if err := os.WriteFile(batsCmd, []byte("@where bash\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(gitCmd)+string(os.PathListSeparator)+os.Getenv("PATH"))
	run, _ := HomeRunners(t.TempDir())
	firstBash := func(name string) string {
		out, _ := run(name, "bash")
		return strings.Split(strings.TrimSpace(string(out)), "\r\n")[0]
	}
	isFixture := func(p string) bool {
		found, err := os.Stat(p)
		if err != nil {
			return false
		}
		want, err := os.Stat(bash)
		return err == nil && os.SameFile(found, want)
	}
	if got := firstBash(batsCmd); !isFixture(got) {
		t.Errorf("first bash bats sees = %q, want %q", got, bash)
	}
	if got := firstBash("where"); isFixture(got) {
		t.Errorf("a non-bats child sees Git Bash first (%q): the PATH prepend leaked", got)
	}
}

func TestSync_APackageBelowItsPinIsMissing(t *testing.T) {
	f, s, tools := newFakePython(t, map[string]string{"python": "3.12.6"})
	f.dists["pyyaml"] = "5.4.1"
	plan, err := s.Plan(tools)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.MissingPackages) != 1 || plan.MissingPackages[0] != "pyyaml" {
		t.Fatalf("want pyyaml missing below its pin, got %+v", plan)
	}
}

// kubectl rejects `--version` as an unknown flag. Probed with it, an install
// at its pin reads as missing and every sync fails naming it; probed with its
// row in versionArgs, it runs at its pin.
func TestSync_ProbesAToolWithoutVersionFlagThroughItsArgs(t *testing.T) {
	tools := []MiseTool{{"kubectl", "1.37.1"}}
	run := func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "mise" && len(args) == 2 && args[0] == "which":
			return []byte("/mise/installs/kubectl\n"), nil
		case name == "/mise/installs/kubectl" && strings.Join(args, " ") == "version --client":
			return []byte("Client Version: v1.37.1\nKustomize Version: v5.8.1\n"), nil
		case name == "/mise/installs/kubectl":
			return []byte("error: unknown flag: " + strings.Join(args, " ") + "\n"), errors.New("exit status 1")
		}
		return nil, errors.New("unexpected call: " + name + " " + strings.Join(args, " "))
	}
	plan, err := MiseSync{ConfigDir: t.TempDir(), Run: run}.Plan(tools)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Missing) != 0 {
		t.Errorf("kubectl at its pin reads as missing: %v", plan.Missing)
	}
}

func TestParseMisePins_ReadsTheLatestMarkerAsAFloorItTracks(t *testing.T) {
	body := "# mise: cli\nAGE_VERSION=1.3.1\n# mise: latest\nCLAUDE_VERSION=2.1.296\n# mise: latest\nAGY_VERSION=1.3.2\n"
	p, err := ParseMisePins([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := names(p.Tools); got != "age, agy, claude" {
		t.Errorf("tools: got %q", got)
	}
	if got := strings.Join(p.Latest, ", "); got != "agy, claude" {
		t.Errorf("latest: got %q", got)
	}
	for _, tool := range p.Tools {
		if tool.Name == "claude" && tool.Version != "2.1.296" {
			t.Errorf("claude's pin is its floor, got %q", tool.Version)
		}
	}
}

func TestParseMisePins_NearMissNamesTheLatestMarker(t *testing.T) {
	_, err := ParseMisePins([]byte("# mise: cli\nAGE_VERSION=1.3.1\n# mise: latests\nCLAUDE_VERSION=2.1.296\n"))
	if err == nil || !strings.Contains(err.Error(), MiseLatestMarker) {
		t.Fatalf("want an error naming %q, got %v", MiseLatestMarker, err)
	}
}

func TestRenderMiseConfig_RendersATrackedToolAsLatest(t *testing.T) {
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}, {"claude", "2.1.296"}}, "", "claude"))
	if !strings.Contains(out, "age = \"1.3.1\"\nclaude = \"latest\"\n") {
		t.Errorf("want age pinned and claude at latest:\n%s", out)
	}
}

// newLatestSync is a sync tracking claude at latest, with claude's newest
// release at newest.
func newLatestSync(t *testing.T, installed map[string]string, newest string) (*fakeMise, MiseSync, []MiseTool) {
	t.Helper()
	cfgDir := t.TempDir()
	tools := []MiseTool{{"age", "1.3.1"}, {"claude", "2.1.296"}}
	f := &fakeMise{installed: installed, pending: tools, newest: map[string]string{"claude": newest},
		cfg: filepath.Join(cfgDir, "conf.d", "dotfiles.toml")}
	return f, MiseSync{ConfigDir: cfgDir, Run: f.run, Latest: []string{"claude"}, CheckUpgrades: true}, tools
}

func TestSync_UpgradesATrackedToolAndIsIdempotent(t *testing.T) {
	f, s, tools := newLatestSync(t, map[string]string{}, "2.1.300")
	if _, err := s.Apply(tools); err != nil {
		t.Fatal(err)
	}
	if f.installed["claude"] != "2.1.300" || len(f.upgrades) != 1 || strings.Join(f.upgrades[0], " ") != "claude" {
		t.Fatalf("want claude upgraded to 2.1.300 by one `mise upgrade claude`, got %s after %v", f.installed["claude"], f.upgrades)
	}
	again, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if again.Pending() || len(f.upgrades) != 1 {
		t.Errorf("second run: want nothing to do, got %+v after %d upgrades", again, len(f.upgrades))
	}
}

// A pin that turns into latest is upgraded by the run that writes the config,
// not by the next one: mise install keeps the installed version.
func TestSync_UpgradesInTheRunThatTurnsAPinIntoLatest(t *testing.T) {
	f, s, tools := newLatestSync(t, map[string]string{"age": "1.3.1", "claude": "2.1.296"}, "2.1.300")
	if err := os.MkdirAll(filepath.Dir(f.cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.cfg, RenderMiseConfig(tools, ""), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if f.installed["claude"] != "2.1.300" || strings.Join(p.Outdated, ",") != "claude" {
		t.Errorf("want claude upgraded and reported in this run, got %s and %+v", f.installed["claude"], p)
	}
}

// #2526: mise skips a release younger than minimum_release_age (24h by
// default), and near that cutoff `mise outdated` and `mise upgrade` can
// disagree on which release is eligible. An upgrade that ran and left the tool
// where it was is upstream lag, not a fault of this machine: the sync reports
// it as behind and succeeds, and the next run tries again.
func TestSync_ALatestToolStillBehindAfterItsUpgradeIsReportedNotFailed(t *testing.T) {
	f, s, tools := newLatestSync(t, map[string]string{}, "2.1.300")
	run := s.Run
	s.Run = func(name string, args ...string) ([]byte, error) {
		if name == "mise" && len(args) > 0 && args[0] == "upgrade" {
			f.upgrades = append(f.upgrades, args[1:])
			return nil, nil // reports success and changes nothing
		}
		return run(name, args...)
	}
	f.installed["claude"] = "2.1.299"
	f.installed["age"] = "1.3.1"
	p, err := s.Apply(tools)
	if err != nil {
		t.Fatalf("an upgrade upstream has not caught up with must not fail the sync: %v", err)
	}
	if strings.Join(p.Behind, ",") != "claude" || len(f.upgrades) != 1 {
		t.Errorf("want claude reported behind after one upgrade, got %+v after %v", p, f.upgrades)
	}
	if again, err := s.Apply(tools); err != nil || len(f.upgrades) != 2 || strings.Join(again.Behind, ",") != "claude" {
		t.Errorf("the next run must try the upgrade again: err %v, upgrades %v, plan %+v", err, f.upgrades, again)
	}
}

// What the upgrade must never do is leave a tool older than it found it: that
// is a fault here, whatever upstream says.
func TestSync_FailsWhenTheUpgradeRegressesALatestTool(t *testing.T) {
	f, s, tools := newLatestSync(t, map[string]string{"age": "1.3.1", "claude": "2.1.299"}, "2.1.300")
	if err := os.MkdirAll(filepath.Dir(f.cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.cfg, s.render(tools), 0o644); err != nil {
		t.Fatal(err)
	}
	run := s.Run
	s.Run = func(name string, args ...string) ([]byte, error) {
		if name == "mise" && len(args) > 0 && args[0] == "upgrade" {
			f.installed["claude"] = "2.1.298"
			return nil, nil
		}
		return run(name, args...)
	}
	_, err := s.Apply(tools)
	if err == nil || !strings.Contains(err.Error(), "claude") || !strings.Contains(err.Error(), "2.1.298") {
		t.Fatalf("want an error naming claude and the older version, got %v", err)
	}
}

// Doctor plans without CheckUpgrades: a newer release upstream is not drift,
// and doctor must not reach the network to learn of it.
func TestSync_PlansNoUpgradeWithoutCheckUpgrades(t *testing.T) {
	f, s, tools := newLatestSync(t, map[string]string{"age": "1.3.1", "claude": "2.1.296"}, "2.1.300")
	s.CheckUpgrades = false
	if _, err := s.Apply(tools); err != nil {
		t.Fatal(err)
	}
	if f.outdates != 0 || len(f.upgrades) != 0 {
		t.Errorf("want no `mise outdated` and no upgrade, got %d and %v", f.outdates, f.upgrades)
	}
}

// #2375: converge removes an install no pin asks for, such as a PowerShell
// left by hand whose shim then dangles. A superseded version of a pinned tool
// is not an orphan: mise schedules it for removal itself, and pruning it here
// would race that schedule. The plan names each orphan and removes nothing.
func TestSync_PrunesAnInstallNoPinAsksForAndIsIdempotent(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}, {"jq", "1.8.2"}}
	f := &fakeMise{installed: map[string]string{"age": "1.3.1", "jq": "1.8.2"}, pending: tools,
		prunable: map[string][]string{"powershell": {"7.5.11"}, "jq": {"1.7.1"}}}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	plan, err := s.Plan(tools)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"powershell@7.5.11"}; !reflect.DeepEqual(plan.Orphans, want) {
		t.Fatalf("plan orphans = %v, want %v", plan.Orphans, want)
	}
	if len(f.prunes) != 0 {
		t.Fatalf("a plan pruned: %v", f.prunes)
	}

	if _, err := s.Apply(tools); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"powershell"}}; !reflect.DeepEqual(f.prunes, want) {
		t.Fatalf("prunes = %v, want only the orphan %v", f.prunes, want)
	}

	again, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if again.Pending() || len(f.prunes) != 1 {
		t.Errorf("second run: want nothing to do and no second prune, got %+v after %v", again, f.prunes)
	}
}

// The config write itself makes orphans: an install the deployed config still
// asked for is not prunable before the write, and is once the write drops it.
// Listed only before the write, it was left installed and failed the check
// after the prune (#2677, measured with http:zsh-autosuggestions).
func TestSync_PrunesWhatTheConfigWriteLeftUnasked(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	cfgDir := t.TempDir()
	cfg := filepath.Join(cfgDir, "conf.d", "dotfiles.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[tools]\nage = \"1.3.1\"\n\"http:zsh-autosuggestions\" = { version = \"0.7.1\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &fakeMise{installed: map[string]string{"age": "1.3.1"}, pending: tools, cfg: cfg,
		keptByConfig: map[string][]string{"http:zsh-autosuggestions": {"0.7.1"}}}
	s := MiseSync{ConfigDir: cfgDir, Run: f.run, Prune: true}

	if _, err := s.Apply(tools); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := [][]string{{"http:zsh-autosuggestions"}}; !reflect.DeepEqual(f.prunes, want) {
		t.Errorf("prunes = %v, want the install the write left unasked %v", f.prunes, want)
	}
}

// A run whose first listing failed prunes nothing ("nothing pruned"); a
// listing that answers after the config write must not prune behind that.
func TestSync_PrunesNothingAfterTheFirstListingFailed(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{}, pending: tools,
		prunable: map[string][]string{"powershell": {"7.5.11"}}, listFailsAt: 1}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	_, err := s.Apply(tools)
	if len(f.prunes) != 0 {
		t.Errorf("pruned after a failed listing: %v", f.prunes)
	}
	if err == nil {
		t.Error("want the run to fail while an orphan is left")
	}
}

// A mise that cannot list what to prune must not cost the tools a run: the
// install goes ahead, nothing is pruned, and the error names the listing.
// The plan still fails, so a dry run reports the same cause.
func TestSync_InstallsWhenTheOrphanListingFails(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{}, pending: tools, listFails: true}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	if _, err := s.Plan(tools); err == nil || !strings.Contains(err.Error(), "mise ls --prunable") {
		t.Fatalf("plan: want the listing's failure, got %v", err)
	}
	_, err := s.Apply(tools)
	if err == nil || !strings.Contains(err.Error(), "nothing pruned: mise ls --prunable") || !strings.Contains(err.Error(), "failed to read a tracked config") {
		t.Errorf("apply: want an error naming the listing and its output, got %v", err)
	}
	if f.installs != 1 || f.installed["age"] != "1.3.1" {
		t.Errorf("want the install to run despite the listing, got %d installs, %v", f.installs, f.installed)
	}
	if len(f.prunes) != 0 {
		t.Errorf("pruned without a listing: %v", f.prunes)
	}
}

// A listing that fails only after the prune must not be reported as nothing
// pruned: the prune ran, and what failed is its verification.
func TestSync_ReportsAPruneItCouldNotVerify(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{"age": "1.3.1"}, pending: tools,
		prunable: map[string][]string{"powershell": {"7.5.11"}}, listFailsFrom: 2}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	_, err := s.Apply(tools)
	if len(f.prunes) != 1 {
		t.Fatalf("want the listed orphan pruned once, got prunes %v", f.prunes)
	}
	if err == nil || strings.Contains(err.Error(), "nothing pruned") {
		t.Fatalf("want an error that does not deny the prune, got %v", err)
	}
	for _, want := range []string{"pruned powershell@7.5.11; cannot verify: mise ls --prunable", "failed to read a tracked config"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("want %q in the error, got %v", want, err)
		}
	}
}

// With no orphan before the sync and a listing that fails after it, nothing
// was pruned and the absence of orphans is what went unchecked.
func TestSync_ReportsAnOrphanCheckItCouldNotRun(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{}, pending: tools, listFailsFrom: 2}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	_, err := s.Apply(tools)
	if f.installs != 1 || len(f.prunes) != 0 {
		t.Fatalf("want one install and no prune, got %d installs, prunes %v", f.installs, f.prunes)
	}
	if err == nil || !strings.Contains(err.Error(), "cannot verify that no orphan remains: mise ls --prunable") {
		t.Errorf("want the unchecked orphan listing named, got %v", err)
	}
}

// Pruning is converge's: doctor and `dotf tools sync` plan the same sync and
// must never ask mise what to delete, let alone delete it.
func TestSync_DoesNotPruneUnlessAsked(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}}
	f := &fakeMise{installed: map[string]string{"age": "1.3.1"}, pending: tools, prunable: map[string][]string{"powershell": {"7.5.11"}}}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run}
	p, err := s.Apply(tools)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Orphans) != 0 || f.listPrune != 0 || len(f.prunes) != 0 {
		t.Errorf("want no prune query and no prune, got orphans %v, %d queries, prunes %v", p.Orphans, f.listPrune, f.prunes)
	}
}

// #2744: before the write, `mise ls --prunable` cannot see a tool the new
// config drops, because the deployed config still asks for it. The plan reads
// both configs' [tools] tables instead, so a dry run can name the prune.
func TestSync_PlanNamesTheToolsAConfigWriteDrops(t *testing.T) {
	kept := []MiseTool{{"age", "1.3.1"}, {"jq", "1.8.2"}}
	write := func(t *testing.T, body []byte) MiseSync {
		t.Helper()
		s := MiseSync{ConfigDir: t.TempDir(), Run: (&fakeMise{installed: map[string]string{}}).run}
		if body != nil {
			if err := os.MkdirAll(filepath.Dir(s.ConfigPath()), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.ConfigPath(), body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return s
	}

	deployed := RenderMiseConfig(append([]MiseTool{{"gh", "2.80.0"}}, kept...), "")
	p, err := write(t, deployed).Plan(kept)
	if err != nil || !p.ConfigChanged || strings.Join(p.Dropped, ",") != "gh" {
		t.Errorf("a config that loses gh: want Dropped [gh], got %+v, %v", p, err)
	}

	p, err = write(t, nil).Plan(kept)
	if err != nil || len(p.Dropped) != 0 {
		t.Errorf("no deployed config: nothing is dropped, got %+v, %v", p, err)
	}

}

// #2760: real mise refuses every command over a deployed config that does not
// parse, so a plan that asks mise anything cannot get past the file the sync
// owns and would rewrite. The plan reports the rewrite as its one change and
// asks mise nothing; Apply writes the rendered config, installs against it,
// and the next plan has nothing to do.
func TestSync_RepairsADeployedConfigThatDoesNotParse(t *testing.T) {
	tools := []MiseTool{{"age", "1.3.1"}, {"jq", "1.8.2"}}
	s := MiseSync{ConfigDir: t.TempDir()}
	f := &fakeMise{installed: map[string]string{}, pending: tools, cfg: s.ConfigPath(), prunable: map[string][]string{}}
	s.Run, s.Prune = f.run, true
	if err := os.MkdirAll(filepath.Dir(s.ConfigPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.ConfigPath(), []byte("[tools\ngh = "), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := s.Plan(tools)
	if err != nil || !p.ConfigChanged || p.UnparsableConfig != s.ConfigPath() {
		t.Fatalf("plan: want the rewrite as its change, got %+v, %v", p, err)
	}
	if f.calls != 0 {
		t.Errorf("plan: mise refuses every command over that file, want none asked, got %d", f.calls)
	}

	if _, err := s.Apply(tools); err != nil {
		t.Fatalf("apply must repair the file it owns: %v", err)
	}
	if body, _ := os.ReadFile(s.ConfigPath()); string(body) != string(s.render(tools)) {
		t.Errorf("apply: want the rendered config, got %q", body)
	}
	if f.installs == 0 {
		t.Errorf("apply: want mise install against the repaired config")
	}
	if again, err := s.Plan(tools); err != nil || again.Pending() || again.UnparsableConfig != "" {
		t.Errorf("the next plan must have nothing to do, got %+v, %v", again, err)
	}
}

// A Windows sync whose bats never reaches its pin because Git Bash is missing
// said only "not running at their pin: bats" (#2346). The error names the
// missing runtime and why it was not found.
func TestBatsRuntimeNote_NamesAMissingGitBash(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "mingw64", "libexec", "git-core")
	withBash := t.TempDir()
	if err := os.MkdirAll(filepath.Join(withBash, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withBash, "bin", "bash.exe"), []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	found := func() ([]byte, error) {
		return []byte(filepath.Join(withBash, "mingw64", "libexec", "git-core") + "\n"), nil
	}
	notUnder := func() ([]byte, error) { return []byte(execPath + "\n"), nil }
	// The standard roots are the host's: on a Windows runner Git is really
	// there, and the cases below mean "nowhere".
	old := gitStandardRoots
	gitStandardRoots = func() []string { return nil }
	t.Cleanup(func() { gitStandardRoots = old })
	noGit := func() ([]byte, error) { return nil, errors.New(`exec: "git": executable file not found in %PATH%`) }
	for _, tc := range []struct {
		name     string
		goos     string
		missing  []string
		execPath func() ([]byte, error)
		want     string // substring; "" means no note at all
	}{
		{"git cannot be run", "windows", []string{"bats"}, noGit, "Git Bash not found: `git --exec-path` failed"},
		{"no bash under Git's root", "windows", []string{"jq", "bats"}, notUnder, "Git Bash not found under " + root},
		{"Git Bash is there", "windows", []string{"bats"}, found, ""},
		{"bats is not the missing tool", "windows", []string{"jq"}, noGit, ""},
		{"not Windows", "linux", []string{"bats"}, noGit, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := batsRuntimeNote(tc.goos, tc.missing, tc.execPath)
			if tc.want == "" && got != "" || !strings.Contains(got, tc.want) {
				t.Errorf("batsRuntimeNote = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

// A from-zero converge installs git and syncs tools in one process, whose PATH
// was captured before git existed: `git --exec-path` fails, yet Git Bash is at
// its standard install root. The lookup finds it there and, when it is nowhere,
// names every root it searched (#3328).
func TestFindGitBash_FallsBackToTheStandardInstallRoots(t *testing.T) {
	noGit := func() ([]byte, error) { return nil, errors.New(`exec: "git": executable file not found in %PATH%`) }
	old := gitStandardRoots
	t.Cleanup(func() { gitStandardRoots = old })

	empty := t.TempDir()
	installed := t.TempDir()
	if err := os.MkdirAll(filepath.Join(installed, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(installed, "bin", "bash.exe")
	if err := os.WriteFile(want, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}

	gitStandardRoots = func() []string { return []string{empty, installed} }
	if got, reason := findGitBash(noGit); reason != "" || got != want {
		t.Errorf("findGitBash = (%q, %q), want (%q, \"\")", got, reason, want)
	}

	gitStandardRoots = func() []string { return []string{empty} }
	_, reason := findGitBash(noGit)
	for _, part := range []string{"`git --exec-path` failed", "also looked under " + empty} {
		if !strings.Contains(reason, part) {
			t.Errorf("reason %q lacks %q", reason, part)
		}
	}

	// git on the PATH still wins over a root.
	gitRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gitRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	onPath := filepath.Join(gitRoot, "bin", "bash.exe")
	if err := os.WriteFile(onPath, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitStandardRoots = func() []string { return []string{installed} }
	viaGit := func() ([]byte, error) {
		return []byte(filepath.Join(gitRoot, "mingw64", "libexec", "git-core") + "\n"), nil
	}
	if got, reason := findGitBash(viaGit); reason != "" || got != onPath {
		t.Errorf("findGitBash via git = (%q, %q), want (%q, \"\")", got, reason, onPath)
	}
}
