---
tags: [spec, tasks, templates]
created: "2026-08-10"
---

# Tasks - HARNESS-064-adversarial-review-trigger

> TDD order. One task = one focused commit. Tick as you go.
>
> **Inline markers**: `[P]` = no dependency on another unchecked task. `[AC<n>]` = satisfies
> acceptance criterion #`<n>` from `proposal.md`.

## Setup

- [x] Branch created from main: `feat/adversarial-review-trigger`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the instruction-surface cap
      was the only one and it was resolved before writing (agy/codex receive the compact payload)

## Implementation

> The bats pin comes first and must be **proved red** against the current `AGENTS.md`: this whole
> spec exists because a rule that nothing checks is a rule that does not fire, so a pin that was
> never seen failing would repeat the defect at one remove.

- [x] [AC4] Write the `agents-md.bats` pin for the trigger (evidence phrasing, the literal
      "verification window", the implementer-cannot-review clause) and **capture it failing**
- [x] [AC1] [AC2] [AC3] Add the proactive trigger paragraph to `AGENTS.md` → pin goes green
- [x] [P] [AC5] Add the Agent-Side Activation Rule to the vault `adversarial-review` SKILL (checks
      the agent runs · how to phrase the proposal · when NOT to propose) and cross-reference it from
      "When to use"
- [x] [AC6] `compile-harness.sh --refresh`, staging **only** `harness/skills/adversarial-review/`;
      confirm `--check` reports no drift
- [x] Emit `features.json` mapping every acceptance criterion to an executable verification command

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous command
- [x] Lint passes (`shellcheck`, `check-bats-names.sh`)
- [x] No unrelated changes in the diff — in particular, no other skill's render swept in by `--refresh`
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder, with `skip-archive` + a rationale section (the spec
      stays active until an independent session reviews it — the first live exercise of that pair)

## Machine-readable features

Emitted as `features.json` alongside this file, following [[pattern-feature-list-as-primitive]].

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running
`verification` and capturing exit code 0, may set that terminal state.
