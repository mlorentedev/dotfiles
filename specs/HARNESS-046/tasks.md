---
tags: [spec, tasks, templates]
created: "2026-08-25"
---

# Tasks - HARNESS-046

> One task = one focused commit. The design predates this spec — `ROSTER.md` has declared each role's phase and forced skills since v1 — so the work is authoring against a settled contract rather than discovering one.

## Setup

- [x] Branch created from main: `feat/invocable-agent-personas`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the three out-of-scope items are stated as gaps, not as unknowns

## Implementation

- [x] [AC4] Audit `ROSTER.md` against the existing `curator` definition and the skills on disk — found two drifts (`architect` under the roster's own `>=3` rule; `curator` row missing `dispose-proposals`)
- [x] [AC4] Correct both drifts in `ROSTER.md`, in the same vault commit as the definitions
- [x] [P] [AC1] Author `architect`, `planner`, `builder`, `reviewer`, `shipper` in `00_meta/agents/definitions/` against curator's established shape
- [x] [P] [AC1] [AC5] Author the `hermes-nan` autonomous catalog entry, pointing at `80_agents/hermes-nan/` and duplicating none of it
- [x] [AC1] Render with `scripts/compile-harness.sh --refresh`; never hand-write a record (they carry `generated_sha`)
- [x] [AC2] Confirm a second refresh pass is byte-identical
- [x] [AC3] Confirm `dotf doctor`'s agent-tier check reads the new records and every tier resolves
- [x] [AC4] Write `check-roster-consistency.py` so the next drift is caught by a check rather than by hand
- [x] [AC4] Confirm that guard can fail — plant a divergence, observe exit 1, restore

## Verification

- [x] All five acceptance criteria exercised and evidence recorded in `features.json`
- [x] `dotf doctor` full sweep unaffected on main: 152 passed, 0 failed
- [ ] Independent review before archive — must not be the implementing session

## Out of scope, tracked elsewhere

- [ ] Wire `dotf agent run --role X` to read the persona definitions — nothing under `cli/internal/agent` references `harness/agents/` today
- [ ] Give `tiers.top` a second entry so `architect` and `curator` have a fallback
