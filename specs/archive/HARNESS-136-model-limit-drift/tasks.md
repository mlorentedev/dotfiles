---
tags: [spec, tasks, templates]
created: "2026-09-21"
---

# Tasks - HARNESS-136-model-limit-drift

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `chore/pi-qwen38-flash-context`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the two
      that mattered (provider scoping, doctor-not-CI) are recorded as resolved
      with the measurement that settled each

## Implementation

The declaration fix and the check are two commits because they are two claims:
the first is a data correction verifiable against published sources, the second
is the mechanism that keeps it true.

- [x] [AC1] Correct the seven declarations against the per-provider catalog,
      cross-checked with the provider's own published table — `51e9ffe`
- [x] [P] [AC2] [AC3] Test both drift directions, asserting the severity each
      earns and that neither leaks into the other
- [x] [AC2] [AC3] `checkModelLimits` compares declaration to catalog and reports
      over-declaration as FAIL, under-declaration as WARN — `27b6a82`
- [x] [P] [AC4] Test that one model id published by two providers resolves per
      provider; this fixes the bug the first implementation pass actually had
- [x] [AC4] Scope the lookup by provider, never by bare id
- [x] [P] [AC5] [AC6] Test the three "cannot compare" states — absent catalog,
      model not published, limit not published — for SKIP-or-silence, never PASS
- [x] [AC7] Test that an unparseable declaration FAILS rather than reporting no
      drift, the failure mode the repo's prohibited-pattern table is about
- [x] [AC8] No `--fix` path; the flag is not passed to the check at all
- [x] Register the check in `doctor.go` beside its sibling `checkModelPins`

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a `features.json` entry with a non-vacuous
      verification command
- [x] Type checks pass — `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`
- [x] Lint passes — `golangci-lint run` at the pinned v2.12.2, 0 issues
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1595, merged; review fixes in the follow-up PR)

## Machine-readable features

`features.json` sits alongside this file. Each acceptance criterion maps to at
least one feature whose `verification` is a single command that exits 0 on pass.
`state` stays `pending` until the harness runs the command and captures its
output as `evidence`.
