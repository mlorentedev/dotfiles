---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "BUG-170-review-not-on-head"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-170-review-not-on-head

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3243-review-not-on-head`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC3] `prtriage.IsReviewCheck`: `review` and `<caller> / review`, nothing else
- [x] [AC1] Failing tests: a failed, skipped, absent, or running `review` on the head lists the PR (repo-wide and `--pr`); cancelled then success on one head does not
- [x] [AC1] Read `commits/<sha>/check-runs` (both names, so no `check_name=` filter) for each non-draft PR inside the comment fan-out; hold one the comments leave clear with the head reading
- [x] [AC2] Failing tests: a draft, and a release-signature diff with a skipped review, stay unlisted; `withoutExempt` drops only exempt diffs
- [x] [AC3] Failing tests, then `Decide`: a skipped or absent review refuses with `review did not run on <sha>`; `manualMerge` uses the matcher

## Closing

- [x] `go test ./...`, `golangci-lint run`, `GOOS=windows` vet and lint
- [ ] `verification.md` filled in; independent review; archive

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-170-review-not-on-head/features.json`):

```json
[
  {
    "id": "BUG-170-review-not-on-head-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
