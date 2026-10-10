package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// fakeMise answers `mise which <tool>` for the tools in installed, and
// `<path> --version` with the version installed records; `mise install` marks
// every pending tool installed at its pin.
type fakeMise struct {
	installed map[string]string // tool -> version
	pending   []MiseTool
	installs  int
}

func (f *fakeMise) run(name string, args ...string) ([]byte, error) {
	switch {
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
	cases := []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"MISE_DATA_DIR": "/d/mise", "XDG_DATA_HOME": "/xdg"}, filepath.Join("/d/mise", "shims")},
		{map[string]string{"XDG_DATA_HOME": "/xdg"}, filepath.Join("/xdg", "mise", "shims")},
		{map[string]string{}, filepath.Join(home, ".local", "share", "mise", "shims")},
	}
	for _, tc := range cases {
		if got := MiseShimsDir(home, func(k string) string { return tc.env[k] }); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.env, got, tc.want)
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
		if name == "mise" && args[0] == "which" {
			return []byte("/bin/age\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: merged, Stdout: stdout}

	p, err := s.Plan(tools)
	if err != nil || len(p.Missing) != 0 {
		t.Fatalf("want age at its pin through the stdout runner, got %+v, %v", p, err)
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
