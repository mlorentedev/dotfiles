---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - CLI-002-repo-structure

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/cli-002-repo-structure`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1/R2 resolved by design; R3 accepted deliberately)

## Implementation

- [x] Write failing `tests/architecture-md.bats`: doc exists, README links it, every real top-level dir appears in the doc's table, boundary section points to AGENTS.md (no duplication) — red confirmed (5/5 fail before the doc exists)
- [x] Author `docs/architecture.md` (dir table, target tree, boundary + epic pointers) + README link — bats guard green (5/5)
- [x] Restructure `cli/`: domain logic to `internal/review/` (package `review`: `Resolve`, `ReadDiff`, `Request`), Cobra wiring to `internal/cmd/` (`New(version)`), `cmd/dot/main.go` entrypoint-only keeping `var version` (ldflags `-X main.version` untouched, R2); tests moved with asserts unchanged (only package decl + `newRootCmd()` -> `New(\"dev\")` call sites); `go test ./...` + `gofmt -l` + `go vet` + `go build` green

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-002-repo-structure/features.json`):

```json
[
  {
    "id": "CLI-002-repo-structure-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
