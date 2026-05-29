---
tags: [spec, tasks, harness-001, epic]
created: "2026-05-28"
---

# Tasks - HARNESS-001-unified-cross-agent-harness

> **Epic coordination, not a single PR's TDD list.** This umbrella ships as a *sequence* of atomic PRs. Each implementation PR has its own `specs/<id>/` with a real TDD `tasks.md`. The order below is the dependency graph; freeze it once PR-1 starts `implementing`.

## Setup

- [x] Vault epic anchor added to `10_projects/dotfiles/11-tasks.md` (HARNESS-001)
- [x] Worktree `feat/harness-001-spec` off `origin/main`
- [x] `proposal.md` complete; engine-vs-consumer boundary drawn (engine owned here)
- [ ] ADR-013 stub drafted (`docs/adr/adr-013-agent-artifact-deploy-engine.md`, `status: proposed`)
- [ ] Manifest format locked (lean: JSON) — resolve at PR-1 kickoff, not here

## PR sequence (the work)

> Tracer-bullet first: smallest payload through the whole pipeline, then scale by consumer. Every PR ≤ ~300 LOC production diff.

- [ ] **PR-1 — Engine core (attribution tracer-bullet)** *(this epic's engine; own spec to be split from SDD-008 or new ENGINE id)*
  - [ ] Define `harness/manifest.json` + schema (one agent: Claude; one enforced pattern: no-attribution)
  - [ ] `compile-harness.sh`: parse manifest → inject `## Overrides of harness defaults` block w/ source-marker into deployed `CLAUDE.md` + `AGENTS.md` → commit artifact
  - [ ] bats: malformed manifest fails; generated marker present; line-cap ≤ 80 assertion
  - [ ] CI drift guard (run-twice-and-diff) fails on hand-edit
  - [ ] Satisfies epic AC1, AC3, AC4, AC5 (Claude path), partial AC7
- [ ] **PR-2 — Windows parity** `compile-harness.ps1` + Pester/PSScriptAnalyzer; cross-OS equivalence test → AC2
- [ ] **PR-3 — All pointer agents** extend manifest to agy + Copilot (`pointer`+overlay) and OpenCode/Pi (`native`); AGY ≤ 50 assertion → completes AC7
- [ ] **PR-4 — SDD-008 consumer** rewire skill deploy through the engine (skill spec stays skill-specific) → AC6
- [ ] **PR-5 — IDEAS-007 consumer** `.agent/<id>/INSTRUCT.md` + registry consumed by the manifest
- [ ] **PR-6 — #156 consumer** full `enforce: true` propagation + contradiction guard; remove contradictory vault carve-outs
- [ ] **PR-7 — #159 consumer** work/personal SSOT mode switch in the manifest

## Dependency graph

```
PR-1 engine core ──┬─> PR-2 win parity ─> PR-3 all agents ──┬─> PR-4 SDD-008
                   │                                         ├─> PR-5 IDEAS-007
                   └─────────────────────────────────────── ├─> PR-6 #156
                                                             └─> PR-7 #159
```

## Closing (epic)

- [ ] All seven PRs merged; each child spec archived under `specs/archive/`
- [ ] Every epic acceptance criterion (AC1–AC8) has matching evidence in `verification.md`
- [ ] ADR-013 promoted from `proposed` → `accepted`; adr-001/008 carry supersede pointers
- [ ] Vault HARNESS-001 anchor ticked with the closing PR link
- [ ] `verification.md` filled in

## Machine-readable features

Epic-level `features.json` maps the engine ACs to executable checks (sibling file). For an umbrella, most features become checkable only as PRs land; entries stay `pending` until their PR sets evidence. **Pass-state gating:** only the harness may set `"state": "passing"` after capturing exit 0. Per [[pattern-feature-list-as-primitive]].
