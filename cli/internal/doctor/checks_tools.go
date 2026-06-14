package doctor

import (
	"fmt"
	"path/filepath"
	"strings"
)

// coreTools is healthcheck.sh section 1's PATH expectation set.
var coreTools = []string{
	"git", "zsh", "bash", "curl", "wget", "jq", "eza",
	"direnv", "node", "npm", "zoxide", "docker", "kubectl", "terraform",
}

// checkCoreTools reproduces healthcheck section 1: the core toolchain is on
// PATH. Names already version-checked by the contract (git, jq) are skipped here
// to avoid double-reporting — the consolidation the two twins never did.
func checkCoreTools(sys *System, c *Contract, rep *Report) {
	rep.Section("Core tools in PATH")
	covered := contractBinaryNames(c)
	for _, tool := range coreTools {
		if covered[tool] {
			continue
		}
		if sys.has(tool) {
			rep.Pass(tool + " found")
		} else {
			rep.Fail(tool + " not in PATH")
		}
	}
}

// toolHome pairs a *_HOME env var with the binary expected under its bin/.
type toolHome struct {
	envVar string
	binary string
}

var versionedHomes = []toolHome{
	{"JAVA_HOME", "java"},
	{"MAVEN_HOME", "mvn"},
	{"PYTHON_HOME", "python3"},
	{"GO_HOME", "go"},
}

// checkVersionedPaths reproduces healthcheck section 2: each *_HOME points at a
// real dir containing its binary. An unset var SKIPs (the tool is optional);
// a present dir missing its binary, or a missing dir, FAILs. Minikube is
// special-cased — its binary sits at the dir root, not under bin/.
func checkVersionedPaths(sys *System, rep *Report) {
	rep.Section("Versioned tool paths")
	for _, t := range versionedHomes {
		dir := sys.Getenv(t.envVar)
		switch {
		case dir == "":
			rep.Skip(t.envVar + " (variable not set)")
		case isExecFile(filepath.Join(dir, "bin", t.binary)):
			rep.Pass(fmt.Sprintf("%s (%s)", t.envVar, dir))
		case isDir(dir):
			rep.Fail(fmt.Sprintf("%s directory exists but %s not found in %s/bin/", t.envVar, t.binary, dir))
		default:
			rep.Fail(fmt.Sprintf("%s directory missing: %s", t.envVar, dir))
		}
	}

	mk := sys.Getenv("MINIKUBE_HOME")
	switch {
	case mk == "":
		rep.Skip("MINIKUBE_HOME (variable not set)")
	case isExecFile(filepath.Join(mk, "minikube")):
		rep.Pass(fmt.Sprintf("MINIKUBE_HOME (%s)", mk))
	case isDir(mk):
		rep.Fail("MINIKUBE_HOME directory exists but minikube binary not found")
	default:
		rep.Fail("MINIKUBE_HOME directory missing: " + mk)
	}
}

// versionedDir pairs a display name with its versions.conf key and the
// APPS_HOME subdirectory prefix the installer lays the version down under.
type versionedDir struct {
	name      string
	key       string
	dirPrefix string
}

var versionMatches = []versionedDir{
	{"Java", "JAVA_VERSION", "jdk-"},
	{"Maven", "MAVEN_VERSION", "apache-maven-"},
	{"Python", "PYTHON_VERSION", "python-"},
	{"Minikube", "MINIKUBE_VERSION", "minikube-"},
	{"Go", "GO_VERSION", "go-"},
}

// checkVersionMatch reproduces healthcheck section 3: the pinned version from
// versions.conf has its corresponding ~/Applications/<tool>-<version> dir. Yarn
// is npm-global pinned (no versioned dir) so it compares the live binary's
// version to the pin: a drift is a WARN, not a FAIL.
func checkVersionMatch(sys *System, cfg *Config, rep *Report) {
	rep.Section("Version match (versions.conf)")
	appsHome := sys.env("APPS_HOME", filepath.Join(sys.home(), "Applications"))

	for _, v := range versionMatches {
		want := cfg.Versions[v.key]
		if want == "" {
			rep.Skip(v.name + " version (not set in versions.conf)")
			continue
		}
		dir := filepath.Join(appsHome, v.dirPrefix+want)
		if isDir(dir) {
			rep.Pass(fmt.Sprintf("%s version %s (directory exists)", v.name, want))
		} else {
			rep.Fail(fmt.Sprintf("%s expected version %s but directory missing: %s", v.name, want, dir))
		}
	}

	pin := cfg.Versions["YARN_VERSION"]
	switch {
	case !sys.has("yarn"):
		rep.Skip("yarn not installed (npm install -g yarn, or re-run setup)")
	case pin == "":
		rep.Skip("YARN_VERSION not set in versions.conf — version match not verified")
	default:
		got, _ := sys.versionLine("yarn")
		if got == pin {
			rep.Pass(fmt.Sprintf("yarn version matches versions.conf (%s)", pin))
		} else {
			rep.Warn(fmt.Sprintf("yarn version drift: installed=%s pinned=%s", got, pin))
		}
	}
}

// toolHomeVars is healthcheck section 5's required-set expectation. DOTFILES_DIR
// is intentionally omitted: it is owned by the env-contract section (which also
// validates its path), so listing it here would double-report.
var toolHomeVars = []string{
	"APPS_HOME", "JAVA_HOME", "MAVEN_HOME", "PYTHON_HOME", "GO_HOME", "MINIKUBE_HOME",
}

// checkToolHomeEnvVars reproduces healthcheck section 5: the tool-home vars are
// set. Unset is a FAIL (the RC files are expected to export them).
func checkToolHomeEnvVars(sys *System, rep *Report) {
	rep.Section("Tool-home environment variables")
	for _, v := range toolHomeVars {
		if sys.Getenv(v) != "" {
			rep.Pass(v + " is set")
		} else {
			rep.Fail(v + " is not set")
		}
	}
}

// optionalTools is healthcheck section 6's advisory set.
var optionalTools = []string{
	"age", "gh", "claude", "gemini", "agy", "bats", "shellcheck", "helm", "ansible", "pip",
}

// checkOptionalTools reproduces healthcheck section 6 folded with the contract's
// optional_binaries: present → PASS (annotated with the contract purpose when
// known), absent → SKIP. dotf itself is reported here with its version-pin
// match (drift → WARN), since dotf is the on-PATH replacement these scripts
// converge into.
func checkOptionalTools(sys *System, cfg *Config, c *Contract, rep *Report) {
	rep.Section("Optional tools")

	purpose := map[string]string{}
	var order []string
	seen := map[string]bool{}
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			order = append(order, name)
		}
	}
	for _, t := range optionalTools {
		add(t)
	}
	if c != nil {
		for _, o := range c.OptionalBinaries {
			purpose[o.Name] = o.Purpose
			add(o.Name)
		}
	}

	for _, name := range order {
		label := name
		if p := purpose[name]; p != "" {
			label = fmt.Sprintf("%s (%s)", name, p)
		}
		if sys.has(name) {
			rep.Pass(label + " found")
		} else {
			rep.Skip(label + " not installed")
		}
	}

	// dotf — the CLI these twins converge into (ADR-020).
	pin := cfg.Versions["DOTF_VERSION"]
	switch {
	case !sys.has("dotf"):
		rep.Skip("dotf not in PATH (run ./scripts/install-dotf.sh or setup)")
	case pin == "":
		rep.Pass("dotf in PATH (DOTF_VERSION not pinned — match not verified)")
	default:
		got := dotfVersion(sys)
		if got == pin {
			rep.Pass(fmt.Sprintf("dotf %s matches versions.conf", pin))
		} else {
			rep.Warn(fmt.Sprintf("dotf version drift: installed=%s pinned=%s (run ./scripts/install-dotf.sh)", got, pin))
		}
	}
}

// dotfVersion returns the trailing field of `dotf version`'s first line (e.g.
// "dotf version 0.2.0" → "0.2.0"), matching the awk '{print $NF}' the twin used.
func dotfVersion(sys *System) string {
	out, err := sys.CommandOutput("dotf", "version")
	if err != nil {
		return "unknown"
	}
	first := out
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		first = out[:i]
	}
	fields := strings.Fields(first)
	if len(fields) == 0 {
		return "unknown"
	}
	return fields[len(fields)-1]
}
