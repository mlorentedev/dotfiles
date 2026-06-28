---
id: "HARNESS-041-board-pickup-self-assign"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-28"
issue: "mlorentedev/knowledge#140"   # repo#NNN — the bitácora item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-041-board-pickup-self-assign

## Why

HARNESS-010 (#270) shipped `bitacora-status.yml`, which moves a board item to **In Progress**
when its issue is **assigned** — making *self-assign at pickup* the start-of-work trigger. But
self-assign is itself a manual habit, and in practice it gets skipped: observed 2026-06-28, both
`knowledge#137` and `#140` were actively worked while **unassigned**, so the Action never fired
and their board status had to be moved by hand. The automation exists; the human step that arms
it is the weak link. We need a **mechanical** trigger for the self-assign, not another reminder.

## What

A global `post-checkout` git hook (added to the GUARD-001 dispatcher, machine-wide via
`core.hooksPath`) that, when a **numbered feature branch** is picked up, **self-assigns** the
linked issue — which then fires the existing `bitacora-status` Action to set In Progress. The
hook does **not** move the board itself: the status transition stays the Action's job (single
source of truth). Branch creation is the mechanical start-of-work signal; the hook automates the
one step that was being forgotten.

After this PR:

1. **`git-hooks/post-checkout`** — dispatcher: runs the pickup helper backgrounded + silenced,
   then chains any repo-local `post-checkout` (a board hiccup must never block/slow a checkout).
2. **`git-hooks/lib/board-pickup.sh`** — parses the issue number from `<prefix>/<issue>-<slug>`,
   resolves the current repo first then the `knowledge` bitácora home (ADR-018), and self-assigns
   the issue **only if open** (`gh issue edit --add-assignee @me`, idempotent at the event layer).
3. **`scripts/install-git-hooks.sh`** — makes `post-checkout` executable on deploy.
4. **Branch convention** — `pattern-git-workflow.md` (knowledge vault) documents `<prefix>/<issue>-<slug>`.

## Out of scope

- **Moving the board status directly** — that is HARNESS-010's Action; this hook only self-assigns
  (no duplicated project/field/option IDs).
- **The branchless / vault-task case** — work with no branch has no mechanical trigger here; it
  stays covered by the AGENTS.md pickup discipline. This PR is the mechanical-force layer only.
- **Rolling `bitacora-status.yml` out to every repo** — that is the OPS rollout; where the Action
  is absent, the `knowledge` fallback still fires (foreign tickets are knowledge issues, ADR-018).

## Risks / open questions

- **Action dependency.** The board only moves if the target repo's `bitacora-status` Action is
  deployed. Mitigation: the knowledge fallback (where the Action is verified present) catches
  foreign-repo tickets per ADR-018. (Resolved.)
- **Re-checkout churn.** Re-checking-out an already-picked-up branch must not re-fire. Mitigation:
  `--add-assignee @me` adds nothing if already assigned, so GitHub emits no `assigned` event.
  (Resolved.)
- **Finished work.** Assigning a closed/Done issue must not resurrect it. Mitigation: the helper
  guards `state == OPEN` before assigning; the Action also guards `state == 'open'`. (Resolved.)
- **Checkout latency.** A board call must never slow `git checkout`. Mitigation: the dispatcher
  backgrounds + silences the helper. (Resolved.)

## Acceptance criteria

- [ ] AC1 — `git-hooks/post-checkout` exists, runs the helper backgrounded, then chains the repo-local hook.
- [ ] AC2 — `board-pickup.sh` parses `<prefix>/<issue>-<slug>`, no-ops on flag≠1 / non-numbered branch.
- [ ] AC3 — the helper self-assigns via `gh issue edit --add-assignee @me`, only for OPEN issues, current-repo-then-knowledge.
- [ ] AC4 — `install-git-hooks.sh` marks `post-checkout` executable on deploy.
- [ ] AC5 — a hermetic bats suite covers parse, open-state guard, resolution order, and fire-and-forget; no non-ASCII `@test` names.

## References

- Issue: `mlorentedev/knowledge#140`. Builds on **HARNESS-010** (#270, `bitacora-status.yml`).
- GUARD-001 (#398/#415/#418) — the machine-wide `core.hooksPath` dispatcher this extends.
- ADR-018 — one shared bitácora board; foreign-repo tickets live as `knowledge` issues.
