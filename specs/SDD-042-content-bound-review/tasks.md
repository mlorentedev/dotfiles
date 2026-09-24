---
tags: [spec, tasks, templates]
created: "2026-09-23"
---

# Tasks - SDD-042-content-bound-review

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/squash-aware-staleness` (worktree `dotfiles-wt-squash-aware-staleness`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC2] Failing tests: normalised contract digest (checkbox fold, CRLF fold, `features.json` state/evidence blanked; AC text edit still differs) → `TestContractDigest*`
- [x] [AC2] Implement `contractDigests(specDir)` and record them in `WriteReviewRequest` (`contract_digests`) [AC6]
- [x] [AC1] Failing real-git tests: squash / rebase / merge landings accepted, AC text edit refused → `TestStaleSquashLanding`, `TestStaleRebaseLanding`, `TestStaleMergeLanding`, `TestStaleContractEditRefused`
- [x] [AC1] Gate decides freshness by digests when the request carries them
- [x] [AC3] Failing test + fix: legacy review with an absent `reviewed_sha` → distinct message → `TestStaleLegacyAbsentObject`
- [x] [P] [AC4] Failing tests: `--force-*` without `--reason` refused; with it, one `review_bypass:` line recording flags, what was overridden, reason, date → `TestArchiveBypassRecorded*`
- [x] [AC4] Implement `BypassReason`, capture of the skipped check's refusal, frontmatter write; CLI `--reason`
- [x] [AC5] Failing test: no refusal names a bypass flag → `TestArchiveRefusalsNameNoBypassFlag`; rewrite the messages
- [x] Update `dotf spec archive --help`, and the skill signature (vault SSOT + `harness/skills/` render)
- [x] [AC6] Launch this spec's own review; its archive runs through the digest path

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/SDD-042-content-bound-review/features.json`):

```json
[
  {
    "id": "SDD-042-content-bound-review-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
