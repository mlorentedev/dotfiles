---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - HARNESS-110

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft`
> state; freeze once you start `implementing`.
>
> `[P]` — no dependency on another unchecked task, safe to run in parallel.
> `[AC<n>]` — helps satisfy acceptance criterion #`<n>` from `proposal.md`.

## Setup

- [x] Branch created from main: `feat/harness-role-join`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Two independent behaviours: the pure join (AC1-AC3) and the hook plumbing (AC4-AC8). Their first
> test tasks carry `[P]`. The join has no I/O and lands first because the hook depends on it.

### The join

- [x] [P] [AC1] Failing test: `ResolveRoles` returns sorted roles for a rule whose skills intersect
      one persona. Fixture built from `LoadPersona`, not from a literal map — the test must break if
      the parse breaks.
- [x] [AC1] Implement `harness.ResolveRoles(suggestion, personas) []string` — pure, sorted, no I/O.
- [x] [AC2] Failing test: both ambiguous rules return their full role set —
      `code-complexity-and-refactor → [builder, reviewer]`, `spec-driven-development → [planner,
      reviewer]`. A rule with no skills returns empty, and empty is not an error.
- [x] [AC2] Make it pass without ranking or tie-breaking.
- [x] [AC3] Failing test: the drift guard goes red when the resolving-rule count drops below 16, and
      red when any persona contributes zero skills.
- [x] [AC3] Implement the guard in Go against `LoadPersona`. Not a python or shell script —
      `check-roster-consistency.py` was repaired once for exactly this class.
- [x] Refactor: extract anything the hook and the guard both need.

### The hook

- [x] [P] [AC5] Failing test: the command reads its payload from **stdin**. No `--prompt` flag is
      added, and none is accepted.
- [x] [AC5] Implement the `--from-hook` stdin mode.
- [x] [AC6] Failing test: each plausible prompt-field spelling is accepted, and the one that arrived
      is recorded. An unrecognised payload records that fact rather than guessing.
- [x] [AC6] Implement, in the shape of `OutcomePayloadUnrecognised`.
- [x] [AC7] Failing test: exit status is 0 for a malformed payload, an empty payload, an unreadable
      persona record, and a `triggers.json` that fails to parse. **Table-driven; every branch that
      can return an error is a row.** Exit 2 erases the user's prompt — this AC is the data-loss
      guard, so it is asserted, never inspected.
- [x] [AC7] Make every failure path fail open.
- [x] [AC8] Failing test: the full match+join completes under the 20 ms budget.
- [x] [AC8] Meet the budget; if the naive path misses it, cache the compiled regexes rather than
      trimming the rule set.
- [x] [AC4] Add the hook-emission entry to `harness/manifest.json` (first of its kind) and bind it
      through the existing deploy path. Verify by consequence: deploy, start a session, observe the
      suggestion — not by asserting the file contains a string.

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a matching `features.json` entry with a non-vacuous command
- [x] `go build ./... && go vet ./... && go test ./...` pass
- [x] `GOOS=windows go vet ./...` passes — the Windows leg compiles the same tree
- [x] `golangci-lint run` passes with the pinned version from `versions.conf`
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in (its work-gate section is already written)
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating: the agent CANNOT write `"state": "passing"` — only
the harness, after running `verification` and capturing exit 0, may set that terminal state.
