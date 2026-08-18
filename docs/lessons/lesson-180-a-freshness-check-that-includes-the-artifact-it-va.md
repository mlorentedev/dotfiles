---
id: lesson-180-a-freshness-check-that-includes-the-artifact-it-va
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 180: A freshness check that includes the artifact it validates is stale by construction

**Context**: CLI-034 gated `dotf spec archive` on an adversarial-review verdict in `specs/<id>/review.md`. A presence check alone is satisfiable by an empty or outdated file — the same alibi `check-spec-gate.sh` added `SPEC_FLOOR=10` to defeat — so the design added a staleness floor: `review.md` records the `reviewed_sha` it examined, and the archive refuses if the spec changed after it. The first formulation was the obvious one: "did anything in `specs/<id>/` change after `reviewed_sha`?"

**Problem**: That predicate is *always true*. `review.md` is itself written after the commit it reviews, so its own commit trips the check — every review would be born stale and the gate would refuse every archive, forever. A second file failed the same way from the other end: `verification.md` carries the archive checklist, which is ticked *at archive time*, so it changes during the very operation being gated. Both were caught by writing the guard tests before trusting the predicate; neither is visible from reading the rule, because the rule sounds right.

**Solution**: Scope the check to the **contract files** the verdict is actually about — `proposal.md`, `tasks.md`, `features.json` — and exclude both the artifact itself and anything written at gate time, with the reason recorded in the code where the list is declared. Two dedicated tests pin the exclusions against a real git history (`TestGitStalenessIgnoresReviewOwnCommit`, `TestGitStalenessIgnoresVerificationMd`), not merely against an injected fake, since a fake would have happily confirmed the broken predicate.

**Rule**: When a check asks "has anything changed since X", enumerate what it may look at — never `*`. The artifact recording the answer, and anything the gated operation itself writes, must be outside the scanned set or the check refutes itself. Write the two negative tests first: "the artifact's own commit does not trip it" is the one that fails on the naive implementation, and it is the one that never appears on a checklist derived from the requirement.

**Sibling**: the same PR surfaced the composition case. `check-spec-gate.sh` requires archive-on-merge, and this new gate requires an independent review before an archive — so the PR *introducing* the second requirement cannot satisfy both, there being no prior producer of the artifact. Two independently correct gates can compose into an unsatisfiable state at the boundary where one is introduced; the bootstrap step needs a declared escape (here `skip-archive` + a written rationale), not a silent bypass.
