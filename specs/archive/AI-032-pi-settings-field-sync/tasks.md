---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - AI-032-pi-settings-field-sync

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

> **Retroactive note**: implemented and functionally verified (against synthetic
> fixtures exercising the real extracted script blocks) before this spec folder
> existed, in the same isolated worktree as AI-033 -- the diff crossed the
> spec-gate's 50-LOC threshold organically. Tasks below record the actual order.

## Setup

- [x] Branch created from `origin/main` in an isolated worktree
      (`feat/pi-settings-field-sync`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Added the field-level sync block to `setup-linux.sh`, layered directly
      after the existing seed-if-missing block: guarded on both `PI_SETTINGS_SRC` and
      `PI_SETTINGS_DST` existing, `jq -e` equality check, atomic `mktemp` + `mv` write
- [x] [AC2] Verified by construction: the jq expression is `.enabledModels = $m`,
      scoped to exactly that key -- no other field is ever named in the write path
- [x] [AC3] Verified: re-running the same extracted block against an
      already-converged fixture takes the "already in sync" branch and performs no
      write (`tests/pi-config.bats`, second `eval` in the same test)
- [x] [AC4] [P] Added the PowerShell parity block to `setup-windows.ps1`, matching
      shape: `Test-Path` guards, `ConvertFrom-Json`/`ConvertTo-Json -InputObject`
      (not piped, to avoid the single-element array collapse), try/catch matching the
      adjacent packages-reconcile block's idiom
- [x] [AC5] Wrote `tests/pi-config.bats`: one test extracts and `eval`s the real
      `setup-linux.sh` block against temp-file fixtures (not a reimplementation);
      one test structurally asserts the `setup-windows.ps1` block's guards and the
      `-InputObject` (non-piped) usage, matching this repo's existing
      grep-based-parity pattern for PowerShell (bats cannot execute pwsh)
- [x] Verified `shellcheck setup-linux.sh` and `bash -n` clean on the new block; the
      PowerShell addition is ASCII-only (grep-verified, no PSScriptAnalyzer available
      in this environment)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a
      non-vacuous verification command
- [x] Type checks pass (`go build ./... && go vet ./...` in `cli/`, unaffected by this
      diff, re-run to confirm no incidental breakage)
- [x] Lint passes (pre-commit hooks: secrets scan, bats @test names, doc-path check,
      message format)
- [x] No unrelated changes in the diff — isolated worktree branched fresh off
      `origin/main`, patch-applied to include only the 3 touched files
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-032-pi-settings-field-sync/features.json`):

```json
[
  {
    "id": "AI-032-pi-settings-field-sync-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
