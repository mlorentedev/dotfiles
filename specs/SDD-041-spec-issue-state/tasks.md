---
tags: [spec, tasks, templates]
created: "2026-09-23"
---

# Tasks - SDD-041-spec-issue-state

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/spec-issue-state` (worktree `dotfiles-wt-spec-issue-state`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [ ] [P] [AC1] Failing tests: frontmatter forms (`owner/name#N`, `name#N`, `#N`, malformed) → `TestIssueStateResolveFrontmatter`
- [ ] [P] [AC1] Failing tests: prose shapes, positive and negative, plus frontmatter precedence → `TestIssueStateResolveProse`, `TestIssueStateFrontmatterWinsOverProse`
- [ ] [AC1] Implement `ResolveIssueLink` in `cli/internal/spec/issuelink.go`
- [ ] [P] [AC1] Failing test: REST output parsing (open, closed, PR, 404, other error) → `TestIssueStateGHLookup`
- [ ] [AC1] Implement `GHIssueStateLookup`
- [ ] [AC1] [AC2] Failing test: audit classification over a fixture tree (archive skipped, non-dirs skipped, deterministic order) → `TestIssueStateAuditClassifies`, `TestIssueStateAuditUnanswerable`
- [ ] [AC2] Implement `AuditIssueState` with a bounded lookup pool
- [ ] [AC2] `dotf spec audit` command, exit contract tested → `TestSpecAuditExitIssueState`
- [ ] [AC3] Failing test: doctor `spec-issue-state` mapping (PASS/FAIL/WARN/Skip) → `TestCheckSpecIssueState`
- [ ] [AC3] Wire `checkSpecIssueState` into the full doctor sweep
- [ ] [AC4] Normalise GOV-004 frontmatter; `TestIssueStateNoActiveSpecIsProseLinked`
- [ ] [AC5] Run `dotf spec audit` in hive and record the result in `verification.md`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/SDD-041-spec-issue-state/features.json`):

```json
[
  {
    "id": "SDD-041-spec-issue-state-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
