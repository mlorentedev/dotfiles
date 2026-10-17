---
tags: [spec, tasks, templates]
created: "2026-10-16"
---

# Tasks - DX-005-zsh-completions

> TDD order. One task = one focused commit.

## Setup

- [x] Branch created from main: `feat/zsh-completions`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Baseline `zsh -ic exit` median recorded in `verification.md` before the first change

## Implementation

PR 1 (this one): `.zshrc` + the `shell-completions` step.

- [x] [AC1] Failing bats for the dump rule: fresh dump trusted, aged dump rebuilt, newer completion file rebuilds
- [x] [AC1] `.zshrc`: rebuild only when stale
- [x] [P] [AC2] Failing bats: fpath order, no duplicates, `menu select`
- [x] [AC2] `.zshrc`: brew `shellenv` above `compinit`, `~/.zsh/completions` and `typeset -U fpath` before it, `menu select`
- [x] [AC4] Failing Go test for `shell-completions` (write, idempotent, skip missing tool, dry run, probe)
- [x] [AC4] Implement the step, register it, extend the registry-order test and the provisioning runbook

PR 2 (follow-up under this spec): zsh-autosuggestions, once the pin and install mechanism are decided.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
