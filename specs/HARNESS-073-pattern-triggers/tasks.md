---
id: "HARNESS-073-pattern-triggers-tasks"
type: spec
status: draft
created: "2026-08-14"
issue: "mlorentedev/dotfiles#980"
tags: [spec, tasks, harness, triggers]
template_version: "1.0"
---

# Tasks: HARNESS-073 File-Path Pattern Trigger Resolution

- [x] Define `harness/triggers.json` schema and default trigger rules <!-- id: 0 -->
- [x] Implement glob matching and trigger resolution in `cli/internal/harness/triggers.go` <!-- id: 1 -->
- [x] Implement unified diff path extraction <!-- id: 2 -->
- [x] Wire `dotf harness triggers` command in `cli/internal/cmd/harness.go` <!-- id: 3 -->
- [x] Add unit tests for trigger matching and CLI flags in `cli/internal/harness/triggers_test.go` and `cli/internal/cmd/harness_test.go` <!-- id: 4 -->
- [x] Verify test suite passes (`go test ./...`) <!-- id: 5 -->
