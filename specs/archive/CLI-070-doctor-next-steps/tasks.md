---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - CLI-070-doctor-next-steps

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/doctor-next-steps`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1][AC2][AC3][AC4] Write `TestNextSteps` (table of FAIL/WARN lines covering the run/re-run/recover-with/upgrade-with verbs, a diagnostic-only backtick span, and a duplicate command) and `TestNextSteps_NoFailNoBlock`
- [x] [AC1][AC2][AC3][AC4] Implement `nextSteps()` + `failRemedyRe` in `doctor.go` to make them pass
- [x] [AC5][AC6] Write `TestRun_NextStepsBlock` (FAIL-with-remedy end to end via `Run()`, and a quick-mode clean run producing no block)
- [x] [AC5][AC6][AC7] Wire `nextSteps()` into `Run()` after `rep.Summary()`; tee `Report`'s writer into a transcript buffer, computing `isColorEnabled` from the real `out` before wrapping it (AC7 — a `MultiWriter` is never an `*os.File`, so color detection must run first)
- [x] Full existing suite re-run (`go test ./internal/doctor/...`, `go test ./...`) — no regressions

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go build ./...`, `go vet ./...`)
- [ ] Lint passes (`golangci-lint run` — not run locally; pin drift on this box, CI is authoritative)
- [x] No unrelated changes in the diff (no scope creep) — pin-floor fix split out to PR #1441
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
