---
tags: [spec, tasks, templates]
created: "2026-06-09"
---

# Tasks - OPS-001-self-deploy-timer

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/self-deploy-timer` (worktree `~/Projects/dotfiles-wt/self-deploy-timer`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (all resolved by the design pass)

## Implementation

- [x] Write failing bats `tests/dotfiles-selfupdate.bats` covering AC1–AC5 (fixture git repo + fake remote + injectable setup stub)
- [x] Implement `scripts/dotfiles-selfupdate.sh` (guard → fetch → ff-only → conditional setup); make AC1–AC5 green
- [x] `shellcheck` clean; bash+zsh pass
- [x] Add `systemd/dotfiles-selfupdate.service` (oneshot, ExecStart = repo script) + `systemd/dotfiles-selfupdate.timer` (OnCalendar=daily, Persistent=true, RandomizedDelaySec) — AC7
- [x] Wire opt-in block in `setup-linux.sh` gated on `DOTFILES_AUTODEPLOY` (1=install+enable, 0=disable+remove, unset=no-op), mirroring the hive-upgrade block — AC6
- [x] Guard test for the opt-in gate + unit structure (AC6/AC7) — incident→guard
- [x] Windows parity: `scripts/dotfiles-selfupdate.ps1` (ASCII-only) + Scheduled Task in `setup-windows.ps1` under `$env:DOTFILES_AUTODEPLOY` — AC8 (runtime verify deferred)
- [x] Docs: `docs/adr/adr-019-*` + `docs/runbooks/guide-self-deploy-timer.md` (healthcheck line intentionally skipped — opt-in would false-positive; `.claude/CLAUDE.md` is gitignored, left untouched)

## Closing

- [x] Every acceptance criterion covered by ≥1 test
- [x] `features.json` present (states stay `pending` — only the harness may mark `passing`)
- [x] Lint passes (shellcheck + ASCII guard)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating: only the harness may set `"state": "passing"` after capturing exit 0.
