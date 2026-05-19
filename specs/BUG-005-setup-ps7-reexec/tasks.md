---
tags: [spec, tasks, bug, windows, powershell]
created: "2026-05-19"
---

# Tasks - BUG-005-setup-ps7-reexec

> TDD order.

## Setup

- [ ] Branch created from main: `fix/BUG-005-setup-ps7-reexec`
- [ ] `proposal.md` complete and acceptance criteria testable

## Implementation (TDD order)

### Tests first (red)

- [ ] `tests/setup-windows.bats`: assert `setup-windows.ps1` contains the PS version check (`PSVersionTable.PSVersion.Major -lt 7`).
- [ ] `tests/setup-windows.bats`: assert the re-exec path references `Get-Command pwsh` and `$PSCommandPath`.
- [ ] `tests/setup-windows.bats`: assert the fail-loud branch references `winget install Microsoft.PowerShell` and exits non-zero.
- [ ] `tests/setup-windows.bats`: assert `setup-linux.sh` does NOT contain `PSVersion` (negative parity — Linux is immune by construction).

### Implementation (green)

- [ ] `setup-windows.ps1`: insert preamble block right after `param()` and before the CONFIGURATION header. Detects `$PSVersionTable.PSVersion.Major -lt 7`, re-execs under pwsh if found, fails loud with winget hint otherwise.

### Refactor / cleanup (still green)

- [ ] Inline comment cites BUG-005, SDD-002, and the upstream PS 7.0 release notes (`ConvertFrom-Json -AsHashtable`).
- [ ] PSScriptAnalyzer clean.

### Local verification

- [ ] All new bats asserts green (grep emulation, full bats in CI).
- [ ] Smoke under Windows PowerShell 5.1: `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` → expect re-exec line + clean merge, no AsHashtable warning.
- [ ] Smoke under pwsh: `pwsh -NoProfile -File .\setup-windows.ps1` → expect no preamble lines, direct execution.

## Closing

- [ ] Every acceptance criterion covered by ≥1 test
- [ ] `verification.md` filled with empirical bytes/output
- [ ] PR opened referencing this spec folder

## Machine-readable features

```json
[
  {
    "id": "BUG-005-f1",
    "behavior": "Windows PowerShell 5.1 + pwsh installed -> setup-windows.ps1 re-executes under pwsh and proceeds",
    "verification": "bats tests/setup-windows.bats -f 'reexec under pwsh'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-005-f2",
    "behavior": "Windows PowerShell 5.1 + pwsh NOT installed -> setup-windows.ps1 exits 1 with actionable error",
    "verification": "bats tests/setup-windows.bats -f 'fail loud'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "BUG-005-f3",
    "behavior": "pwsh 7+ direct invocation -> preamble no-op, script proceeds",
    "verification": "bats tests/setup-windows.bats -f 'pwsh direct'",
    "state": "pending",
    "evidence": ""
  }
]
```
