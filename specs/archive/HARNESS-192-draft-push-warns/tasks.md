---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "HARNESS-192-draft-push-warns"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-192-draft-push-warns

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/3364-draft-push-warns` (from `75cccbd0`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC4] Red: `tests/spec-gate-archive.bats` AC9 tests (draft warns and passes; `false` and unset still refuse; `skip-archive` without rationale still refuses on a draft; `dotf gate spec` ignores the event's `draft`; `spec-gate-pr.sh` never names the variable)
- [x] [AC3] Red: `tests/spec-gate-prepush.bats` (stub logs `SDD_PR_DRAFT`; draft → `true`, ready or missing field → `false`, no PR → unset; the `--jq` expression carries `draft`)
- [x] [AC3] Green: `scripts/spec-gate-prepush.sh` resolves `draft` and exports `SDD_PR_DRAFT`
- [x] [AC1] [AC2] Green: `scripts/check-spec-gate.sh` warns and returns 0 on `SDD_PR_DRAFT=true`, after the `skip-archive` branch
- [x] [AC1] [AC4] Green: `cli/internal/cmd/gate_spec_rules.go` prints the same warning on the same variable; the event's `draft` stays unread
- [x] Lesson 590: a merge-time invariant checked at push time blocks the fix it needs

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet`, both GOOS)
- [x] Lint passes (shellcheck; pinned golangci-lint, both GOOS)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

`features.json` beside this file maps AC1..AC4 to the tests above; `state: verified` is written only with the harness's captured evidence.
