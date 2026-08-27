---
tags: [spec, verification, orca, cli, deploy]
created: "2026-08-27"
---

# Verification - CLI-051-orca-config-sync

## Evidence

- [x] Declarative deployment -> `ai/deploy.json` entry `orca-keybindings`, verified via `go test ./internal/deploy/...` and `dotf deploy orca-keybindings`.
- [x] Settings & keybindings export -> `dotf orca export` extracts `ai/orca/keybindings.json` and `ai/orca/settings.json` (142 settings keys), verified via `TestExport_ExtractsKeybindingsAndSettings`.
- [x] Safe baseline tuning -> `dotf orca tune` (with `--dry-run` and active process guards), verified via `TestTune_AppliesBaselineAndCreatesBackup` and `TestTune_GuardsAgainstRunningProcess`.
- [x] Unit test suite -> `go test ./internal/...` PASS (100% green across all packages).

## Test status

- Test suite: `go test ./internal/cmd/... ./internal/deploy/... ./internal/doctor/... ./internal/orca/...` -> PASS (0 failures).
- Manual smoke test:
  - `dotf orca export` executed successfully against live system, producing `ai/orca/keybindings.json` and `ai/orca/settings.json`.
  - `dotf deploy orca-keybindings` executed successfully, reporting `deployed` on first run and `in sync` on subsequent runs.
  - `dotf orca tune --dry-run` planned correct baseline differences without modifying live state.
- No regressions: verified clean pass across entire Go CLI suite.

## Decisions made during implementation

- Kept `settings.json` export scoped strictly to `data["settings"]` to strip all ephemeral runtime keys (`orchestration.db`, worktree sessions, GitHub cache).
- Implemented `ProcessChecker` abstraction in `internal/orca` to guarantee determinism in tests while enforcing process safety on Linux and Windows.

## Promotion candidates

- [x] Lesson for `docs/lessons/`: recorded `docs/lessons/lesson-234-orchestrating-orca-ade-declarative-configuration-and-bi.md`.
- [ ] ADR-worthy decision: no (follows ADR-020 and CLI-039).
- [ ] Pattern candidate: no (follows existing agent configuration patterns).
