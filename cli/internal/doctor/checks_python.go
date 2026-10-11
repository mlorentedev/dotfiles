package doctor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// pythonFloor is the oldest Python the suite runs on (#2062): its TOML readers
// import tomllib, new in 3.11. The exact version mise installs is
// PYTHON_VERSION in versions.conf; this is only the floor the Python a shell
// resolves must clear.
const pythonFloor = "3.11.0"

var pythonVersionRE = regexp.MustCompile(`[0-9]+\.[0-9]+(\.[0-9]+)?`)

// pythonCommand is the name a shell runs Python by: Windows installs, mise's
// among them, carry python.exe and no python3.
func pythonCommand(sys *System) string {
	if sys.GOOS == "windows" {
		return "python"
	}
	return "python3"
}

// checkPython requires the Python a shell resolves to clear pythonFloor and to
// import the packages versions.conf marks "# mise: python-package" at their
// pin (#2062, ADR-044); a missing package only warns where mise is absent. mise provides both through `dotf tools sync`, which
// checkMiseTools runs under fix; this check only probes what the shell gets,
// so a Python mise installed but the shell does not reach still fails, and
// says so.
func checkPython(sys *System, cfg *Config, rep *Report) {
	rep.Section("Python")
	var pins tools.MisePins
	if cfg.VersionsPath != "" {
		raw, err := os.ReadFile(cfg.VersionsPath)
		if err != nil {
			rep.Fail("versions.conf unreadable: " + err.Error())
			return
		}
		// A versions.conf the parser rejects is checkMiseTools' failure to
		// report; the floor still applies without its pins.
		pins, _ = tools.ParseMisePins(raw)
	}
	py := pythonCommand(sys)
	v := ""
	if sys.has(py) {
		line, _ := sys.versionLine(py)
		v = pythonVersionRE.FindString(line)
	}
	switch {
	case v == "":
		rep.Fail(fmt.Sprintf("%s not on PATH: dotf requires Python >= %s (%s)", py, pythonFloor, pythonRemedy(sys, pins)))
		return
	case !atLeast(v, pythonFloor):
		rep.Fail(fmt.Sprintf("%s on PATH is %s: dotf requires Python >= %s (%s)", py, v, pythonFloor, pythonRemedy(sys, pins)))
		return
	}
	rep.Pass(fmt.Sprintf("%s %s", py, v))
	if missing := packagesMissing(sys, py, pins.PythonPackages); len(missing) > 0 {
		// Without mise nothing manages the packages, as checkMiseTools says of
		// the CLIs; Windows runs no mise until the ADR-044 Windows wave (#2013).
		if !sys.has("mise") {
			rep.Warn(fmt.Sprintf("%s cannot import at their pin: %s (mise not on PATH, so nothing manages them; install mise, then run: dotf tools sync)", py, strings.Join(missing, ", ")))
			return
		}
		rep.Fail(fmt.Sprintf("%s cannot import at their pin: %s (%s)", py, strings.Join(missing, ", "), packagesRemedy(sys, pins.PythonPackages)))
		return
	}
	if len(pins.PythonPackages) > 0 {
		rep.Pass(fmt.Sprintf("%d python package(s) at their pin", len(pins.PythonPackages)))
	}
}

// pythonRemedy says what clears a Python below the floor. A sync installs one
// only when versions.conf pins it for mise; when mise already resolves one
// that clears the floor, a sync changes nothing, because the shell is not
// using it.
func pythonRemedy(sys *System, pins tools.MisePins) string {
	pinned := false
	for _, t := range pins.Tools {
		pinned = pinned || t.Name == "python"
	}
	switch {
	case !pinned:
		return "versions.conf pins no PYTHON_VERSION under \"" + tools.MiseMarker + "\", so no sync installs one: install Python >= " + pythonFloor
	case !sys.has("mise"):
		return "install mise, then run: dotf tools sync"
	}
	path, err := sys.CommandStdoutDir(sys.home(), "mise", "which", pythonCommand(sys))
	path = strings.TrimSpace(path)
	if err != nil || path == "" {
		return "run: dotf tools sync"
	}
	out, _ := sys.CommandOutput(path, "--version")
	if mv := pythonVersionRE.FindString(out); mv != "" && atLeast(mv, pythonFloor) {
		return fmt.Sprintf("mise's python is %s at %s, but this shell does not resolve it: activate mise in the shell, or put its shims ahead of the system python on PATH", mv, path)
	}
	return "run: dotf tools sync"
}

// packagesMissing names the packages python does not carry at or above their
// pin.
func packagesMissing(sys *System, python string, pkgs []tools.MiseTool) []string {
	var missing []string
	for _, p := range pkgs {
		out, err := sys.CommandOutput(python, "-c", tools.PyDistVersion, p.Name)
		got := pythonVersionRE.FindString(out)
		if err != nil || got == "" || !atLeast(got, p.Version) {
			missing = append(missing, p.Name)
		}
	}
	return missing
}

// packagesRemedy says what clears a package the shell's Python lacks, on a
// machine with mise. The sync installs the packages into mise's Python only,
// so when that one already carries them, the shell is running another Python
// and a sync changes nothing.
func packagesRemedy(sys *System, pkgs []tools.MiseTool) string {
	path, err := sys.CommandStdoutDir(sys.home(), "mise", "which", pythonCommand(sys))
	path = strings.TrimSpace(path)
	if err == nil && path != "" && len(packagesMissing(sys, path, pkgs)) == 0 {
		return fmt.Sprintf("mise's python at %s has them, but this shell runs another python: activate mise in the shell, or put its shims ahead of it on PATH", path)
	}
	return "run: dotf tools sync"
}
