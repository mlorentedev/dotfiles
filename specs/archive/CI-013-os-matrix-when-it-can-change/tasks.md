---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "CI-013-os-matrix-when-it-can-change"
type: spec-tasks
status: active
owner: manu
---

# Tasks - CI-013-os-matrix-when-it-can-change

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `ci/os-matrix-when-it-can-change`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] Open questions in `proposal.md` are owner decisions that do not block this change

## Implementation

- [x] [AC1] [AC2] Write failing tests: `tests/ci-os-matrix.bats` (structural gate rule, OS-list outputs of cli and cli-smoke, shared filter step, filter/allow-list disjointness, integration cache)
- [x] [AC4] Move the docs-only set to `.github/filters/code.yml`; `tests/ci-path-filtering.bats` reads it
- [x] [AC1] [AC2] cli.yml: `test_os` output from `cli-changes`, `test` takes `fromJSON` of it; amend `tests/cli-go-tests-unfiltered.bats`
- [x] [AC1] [AC2] cli-smoke.yml: `cli-smoke-changes` job and `smoke_os` output
- [x] [AC3] ci.yml integration: buildx with a `type=gha` cache, written on push only; amend `tests/integration-builds-dotf.bats`
- [x] Mutation check: a literal OS list, an exclusion ci.yml calls code, and a push branch dropped from the expression each turn a test red
- [x] Lesson 578

## Verification

- [x] `bats` over every file referencing the changed workflows (315 tests) and `actionlint`
- [ ] AC1 and AC3 live proof after merge (recorded in `verification.md`)
