---
tags: [spec, tasks, templates]
created: "2026-09-26"
---

# Tasks - AI-046-pi-nan-provider

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/pi-nan-provider`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (owner chose A, 2026-09-26)

## PR-A: install the package next to today's provider

- [x] Measure provider coexistence and offline start on pi 0.87.1 (`verification.md`)
- [x] [AC1] Declare `npm:@gtrabanco/pi-nan-provider@0.7.0` in `ai/pi/packages.json` (covered by the existing pin contract, `tests/pi-packages.bats`)
- [x] [AC4] Failing test: the deploy entry for `nan-provider.json` is `merge` and sets `mediaMcp: false`
- [x] [AC4] `ai/pi/nan-provider.json` plus its `ai/deploy.json` entry
- [x] Document the package in `ai/pi/README.md`
- [ ] PR opened, triaged, merged by the owner

## PR-B: the package owns the ids

- [ ] [AC2] CI test: install the pinned package, dummy key, assert every `nan/*` id in `enabledModels` is in `pi --list-models nan`
- [ ] Live prompt test without our `timeoutSeconds: 300` and `compat.supportsDeveloperRole`; keep them as `modelOverrides` if NaN needs them
- [ ] [AC3] Remove the NaN models from `models.json`, and rework its consumers (`pi-config.bats`, `guard-pi-models-schema.bats`, `reviewer-pool.bats`, `checks_model_limits.go`, `render_test.go`, `deploy_test.go`, `model-pins.json`) with the test-deletion evidence ledger
- [ ] #1772's opencode/pi context parity test must not go vacuous: fail on an empty intersection, or retarget it at the package snapshot
- [ ] [AC6] Measure the model-switch guard once
- [ ] Announce the first `dotf pi packages apply` to peers (it also removes pi-memory on msi, HARNESS-139 AC8)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test or recorded measurement
- [ ] `verification.md` filled in
- [ ] Independent adversarial review before archive

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-046-pi-nan-provider/features.json`):

```json
[
  {
    "id": "AI-046-pi-nan-provider-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
