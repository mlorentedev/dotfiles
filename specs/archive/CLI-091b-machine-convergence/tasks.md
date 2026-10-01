---
tags: [spec, tasks, templates]
created: "2026-09-29"
---

# Tasks - CLI-091b-machine-convergence

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/tools-install-dry-run`
- [x] `proposal.md` is complete for the rows in flight (B4, #1381)
- [x] No open questions left in `proposal.md` "Risks / open questions" for those rows

## Implementation

Row B4 and #1381 (first PR):

- [x] [P] [AC2] Failing tests: `TestResolveCatalogPath*` in `cli/internal/env/env_test.go`, and `TestToolsList_CheckoutCatalogWins` and `TestToolsList_MirrorWithoutCheckout` in `cli/internal/cmd/tools_test.go`
- [x] [AC2] `env.ResolveCatalogPath`, used by `dotf tools list` and `dotf tools install`
- [x] [P] [AC1] Failing tests: `TestInstallerPlan` in `cli/internal/tools/install_test.go` and `TestToolsInstall_DryRun` in `cli/internal/cmd/tools_test.go`
- [x] [AC1] `Installer.Plan` (same probe and `decideAction` as `Install`) and `dotf tools install --dry-run`

Later rows add their tasks here when they start: B5 (#1262, #1265), then B1-B3 and B6-B10.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (golangci-lint 2.12.2, 0 issues; CI `cli-lint` green on #1848)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1848, `a94ae34`)

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-091b-machine-convergence/features.json`):

```json
[
  {
    "id": "CLI-091b-machine-convergence-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
