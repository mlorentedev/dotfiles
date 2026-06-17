package doctor

import (
	"fmt"
	"path/filepath"
	"strings"
)

// checkGuardHooks verifies the GUARD-001 memory-sink dispatcher is wired
// machine-wide via git's global core.hooksPath. The dispatcher (the tracked
// git-hooks/ dir under DOTFILES_DIR, shipped in #409) only runs in a repo when
// core.hooksPath points at it — so this check is what actually ACTIVATES the
// guard; without it the dispatcher is inert.
//
// Safety-first, because a global core.hooksPath has machine-wide blast radius:
// an unrelated pre-existing value is NEVER clobbered. It is surfaced as a WARN so
// the human resolves the conflict deliberately. Only an unset hooksPath is wired
// (and only under --fix); an already-correct one is a no-op (idempotent re-run).
func checkGuardHooks(sys *System, cfg *Config, rep *Report, fix bool) {
	rep.Section("GUARD memory-sink hooks")

	target := filepath.Join(cfg.DotfilesDir, "git-hooks")

	// Wiring core.hooksPath at a dir without the dispatcher would point git at a
	// missing hook — refuse and tell the user to deploy first.
	if !pathExists(filepath.Join(target, "pre-commit")) {
		rep.Fail("dispatcher not found at " + target + " — run dotfiles setup to deploy git-hooks/")
		return
	}

	current := gitGlobalHooksPath(sys)
	switch {
	case current == target:
		rep.Pass("core.hooksPath wired to the GUARD dispatcher")
	case current == "":
		if !fix {
			rep.Fail("core.hooksPath unset — GUARD memory-sink guard inactive; run `dotf doctor --fix`")
			return
		}
		if err := setGitGlobalHooksPath(sys, target); err != nil {
			rep.Fail(fmt.Sprintf("could not wire core.hooksPath: %v", err))
			return
		}
		rep.Fix("wired core.hooksPath → " + target)
	default:
		// Unrelated pre-existing value: preserve it, never clobber.
		rep.Warn(fmt.Sprintf("core.hooksPath is %s (not the GUARD dispatcher) — preserving it; "+
			"GUARD inactive. Point it at %s, or chain the dispatcher from your hooks, manually.", current, target))
	}
}

// gitGlobalHooksPath returns the global core.hooksPath, or "" when unset.
// `git config --global --get` exits non-zero for an absent key, which the System
// fake/real both surface as an error — mapped here to "".
func gitGlobalHooksPath(sys *System) string {
	out, err := sys.CommandOutput("git", "config", "--global", "--get", "core.hooksPath")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// setGitGlobalHooksPath wires the dispatcher dir machine-wide.
func setGitGlobalHooksPath(sys *System, target string) error {
	_, err := sys.CommandOutput("git", "config", "--global", "core.hooksPath", target)
	return err
}
