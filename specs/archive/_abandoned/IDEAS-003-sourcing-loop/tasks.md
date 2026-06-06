---
tags: [spec, tasks, ideas-003]
created: "2026-05-25"
---

# Tasks - IDEAS-003-sourcing-loop

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [ ] Branch created from main: `refactor/IDEAS-003-sourcing-loop`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left (R1-R5 mitigated; pick `__src_f` per R2 decision)
- [ ] Capture pre-refactor `profile-shell` baseline (5 runs, both bash and zsh) → record in PR description

## Implementation

> TDD order. Parity test first, then refactor, then drift check.

- [ ] Write failing bats `tests/sourcing-loop.bats` #1: dump pre-refactor env state (exports + aliases + functions) into `/tmp/pre-state`. Mark test pending — implementation will fill in `/tmp/post-state` comparison.
- [ ] Confirm the test reads pre-state correctly by capturing it from current `.zshrc` source order. Commit baseline.
- [ ] Refactor `.zshrc`: replace explicit source block with the brace-expanded loop. Use `__src_f` per R2.
- [ ] Re-run parity test: confirm post-state == pre-state for zsh.
- [ ] Refactor `.bashrc` identically.
- [ ] Confirm parity test passes for bash.
- [ ] Run `profile-shell --shell zsh` and `--shell bash` — record numbers, assert ±10%.
- [ ] Re-deploy via `setup-linux.sh` (or symlink refresh). Run drift detector. Exit 0 expected.
- [ ] Verify shell startup parity: open fresh terminal, confirm no warnings/errors.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks N/A (shell)
- [ ] Lint passes (no new `.sh` files; rc files validated by bash -n / zsh -n)
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.

Minimal `features.json` skeleton (drop into `<repo>/specs/IDEAS-003-sourcing-loop/features.json`):

```json
[
  {
    "id": "IDEAS-003-sourcing-loop-f1",
    "behavior": ".zshrc uses brace-expanded loop for .zsh/*.zsh sourcing",
    "verification": "grep -qE 'for __src_f in.*\\$HOME/\\.zsh/\\{' .zshrc",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-003-sourcing-loop-f2",
    "behavior": ".bashrc uses identical pattern",
    "verification": "grep -qE 'for __src_f in.*\\$HOME/\\.zsh/\\{' .bashrc",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-003-sourcing-loop-f3",
    "behavior": "Env state parity pre/post refactor (zsh + bash)",
    "verification": "bats tests/sourcing-loop.bats --filter 'parity'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-003-sourcing-loop-f4",
    "behavior": "Shell startup time within ±10% of baseline",
    "verification": "scripts/shell-profile.sh --assert-within 10",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-003-sourcing-loop-f5",
    "behavior": "Drift detector clean post-deploy",
    "verification": "scripts/drift-detector.sh",
    "state": "pending",
    "evidence": ""
  }
]
```
