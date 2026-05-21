---
tags: [spec, tasks]
created: "2026-05-21"
---

# Tasks - WIN-001-healthcheck-ps1

> TDD order. One task = one focused commit.

## Setup

- [x] Branch created from main: `feat/WIN-001-healthcheck-ps1`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Implement `scripts/healthcheck.ps1` mirroring `healthcheck.sh` 12-section layout (parity numbering, SKIP for Linux-only sections, Write-Pass/Fail/Skip/Section helpers, exit code policy)
- [x] Write `tests/healthcheck-ps1.bats` (parity asserts: 12 sections, helpers defined, SKIP for sec 9/11/12, PSScriptAnalyzer)
- [x] Wire `healthcheck.ps1` into `setup-windows.ps1` as section 8d (post-doctor, non-fatal, references `hc` alias in the warn message)
- [x] Add `hc` function to `powershell/profile.ps1` (mirrors Linux `hc` alias semantics)
- [x] Add parity asserts in `tests/setup-windows.bats` (invocation present, non-fatal, ordering vs doctor)
- [x] Add parity asserts in `tests/powershell-profile.bats` (`hc` function defined, references healthcheck.ps1)
- [x] PSScriptAnalyzer clean on `setup-windows.ps1`, `powershell/profile.ps1`, `scripts/healthcheck.ps1`
- [x] Empirical smoke run on this Windows machine: 59 passed / 18 failed / 15 skipped, exit code 1 (correct)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] PSScriptAnalyzer / PowerShell parse clean
- [x] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] CI green
- [ ] Archive: `specs/WIN-001-healthcheck-ps1/` -> `specs/archive/`

## Sibling tickets opened in vault during this PR

- [WIN-001b-healthcheck-auto-wire-linux](https://github.com/mlorentedev/dotfiles) — Linux parity of the auto-wire decision (P1, blocked by WIN-001 + 1 week UX validation).
- [REFACTOR-003-diff-check-ps1](https://github.com/mlorentedev/dotfiles) — PowerShell port of `diff-check.sh` to unSKIP healthcheck.ps1 section 12 (P2, independent).
