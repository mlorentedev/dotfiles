---
tags: [spec, tasks, templates]
created: "2026-11-13"
id: "HARNESS-196-vendor-security-audit-web-perf"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-196-vendor-security-audit-web-perf

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `chore/3456-harness-rerender` (the spec was added after the branch; the work predates it)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC2] Failing test first: `TestTheCompiledDependencyMapMatchesTheRecords` failed with the new record in place ("security-audit: the compiled map says [], its record's requires: says [test]")
- [x] [AC2] Add `security-audit: {"test"}` to `DefaultSkillDependencies`
- [x] [AC3] Failing test first: `HERMES-018` failed ("security-audit is vendored but absent from ATTRIBUTION.md")
- [x] [AC3] Add the `security-audit` and `web-perf` rows to `ATTRIBUTION.md`, with the licenses read from upstream
- [x] [AC1] Correct the vault source first (vault `11bacfaf`: web-perf licence, reviewer row in ROSTER.md), then re-render the records from the vault with `dotf harness refresh`
- [x] [AC4] Compare the three vendored scripts with upstream at `c1c8a8c1471069fb0e188eeaff69b8e8db6564a8`
- [x] [AC5] Read both validators in full; record imports, side effects and deploy path in `verification.md`

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-196-vendor-security-audit-web-perf/features.json`):

```json
[
  {
    "id": "HARNESS-196-vendor-security-audit-web-perf-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
