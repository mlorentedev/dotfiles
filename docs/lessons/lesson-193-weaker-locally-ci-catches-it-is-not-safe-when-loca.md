---
id: lesson-193-weaker-locally-ci-catches-it-is-not-safe-when-loca
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 193: "Weaker locally, CI catches it" is not safe when local and CI share the same script

**Context**: BUG-061, fixing a spec-gate false negative where a PR that correctly archived its spec in the same change could not be pushed locally — `check-spec-gate.sh`'s archive-on-merge credit only fires when `SDD_PR_BODY` names a closing keyword, and that variable is empty on every local pre-push run by design. The filed issue offered three fix options; option 2 read "credit an archive move unconditionally in the pre-push tier... weaker, but the #397 protection against a gratuitous archive-move dodging the gate still holds in CI."

**Problem**: option 2 was implemented as stated — locally (empty `SDD_PR_BODY`), any spec that genuinely transitioned from active-at-base to archived-at-head was credited toward the LOC-threshold's "spec folder touched" requirement, without needing a closing-issue link. Two existing regression tests (`tests/check-spec-gate.bats`, the #397 pair) immediately went red: a PR bundling 60 LOC of unrelated production code with a genuine archive of an unrelated spec now passed locally, exactly the "gratuitous archive-move dodging the gate" #397 already closed. The flaw in the reasoning: "CI catches it" only holds when local and CI run *different* logic. Here they run the *same* `check-spec-gate.sh`, gated only by whether `SDD_PR_BODY` happens to be set — so a check made unconditionally permissive "when there's no PR body" is exactly as permissive in CI whenever CI itself has no PR body to supply (or, more subtly, it normalizes local runs to a genuinely weaker invariant than the one the script's own tests pin, which is a contradiction the tests exist to catch). A diff-only heuristic cannot distinguish #854's legitimate author (archiving the spec they are actually implementing) from #397's attacker (archiving something unrelated to bulk-legitimize other code) — their diffs are structurally identical, and no amount of "was this transition genuine" narrowing closes that gap without the PR-body linkage the local run doesn't have.

**Solution**: discard the local-credit approach entirely and implement option 1 instead — `scripts/spec-gate-prepush.sh`, a wrapper mirroring the existing CI adapter (`spec-gate-pr.sh`) that resolves the current branch's live PR via `gh pr view` (no token needed locally; the developer's own `gh auth`) and forwards real `SDD_LABELS`/`SDD_PR_BODY`/`SDD_PR_AUTHOR` to the unmodified gate. Unlike its CI sibling, it falls THROUGH to running the gate with no PR context on any resolution failure (no `gh`, no `jq`, unauthenticated, no PR open yet) rather than failing closed — "no PR context" is the ordinary local baseline here, not a stale-data hazard to guard against. `check-spec-gate.sh` itself needed no change beyond a message pointing at the wrapper and the manual `SDD_PR_BODY=...` override.

**Rule**: when a fix option is phrased as "loosen X here, Y elsewhere still catches it," check first whether "elsewhere" is actually a different code path or just the same code path under different environment variables — if a script is shared between an advisory local tier and an authoritative CI tier, any relaxation gated only on which env vars happen to be set is live everywhere those vars happen to be unset, not only in the tier the fix was aimed at. The repo's own existing regression tests caught this on the first test run after implementing the "obvious" reading of the option — proof that re-running the FULL adjacent test suite (not just new tests for the change at hand) before trusting an implementation is itself the guard.

**Tags**: `testing`, `shell`, `sdd`, `ci`
