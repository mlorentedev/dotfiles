---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - WIN-013-scripts-dir-contract

## Setup

- [x] Branch: `fix/scripts-dir-contract` from `origin/main`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [AC1] Failing test: contract default == required_path_entries.windows[0] == setup's `$ScriptsDir` shape (`tests/env-contract.bats`)
- [x] [AC1] `setup-windows.ps1` `$ScriptsDir = "$DotfilesDest\scripts"`; `env-contract.json` `required_path_entries.windows[0]`
- [x] [AC2] Failing test: the seven retired names and the legacy-copy sweep are present, the User PATH is untouched (`tests/setup-windows.bats`)
- [x] [AC2] Removal block after "Deploying scripts..." (retired list x both dirs; deployed list x legacy dir)
- [x] [AC3] Remove the WIN-013 row from `doctor-gate-known-failures.txt`; Pester guard asserts the parser count (0 allowed)
- [x] [AC4] `claude-mem-heal`, `diff-check`, `healthcheck|doctor` guards refute `Copy-Item`/invocation instead of mention
- [x] [AC5] CI `test-windows` gate: `0 known runner-only FAIL(s), 0 unexpected, 0 stale`

## Verification

- [x] bats: env-contract, setup-windows, ci-windows-doctor-gate, profile-heal-ps1
- [x] Pester: doctor-gate
- [x] PSScriptAnalyzer: no new findings vs main; parse clean; ASCII delta 0
- [x] `verification.md` records the evidence
