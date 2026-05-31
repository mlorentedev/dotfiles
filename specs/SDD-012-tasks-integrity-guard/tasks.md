---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - SDD-012-tasks-integrity-guard

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Worktree off origin/main: `feat/sdd-012-tasks-integrity-guard`
- [x] `proposal.md` complete; acceptance criteria testable
- [~] Vault gate entry in `11-tasks.md` — DEFERRED (vault checkout busy with a parallel session); scaffolded with `--force-no-vault`, entry to be added when the vault is free (tracked, not skipped)

## Implementation (TDD)

- [x] Write failing bats `tests/check-backlog-integrity.bats` with fixtures (dup, contradiction, clean+sub-id, usage, missing, dispatcher) — RED (exit 127)
- [x] Implement `scripts/check-backlog-integrity.sh` (sed extract `status<TAB>id` + plain awk aggregate; greedy `[a-z]?` keeps WIN-002a distinct) — bats GREEN
- [x] Wire `vault check-tasks` into `scripts/vault.sh` dispatcher (+ usage)
- [x] Add GUI-independent section "7/7 Backlog Integrity" to `vault-health.sh` scanning every `10_projects/*/11-tasks.md` (auto-surfaces at SessionStart)

## Closing

- [x] Every AC (1-4) covered by a test/check; AC5 is the deferred vault follow-up
- [x] `features.json` present with non-vacuous verification commands
- [x] shellcheck clean + `bash -n` clean on all 3 scripts
- [x] No unrelated changes (diff = guard + test + dispatcher + vault-health section + spec)
- [x] `verification.md` filled
- [ ] PR opened referencing this spec folder

## Follow-up (vault, separate from the guard PR)

- [ ] **AC5** — when the vault checkout is free: add the SDD-012 backlog entry, then consolidate `10_projects/dotfiles/11-tasks.md` to one canonical list (merge the 30 duplicates, resolve the 3 contradictions: IDEAS-007 / SDD-004 / BUG-022) until `check-backlog-integrity.sh 10_projects/dotfiles/11-tasks.md` exits 0.

## Machine-readable features

Sibling `features.json` per [[pattern-feature-list-as-primitive]]. The agent may not set `"state": "passing"` — only the harness, after running `verification` (exit 0), sets that terminal state.
