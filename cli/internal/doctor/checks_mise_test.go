package doctor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

const miseVersions = "# mise: cli\nJQ_VERSION=1.7.1\n# mise: cli\nAGE_VERSION=1.2.1\n"

// miseFixture is a machine where mise resolves jq and age; ageVersion is what
// the installed age reports, and synced writes the config the sync renders.
func miseFixture(t *testing.T, goos, ageVersion string, synced bool) (*System, *Config) {
	t.Helper()
	home := t.TempDir()
	versions := filepath.Join(home, "versions.conf")
	writeFile(t, versions, miseVersions)
	if synced {
		pins, err := tools.ParseMiseTools([]byte(miseVersions))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(home, ".config", "mise", "conf.d", "dotfiles.toml"), string(tools.RenderMiseConfig(pins, "")))
	}
	s := newSys(map[string]string{"HOME": home}, []string{"mise"}, map[string]string{
		"mise which jq":    "/m/jq\n",
		"/m/jq --version":  "jq-1.7.1",
		"mise which age":   "/m/age\n",
		"/m/age --version": "v" + ageVersion,
		// What mise provides for the pins: a pin can bring a companion.
		"mise bin-paths --bin-names age@1.2.1 jq@1.7.1": "age\nage-keygen\njq\n",
		"mise install":              "",
		"mise ls --prunable --json": "{}",
	})
	s.GOOS = goos
	return s, &Config{VersionsPath: versions}
}

func TestCheckMiseTools_AtTheirPinPassesOnEveryOS(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		s, cfg := miseFixture(t, goos, "1.2.1", true)
		var b bytes.Buffer
		rep := capture(&b)
		checkMiseTools(s, cfg, rep, false)
		if rep.Failures() != 0 || rep.Warnings() != 0 || !strings.Contains(b.String(), "2 CLI(s) at their pin") {
			t.Errorf("%s: want a clean pass\n%s", goos, b.String())
		}
	}
}

func TestCheckMiseTools_ACLIBelowItsPinFailsNamingTheSync(t *testing.T) {
	s, cfg := miseFixture(t, "darwin", "1.1.0", true)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 1 || !strings.Contains(b.String(), "age") || !strings.Contains(b.String(), "dotf tools sync") {
		t.Errorf("want one FAIL naming age and the sync\n%s", b.String())
	}
}

func TestCheckMiseTools_AStaleRenderedConfigWarns(t *testing.T) {
	s, cfg := miseFixture(t, "linux", "1.2.1", false)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "dotf tools sync") {
		t.Errorf("want one WARN naming the sync\n%s", b.String())
	}
}

// No mise: the pins are not managed, which the core-tools check does not see
// when a CLI is on PATH from elsewhere. Advisory, until mise ships in the
// catalog (PLAT-001c T1b).
func TestCheckMiseTools_NoMiseWarns(t *testing.T) {
	s, cfg := miseFixture(t, "linux", "1.2.1", true)
	s.LookPath = newSys(nil, nil, nil).LookPath
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "mise not on PATH") {
		t.Errorf("want one WARN\n%s", b.String())
	}
}

// shadowFixture is miseFixture at the pin (mise provides age, age-keygen and
// jq); local is what ~/.local/bin holds, a regular
// file per name. The deleted setup installers put copies there (#2013 W2), and
// a process without mise activated (a GUI app, launchd, cron) finds them first.
func shadowFixture(t *testing.T, local ...string) (*System, *Config, string) {
	t.Helper()
	s, cfg := miseFixture(t, "darwin", "1.2.1", true)
	home := s.Getenv("HOME")
	for _, n := range local {
		writeFile(t, filepath.Join(home, ".local", "bin", n), "#!/bin/sh\n")
	}
	return s, cfg, home
}

func TestCheckMiseTools_ACopyInLocalBinThatShadowsMiseWarnsNamingIt(t *testing.T) {
	s, cfg, home := shadowFixture(t, "age-keygen", "jq", "dotf")
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	out := b.String()
	if rep.Failures() != 0 || rep.Warnings() != 1 {
		t.Fatalf("want one WARN\n%s", out)
	}
	// A companion binary counts: mise names it, not the pin.
	for _, want := range []string{filepath.Join(home, ".local", "bin", "age-keygen"), filepath.Join(home, ".local", "bin", "jq"), "dotf doctor --fix"} {
		if !strings.Contains(out, want) {
			t.Errorf("WARN must name %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "dotf\n") || strings.Contains(out, filepath.Join("bin", "dotf")) {
		t.Errorf("a file mise does not provide is not a leftover\n%s", out)
	}
}

// --fix replaces a copy with a link to mise's shim rather than deleting it: a
// consumer whose PATH lists ~/.local/bin explicitly (a systemd unit, a plist)
// keeps resolving the tool, now at its pin, instead of finding nothing.
func TestCheckMiseTools_FixReplacesACopyWithALinkToMiseShim(t *testing.T) {
	s, cfg, home := shadowFixture(t, "jq", "dotf")
	bin := filepath.Join(home, ".local", "bin")
	shim := filepath.Join(home, ".local", "share", "mise", "shims", "jq")
	writeFile(t, shim, "#!/bin/sh\n")
	if err := os.Symlink("/m/age", filepath.Join(bin, "age")); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Errorf("a repaired copy is not a warning\n%s", b.String())
	}
	if got, err := os.Readlink(filepath.Join(bin, "jq")); err != nil || got != shim {
		t.Errorf("~/.local/bin/jq must now link to mise's shim %q, got %q (%v)", shim, got, err)
	}
	if got, _ := os.Readlink(filepath.Join(bin, "age")); got != filepath.FromSlash("/m/age") {
		t.Errorf("a symlink that was already there must be left alone, got %q", got)
	}
	if fi, err := os.Lstat(filepath.Join(bin, "dotf")); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("a file mise does not provide must be left alone: %v", err)
	}
	// Idempotent: the link is a symlink, so a second run has nothing to repair.
	b.Reset()
	rep = capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Warnings() != 0 || strings.Contains(b.String(), "replaced") {
		t.Errorf("second run must be clean\n%s", b.String())
	}
}

// A link to a shim that does not exist would break the consumer the copy still
// serves, so the copy stays and the WARN names the reshim.
func TestCheckMiseTools_FixKeepsTheCopyWhenMiseHasNoShimForIt(t *testing.T) {
	s, cfg, home := shadowFixture(t, "jq")
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if fi, err := os.Lstat(filepath.Join(home, ".local", "bin", "jq")); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("the copy must stay when there is no shim to link to: %v", err)
	}
	if rep.Warnings() != 1 || !strings.Contains(b.String(), "mise reshim") {
		t.Errorf("want one WARN naming mise reshim\n%s", b.String())
	}
}

// A pin below its pin is a FAIL already; the leftover question is only asked
// once mise's own copy is proven, which is what makes removing one safe.
func TestCheckMiseTools_LeftoversAreNotTouchedWhileAPinIsMissing(t *testing.T) {
	s, cfg, home := shadowFixture(t, "jq")
	// The shim exists, so only the guard on the pins can stop the link.
	writeFile(t, filepath.Join(home, ".local", "share", "mise", "shims", "jq"), "#!/bin/sh\n")
	s.CommandOutputDir = newSys(nil, nil, map[string]string{"mise which jq": "/m/jq\n", "/m/jq --version": "jq-1.7.1", "mise which age": "/m/age\n", "/m/age --version": "v1.0.0"}).CommandOutputDir
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if fi, err := os.Lstat(filepath.Join(home, ".local", "bin", "jq")); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("nothing may be replaced before every pin runs through mise: %v\n%s", err, b.String())
	}
}

func TestCheckMiseTools_BinPathsFailingWarnsNamingTheCommand(t *testing.T) {
	s, cfg, _ := shadowFixture(t, "jq")
	prev := s.CommandStdoutDir
	s.CommandStdoutDir = func(dir, name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "bin-paths" {
			return "", errors.New("exit status 1")
		}
		return prev(dir, name, args...)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "mise bin-paths") {
		t.Errorf("want one WARN naming mise bin-paths\n%s", b.String())
	}
}

// mise names executables; an entry that is a path instead would, joined onto
// ~/.local/bin, reach a file outside it. It is reported and never touched.
func TestCheckMiseTools_AnEntryThatIsAPathIsNeverTouched(t *testing.T) {
	s, cfg, home := shadowFixture(t)
	outside := filepath.Join(home, "notes")
	writeFile(t, outside, "mine\n")
	// Where the same path lands under the shims dir, so only the name check
	// stands between the fix and the rename.
	writeFile(t, filepath.Join(home, ".local", "share", "mise", "shims", "..", "..", "notes"), "#!/bin/sh\n")
	prev := s.CommandStdoutDir
	s.CommandStdoutDir = func(dir, name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "bin-paths" {
			return "../../notes\n..\n", nil
		}
		return prev(dir, name, args...)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if fi, err := os.Lstat(outside); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("a file outside ~/.local/bin was touched: %v\n%s", err, b.String())
	}
	if rep.Warnings() != 1 || !strings.Contains(b.String(), "not file names") || strings.Contains(b.String(), "replaced") {
		t.Errorf("want one WARN naming the odd entries and no repair\n%s", b.String())
	}
}

// Under fix the drift is cleared by the sync itself: the config is written and
// mise install runs, and the section ends clean.
func TestCheckMiseTools_FixRunsTheSync(t *testing.T) {
	s, cfg := miseFixture(t, "darwin", "1.2.1", false)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 0 || !strings.Contains(b.String(), "ran the mise sync") {
		t.Errorf("want the sync run and a clean section\n%s", b.String())
	}
	if _, err := os.Stat(filepath.Join(s.home(), ".config", "mise", "conf.d", "dotfiles.toml")); err != nil {
		t.Errorf("the fix wrote no config: %v", err)
	}
}

func TestCheckMiseTools_APythonPackageMissingFailsNamingTheSync(t *testing.T) {
	home := t.TempDir()
	versions := filepath.Join(home, "versions.conf")
	body := "# mise: cli\nPYTHON_VERSION=3.12.6\n# mise: python-package\nPYYAML_VERSION=6.0.3\n"
	writeFile(t, versions, body)
	s := newSys(map[string]string{"HOME": home}, []string{"mise"}, map[string]string{
		"mise which python":   "/m/python\n",
		"/m/python --version": "Python 3.12.6",
	})
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, &Config{VersionsPath: versions}, rep, false)
	if rep.Failures() != 1 || !strings.Contains(b.String(), "python packages missing from mise's python at their pin: pyyaml") {
		t.Errorf("want one FAIL naming pyyaml\n%s", b.String())
	}
}

// shellPathFixture is miseFixture at the pin, on darwin, with zsh and bash on
// PATH. paths maps "zsh -c" and "bash -lc" to the PATH that shell starts with
// from an empty environment; env records what each probe was given.
func shellPathFixture(t *testing.T, paths map[string]string, env *[][]string) (*System, *Config, string) {
	t.Helper()
	return shellPathFixtureAt(t, "1.2.1", paths, env)
}

func shellPathFixtureAt(t *testing.T, ageVersion string, paths map[string]string, env *[][]string) (*System, *Config, string) {
	t.Helper()
	s, cfg := miseFixture(t, "darwin", ageVersion, true)
	home := s.Getenv("HOME")
	s.LookPath = newSys(nil, []string{"mise", "zsh", "bash"}, nil).LookPath
	s.CommandOutputEnv = func(e []string, name string, args ...string) (string, error) {
		*env = append(*env, e)
		out, ok := paths[name+" "+args[0]]
		if !ok {
			return "", errors.New("unexpected probe " + name + " " + strings.Join(args, " "))
		}
		// The fixtures write PATH with ':'; the check splits with the host's
		// separator, so a Windows host (whose HOME holds a ':') reads it too.
		return strings.NewReplacer("HOME", home, ":", string(filepath.ListSeparator)).Replace(out), nil
	}
	return s, cfg, home
}

// A shell an agent or sshd runs reads no interactive rc, so `mise activate`
// never reaches it: only the shims dir on its startup PATH runs the pins
// (#2013 W2b). The probe starts each shell from an empty environment, so the
// answer is what the deployed files give, not what doctor's own shell has.
func TestCheckMiseTools_NonInteractiveShellsStartWithTheShimsFirst(t *testing.T) {
	var env [][]string
	s, cfg, home := shellPathFixture(t, map[string]string{
		"zsh -c":   "HOME/.local/share/mise/shims:HOME/.local/bin:/usr/bin:/bin",
		"bash -lc": "HOME/.local/share/mise/shims:HOME/.local/bin:/usr/bin:/bin\n",
	}, &env)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 0 || rep.Warnings() != 0 || !strings.Contains(b.String(), "zsh -c, bash -lc") {
		t.Errorf("want a clean pass naming both shells\n%s", b.String())
	}
	if len(env) != 2 {
		t.Fatalf("probes = %d, want 2", len(env))
	}
	for _, e := range env {
		var paths []string
		for _, kv := range e {
			if strings.HasPrefix(kv, "PATH=") {
				paths = append(paths, kv)
			}
		}
		if !contains(e, "HOME="+home) || len(paths) != 1 || paths[0] != "PATH=/usr/bin:/bin" {
			t.Errorf("probe env %v: want HOME and only the bare system PATH, never doctor's own", e)
		}
	}
}

func TestCheckMiseTools_ANonInteractiveShellWithoutTheShimsWarns(t *testing.T) {
	cases := []struct {
		name, zsh, want string
	}{
		{"no shims", "HOME/.local/bin:/usr/bin:/bin", "`zsh -c` starts without mise's shims"},
		{"local bin first", "HOME/.local/bin:HOME/.local/share/mise/shims:/usr/bin", "`zsh -c` puts " + filepath.Join("HOME", ".local", "bin")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var env [][]string
			s, cfg, home := shellPathFixture(t, map[string]string{
				"zsh -c":   tc.zsh,
				"bash -lc": "HOME/.local/share/mise/shims:/usr/bin",
			}, &env)
			var b bytes.Buffer
			rep := capture(&b)
			checkMiseTools(s, cfg, rep, false)
			want := strings.ReplaceAll(tc.want, "HOME", home)
			if rep.Warnings() != 1 || !strings.Contains(b.String(), want) || !strings.Contains(b.String(), "dotf deploy") {
				t.Errorf("want one WARN saying %q and naming dotf deploy\n%s", want, b.String())
			}
		})
	}
}

// The startup PATH is a fact about the deployed files, not about the pins: a
// pin that is not installed yet must not hide a shell that would never run it
// through mise anyway.
func TestCheckMiseTools_ShellProbeRunsWhenAPinIsMissing(t *testing.T) {
	var env [][]string
	s, cfg, _ := shellPathFixtureAt(t, "1.0.0", map[string]string{
		"zsh -c":   "/usr/bin:/bin",
		"bash -lc": "HOME/.local/share/mise/shims:/usr/bin",
	}, &env)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	if rep.Failures() != 1 || rep.Warnings() != 1 || !strings.Contains(b.String(), "`zsh -c` starts without") {
		t.Errorf("want the pin FAIL and the shell WARN both\n%s", b.String())
	}
}

// bash -lc reads the first of ~/.bash_profile, ~/.bash_login and ~/.profile,
// so where one of the first two exists the deployed .profile is never read
// and `dotf deploy` alone cannot fix it: the WARN says so.
func TestCheckMiseTools_ABashProfileThatHidesProfileIsNamed(t *testing.T) {
	var env [][]string
	s, cfg, home := shellPathFixture(t, map[string]string{
		"zsh -c":   "HOME/.local/share/mise/shims:/usr/bin",
		"bash -lc": "/usr/bin:/bin",
	}, &env)
	writeFile(t, filepath.Join(home, ".bash_login"), "")
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, false)
	want := filepath.Join(home, ".bash_login") + " exists, so bash -l never reads ~/.profile"
	if rep.Warnings() != 1 || !strings.Contains(b.String(), want) {
		t.Errorf("want one WARN saying %q\n%s", want, b.String())
	}
}

// Windows has no zsh -c or bash -lc to probe: what `pwsh -NoProfile` inherits
// is the user environment, a later row (env persist).
func TestCheckMiseTools_NoShellProbeOnWindows(t *testing.T) {
	var env [][]string
	s, cfg, _ := shellPathFixture(t, nil, &env)
	s.GOOS = "windows"
	var b bytes.Buffer
	checkMiseTools(s, cfg, capture(&b), false)
	if len(env) != 0 {
		t.Errorf("probed %d shells on windows\n%s", len(env), b.String())
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// #2375: an install no pin asks for leaves a shim that cannot run. Doctor
// names it and the converge that removes it; --fix removes nothing.
func TestCheckMiseTools_AnInstallNoPinAsksForWarnsNamingTheConverge(t *testing.T) {
	s, cfg := miseFixture(t, "darwin", "1.2.1", true)
	stdout := s.CommandStdoutDir
	s.CommandStdoutDir = func(dir, name string, args ...string) (string, error) {
		if name == "mise" && strings.Join(args, " ") == "ls --prunable --json" {
			return `{"powershell": [{"version": "7.5.11"}], "jq": [{"version": "1.6"}]}`, nil
		}
		return stdout(dir, name, args...)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Warnings() != 1 || !strings.Contains(b.String(), "powershell@7.5.11") || strings.Contains(b.String(), "jq@1.6") ||
		!strings.Contains(b.String(), "dotf converge --only tools") {
		t.Errorf("want one WARN naming only the orphan and the converge\n%s", b.String())
	}
}
