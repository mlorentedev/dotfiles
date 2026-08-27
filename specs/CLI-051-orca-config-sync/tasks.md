# Tasks: CLI-051-orca-config-sync

## Phase 1: Baseline Assets & Manifest
- [x] Create `ai/orca/keybindings.json` with the Linux tab navigation defaults.
- [x] Register `orca-keybindings` in `ai/deploy.json`.

## Phase 2: Core Orca Package in Go
- [x] Implement `cli/internal/orca/export.go` to export keybindings and clean settings.
- [x] Implement `cli/internal/orca/tune.go` with process checking and atomic updates.
- [x] Add unit tests in `cli/internal/orca/orca_test.go`.

## Phase 3: CLI Subcommand Wiring & Doctor Checks
- [x] Create `cli/internal/cmd/orca.go` exposing `dotf orca export` and `dotf orca tune`.
- [x] Wire `orca` command into `cli/internal/cmd/root.go`.
- [x] Extend `cli/internal/doctor/checks_orca.go` to inspect keybindings deployment.

## Phase 4: Documentation & Lessons
- [x] Update `ai/orca/ORCA.md` with synchronization and deployment instructions.
- [x] Record lesson in `docs/lessons/`.
- [x] Verify all tests and CLI executions.
