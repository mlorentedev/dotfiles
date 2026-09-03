package doctor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// coreTools is healthcheck.sh section 1's PATH expectation set.
var coreTools = []string{
	"git", "zsh", "bash", "curl", "wget", "jq", "eza",
	"direnv", "node", "npm", "zoxide", "docker", "kubectl",
}

// posixOnlyTools are absent on Windows by design — skip them there instead of
// reporting a false failure (Windows uses pwsh, not zsh; direnv is POSIX-shell).
// wget joined on the first CI doctor run (TEST-003/#1298): no Windows setup
// block installs it and no Windows script calls it (curl and Invoke-WebRequest
// cover the role), so expecting it there failed every fresh box for a tool the
// repo never provisions.
var posixOnlyTools = map[string]bool{"zsh": true, "direnv": true, "wget": true}

// checkCoreTools reproduces healthcheck section 1: the core toolchain is on
// PATH. Names already version-checked by the contract (git, jq) are skipped here
// to avoid double-reporting — the consolidation the two twins never did.
func checkCoreTools(sys *System, c *Contract, rep *Report) {
	rep.Section("Core tools in PATH")
	covered := contractBinaryNames(c)
	win := sys.GOOS == "windows"
	for _, tool := range coreTools {
		if covered[tool] {
			continue
		}
		if win && posixOnlyTools[tool] {
			rep.Skip(tool + " (POSIX-only; not expected on Windows)")
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
		case isExecFile(filepath.Join(dir, "bin", t.binary)),
			sys.GOOS == "windows" && isExecFile(filepath.Join(dir, "bin", t.binary+".exe")):
			// The .exe form: a JAVA_HOME set by the GitHub runner's toolcache
			// holds java.exe, and the extensionless probe reported it missing
			// (TEST-003/#1298 first run).
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
	if cfg.VersionsPath != "" {
		// Provenance (#697): show which versions.conf the pins came from, so a stale
		// deployed copy producing nonsensical drift directions is self-diagnosing.
		rep.Info("versions.conf: " + cfg.VersionsPath)
	}
	if sys.GOOS == "windows" {
		// Windows installs these via winget, not ~/Applications/<tool>-<version>,
		// so the versioned-dir check does not apply (parity with healthcheck.ps1
		// section 3, which skips when APPS_HOME is unset).
		rep.Skip("versioned tool dirs (Windows uses winget, not APPS_HOME)")
	} else {
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
	}

	// yarn is npm-global (cross-platform) — always checked. Its pin lives in
	// packages.json since OPS-042 (#1336), where `dotf tools install` reads it;
	// versions.conf no longer carries YARN_VERSION.
	if !sys.has("yarn") {
		rep.Skip("yarn not installed (run `dotf tools install yarn`; needs Node.js on PATH)")
	} else {
		got, _ := sys.versionLine("yarn")
		matchPinFloorFrom(rep, "yarn", got, catalogPin(sys, cfg, "yarn"), "packages.json")
	}

	checkGitWindowsFloor(sys, cfg, rep)
}

// gitFloorKey is the versions.conf pin checkGitWindowsFloor consumes. Like every
// pin in that file it is a MINIMUM (REFACTOR-013), and unlike the others no
// installer provisions it — git comes from the OS package manager — so doctor is
// its only consumer.
const gitFloorKey = "GIT_VERSION"

// gitVersionRe pulls the numeric triple out of `git --version`. git-for-windows
// reports "git version 2.55.0.windows.5", so the match is anchored on the shape
// of the triple, not on the end of the line.
var gitVersionRe = regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)

// checkGitWindowsFloor reports whether git-for-windows meets the floor that
// carries the upstream fix for BUG-069 (#912).
//
// Measured on 2.53.0, MSYS bash could not open a hook handed to it at a `C:/`
// drive path — the form both `dotf hooks install` and checkGuardHooks write to
// core.hooksPath — so every commit aborted with "No such file or directory".
// (The measurement predates CLI-072 and named install-git-hooks.ps1, the twin
// `dotf hooks install` replaced; the path form it writes is unchanged.)
// Measured again on 2.55.0.windows.5 the same value works: the defect was fixed
// by git-for-windows between the two measurements (lesson 239). What survives a
// bug fixed upstream is not a workaround but the floor: the toolchain version
// from which the repo's wiring is known to execute.
//
// A WARN, not a FAIL, for two reasons. The floor is a version heuristic
// bracketed by two measurements, not an observed failure on this box; and
// doctor's precedent for a tool one release behind its pin is a WARN (yarn,
// opencode). It is Windows-only because the hooksPath defect was git-for-
// windows', and the env contract's own git min_version already FAILs a
// genuinely ancient git on every OS.
func checkGitWindowsFloor(sys *System, cfg *Config, rep *Report) {
	pin := cfg.Versions[gitFloorKey]
	switch {
	case sys.GOOS != "windows":
		rep.Skip("git-for-windows floor (Windows-only; the C:/ hooksPath defect was git-for-windows', #912)")
	case pin == "":
		rep.Skip(gitFloorKey + " not set in versions.conf — git-for-windows floor not verified")
	case !sys.has("git"):
		// The env contract FAILs an absent git; repeating it here would report
		// one defect twice.
		rep.Skip("git not in PATH — git-for-windows floor unverifiable (the env contract owns that FAIL)")
	default:
		raw, _ := sys.versionLine("git")
		m := gitVersionRe.FindStringSubmatch(raw)
		if len(m) < 2 {
			rep.Warn(fmt.Sprintf("git version unparseable: %q — git-for-windows floor %s not verified", raw, pin))
			return
		}
		if atLeast(m[1], pin) {
			rep.Pass(fmt.Sprintf("git %s meets the git-for-windows floor %s (#912)", m[1], pin))
			return
		}
		rep.Warn(fmt.Sprintf("git %s is below the git-for-windows floor %s — before 2.55 MSYS bash could not open a C:/-form core.hooksPath, so the GUARD hooks fail on every commit; upgrade with `winget upgrade Git.Git` (#912, lesson 239)", m[1], pin))
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
	win := sys.GOOS == "windows"
	for _, v := range toolHomeVars {
		switch {
		case sys.Getenv(v) != "":
			rep.Pass(v + " is set")
		case win:
			// Linux-deploy vars; optional on Windows (winget-managed toolchains).
			rep.Skip(v + " (optional on Windows — Linux-deploy var)")
		default:
			rep.Fail(v + " is not set")
		}
	}
}

// optionalTools is healthcheck section 6's advisory set. terraform lives here
// (not in coreTools): it is an optional IaC tool, so its absence is a SKIP, not a
// FAIL that pushes doctor to exit 1 on a machine that never wanted it (BUG-052).
var optionalTools = []string{
	"age", "gh", "claude", "gemini", "agy", "bats", "shellcheck", "helm", "ansible", "pip", "terraform",
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

	checkDotfVersion(sys, cfg, rep)
}

// checkDotfVersion reports dotf itself — the CLI these twins converge into
// (ADR-020) — against the versions.conf pin. Absent → SKIP; unpinned → PASS
// unverified; a stale release → FAIL; a source build ("dev") → SKIP, since
// the pin exists to catch a stale release and CI runs the PR's own build.
func checkDotfVersion(sys *System, cfg *Config, rep *Report) {
	pin := cfg.Versions["DOTF_VERSION"]
	switch {
	case !sys.has("dotf"):
		rep.Skip("dotf not in PATH (run ./scripts/install-dotf.sh or setup)")
	case pin == "":
		rep.Pass("dotf in PATH (DOTF_VERSION not pinned — match not verified)")
	default:
		got := dotfVersion(sys)
		switch got {
		case pin:
			rep.Pass(fmt.Sprintf("dotf %s matches versions.conf", pin))
		case "dev":
			// A source build (cli/cmd/dotf/main.go's default): deliberate on a
			// dev box, and what CI runs after building the PR under test. The
			// pin exists to catch a STALE RELEASE binary, which this is not.
			rep.Skip("dotf is a source build (dev) — the versions.conf pin does not apply")
		default:
			// FAIL, not WARN (OPS-025/#869): a stale dotf binary means every guard
			// merged into doctor since it was built is not running at all on this
			// machine — a categorically worse gap than an ordinary tool being one
			// version behind, where the check itself still runs.
			rep.Fail(fmt.Sprintf("dotf version drift: installed=%s pinned=%s (run ./scripts/install-dotf.sh)", got, pin))
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
