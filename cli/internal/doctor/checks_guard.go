package doctor

import (
	"fmt"
	"path/filepath"
	"runtime"
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
	case samePath(current, target):
		rep.Pass("core.hooksPath wired to the GUARD dispatcher")
	case isGuardDispatcher(current):
		// BUG-040: the question is whether a GUARD dispatcher RUNS, not whether
		// the path matches the deploy mirror. Developing the hooks from a repo
		// checkout points hooksPath at an equivalent dispatcher — active, and
		// previously reported INACTIVE on every run. Name the path so the
		// divergence from the mirror stays visible instead of silently blessed.
		rep.Pass("GUARD dispatcher active via " + current + " (not the deploy mirror " + target + ")")
	default:
		// Unrelated pre-existing value: preserve it, never clobber.
		rep.Warn(fmt.Sprintf("core.hooksPath is %s (not the GUARD dispatcher) — preserving it; "+
			"GUARD inactive. Point it at %s, or chain the dispatcher from your hooks, manually.", current, target))
	}

	// Everything above answers a question about the MACHINE. git resolves
	// core.hooksPath local-over-global, so a correctly wired machine says nothing
	// about whether the guard fires in any particular repo — which is how a
	// repo-local override left the guard inert in the dotfiles checkout itself
	// while this section reported "all ok", before AND after the override was
	// removed (BUG-056). Probe the repos that matter by effect.
	for _, repo := range guardProbeRepos(sys, cfg) {
		switch {
		case guardRunsIn(sys, repo):
			rep.Pass("guard fires in " + repo)
		case effectiveHooksPath(sys, repo) != "":
			rep.Fail("guard INACTIVE in " + repo + " — core.hooksPath is overridden there (" +
				effectiveHooksPath(sys, repo) + "); unset it with `git -C " + repo + " config --unset core.hooksPath`")
		default:
			rep.Fail("guard INACTIVE in " + repo + " — the pre-commit hook git runs there is not a GUARD dispatcher")
		}
	}
}

// guardProbeRepos lists the git checkouts worth probing by effect: the dotfiles
// repo (where the guard is developed, and the worst place for it to be silently
// off) and the vault (the guard's single sink, so the one repo whose memory
// artifacts are the exception). Absent or non-git paths are dropped rather than
// reported — a machine without the vault is a valid state, already SKIPped by
// checkVault.
//
// Both come from the environment contract ONLY, never from resolveRepoDir's
// cwd-walk fallback. Two reasons, and the second is the one that bit:
//
//   - Probing "whatever repo the caller happens to be sitting in" is surprising
//     for a machine-health command, and its output would change with cwd.
//   - It reaches ambient filesystem state, so the unit tests stopped being unit
//     tests: in CI the walk found the real checkout and every pre-existing
//     guard case went red on a machine property none of them were asserting.
//     That is the same defect this whole change exists to prevent, committed
//     while writing the prevention — a test that does not isolate from the
//     machine ends up measuring the machine.
func guardProbeRepos(sys *System, cfg *Config) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range []string{
		sys.Getenv("DOTFILES_REPO_DIR"),
		sys.Getenv("VAULT_PATH"),
	} {
		if r == "" || seen[r] || !isDir(filepath.Join(r, ".git")) {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// isGuardDispatcher reports whether dir holds a GUARD-001 dispatcher: the
// pre-commit entrypoint AND the memory-sink guard it delegates to. Structural
// rather than a marker grep — pre-commit execs lib/memory-sink-guard.sh, so
// without that script the guard cannot run no matter what the files say.
func isGuardDispatcher(dir string) bool {
	return pathExists(filepath.Join(dir, "pre-commit")) &&
		pathExists(filepath.Join(dir, "lib", "memory-sink-guard.sh"))
}

// samePath compares two hooksPath values as paths, not as bytes. git reports
// core.hooksPath with forward slashes even on Windows, so comparing it to a
// filepath.Join target is a false negative there; trailing separators and
// drive-letter case are the same class of noise.
func samePath(a, b string) bool {
	norm := func(p string) string {
		return filepath.Clean(filepath.FromSlash(strings.TrimRight(p, `/\`)))
	}
	na, nb := norm(a), norm(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(na, nb)
	}
	return na == nb
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
