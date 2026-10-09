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
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}, {"golangci-lint", "2.12.2"}}))
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
	out := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}}))
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
