---
tags: [spec, tasks, claude-mem, epipe]
created: "2026-06-06"
---

# Tasks - 2026-05-27-claude-mem-heal-consumer-epipe

> TDD order. One task = one focused commit (squashed into one PR). Tick as you go.

## Setup

- [x] Branch created from main: `fix/claude-mem-heal-consumer-epipe`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Vault entry flipped `[~]` → `[ ]` (resumed), Option-A decision recorded

## Implementation (TDD)

- [x] **RED** — extended `tests/claude-mem-heal.bats` with the functional EPIPE
  guard (AC1) + structural tests (AC2) + idempotency (AC4). Structural tests
  confirmed FAILING against the old `head -n1`-emitting code.
- [x] **GREEN (.sh)** — `heal_hooks_json`: detect `break; }; done` OR `head -n1`;
  sed-substitute both to `}; done | sed -n 1p`; kept BUG-018 directive sub;
  refreshed comments + log; dropped 2 dead assignments. bats green.
- [x] **GREEN (.ps1)** — `Repair-HooksJson`: drain form → `sed -n 1p`; detect +
  convert existing `head -n1`; refreshed comments + log.
- [x] **PARITY** — added `tests/claude-mem-heal-ps1.bats` (AC3): drives
  `Repair-HooksJson` via `pwsh` (cygpath path bridge), same outcome + idempotency.
- [x] **LIVE (AC5)** — healed a copy of the real deployed `13.3.0/hooks/hooks.json`:
  7 × `head -n1` → `sed -n 1p`; second run silent; valid JSON; surgical diff.

## Closing

- [x] Every acceptance criterion (AC1–AC5) is covered by ≥1 test or recorded evidence
- [x] `bats tests/claude-mem-heal.bats tests/claude-mem-heal-ps1.bats` green (25/26; the 1 failure is a pre-existing Windows-only symlink env issue)
- [x] No unrelated changes in the diff (no `.mcp.json` / no `init-spec` changes)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder ([#242](https://github.com/mlorentedev/dotfiles/pull/242))
