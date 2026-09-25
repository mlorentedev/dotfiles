---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - HARNESS-145-doctrine-triggers-ledger-agy

## Evidence

- [ ] AC1 -> S1: bats `doctrine: the committed records keep the capped agy surface under 8000 characters`
- [ ] AC2 -> S2: bats `triggers: --refresh fails naming the trigger and pattern, and writes nothing`, `TestCheckTriggerTargets_DanglingTriggerFailsAndIsNamed`
- [ ] AC3 -> S3: `TestSessionsWithNoIDShareNoLedger` (fails on the old code)
- [ ] AC4 -> S4: `TestAgyAnswersInItsOwnProtocol`, `TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry`, `TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands`
- [ ] AC5 -> operational tail: an `agy` decision record with a real conversation id

Each slice PR fills its own section below and nothing else.

## S1 evidence

_Filled by the S1 PR._

## S2 evidence

_Filled by the S2 PR._

## S3 evidence

_Filled by the S3 PR._

## S4 evidence

_Filled by the S4 PR._

## Decisions made during implementation

- The slim records are derived from the vault section (the SSOT); the original wording stays beside the dense text in the vault so nothing is lost. The cost is duplicated rules, accepted and recorded as a risk in `proposal.md`.
- A format an older binary cannot emit lives under a manifest key that binary never reads (`agents.bind_named`), because an older binary treats every target in `agents.bind` as Claude's shape.
- An empty session key means "cannot store": the call is allowed and journaled as `session-unscoped` rather than mapped to a shared bucket.
- agy answers `ask`, never `allow`: `allow` approves without asking, which would make measurement plumbing a permission grant.
- The eight dangling triggers were kept and remapped, not pruned: the rules route prompts to a persona.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`: yes, lessons 292 and 293 (S4)
- [x] ADR-worthy decision for the repo's `docs/adr/`: amendment to ADR-027 and the ADR-010 hooks row (S4)
- [ ] New pattern candidate for `00_meta/patterns/`: no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-145-doctrine-triggers-ledger-agy/` -> `specs/archive/HARNESS-145-doctrine-triggers-ledger-agy/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed
