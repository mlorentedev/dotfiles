package doctor

import (
	"path/filepath"

	"github.com/mlorentedev/dotfiles/cli/internal/memlink"
)

// checkAutoMemoryLink verifies the Claude auto-memory directory is a link
// (junction on Windows, symlink on POSIX) into the knowledge vault, so /handoff
// lands the session continuity block in the single sink — the vault — and not in
// ~/.claude or the code repo. That link is otherwise created only by the silent
// session-start hook; on a machine where it is missing, /handoff orphans the
// block (knowledge#120, observed 2026-06-19). Surfacing + repairing it from the
// one command users actually run for setup drift is the #551 acceptance.
//
// Mirrors checkVaultHooks: verify always, repair only under --fix, idempotent.
// Crucially it shares memlink with the session-start adapter, so doctor and the
// hook compute an identical target on every OS — and, like memlink.Ensure, it
// never overwrites a real data directory (the divergence case is a WARN to
// reconcile by hand, not a destructive fix).
func checkAutoMemoryLink(sys *System, start string, rep *Report, fix bool) {
	rep.Section("Auto-memory vault link")

	// Same ADR-025 seam + legacy default as checkVault / checkVaultHooks, so a
	// path-default fix is inherited here too.
	vault := sys.env("VAULT_PATH", filepath.Join(sys.home(), "Projects", "knowledge"))
	target := memlink.ClaudeMemoryTarget(sys.home(), start)

	switch memlink.Status(start, target, "", vault) {
	case memlink.StateLinked:
		rep.Pass("auto-memory linked to the vault: " + target)

	case memlink.StateNoSource:
		// No vault memory source for this project — a valid state (not every repo
		// has a vault project), so SKIP, not FAIL.
		rep.Skip("no vault memory source for this project — nothing to link")

	case memlink.StateRealDir:
		// The agent's own data lives here, diverged from the vault. Repairing would
		// mean destroying it, which memlink refuses by contract — so do NOT --fix;
		// flag it for a manual reconcile (knowledge#120).
		rep.Warn("auto-memory is a real directory, not a vault link: " + target +
			" — /handoff writes here, not the vault. Reconcile by hand (merge into the vault source, then relink); --fix will not overwrite real data.")

	case memlink.StateRepairable:
		if !fix {
			rep.Fail("auto-memory not linked to the vault — run `dotf doctor --fix` so /handoff reaches the vault")
			return
		}
		if msg, _ := memlink.Ensure(start, target, "", vault); msg != "" {
			rep.Fix("linked auto-memory to the vault: " + target)
		} else {
			rep.Fail("could not create the auto-memory link at " + target + " (check permissions)")
		}
	}
}
