---
tags: [spec, tasks, templates]
created: "2026-08-07"
---

# Tasks - BUG-040-guard-gate-effectiveness

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/guard-hooks-effectiveness`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Go first: it owns the durable check (`dotf doctor`) and has the test harness. The
two shell twins then mirror the same three tiers — they cannot be collapsed into
the Go one here, because the deployed `dotf` release carries no source tree and so
cannot place `git-hooks/` itself (ADR-020 C7). Collapsing the *wiring* half is
tracked separately.

- [x] [AC1] Write failing test: hooksPath at an equivalent dispatcher elsewhere must not report INACTIVE
- [x] [AC3] Write failing test: separator/trailing-slash variants of the target normalize to the tier-1 PASS
- [x] [AC2] Write test: a `pre-commit` without `lib/memory-sink-guard.sh` still WARNs (guards against over-relaxing)
- [x] [AC1] [AC3] Implement `isGuardDispatcher` + `samePath`, rewrite the switch as three tiers (`checks_guard.go`)
- [x] [AC4] Confirm the 5 pre-existing tests still pass unchanged (unset/fix/idempotent/missing/foreign)
- [x] [AC1] [AC2] [AC3] Mirror the three tiers in `scripts/install-git-hooks.sh` (`is_guard_dispatcher`, `-ef` for same-inode)
- [x] [AC1] [AC2] [AC3] Mirror the three tiers in `scripts/install-git-hooks.ps1` (`Test-GuardDispatcher`, `Test-SameHooksPath`)
- [ ] [AC5] Add the bats + Pester regression cases for the shell twins
- [ ] Re-run the full `go test ./...`, bats and PSScriptAnalyzer suites

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
