---
tags: [spec, tasks, templates]
created: "2026-11-09"
id: "HARNESS-158-review-scope"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-158-review-scope

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/harness-158-review-scope`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC3] `SpecFolderNames` returns every folder name the base resolver followed (one trace, `traceSpecFolder`, so the two cannot disagree), with a test on a renamed fixture
- [x] [AC1] [AC2] [AC3] Failing tests for `spec.OwnCommits(repoRoot, base, head, paths, issue)` on a fixture history: two slices, interleaved and trailing unrelated commits, an issue-only slice, a `#17270` decoy, a merge commit
- [x] [AC1] [AC2] [AC3] Implement `OwnCommits` (`git log --no-merges` over the folder paths, union `--grep` on the issue)
- [x] [AC4] `ReviewScope` carries the list and the range size; `writeReviewScope` names the commits; `describeScope` prints `N of M`; test that an initial scope is unchanged
- [x] [AC4] `reviewRange` reads the spec's issue and fills the list
- [x] [AC5] Run the derivation against HARNESS-142's recorded base and head, record the output in `verification.md`

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `go build`, `go vet`, `go test ./...`, `golangci-lint run`, and the `GOOS=windows` vet and lint pass
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-158-review-scope/features.json`):

```json
[
  {
    "id": "HARNESS-158-review-scope-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
