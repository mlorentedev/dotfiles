---
id: lesson-054-safety-net-fixes-must-be-audited-against-the-same-
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 054: Safety-net fixes must be audited against the same bug-class they paper over

**Context:** BUG-022 (PR #87, 2026-05-21) was a fix for BUG-015's hook-resolution probe. The probe originally re-executed the same `break; }; done` EPIPE-race pattern as the upstream claude-mem hooks. BUG-022 appended `head -n1` after the while-loop to make the pipeline race-free. The same day, hours later, BUG-023 surfaced: the BUG-022 fix STILL raced under `set -euo pipefail` when 2+ candidates matched in cache (this user had 6 versions: 12.7.4 -> 13.3.0). `head -n1` closed the consumer; leftover printfs in the while loop got EPIPE; pipefail propagated 141; `set -e` killed healthcheck.sh mid-section 4/12. setup-linux.sh logged a false-positive WARNING. The fix was a half-fix for the same bug-class.
**Problem:** When a safety-net fix patches a bug-class (race condition, escape bug, error-suppression pattern), it's tempting to believe the patch fully closes the class. But a partial patch can leave a structurally identical sub-case open -- exactly the same bug-class, just in a different scenario count (0-1 matches vs 2+ matches; single producer vs multiple producers; single-arg vs varargs). The next time someone reads the code, the fix looks defensive and complete, masking the residual vulnerability. Cascade-cost lesson applies recursively: every safety-net iteration is itself a candidate for the same audit.
**Solution:** When shipping a fix for a bug-class, before merging, ask: 'could the same bug-class symptom still fire in a different cardinality, a different producer/consumer count, or a different shell mode (pipefail on/off, set -e on/off, strict-mode on/off)?' If the answer is maybe, explicitly enumerate the cases and add a bats parity assert with BOTH positive (new pattern present) AND negative (old broken pattern absent) for each cardinality. Concrete pattern for pipe-with-early-close races: materialize candidates into a variable first, then iterate in pure bash with `break` -- no pipe at all means no consumer-close, no EPIPE, no pipefail propagation. Avoid `done | head -n1` form entirely; it is a half-fix dressed up as a full one. Companion to the 'detection probe must use the race-free pattern' lesson -- applies it recursively to the fix itself.
**Tags:** `#safety-net` `#audit-discipline` `#race-condition` `#bash` `#pipefail` `#cascade-cost` `#claude-mem`
