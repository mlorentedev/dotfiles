---
id: "HARNESS-064-adversarial-review-trigger"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-10"
issue: "mlorentedev/dotfiles#879"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #879; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# HARNESS-064-adversarial-review-trigger

## Why

<!-- from issue #879: HARNESS-064: adversarial-review has no trigger, only a block — move it into the verification window -->

CLI-034 bound the adversarial-review **artifact** to `dotf spec archive`: a spec can no longer be
archived without a fresh, passing `review.md`. It did not bind the **moment**. Nothing tells an agent
to run the review while the review can still change anything, so the requirement is discovered when
the archive refuses — after the PR has merged, which is the point at which acting on a reviewer's
finding is most expensive. PR #877 proved this concretely on its own merge: no review existed during
the work, `check-spec-gate.sh` requires archive-on-merge, and the PR had to take the `skip-archive`
escape to land at all. A gate that is only discoverable by tripping over it trains people to reach
for the escape, which is how an enforcement layer decays into paperwork.

## What

An always-on trigger that puts the review in the **verification window** — after implementation,
before the PR merges — mirroring the two-surface split the `spec` skill already uses:

- `AGENTS.md` carries the short always-on trigger: when a spec's implementation is complete and a PR
  is about to be opened or merged, the agent **proposes** `/adversarial-review <feature-id>` itself,
  naming the spec and stating the evidence (that `dotf spec archive` will otherwise refuse).
- The `adversarial-review` SKILL becomes the SSOT for *how* the agent decides and *how* it phrases
  the proposal — the checks it runs, the template, and the "when NOT to propose" debounce.

Deliberately a **proposal, never an automatic action**: the review's entire value is independence,
so the human decides which session or agent supplies it. An agent that proposed and then answered
its own proposal would produce the self-review the skill explicitly forbids.

## Out of scope

- Automatic invocation of the review, or any change that makes the agent supply its own verdict.
- Any change to the CLI-034 gate itself (`checkReviewGate`, the verdict grammar, the staleness floor).
- The NaN/CI judge lane — decided (CI, not hermes) but blocked on AI-001 (`knowledge#150`).
- Retro-fitting reviews onto the ~218 already-archived specs.

## Risks / open questions

- **Noise.** A trigger that fires on every change becomes wallpaper and gets ignored, taking the
  Discipline Gate's credibility with it. Mitigation: the skill carries an explicit "when NOT to
  propose" list (no active spec, already reviewed, user declined once for this change), mirroring
  the `spec` skill's once-per-change debounce.
- **Independence theatre.** The agent proposing *and* supplying the review voids the point. The
  trigger text must state that the implementer cannot be the reviewer — the same trap #875 is being
  held open to avoid.
- **Instruction-surface caps.** `AGENTS.md` is already 25603 chars. Verified before writing: agy caps
  each rules file at 12000 and codex stops at 32 KiB, but both receive the **compact** doctrine
  payload (~2 KB), not the whole file — so growing `AGENTS.md` does not breach either.
- **Render drift.** Skills are edited in the vault SSOT (`00_meta/skills/`), never in
  `harness/skills/`, or the next `compile-harness.sh --refresh` silently reverts the edit (CLI-005).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — `AGENTS.md` carries the trigger, phrased as a **proposal** that states its evidence: the
      feature-id and the fact that `dotf spec archive` will refuse without `review.md`.
- [ ] AC2 — the trigger names the window with the literal phrase **"verification window"**, not
      "before archiving": the point is that the review lands while it can still change the outcome.
- [ ] AC3 — the trigger states that the implementing session cannot supply the review, so the
      proposal cannot be self-answered.
- [ ] AC4 — `tests/agents-md.bats` pins AC1–AC3 by content, and the pin fails against the current
      `AGENTS.md` before the change lands.
- [ ] AC5 — the `adversarial-review` SKILL carries the full Agent-Side Activation Rule (checks,
      phrasing template, when NOT to propose) and its "When to use" cross-references the trigger.
- [ ] AC6 — the skill is edited in the vault SSOT and its `harness/` render regenerated via
      `compile-harness.sh --refresh`; `--check` reports no drift.

## References

- Bitácora board: `mlorentedev/dotfiles#879` (see the `issue:` frontmatter field)
- `specs/CLI-034-spec-archive-review-gate/` — the gate this completes (#875, PR #877)
- PR #885 — the escape-path repair that made this discoverable; its body records the `skip-archive`
  pair as still unexercised, which this spec's own PR will exercise
- `00_meta/skills/spec/SKILL.md` → "Agent-Side Activation Rule" — the two-surface pattern mirrored here
- `00_meta/patterns/pattern-spec-driven-development.md`
