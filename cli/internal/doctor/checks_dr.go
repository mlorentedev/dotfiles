package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
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

func checkDisasterRecovery(sys *System, cfg *Config, rep *Report) {
	rep.Section("Disaster recovery")

	// The escrow. Absent is a SKIP, not a FAIL: #586 item 5 has not shipped, so
	// its absence is a known state rather than a regression. It becomes a live
	// freshness check the moment `dotf secrets backup` produces one.
	escrow := filepath.Join(cfg.DotfilesDir, "sensitive", "dr", "bitwarden-export.age")
	switch info, err := os.Stat(escrow); {
	case err != nil:
		rep.Skip("no DR escrow at " + escrow + " — `dotf secrets backup` has not run (#586 item 5)")
	case sys.Now().Sub(info.ModTime()) > escrowMaxAge:
		rep.Warn(fmt.Sprintf("DR escrow is %d days old — re-run `dotf secrets backup`; secrets added since are not in it",
			int(sys.Now().Sub(info.ModTime()).Hours()/24)))
	default:
		rep.Pass("DR escrow present and fresh")
	}

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
