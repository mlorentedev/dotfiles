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
		writeFile(t, filepath.Join(home, ".config", "mise", "conf.d", "dotfiles.toml"), string(tools.RenderMiseConfig(pins)))
	}
	s := newSys(map[string]string{"HOME": home}, []string{"mise"}, map[string]string{
		"mise which jq":    "/m/jq\n",
		"/m/jq --version":  "jq-1.7.1",
		"mise which age":   "/m/age\n",
		"/m/age --version": "v" + ageVersion,
		// What mise provides for the pins: a pin can bring a companion.
		"mise bin-paths --bin-names age@1.2.1 jq@1.7.1": "age\nage-keygen\njq\n",
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

func TestCheckMiseTools_FixRemovesTheShadowingCopiesOnly(t *testing.T) {
	s, cfg, home := shadowFixture(t, "jq", "dotf")
	bin := filepath.Join(home, ".local", "bin")
	if err := os.Symlink("/m/age", filepath.Join(bin, "age")); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Failures() != 0 || rep.Warnings() != 0 {
		t.Errorf("a repaired leftover is not a warning\n%s", b.String())
	}
	if _, err := os.Lstat(filepath.Join(bin, "jq")); !os.IsNotExist(err) {
		t.Errorf("the shadowing jq was not removed: %v", err)
	}
	for _, keep := range []string{"dotf", "age"} {
		if _, err := os.Lstat(filepath.Join(bin, keep)); err != nil {
			t.Errorf("%s must survive the fix (not mise's, or a deliberate symlink): %v", keep, err)
		}
	}
	// Idempotent: a second run has nothing to repair.
	b.Reset()
	rep = capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if rep.Warnings() != 0 || strings.Contains(b.String(), "removed") {
		t.Errorf("second run must be clean\n%s", b.String())
	}
}

// A pin below its pin is a FAIL already; the leftover question is only asked
// once mise's own copy is proven, which is what makes removing one safe.
func TestCheckMiseTools_LeftoversAreNotTouchedWhileAPinIsMissing(t *testing.T) {
	s, cfg, home := shadowFixture(t, "jq")
	s.CommandOutputDir = newSys(nil, nil, map[string]string{"mise which jq": "/m/jq\n", "/m/jq --version": "jq-1.7.1", "mise which age": "/m/age\n", "/m/age --version": "v1.0.0"}).CommandOutputDir
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep, true)
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "jq")); err != nil {
		t.Errorf("nothing may be removed before every pin runs through mise: %v\n%s", err, b.String())
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
