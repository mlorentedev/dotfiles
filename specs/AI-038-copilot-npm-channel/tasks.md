---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - AI-038-copilot-npm-channel

## Setup

- [x] Branch: `feat/copilot-npm-channel` from `origin/main`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [AC1] Failing tests: catalog entry present, winget row absent, no Linux installer line (`tests/copilot-config.bats`, `tests/setup-windows.bats`)
- [x] [AC1] `packages.json` entry; winget row removed; both setup blocks re-worded to name the catalog
- [x] [AC2] `ai/copilot/config.json` `autoUpdate: false` + bats guard
- [x] [AC3] Failing test `TestCheckCopilot_PinMatchByStatus` (five rows by status)
- [x] [AC3] `checkCopilot` (semverOf + catalogPin + matchPinFrom), registered after `checkOpenCode`
- [x] [AC4] ADR-036 table + dated amendment
- [x] [AC5] Box verification recorded in `verification.md`

## Verification

- [x] Go loop: build, vet (host, `GOOS=windows`, `GOOS=linux`), test, golangci-lint
- [x] bats: copilot-config, setup-windows, opencode (catalog shape), verify-setup parse
- [x] `verification.md` records the evidence
