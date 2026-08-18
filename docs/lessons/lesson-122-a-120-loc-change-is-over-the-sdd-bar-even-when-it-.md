---
id: lesson-122-a-120-loc-change-is-over-the-sdd-bar-even-when-it-
type: lesson
status: active
created: "2026-06-23"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 122: A ~120-LOC change is over the SDD bar even when it "obviously" mirrors an existing check

**Context**: OPS-016 added a `checkVaultHooks` diagnostic to `dotf doctor` (#553) — a near-copy of the existing `checkGuardHooks`. Because the GitHub issue read like a complete mini-spec and the change mirrored an established pattern, it was built directly with no `specs/<id>/` folder.

**Problem**: `spec-gate` CI failed — 297 LOC of production diff (≥ the 50-LOC threshold) with no active spec folder touched. "The issue is the spec" is not a shortcut this repo honors: SDD Tier 4 is an automated gate, not a judgment call. Blind spot inside the blind spot: the gate counts `_test.go` files (they are not under `tests/`), so the test file inflated the diff — but even the 119 non-test LOC were over threshold.

**Solution**: Authored the spec retroactively via `dotf spec init OPS-016-… --issue 549` (work-gated on the open issue, ADR-018), filled proposal/tasks/verification + features.json with the real acceptance criteria and evidence, and archived it via `dotf spec archive` once merged. Gate green on the next push.

**Rule**: For any change ≳50 LOC of production diff, scaffold `specs/<id>/` FIRST — never lean on "the issue already explains it" or "this just mirrors an existing check". The gate enforces it mechanically and counts `_test.go` toward the threshold. Corollary on placement: when the change is *pure behaviour* (no repo asset to deploy), its home is a `dotf doctor` check, not a new bootstrap script (ADR-020 C7) — provisioning that just runs `pre-commit install` is behaviour, so it converges into the CLI checker, not shell.

---
