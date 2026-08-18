---
tags: [spec, tasks, templates]
created: "2026-08-18"
---

# Tasks - TOOL-017-deterministic-review-triage

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/tool-017-deterministic-triage`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] **Open question resolved before implementation starts:** the classification
      marker does not exist yet. Either `.pr_agent.toml` is asked to emit a stable
      one (TOOL-013's surface, coordinate with whoever holds it) or every finding
      is `unclassified` and every PR escalates — fail-closed but useless. This
      gates the value of everything below.

## Implementation

> TDD order. The reproduction harness comes first, because every later task
> depends on it and it is the piece most able to be silently vacuous.

- [ ] `[AC5]` reproduction harness: apply a mutation, **verify by checksum that it landed**, report `did-not-land` as a state distinct from `did-not-reproduce`
- [ ] `[AC5]` red-direction: an invalid mutation must report `did-not-land`, never a clean result — the exact failure two sessions hit on 2026-08-17
- [ ] `[AC2]` `[P]` read reviewer identity + notice markers from `harness/review-attestation.json`; no second copy of the distinction
- [ ] `[AC2]` red-direction: adding a reviewer to the registry changes triage behaviour with no code edit
- [ ] `[AC3]` extract a finding's class from a declared marker; absent marker → `unclassified`
- [ ] `[AC3]` red-direction: a finding whose class appears only in prose is `unclassified`, not parsed
- [ ] `[AC4]` gate `apply` on a successful reproduction for REAL findings
- [ ] `[AC6]` record a refuted REAL finding with its evidence, distinct from a skip
- [ ] `[AC1]` trigger from the reviewer-finished signal (GUARD-003's `workflow_run`)
- [ ] `[AC7]` closing guard: a PR with an undispositioned reviewer comment cannot close or merge
- [ ] `[AC7]` red-direction: a PR with one undispositioned comment fails the guard
- [ ] `[AC8]` every check above has a red-direction test — asserted, not assumed

## Verification

- [ ] `go build ./... && go vet ./... && go test ./...`
- [ ] `bats tests/*.bats`
- [ ] `shellcheck` clean
- [ ] Red-teamed: removing each fix turns its guard red, **and each mutation is confirmed to have landed**
- [ ] Replayed against #1059's real payload: the REAL finding must come back **refuted**, and the multi-line hole must not be claimed as the reviewer's finding

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion, each
with a shell command whose exit 0 is the pass condition.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the
harness, after running `verification` and capturing exit code 0, may set that
terminal state. Every entry starts `pending` with empty `evidence`.
