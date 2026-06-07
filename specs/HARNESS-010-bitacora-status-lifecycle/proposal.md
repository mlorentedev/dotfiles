---
id: "HARNESS-010-bitacora-status-lifecycle"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-07"
issue: "mlorentedev/dotfiles#270"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-010-bitacora-status-lifecycle

## Why

The bitácora (GitHub Project #1) has a `Status` field but **no rule tells the agent when to transition it**, and **nothing flips it automatically**. Observed 2026-06-06: HARNESS-006 (#261) was actively implemented while still in `Backlog` — its status had to be moved by hand. The mechanical board exists (built-in workflows set `Backlog` on add and `Done` on close); the *start-of-work* and *blocked* transitions are pure manual discipline that, in practice, gets skipped. A board whose `Status` lags reality is worthless as a real-time picture across repos.

## What

After this PR:

1. **Doctrine** — `AGENTS.md` codifies the status-lifecycle as a cross-agent rule: a Standing Order plus operational touchpoints in the Neural Hive Loop (pickup → self-assign; blocker → `Blocked`; close → `Done`). This is the canonical, agent-readable contract.
2. **Automation** — a new per-repo GitHub Action (`.github/workflows/bitacora-status.yml`) listens on `issues: [assigned]` and sets the issue's bitácora `Status = In Progress`. Self-assigning when you pick up an issue (the new Phase-1 habit) now flips the board automatically — the previously-manual step is gone.
3. **Rollout contract** — the bitácora runbook registers the new workflow in the per-repo deployment bundle (§4/§7) so OPS-002 (#258) rolls out *both* workflows (add-to-project + bitacora-status) to every linked repo in one pass.
4. **Skill wiring** — `/handoff` reconciles board status at session end: any issue worked this session must be `In Progress`/`Blocked`/`Done`, never left in `Backlog`.

## Out of scope

- Deploying the workflow to all repos — that is **OPS-002 (#258)**; this PR ships + validates it on `dotfiles` and registers it as the template.
- Auto-detecting `Blocked` — there is no reliable machine signal; it stays a doctrine-driven manual transition (one documented `gh` command).
- A "PR-linked" trigger — a PR opens *after* work starts, so it is a late signal; `assigned` fires at the true start.
- A cross-platform helper script (`.sh`/`.ps1`) — the `gh project item-edit` one-liner already in the runbook §5 is sufficient; a new script would add Windows-parity + test surface for no gain.

## Risks / open questions

- **Self-assign dependency.** The `assigned` trigger only fires if someone self-assigns. Mitigation: codify the habit in `AGENTS.md` Phase 1; `/handoff` reconciliation catches misses. (Resolved.)
- **PAT scope.** `BITACORA_PAT` must carry `project` + `repo` scope per repo. Verified present in `dotfiles` and `knowledge`. (Resolved.)
- **Add/edit race.** If `assigned` fires before the project auto-add, the item may not exist. Mitigation: the Action does `item-add` (idempotent — verified: re-adding #270 returned the existing item id, no duplicate) then `item-edit`; add-then-edit is self-sufficient. (Resolved.)
- **Closed issues.** Assigning a closed issue should not resurrect it to `In Progress`. Mitigation: `if: github.event.issue.state == 'open'` guard. (Resolved.)

## Acceptance criteria

- [ ] AC1 — `AGENTS.md` contains a status-lifecycle rule covering start→`In Progress`, blocker→`Blocked`, close→`Done`, as a Standing Order and as Neural Hive Loop touchpoints.
- [ ] AC2 — `.github/workflows/bitacora-status.yml` exists, triggers on `issues: [assigned]`, guards `state == 'open'`, and sets `Status = In Progress` via `gh` + `BITACORA_PAT` using the canonical project/field/option IDs.
- [ ] AC3 — the workflow file is valid YAML.
- [ ] AC4 — runbook §4/§7 registers `bitacora-status.yml` in the per-repo bundle; §5 marks `In Progress` as automated-on-assign while keeping `Blocked` manual.
- [ ] AC5 — `/handoff` SKILL.md has a board-status reconciliation step.

## References

- Issue: `mlorentedev/dotfiles#270` (HARNESS-010); related #265 (HARNESS-009), epic #244 (Flow v2), #258 (OPS-002 rollout).
- Runbook: `docs/runbooks/guide-bitacora-setup.md` (§5 status lifecycle, §7 workflow template) — OPS-004.
- ADR: `docs/adr/adr-018-de-vault-task-placement.md` (task state lives in the bitácora).
