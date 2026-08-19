package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// Disaster-recovery readiness.
//
// A runbook nobody has executed is a hypothesis with formatting. #848 exists
// because someone finally asked to *run* the recovery chain rather than read it,
// and step 1 — "restore the age key from its offline backup", the step every
// later step depends on — turned out to have no instructions at all. Nothing in
// CI could have found that: the runbook was present, well-written, and wrong by
// omission.
//
// So this check does not verify the runbook. It surfaces the two facts that
// decay silently between drills: whether a DR escrow exists and how old it is,
// and when the recovery chain was last actually executed.

// drillMaxAge is how long a recovery drill stays meaningful. Deliberately
// generous — the point is to catch "never" and "years ago", not to nag.
const drillMaxAge = 180 * 24 * time.Hour

// escrowMaxAge is tighter: an escrow older than this predates enough secret
// churn that restoring from it would silently lose credentials.
const escrowMaxAge = 90 * 24 * time.Hour

// drillMarkerPath is touched by whoever completes a recovery drill. Kept in the
// deploy dir rather than the repo so it records what happened on THIS machine —
// a drill is a property of a box and its USB, not of the source tree.
func drillMarkerPath(cfg *Config) string {
	return filepath.Join(cfg.DotfilesDir, ".dr-drill")
}

// reportAbsentEscrow renders the missing-escrow state at the severity its real
// exposure earns: `live` is how many registry entries resolve through Bitwarden,
// and regErr is non-nil when the registry could not be read to find out.
//
// Split out from checkDisasterRecovery so the severity rule — the whole point of
// #997 — is testable on its own, the same way checkBWSyncAge is split out of the
// reach check.
//
// No rep.Fix here, deliberately: this check performs no repair, and the
// neighbouring reach check documents what happens when a remediation string is
// emitted as a Fix — a read-only run reports "Applied 1 fix action(s)" for a
// repair that never happened. The command belongs in the message.
func reportAbsentEscrow(rep *Report, escrow string, live int, regErr error) {
	if regErr != nil {
		// Exposure is UNKNOWN, and the two wrong answers are both tempting.
		//
		// Silence (the reach check's "degrade to advisory, treat as unexposed")
		// would emit a SKIP reading "nothing depends on it" — precisely the
		// sentence #997 exists to delete, now printed on a machine where it was
		// never checked. FAIL would go red from any healthy machine: env.RepoDir
		// falls back to a cwd walk-up for .git, so `dotf doctor` from inside any
		// other git repo lands here, and a doctor that is red where nothing is
		// wrong is one nobody reads.
		//
		// So: audible, but honest about what it does not know.
		rep.Warn(fmt.Sprintf(
			"no DR escrow at %s, and the registry could not be read (%v) — cannot tell whether any secret depends on it",
			escrow, regErr))
		return
	}
	if live > 0 {
		// The count carries the argument. "No backup" is abstract; "28 secrets
		// exist only on a remote server you do not control" is what makes an
		// operator act.
		//
		// The BW_SESSION form is named plainly, because for `backup` it is not a
		// workaround — it is the invocation, permanently.
		//
		// Everything else in the write path is moving to the bw serve daemon
		// (#993), and it was tempting to phrase this as "broken until that
		// lands". It is not: `bw serve`'s router exposes /status, /sync, /unlock,
		// /lock, /generate, /list/object/:object, /object/:object[/:id],
		// /attachment, /move, /send/*, /device-approval/* — and no export route
		// at all. The only "/export" in the shipped bundle is an ORGANIZATION
		// export against the cloud API, not a personal-vault serve route. So
		// `bw export` has no daemon path to migrate to, and this command will
		// need a CLI session for as long as that remains true. Citing #993 here
		// would have made the message wrong the day #993 merged.
		rep.Fail(fmt.Sprintf(
			"no DR escrow at %s — %d secret(s) resolve through Bitwarden and have no local copy, "+
				"so the remote account is their only copy; create one with "+
				"`BW_SESSION=\"$(bw unlock --raw)\" dotf secrets backup` "+
				"(the export path has no bw serve endpoint, so it needs a CLI session)",
			escrow, live))
		return
	}
	// Nothing resolves through Bitwarden yet: every secret still has its age
	// copy on disk, so the escrow is a convenience and its absence costs
	// nothing. Staying quiet here is what keeps the FAIL above credible.
	rep.Skip("no DR escrow at " + escrow + " — no secret resolves through Bitwarden yet, so nothing depends on it")
}

func checkDisasterRecovery(sys *System, cfg *Config, rep *Report) {
	rep.Section("Disaster recovery")

	// The escrow. SEVERITY IS KEYED TO REAL EXPOSURE, not to a flat policy —
	// the same rule checkBitwardenReach already applies in this package, and for
	// the same reason.
	//
	// The old behaviour was a flat SKIP, justified by "#586 item 5 has not
	// shipped, so its absence is a known state". That justification expired
	// without anyone noticing (#997). While every registry entry carried an
	// `age:` pointer, an absent escrow really did cost nothing: each secret had
	// a local encrypted copy. Then `migrate` began dropping that pointer as it
	// flipped entries to Bitwarden (#971), and for those secrets the remote
	// account became the ONLY copy in existence. An absent escrow stopped being
	// an inconvenience and became a single point of total loss — still reported
	// as "nothing to check here".
	//
	// So: SKIP while nothing depends on it, FAIL from the first bw-backed entry
	// onward. This is the class named by #972 and #898 — a check whose only
	// reachable branch is the no-op one proves nothing, and a SKIP is
	// indistinguishable from "no problem here".
	live, regErr := sys.BWBackedSecrets()

	escrow := filepath.Join(cfg.DotfilesDir, "sensitive", "dr", "bitwarden-export.age")
	switch info, err := os.Stat(escrow); {
	case os.IsNotExist(err):
		reportAbsentEscrow(rep, escrow, live, regErr)
	case err != nil:
		// A stat error is NOT proof of absence, and the distinction only started
		// to matter with this change: as a SKIP, treating "cannot read" as
		// "not there" was harmless, but as a FAIL it asserts something the check
		// never established — "these 28 secrets have no local copy" — when the
		// escrow may be sitting right there behind a permission or I/O error.
		// Claiming an unverified fact is the exact failure this whole check
		// exists to stop, so it must not be committed in the fix for it.
		rep.Warn(fmt.Sprintf(
			"cannot inspect DR escrow at %s (%v) — neither its presence nor its freshness was established",
			escrow, err))
	case sys.Now().Sub(info.ModTime()) > escrowMaxAge:
		rep.Warn(fmt.Sprintf("DR escrow is %d days old — re-run `dotf secrets backup`; secrets added since are not in it",
			int(sys.Now().Sub(info.ModTime()).Hours()/24)))
	default:
		rep.Pass("DR escrow present and fresh")
	}

	// Age is a cheaper question than the one that matters. Measured on this machine
	// 2026-08-19: the escrow was four days old — comfortably inside escrowMaxAge and
	// reported "fresh" by the branch above — while the live vault already held one
	// item it did not. Worse, a DELETION is invisible to any age comparison: the
	// deleted item simply stops existing, and every survivor can predate the escrow.
	//
	// So this asks the other question, from a manifest written beside the escrow at
	// backup time (#1077).
	checkEscrowDescribesVault(sys, filepath.Dir(escrow), rep)

	// The drill. This is the one that matters, and the one no other check can
	// stand in for: an escrow that exists proves a file was written, never that
	// anyone can restore from it.
	marker := drillMarkerPath(cfg)
	switch info, err := os.Stat(marker); {
	case err != nil:
		rep.Warn("no recovery drill recorded — run the chain in docs/runbooks/guide-secrets-governance.md " +
			"§ RECOVER against the real offline backup, then `touch " + marker + "`")
	case sys.Now().Sub(info.ModTime()) > drillMaxAge:
		rep.Warn(fmt.Sprintf("last recovery drill was %d days ago — re-run it; an untested backup is a supposition",
			int(sys.Now().Sub(info.ModTime()).Hours()/24)))
	default:
		rep.Pass(fmt.Sprintf("recovery drill run %d days ago",
			int(sys.Now().Sub(info.ModTime()).Hours()/24)))
	}
}

// checkEscrowDescribesVault compares what the escrow contained with what the vault
// holds now. Three outcomes, and the two SKIPs are as load-bearing as the compare:
//
//   - No manifest: every escrow written before this feature is in that state,
//     including the one on this machine until `backup` next runs. FAILing there
//     would turn doctor red everywhere on merge — the deploy-skew shape measured
//     twice this week (#992) — so it SKIPs and names the remedy.
//   - No session: the daemon is locked or absent. An unchecked escrow reported as
//     fresh is this check's own defect arriving through the check itself, so it
//     SKIPs with the reason and never passes.
//   - Compared: silent when the digests agree, and Warn — not Fail — when they do
//     not. A stale escrow is expected after any mutation and is remediable by one
//     command; a section that goes red after every `rotate` until someone re-runs
//     backup is one people learn to scroll past, which is the failure this whole
//     area exists to prevent.
func checkEscrowDescribesVault(sys *System, escrowDir string, rep *Report) {
	blob, err := os.ReadFile(filepath.Join(escrowDir, secrets.ManifestFileName))
	if err != nil {
		rep.Skip("escrow has no manifest, so drift against the vault is unknown — " +
			"re-run `dotf secrets backup` to mint one")
		return
	}
	var stored secrets.EscrowManifest
	if err := json.Unmarshal(blob, &stored); err != nil {
		rep.Warn(fmt.Sprintf("escrow manifest is unreadable (%v) — re-run `dotf secrets backup`", err))
		return
	}
	if sys.BWItemRevisions == nil {
		rep.Skip("no vault listing available, so escrow drift was not checked")
		return
	}
	items, err := sys.BWItemRevisions()
	if err != nil {
		rep.Skip(fmt.Sprintf("could not list the vault, so escrow drift was not checked: %v", err))
		return
	}
	live, err := secrets.ManifestFromItems(items)
	if err != nil {
		rep.Skip(fmt.Sprintf("could not describe the live vault, so escrow drift was not checked: %v", err))
		return
	}
	if diff := stored.Differs(live); diff != "" {
		rep.Warn("DR escrow no longer describes the vault: " + diff + " Re-run `dotf secrets backup`.")
		return
	}
	rep.Pass(fmt.Sprintf("DR escrow still describes the vault (%d items)", stored.Count))
}
