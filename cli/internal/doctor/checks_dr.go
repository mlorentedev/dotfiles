package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
		// The escrow permanently needs a bw CLI session; since #1008 the command
		// acquires it itself with one prompt, so the remedy is the bare command.
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
				"`dotf secrets backup` from a terminal (it prompts for the master password: "+
				"the export path has no bw serve endpoint, so it unlocks the bw CLI for that "+
				"one run; elsewhere, prefix BW_SESSION=\"$(bw unlock --raw)\" from a shell that has one)",
			escrow, live))
		return
	}
	// Nothing resolves through Bitwarden yet: every secret still has its age
	// copy on disk, so the escrow is a convenience and its absence costs
	// nothing. Staying quiet here is what keeps the FAIL above credible.
	rep.Skip("no DR escrow at " + escrow + " — no secret resolves through Bitwarden yet, so nothing depends on it")
}

// escrowPath locates the DR escrow the same way its writer places it: in the
// CHECKOUT (`dotf secrets backup` writes env.RepoSensitiveDir + "dr", because
// the escrow is version-controlled state — #635 durability class), falling
// back to the deploy mirror only when no checkout is found. The mirror is
// copy-only and by construction never fresher than the checkout; reading it
// first only ever agreed on Linux because setup-linux.sh happens to copy
// sensitive/ recursively. setup-windows.ps1 did not, and doctor there reported
// total-loss risk over an escrow sitting in the checkout (WIN-011/#1292).
func escrowPath(sys *System, cfg *Config) string {
	const rel = "bitwarden-export.age"
	if repo := resolveRepoDir(sys); repo != "" {
		if p := filepath.Join(repo, "sensitive", "dr", rel); pathExists(p) {
			return p
		}
	}
	return filepath.Join(cfg.DotfilesDir, "sensitive", "dr", rel)
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

	escrow := escrowPath(sys, cfg)
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
		checkEscrowDescribesVault(sys, filepath.Dir(escrow), rep)
	default:
		rep.Pass("DR escrow present and fresh")
		checkEscrowDescribesVault(sys, filepath.Dir(escrow), rep)
	}

	// The drill. This is the one that matters, and the one no other check can
	// stand in for: an escrow that exists proves a file was written, never that
	// anyone can restore from it.
	checkDrillRecord(sys, filepath.Dir(escrow), rep)
}

// checkDrillRecord reads the record `dotf secrets usb verify` commits beside the
// escrow (#1770). It replaced a hand-touched marker in the deploy dir, which
// recorded that someone ran `touch`, on one machine only: the record is written
// only by a verification that passed, and travels with the checkout, so every
// machine reads the same answer. It is checked against the manifest as well as
// the clock, because a stick verified yesterday against last month's escrow
// restores last month's vault.
func checkDrillRecord(sys *System, escrowDir string, rep *Report) {
	const remedy = "mount the offline stick and run `dotf secrets usb backup --mount <dir>`, then commit " +
		"sensitive/dr/" + secrets.DrillRecordFileName
	rec, err := secrets.ReadDrillRecord(escrowDir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		rep.Warn("no recovery drill recorded — " + remedy)
		return
	case err != nil:
		rep.Warn(fmt.Sprintf("cannot read the drill record (%v) — %s", err, remedy))
		return
	}
	at, err := time.Parse(time.RFC3339, rec.VerifiedAt)
	if err != nil {
		rep.Warn(fmt.Sprintf("the drill record's verified_at %q is not a time — %s", rec.VerifiedAt, remedy))
		return
	}
	days := int(sys.Now().Sub(at).Hours() / 24)
	if sys.Now().Sub(at) > drillMaxAge {
		rep.Warn(fmt.Sprintf("the offline copy was last verified %d days ago — %s; an untested backup is a supposition", days, remedy))
		return
	}
	if stored, err := readEscrowManifest(escrowDir); err == nil && stored.Digest != rec.EscrowDigest {
		rep.Warn("the offline copy was verified against an older escrow, so it would restore an older vault — " + remedy)
		return
	}
	rep.Pass(fmt.Sprintf("offline copy verified %d days ago on %s (%d items)", days, rec.Host, rec.Items))
}

func readEscrowManifest(dir string) (secrets.EscrowManifest, error) {
	var m secrets.EscrowManifest
	blob, err := os.ReadFile(filepath.Join(dir, secrets.ManifestFileName))
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(blob, &m)
}

// checkEscrowDescribesVault compares what the escrow contained with what the vault
// holds now. Three outcomes, and the two SKIPs are as load-bearing as the compare:
//
//   - No manifest: every escrow written before this feature is in that state,
//     including the one on this machine until `backup` next runs. FAILing there
//     would turn doctor red everywhere on merge — the deploy-skew shape measured
//     twice this week (#992) — so it SKIPs and names the remedy.
//   - No session, or no sync: the daemon is locked or absent, or it could not pull
//     the vault first. An unchecked escrow reported as fresh is this check's own
//     defect arriving through the check itself, so it SKIPs with the reason and
//     never passes.
//   - Compared: silent when the digests agree, and Warn — not Fail — when they do
//     not. A stale escrow is expected after any mutation and is remediable by one
//     command; a section that goes red after every `rotate` until someone re-runs
//     backup is one people learn to scroll past, which is the failure this whole
//     area exists to prevent.
func checkEscrowDescribesVault(sys *System, escrowDir string, rep *Report) {
	manifestPath := filepath.Join(escrowDir, secrets.ManifestFileName)
	blob, err := os.ReadFile(manifestPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		rep.Skip("escrow has no manifest, so drift against the vault is unknown — " +
			"re-run `dotf secrets backup` to mint one")
		return
	case err != nil:
		// Absence and unreadability are different facts, and the escrow check
		// twenty lines above says so about itself: "a stat error is NOT proof of
		// absence". Reporting a permission error as "no manifest yet" would send
		// the reader to run backup, which will not fix a permission error.
		rep.Warn(fmt.Sprintf("cannot read the escrow manifest at %s (%v) — drift against the vault was not checked",
			manifestPath, err))
		return
	}
	var stored secrets.EscrowManifest
	if err := json.Unmarshal(blob, &stored); err != nil {
		rep.Warn(fmt.Sprintf("escrow manifest is unreadable (%v) — re-run `dotf secrets backup`", err))
		return
	}
	if stored.Digest == "" || stored.Count == 0 {
		rep.Warn("escrow manifest carries no digest — re-run `dotf secrets backup`")
		return
	}
	if sys.BWItemRevisions == nil || sys.BWSync == nil {
		rep.Skip("no synced vault listing available, so escrow drift was not checked")
		return
	}
	// bw serve lists from its own cache, so an unsynced listing can equal an old
	// escrow after the vault has moved on: a false pass, in the direction that
	// loses data (#1820). A failed sync leaves a cache of unknown age, and that is
	// refused rather than compared.
	if err := sys.BWSync(); err != nil {
		rep.Skip(fmt.Sprintf("could not sync the vault, so escrow drift was not checked: %v", err))
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
