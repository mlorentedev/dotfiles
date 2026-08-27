package doctor

import (
	"fmt"
	"os"
	"path/filepath"
)

// piQuarantineDir is where --fix moves a shadowing link, and it is deliberately
// OUTSIDE ~/.pi/agent/extensions/.
//
// The obvious place is a .disabled/ subdirectory beside the link. It is wrong,
// and wrong in this spec's signature way. pi's own docs state the two
// auto-discovery patterns (docs/extensions.md, measured against 0.84.2):
//
//	~/.pi/agent/extensions/*.ts        global
//	~/.pi/agent/extensions/*/index.ts  global, subdirectory form
//
// A .disabled/ directory holding an index.ts matches the second pattern. Whether
// pi's glob skips a leading dot is not documented either way, so the safe
// reading is that quarantining there is a candidate extension named ".disabled"
// carrying the same file — the same tool, the same conflict, one directory
// deeper. A repair that relocates a defect inside the surface that reads it
// reports success and changes nothing. Moving it out of the scanned tree
// removes the question rather than betting on the answer.
const piQuarantineDir = ".disabled-extensions"

// repairPiShadows quarantines the links checkPiExtensions found. It runs ONLY
// under --fix, so the user has opted in; the check itself never writes.
//
// WHAT MOVES AND WHAT DOES NOT. Only the link under ~/.pi is in scope. Its
// target lives inside pi's own npm package —
// .../pi-coding-agent/examples/extensions/subagent/index.ts — and is never
// touched, which is why quarantine loses nothing: the example code this link
// pointed at survives in the package, reinstalled on every pi upgrade.
//
// Quarantine over deletion because these links were a deliberate choice on
// 2026-08-09. They are unreproducible and they stop pi starting, so they must
// stop being loaded; but the repo has no standing to decide the user no longer
// wants them, only that pi cannot keep both. Renaming says that and stays
// reversible; os.Remove says more than doctor knows.
//
// The relative shape is preserved (extensions/subagent/index.ts becomes
// .disabled-extensions/subagent/index.ts) so restoring one is a mv with the two
// paths visible, not an archaeology exercise.
func repairPiShadows(sys *System, rep *Report, shadows []piShadow) {
	moved := 0
	for _, s := range shadows {
		if !s.collides {
			// An unconfirmed hand-wired link is unreproducible but is not
			// currently breaking anything. --fix repairs the outage it can name;
			// it does not tidy the directory on the user's behalf.
			continue
		}

		extRoot := filepath.Join(sys.home(), ".pi", "agent", "extensions")
		rel, err := filepath.Rel(extRoot, s.link)
		if err != nil {
			rep.Warn(fmt.Sprintf("cannot place %s under quarantine: %v", short(sys, s.link), err))
			continue
		}
		dst := filepath.Join(sys.home(), ".pi", "agent", piQuarantineDir, rel)

		// Never clobber an earlier quarantine. Two runs that both "succeed"
		// while the second silently destroys the first is the failure mode this
		// whole check exists to catch.
		if _, err := os.Lstat(dst); err == nil {
			rep.Warn(fmt.Sprintf("%s already quarantined at %s — leaving %s in place, resolve by hand",
				rel, short(sys, dst), short(sys, s.link)))
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			rep.Warn(fmt.Sprintf("cannot create %s: %v", short(sys, filepath.Dir(dst)), err))
			continue
		}
		if err := os.Rename(s.link, dst); err != nil {
			rep.Warn(fmt.Sprintf("cannot quarantine %s: %v", short(sys, s.link), err))
			continue
		}
		rep.Fix(fmt.Sprintf("quarantined %s -> %s (restore with mv, the link target was not touched)",
			short(sys, s.link), short(sys, dst)))
		moved++
	}

	if moved > 0 {
		rep.Info("re-check that pi now starts: pi -p 'ok'")
	}
}
