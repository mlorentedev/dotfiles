---
id: "HARNESS-061-effectiveness-probes"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#852"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-061-effectiveness-probes

## Why

<!-- from issue #852: HARNESS-061: verify behaviour, not representation — effectiveness probes, stub pairing, DR drills -->

One session surfaced four defects sharing a single shape: a check verified a *representation* of a guard rather than the guard's *behaviour*, so each read healthy while the thing it guarded was off. `git commit` had been broken machine-wide since #765 under 15 passing tests (BUG-055); the memory-sink guard was inert in the very repo that ships it while `dotf doctor` reported "all ok" (BUG-056); `gitleaks` ran three times per commit, printed on screen for hours; and the disaster-recovery runbook's step 1 had no instructions. None presented as a failure — which is precisely why no audit found them, since an audit reads representations too.

## What

`dotf doctor` answers "does this guard fire here?" instead of "is a file at this path?", and the test suite makes the BUG-055 shape fail loudly instead of passing silently.

- Hook checks resolve what git will **actually run** for a repo — effective config (local beating global), the common git dir, the executable bit — and report per repo, so a repo-local override is detected and named rather than masked by correct global wiring.
- The vault secret gate reports on whether the gate runs. It previously keyed on `.git/hooks/<stage>`, a file `pre-commit install` **refuses** to create while `core.hooksPath` is set, so it read INACTIVE on a demonstrably working gate.
- A bats suite that stubs a third-party binary must pair with a `<name>-real.bats` or carry an explicit exemption with its reason; CI fails on a new unpaired suite.
- `dotf doctor` surfaces DR escrow age and when the recovery chain was last actually executed.

## Out of scope

- Retrofitting real-dependency tests to the nine currently-exempt suites. The guard records them as a backlog; converting them is separate work.
- The prune design (#802 / #843) — same session, unrelated shape.
- Scheduling or enforcing the DR drill itself. The check surfaces staleness; performing the drill is a calendar commitment, not code, and this spec does not pretend otherwise.
- Converting `checkOrcaHook`, the third identity-shaped check. It reads a JSON field rather than a hook path, so the probe helpers do not apply unchanged.

## Risks / open questions

- **An execution probe would be stronger than resolution.** Running the hook is the only fully behavioural answer, but the vault's pre-push stage runs gitleaks (9–60s), which is unacceptable in a `doctor` run. Resolved by resolving-and-following for expensive stages and documenting the split at the call site; the guard's own pre-commit stage is cheap enough that its dispatcher membership is equivalent to "it runs".
- **`stageReachesPreCommit` accepts a dispatcher as a live gate.** This is correct only while the dispatcher's fallback works — which is exactly what BUG-055 broke. Mitigated because the fallback now has its own real-dependency test (`precommit-fallback-real.bats`), so the two guards cover each other rather than sharing an assumption.
- **The exemption table is a backlog that could become permanent.** Mitigated by a staleness test: an entry whose suite gains a real sibling, or whose suite is deleted, fails the build.

## Acceptance criteria

- [ ] A guard check cannot pass by reading global config while git resolves an effective value; a repo-local override is detected and the affected repo named, with the remedy stated.
- [ ] The vault secret gate passes when the gate reaches pre-commit through the dispatcher fallback with no `.git/hooks/<stage>` present, and does not pass for a dispatcher with no config to act on.
- [ ] A non-executable hook resolves as absent, so the probe does not reintroduce the file-exists question it replaced.
- [ ] A new bats suite stubbing a third-party binary fails until it pairs with a real-dependency test or is explicitly exempted.
- [ ] The exemption list cannot go stale silently.
- [ ] `dotf doctor` reports DR escrow age and drill recency, warning (never failing) when unproven — nothing is broken, it is untested.
- [ ] Every new check has a red-direction test that fails when the guarded thing is broken.

## References

- Bitácora board: `mlorentedev/dotfiles#852`
- Precedent in-tree: `System.AgeRoundTrip` — proves a key decrypts rather than that a file exists
- Prior rulings of the same principle: BUG-040 (path identity vs. effectiveness), #839 (BUG-056), #806 (the vault gate's false FAIL)
- Related patterns: `00_meta/patterns/pattern-verify-state-before-acting.md` — "an executable on PATH is usable → **run it**"
