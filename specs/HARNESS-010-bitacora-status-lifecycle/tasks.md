---
tags: [spec, tasks]
created: "2026-06-07"
---

# Tasks - HARNESS-010-bitacora-status-lifecycle

> One task = one focused change. Tick as you go.

## Setup

- [x] Branch created from `origin/main`: `harness-010-bitacora-status-lifecycle` (worktree)
- [x] `proposal.md` complete; acceptance criteria testable; no open questions left

## Implementation

- [ ] AC2 — add `.github/workflows/bitacora-status.yml` (`issues:[assigned]` → `Status=In Progress`, `state==open` guard, idempotent `item-add` + `item-edit`)
- [ ] AC1 — codify the status-lifecycle rule in `AGENTS.md` (Standing Order + Neural Hive Loop Phase 1/2/3 touchpoints)
- [ ] AC4 — update vault runbook `00_meta/runbooks/bitacora-project-setup.md` §4/§5/§7 (register workflow in per-repo bundle; mark In Progress automated-on-assign)
- [ ] AC5 — wire board-status reconciliation into vault `00_meta/skills/handoff/SKILL.md`
- [ ] AC3 — confirm workflow YAML parses

## Closing

- [ ] Every acceptance criterion covered by a `features.json` entry with a non-vacuous verification command
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder (`Closes #270`)

## Machine-readable features

See sibling `features.json`. **Pass-state gating:** only the harness may set `"state": "passing"`, after running `verification` with exit 0 and capturing `evidence`. The vault `/handoff` step (AC5) and the live `assigned → In Progress` behaviour are verified by observation (recorded in `verification.md`), since they cannot run in this repo's CI.
