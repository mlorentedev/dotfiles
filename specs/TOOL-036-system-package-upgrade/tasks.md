---
tags: [spec, tasks, templates]
created: "2026-11-10"
id: "TOOL-036-system-package-upgrade"
type: spec-tasks
status: active
owner: manu
---

# Tasks - TOOL-036-system-package-upgrade

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/system-package-upgrade`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation (PR 1, this branch)

- [x] [AC1] Failing test: darwin plan lists only outdated catalog formulae/casks as changes, refresh as info
- [x] [AC1] Implement `systemUpgrade` reconciler (darwin listing + plan) in `cli/internal/converge/sysupgrade.go`
- [x] [AC2] [AC3] Failing test: apply upgrades only catalog formulae/casks, then cleanup + autoremove; second plan is `changed=0`
- [x] [AC2] [AC3] Implement darwin apply + housekeeping
- [x] [P] [AC4] Failing test + implement linux: `sudo -n true` gate, skip reason, `--only-upgrade` on outdated catalog packages
- [x] [P] [AC5] Failing test + implement windows: per-id `winget upgrade`, one failure isolated and named in the step error
- [x] [AC6] Failing test + implement: refresh failure is a skip with the manager's error
- [x] Register the step after `tools` in `Registry` and in every test that enumerates steps
- [x] [AC7] `packages.json` `$comment` and ADR-036 amendment

## Follow-up (PR 1b, `fix/winget-outdated-id-column`)

#3287 merged at 6debd6ae before the fix for its last review finding was pushed, so that fix lands here.

- [x] [AC5] Failing test: a truncated Name cell (`Example application Git…`) must not mark `Git.Git` as outdated
- [x] [AC5] `wingetIDs` reads only the Id column, located from the header line by rune position; a truncated Id cell (`…`) still prefix-matches
- [x] [AC5] Proposal AC5 says only the ids that upgraded are reported, and that the id is read from the Id column

## Follow-up (PR 2)

- [ ] `dotf update` runs `system-upgrade` on its daily tick when HEAD did not move, and records the outcome in `last.json`

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` and `golangci-lint run` pass
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in, including a real plan → apply → `changed=0` run on the owner's Mac
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-036-system-package-upgrade/features.json`):

```json
[
  {
    "id": "TOOL-036-system-package-upgrade-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
