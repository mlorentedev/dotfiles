---
tags: [spec, tasks]
created: "2026-08-28"
---

# Tasks - AI-039-copilot-settings-merge

## Setup

- [x] Branch: `feat/copilot-settings-merge` from `origin/main` (worktree `dotfiles-wt-copilot-settings`)
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md`

## Implementation

- [x] [P] [AC1] Failing tests in `cli/internal/deploy`: merge preserves unmanaged keys, creates an absent destination, tolerates a `//` header, second run unchanged (mtime)
- [x] [AC1] `Config.Strategy` (`replace` default | `merge`), `PlanConfig` for non-rendered entries, `Deploy` merges through it
- [x] [P] [AC2] Failing tests: plan / dry-run of a non-rendered entry leaves no destination directory; unknown strategy and rendered merge rejected by name
- [x] [AC2] Compare hoisted above staging for non-rendered entries; `ParseManifest` validates `strategy`
- [x] [P] [AC3] `ai/copilot/settings.json` (model, includeCoAuthoredBy, autoUpdate), `ai/copilot/config.json` trimmed to `trustedFolders`, three manifest entries gated with `requires: copilot`, both setups copy only `copilot-instructions.md`
- [x] [AC3] `Config.Requires` + `dotf deploy` skip (`deploy_requires_test.go`); `tests/copilot-config.bats` rewritten against the documented-key set (1.0.81), manifest shape, explicit copy; `tests/env-contract.bats` comment follows the model to `settings.json`
- [x] [P] [AC4] Failing test `TestCheckDeployManifest_ByStatus` (PASS with counts / WARN naming the entry / SKIP without repo / no directory created)
- [x] [AC4] `checkDeployManifest` registered after `checkDeployDrift`
- [x] [AC5] Box: `dotf deploy` → merged, per-box keys intact; second run `in sync`; `dotf doctor` green; `copilot -p` smoke
- [x] Mutation checks: merge from an empty object fails AC1's tests; MkdirAll before the compare fails AC2's test

## Verification

- [x] Go loop: build, vet (host, `GOOS=linux`), test, golangci-lint (0 issues)
- [x] bats: copilot-config, env-contract, setup-linux, setup-windows; parse + ASCII delta 0 + CRLF intact on `setup-windows.ps1`; shellcheck on `setup-linux.sh`
- [x] `verification.md` records the evidence; `features.json` per AC
