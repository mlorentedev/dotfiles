package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

const pythonVersions = "# mise: cli\nPYTHON_VERSION=3.12.6\n# mise: python-package\nPYYAML_VERSION=6.0.3\n"

// pythonFixture is a machine whose versions.conf is body, with onPath and the
// command answers given.
func pythonFixture(t *testing.T, goos, body string, onPath []string, cmdOut map[string]string) (*System, *Config) {
	t.Helper()
	home := t.TempDir()
	versions := filepath.Join(home, "versions.conf")
	writeFile(t, versions, body)
	s := newSys(map[string]string{"HOME": home}, onPath, cmdOut)
	s.GOOS = goos
	return s, &Config{VersionsPath: versions}
}

func runCheckPython(s *System, cfg *Config) (*Report, string) {
	var b bytes.Buffer
	rep := capture(&b)
	checkPython(s, cfg, rep)
	return rep, b.String()
}

func TestCheckPython_AtTheFloorWithItsPackagesPasses(t *testing.T) {
	s, cfg := pythonFixture(t, "darwin", pythonVersions, []string{"python3"}, map[string]string{
		"python3 --version":                             "Python 3.12.6",
		"python3 -c " + tools.PyDistVersion + " pyyaml": "6.0.3\n",
	})
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 0 || !strings.Contains(out, "python3 3.12.6") || !strings.Contains(out, "1 python package(s) at their pin") {
		t.Errorf("want a clean pass\n%s", out)
	}
}

// Windows installs, mise's among them, carry python.exe and no python3.
func TestCheckPython_ProbesPythonOnWindows(t *testing.T) {
	s, cfg := pythonFixture(t, "windows", pythonVersions, []string{"python"}, map[string]string{
		"python --version": "Python 3.12.6",
		"python -c " + tools.PyDistVersion + " pyyaml": "6.0.3\n",
	})
	if rep, out := runCheckPython(s, cfg); rep.Failures() != 0 || !strings.Contains(out, "python 3.12.6") {
		t.Errorf("want a pass through python\n%s", out)
	}
}

func TestCheckPython_BelowTheFloorFailsNamingTheSync(t *testing.T) {
	s, cfg := pythonFixture(t, "darwin", pythonVersions, []string{"python3", "mise"}, map[string]string{
		"python3 --version": "Python 3.9.6",
	})
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 1 || !strings.Contains(out, "3.9.6") || !strings.Contains(out, "run: dotf tools sync") {
		t.Errorf("want one FAIL naming 3.9.6 and the sync\n%s", out)
	}
}

// mise already has a Python that clears the floor: a sync changes nothing, the
// shell is not using it, and the failure says that instead.
func TestCheckPython_BelowTheFloorWithMisePythonNamesThePath(t *testing.T) {
	s, cfg := pythonFixture(t, "darwin", pythonVersions, []string{"python3", "mise"}, map[string]string{
		"python3 --version":    "Python 3.9.6",
		"mise which python3":   "/m/python3\n",
		"/m/python3 --version": "Python 3.12.6",
	})
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 1 || !strings.Contains(out, "mise's python is 3.12.6 at /m/python3") || strings.Contains(out, "run: dotf tools sync") {
		t.Errorf("want one FAIL naming mise's python and the PATH, not the sync\n%s", out)
	}
}

func TestCheckPython_AbsentFails(t *testing.T) {
	s, cfg := pythonFixture(t, "linux", pythonVersions, nil, nil)
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 1 || !strings.Contains(out, "python3 not on PATH") || !strings.Contains(out, "install mise") {
		t.Errorf("want one FAIL naming the missing python3 and mise\n%s", out)
	}
}

// With no python pinned for mise, no sync installs one, so the remedy cannot
// be the sync.
func TestCheckPython_WithNoPythonPinTheRemedyIsNotTheSync(t *testing.T) {
	s, cfg := pythonFixture(t, "darwin", "# mise: cli\nAGE_VERSION=1.3.1\n", []string{"python3", "mise"}, map[string]string{
		"python3 --version": "Python 3.9.6",
	})
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 1 || !strings.Contains(out, "pins no PYTHON_VERSION") || strings.Contains(out, "run: dotf tools sync") {
		t.Errorf("want one FAIL saying nothing pins python\n%s", out)
	}
}

func TestCheckPython_APackageMissingOrBelowItsPinFails(t *testing.T) {
	for name, answer := range map[string]map[string]string{
		"missing": {"python3 --version": "Python 3.12.6"},
		"below":   {"python3 --version": "Python 3.12.6", "python3 -c " + tools.PyDistVersion + " pyyaml": "5.4.1\n"},
	} {
		s, cfg := pythonFixture(t, "darwin", pythonVersions, []string{"python3", "mise"}, answer)
		rep, out := runCheckPython(s, cfg)
		if rep.Failures() != 1 || !strings.Contains(out, "cannot import at their pin: pyyaml") {
			t.Errorf("%s: want one FAIL naming pyyaml\n%s", name, out)
		}
	}
}

// The sync installs packages into mise's Python only: when that one has them,
// the shell is running another Python, and the remedy says so, not the sync.
func TestCheckPython_APackageOnlyMisePythonHasNamesThePath(t *testing.T) {
	s, cfg := pythonFixture(t, "darwin", pythonVersions, []string{"python3", "mise"}, map[string]string{
		"python3 --version":  "Python 3.12.6",
		"mise which python3": "/m/python3\n",
		"/m/python3 -c " + tools.PyDistVersion + " pyyaml": "6.0.3\n",
	})
	rep, out := runCheckPython(s, cfg)
	if rep.Failures() != 1 || !strings.Contains(out, "mise's python at /m/python3 has them") || strings.Contains(out, "run: dotf tools sync") {
		t.Errorf("want one FAIL naming mise's python and the PATH, not the sync\n%s", out)
	}
}

// Without mise nothing manages the packages, as checkMiseTools reports for the
// CLIs: a missing package warns, naming that, and is no failure (#2062; on
// Windows mise waits for the ADR-044 Windows wave, #2013). The floor still
// fails without mise: TestCheckPython_AbsentFails.
func TestCheckPython_APackageMissingWithoutMiseWarns(t *testing.T) {
	for _, goos := range []string{"windows", "darwin"} {
		py := "python3"
		if goos == "windows" {
			py = "python"
		}
		s, cfg := pythonFixture(t, goos, pythonVersions, []string{py}, map[string]string{
			py + " --version": "Python 3.12.10",
		})
		rep, out := runCheckPython(s, cfg)
		if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(out, "cannot import at their pin: pyyaml") || !strings.Contains(out, "mise not on PATH") {
			t.Errorf("%s: want one WARN naming pyyaml and the missing mise, no FAIL\n%s", goos, out)
		}
	}
}
