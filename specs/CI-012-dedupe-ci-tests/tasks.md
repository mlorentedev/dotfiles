---
tags: [spec, tasks, templates]
created: "2026-10-07"
---

# Tasks - CI-012-dedupe-ci-tests

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/CI-012-dedupe-ci-tests`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Rows N4, N5 and N7 of #2059 (this PR). N1, N2, N3, N6 and N8 follow as their own PRs.

- [x] [AC3] Add the Go twins that did not exist: top tier through the command, saturated pool through the command, probe reaching a stub harness; pin `exit` presence in the record test
- [x] [AC3] Mutate production code once per deleted case and confirm its twin goes red (table in the PR body)
- [x] [AC3] Delete 11 of 12 `dotf-agent-run.bats` cases, keep the pipe case
- [x] [AC1] Write `tests/dotf-bin-helper.bats` (helper outcomes, fail-open detector with negative control) before the helper
- [x] [AC1] [AC2] `tests/lib/dotf-bin.bash`; five files resolve the binary through it
- [x] [AC2] [AC4] `ci.yml` test job: build once and export `DOTF_BIN`; junit report with timing; upload artifact
- [ ] Remaining rows: N1, N2, N3, N6, N8

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CI-012-dedupe-ci-tests/features.json`):

```json
[
  {
    "id": "CI-012-dedupe-ci-tests-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
