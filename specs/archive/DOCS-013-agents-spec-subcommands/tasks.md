---
id: "DOCS-013-tasks"
type: spec
status: draft
owner: manu
tags: [spec, tasks, templates]
created: "2026-08-18"
---

# Tasks - DOCS-013-agents-spec-subcommands

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Worktree created from main (`dotfiles-wt-router`, branch `feat/harness-router-and-slimming`)
- [x] Proposal initialized and issue #1016 linked

## Implementation (TDD Sequence)

- [x] [P] [AC2] **TDD (Red)**: Write `TestSpecSubcommandsProseMatchesCode` in `cli/internal/spec/drift_test.go`. Verify it fails on current `AGENTS.md` (which claims `dotf spec fill`).
- [x] [AC1] **TDD (Green)**: Update `AGENTS.md` and `ai/claude/CLAUDE.md` § Spec-Driven Development to correctly document CLI subcommands (`init`, `review`, `archive`) and separate skill-only conversational steps.
- [x] [AC2] **TDD (Verify Green)**: Run `go test ./cli/internal/spec/...` and verify `TestSpecSubcommandsProseMatchesCode` passes.
- [x] [AC3] **Mutation Testing**: Temporarily inject an invalid subcommand into `AGENTS.md` (e.g. `dotf spec nonexistent`), assert test goes RED, then revert mutation and verify test returns to GREEN.

## Closing

- [x] All tests in `cli/` pass (`go test ./...`)
- [x] `features.json` populated with verification commands and captured evidence
- [x] `verification.md` filled in with mutation test evidence

