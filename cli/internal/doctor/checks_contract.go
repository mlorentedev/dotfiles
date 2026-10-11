package doctor

import (
	"fmt"
	"regexp"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

// checkContractEnvVars reproduces doctor.sh section 1: each declared structural
// env var is present (or has an applicable default), and path_exists vars point
// at a real directory. In --fix mode an unset var with a default is reported as
// a FIX with the exact profile line to add — a Go subprocess cannot export into
// the parent shell, so "applying" the default means telling the user how to
// persist it, not mutating an ephemeral child environment.
func checkContractEnvVars(sys *System, c *Contract, rep *Report, fix bool) {
	rep.Section("Environment variables (contract" + contractScope(sys) + ")")
	osName := envpkg.ContractOS(sys.GOOS)
	for _, e := range c.EnvVars {
		// A var scoped to a different OS than this one does not apply here.
		if !envpkg.AppliesOn(e.RequiredOn, sys.GOOS) {
			rep.Pass(fmt.Sprintf("%s (%s-scoped, skipped on %s)", e.Name, e.RequiredOn, osName))
			continue
		}

		current := sys.Getenv(e.Name)
		if current == "" {
			raw, _, _ := envpkg.ForOS(e.Default, sys.GOOS)
			def := expandHome(sys, raw)
			if def == "" {
				if e.requiredHere() {
					rep.Fail(e.Name + " unset and no default available (required)")
				} else {
					rep.Pass(e.Name + " unset (optional, no default)")
				}
				continue
			}
			if fix {
				rep.Fix(fmt.Sprintf("%s unset — add to your shell profile: export %s=%q", e.Name, e.Name, def))
			} else if e.requiredHere() {
				rep.Warn(fmt.Sprintf("%s unset (required); default %q — run --fix or set in profile", e.Name, def))
			} else {
				rep.Warn(fmt.Sprintf("%s unset; default %q would be reported with --fix", e.Name, def))
			}
			current = def
		}

		// Validation runs against the resolved value (set or defaulted), exactly
		// as doctor.sh did — so a default pointing at a missing dir still FAILs.
		if e.Validation == "path_exists" {
			if isDir(current) {
				rep.Pass(fmt.Sprintf("%s=%s (path exists)", e.Name, current))
			} else {
				rep.Fail(fmt.Sprintf("%s=%s (path does not exist)", e.Name, current))
			}
		} else {
			rep.Pass(fmt.Sprintf("%s=%s", e.Name, current))
		}
	}
}

// checkContractPath reproduces doctor.sh section 2: every required PATH entry
// for this OS is actually on PATH. A miss is a WARN (advisory: the shell profile
// will set it on next login), never a hard FAIL.
func checkContractPath(sys *System, c *Contract, rep *Report) {
	rep.Section("PATH entries (contract" + contractScope(sys) + ")")
	entries, _, ok := envpkg.ForOS(c.RequiredPathEntries, sys.GOOS)
	if !ok {
		// An OS the contract has no key for checks nothing, and an empty
		// section would read as a clean PATH.
		rep.Warn("no required PATH entries declared for " + envpkg.ContractOS(sys.GOOS) + " — add its key to env-contract.json")
		return
	}
	for _, entry := range entries {
		expanded := expandHome(sys, entry)
		if pathContains(sys, expanded) {
			rep.Pass(expanded + " in PATH")
		} else {
			rep.Warn(expanded + " not in PATH — check shell profile")
		}
	}
}

// checkRequiredBinaries reproduces doctor.sh section 3: each required binary is
// on PATH and, when pinned, meets its minimum version. An unparseable version
// is a WARN (present but couldn't verify), a too-old version is a FAIL.
func checkRequiredBinaries(sys *System, c *Contract, rep *Report) {
	rep.Section("Required binaries (contract)")
	for _, b := range c.RequiredBinaries {
		if !sys.has(b.Name) {
			rep.Fail(b.Name + " missing")
			continue
		}
		if b.MinVersion == "" {
			rep.Pass(b.Name + " on PATH")
			continue
		}
		raw, _ := sys.versionLine(b.Name)
		re, err := regexp.Compile(b.VersionPattern)
		if err != nil {
			rep.Warn(fmt.Sprintf("%s on PATH but version_pattern invalid: %v", b.Name, err))
			continue
		}
		m := re.FindStringSubmatch(raw)
		if len(m) < 2 {
			rep.Warn(fmt.Sprintf("%s on PATH but version unparseable: %q (pattern: %s)", b.Name, raw, b.VersionPattern))
			continue
		}
		actual := m[1]
		if atLeast(actual, b.MinVersion) {
			rep.Pass(fmt.Sprintf("%s %s (>= %s)", b.Name, actual, b.MinVersion))
		} else {
			rep.Fail(fmt.Sprintf("%s %s is older than minimum %s", b.Name, actual, b.MinVersion))
		}
	}
}

// contractScope names, in a section header, the contract key this OS reads,
// and the key it falls back to where an entry declares none of its own, so a
// darwin report never reads as a linux one (#2013 P2).
func contractScope(sys *System) string {
	scope := ", " + envpkg.ContractOS(sys.GOOS)
	if fb := envpkg.FallbackOS(sys.GOOS); fb != "" {
		scope += "; undeclared keys read " + fb
	}
	return scope
}

// contractBinaryNames returns the set of binary names already version-checked by
// the contract, so the core-tools section can avoid double-reporting them.
func contractBinaryNames(c *Contract) map[string]bool {
	names := map[string]bool{}
	if c == nil {
		return names
	}
	for _, b := range c.RequiredBinaries {
		names[b.Name] = true
	}
	return names
}
