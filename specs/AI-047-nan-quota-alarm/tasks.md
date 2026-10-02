---
tags: [spec, tasks, templates]
created: "2026-09-26"
---

# Tasks - AI-047-nan-quota-alarm

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/nan-quota-alarm`
- [x] Owner chose (A), 2026-09-26
- [x] Measure `/v1/usage`: window, paging, UTC dates, accepted User-Agents (`verification.md`)

## PR-1: the check

- [x] [AC1] [AC3] Failing tests for `nanquota.Evaluate` against a recorded fixture, then the package
- [x] Declared table `harness/nan-quotas.json`, closed-world (metered + unmetered)
- [x] [AC3] [AC4] [AC5] Failing doctor tests against fixtures (unserved id, outage, no key, bw gate, key never printed), then `checks_nan_quota.go` and the `HTTPGetBody` seam
- [x] [AC2] Live run, recorded
- [x] PR opened, triaged, merged by the owner (#1772, which it depended on, merged 2026-09-27)

## PR-1b: watch the account, not the bindings (2026-10-01)

- [x] [AC7] Failing tests: an unbound metered model at 83% WARNs, an undeclared model with usage WARNs, a tier-hidden unbound model is ignored
- [x] [AC1] [AC2] Metered under the threshold is INFO (shown by default), unmetered is PASS (shown with `--verbose`)
- [x] [AC7] Doctor test at default verbosity: `TestCheckNaNQuota_WatchesMeteredModelsNothingBinds`
- [x] [AC2] Live run, recorded

## PR-2: hermes

- [x] [AC6] Vault: remove the single-pool quota section from `00_meta/agents/scripts/budget-report.sh` and its false "no usage endpoint" header; point the digest reader at `dotf doctor`
- [x] Measure whether `/v1/usage` is per key or per member — moot as deployed: one key (resolved by inspection, `proposal.md` Risks)

## Closing

- [x] Every acceptance criterion covered by a test or a recorded measurement
- [ ] Independent adversarial review before archive

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-047-nan-quota-alarm/features.json`):

```json
[
  {
    "id": "AI-047-nan-quota-alarm-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
