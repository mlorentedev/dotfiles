package doctor

import (
	"fmt"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// checkSkillRecordsAgainstVault compares each committed harness record with the
// vault source it was refreshed from. The vault is the source of record for
// skills and agents; `compile-harness.sh --refresh` copies it into the
// committed harness/{skills,agents}/<name>/ records and stamps `generated_from`
// (the vault path) and `generated_sha` (the first 16 hex of the source's
// sha256). `--check` validates the records offline and never reads the vault,
// so a vault edit nobody refreshed left main's record stale with nothing
// saying so: on 2026-10-13 msi's checkout carried the refreshed teardown skill
// as an "uncommitted change" while main was the stale side.
//
// WARN, not FAIL: either side can be the older one. A vault edit waits for a
// refresh PR, and a machine whose vault is not pulled yet sees the records
// ahead of it.
func checkSkillRecordsAgainstVault(sys *System, rep *Report) {
	repo := resolveRepoDir(sys)
	if repo == "" {
		return
	}
	// The resolver, not a literal default: a vault guessed at ~/Projects/knowledge
	// would compare against whatever happens to live there (ADR-025).
	vault := sys.vaultPath()
	if vault == "" || !isDir(vault) {
		rep.Skip("no vault resolves on this machine (VAULT_PATH, machine.json): committed harness records not compared with their source")
		return
	}

	d := harness.CompareWithVault(repo, vault)
	stale, unstamped, unread, checked := d.Stale, d.Unstamped, d.Unread, d.Checked

	if len(unread) > 0 {
		rep.Warn(fmt.Sprintf("cannot read %s, so those records were not compared with the vault", strings.Join(unread, ", ")))
	}
	if len(unstamped) > 0 {
		rep.Warn(fmt.Sprintf("%d committed harness record(s) carry no generated_from/generated_sha, so they cannot be compared with the vault: %s "+
			"(run `dotf harness refresh` in a worktree and open a PR)", len(unstamped), strings.Join(unstamped, ", ")))
	}
	if len(d.Edited) > 0 {
		rep.Warn(fmt.Sprintf("%d committed harness record(s) differ from the render of a vault source that has not changed: %s. "+
			"The next refresh reverts that edit: carry it to the vault source, then run `dotf harness refresh`", len(d.Edited), strings.Join(d.Edited, ", ")))
	}
	if len(stale) > 0 {
		rep.Warn(fmt.Sprintf("%d committed harness record(s) differ from their vault source: %s. "+
			"If the vault is the newer side, run `dotf harness refresh` in a worktree and open a PR; "+
			"if the records are, pull the vault", len(stale), strings.Join(stale, ", ")))
		return
	}
	if checked > 0 && len(d.Edited) == 0 && len(unstamped) == 0 && len(unread) == 0 {
		rep.Pass(fmt.Sprintf("%d committed records match their vault source", checked))
	}
}
