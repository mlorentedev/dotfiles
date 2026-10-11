package doctor

import (
	"fmt"
	"path/filepath"
	"strings"
)

// checkVaultHooks verifies the knowledge vault's LOCAL pre-commit/pre-push git
// hooks are installed — the vault's only secret gate.
//
// Why this lives here and not in a bootstrap script: the vault runs no CI (its
// feedback-no-ci-for-vault rule — obsidian-git auto-pushes on a timer, so a
// push-triggered pipeline would run almost continuously) and its server-side
// backstop was removed on 2026-06-23 (knowledge#114). So gitleaks at pre-push is
// the only thing standing between a pasted secret and a public push. But
// .git/hooks/ is NOT version-controlled, so a fresh clone has no gate until
// `pre-commit install` runs — a silent gap on every new machine (knowledge#549 /
// OPS-016). Provisioning the gate is pure behavior (no repo asset to deploy), so
// it belongs in the consolidated `dotf doctor` checker, not a new shell script.
//
// Mirrors checkGuardHooks: verify always, repair only under --fix, idempotent.
// The vault is OPTIONAL — absent → SKIP (not every machine syncs it). But when
// the vault IS present and the gate is missing, --fix installs it, and a missing
// pre-commit binary is a loud FAIL, never a silent skip (the #549 acceptance:
// "rather than silently skipping the gate").
func checkVaultHooks(sys *System, rep *Report, fix bool) {
	rep.Section("Knowledge vault hooks (secret gate)")

	// Same ADR-025 seam as checkVault — VAULT_PATH with the legacy default. No
	// hardcode, so a path-default fix (e.g. #551) is inherited here for free.
	vault := sys.env("VAULT_PATH", filepath.Join(sys.home(), "Projects", "knowledge"))

	// Vault absent -> nothing to protect. SKIP, not FAIL: a machine that never
	// syncs the vault is a valid state. Asks git rather than assuming
	// <vault>/.git is a directory — it is a gitdir: pointer FILE in a linked
	// worktree, which used to make this SKIP a checkout that was genuinely
	// present (#806).
	if !isGitCheckout(sys, vault) {
		rep.Skip("no vault checkout at " + vault + " — no secret gate to provision")
		return
	}
	if !pathExists(filepath.Join(vault, ".pre-commit-config.yaml")) {
		rep.Skip(vault + " has no .pre-commit-config.yaml — not the knowledge vault?")
		return
	}

	// The gate is "active" only when BOTH stages reach pre-commit's gates:
	// gitleaks + the hardcoded-path guard live on pre-push, the structural checks
	// on pre-commit (the vault's .pre-commit-config.yaml). A bare or hand-rolled
	// hook squatting the path still does not count — but a GUARD dispatcher does,
	// because it chains into `pre-commit hook-impl` and genuinely runs them.
	//
	// Resolve what git will RUN for each stage, not whether pre-commit's own
	// generated file sits in .git/hooks. Once core.hooksPath is set machine-wide
	// `pre-commit install` refuses outright, so that file is one the design
	// guarantees absent — and this check reported the gate INACTIVE while gitleaks
	// was verifiably running through the GUARD dispatcher's fallback.
	//
	// Every route reaches gitleaks only through the pre-commit binary, so a
	// missing binary is an inactive gate whatever the hooks look like. Checked
	// first and in check mode too: a Mac with the dispatcher deployed and no
	// pre-commit installed reported "gate active" while no commit was scanned.
	preCommit := preCommitPath(sys)
	if preCommit == "" {
		rep.Fail("vault secret gate INACTIVE — pre-commit not on PATH; install it with `dotf tools install pre-commit`")
		return
	}
	preCommitOK := stageReachesPreCommit(sys, vault, "pre-commit")
	prePushOK := stageReachesPreCommit(sys, vault, "pre-push")
	if preCommitOK && prePushOK {
		rep.Pass("vault pre-commit + pre-push hooks installed (gitleaks gate active)")
		return
	}

	missing := missingHooks(preCommitOK, prePushOK)
	if !fix {
		rep.Fail("vault secret gate INACTIVE — " + missing + " not installed; run `dotf doctor --fix`")
		return
	}

	// `pre-commit install` resolves the git repo from its working directory, so it
	// must run with cwd = vault (hence CommandOutputDir, not CommandOutput).
	out, err := sys.CommandOutputDir(vault, preCommit, "install",
		"--hook-type", "pre-commit", "--hook-type", "pre-push")
	if err != nil {
		rep.Fail(fmt.Sprintf("`pre-commit install` failed in %s: %v (%s)", vault, err, firstLine(out)))
		return
	}
	rep.Fix("installed vault pre-commit + pre-push hooks in " + vault)
}

// preCommitPath resolves pre-commit the way the dispatcher does
// (git-hooks/lib/chain-local-hook.sh): PATH first, then uv's tool bin dir in
// uv's documented order (UV_TOOL_BIN_DIR, XDG_BIN_HOME, ~/.local/bin), where
// `dotf tools install` puts it. A GUI launcher's PATH has no ~/.local/bin, and
// the dispatcher still runs the gate there, so doctor must not FAIL it. The
// bare name when PATH has it, the absolute path from uv's dir otherwise, ""
// when neither has it.
func preCommitPath(sys *System) string {
	if sys.has("pre-commit") {
		return "pre-commit"
	}
	dir := sys.env("UV_TOOL_BIN_DIR", sys.env("XDG_BIN_HOME", filepath.Join(sys.home(), ".local", "bin")))
	p := filepath.Join(dir, "pre-commit")
	if sys.GOOS == "windows" {
		p += ".exe"
	}
	if isExecFile(p) {
		return p
	}
	return ""
}

// missingHooks renders the human list of absent stages for the FAIL message.
func missingHooks(preCommitOK, prePushOK bool) string {
	switch {
	case !preCommitOK && !prePushOK:
		return "pre-commit + pre-push hooks"
	case !prePushOK:
		return "pre-push hook (gitleaks)"
	default:
		return "pre-commit hook"
	}
}

// firstLine collapses command output to its first non-empty line for a terse FAIL.
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}
