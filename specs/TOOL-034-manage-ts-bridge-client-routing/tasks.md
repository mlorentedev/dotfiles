---
tags: [spec, tasks, templates]
created: "2026-11-05"
id: "TOOL-034-manage-ts-bridge-client-routing"
type: spec-tasks
status: active
owner: manu
---

# Tasks - TOOL-034-manage-ts-bridge-client-routing

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from `origin/main`: `feat/3219-ts-bridge-client-routing`
- [x] Work-gate verified: #3219 is open, assigned and In Progress
- [x] ADR-058 and Regla-del-3 audit are complete
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### PR 1 - Generic checksummed archive source

- [x] [P] [AC1] [AC2] Add failing catalog tests for a `github-archive` source with exact per-platform member declarations and strict validation.
- [x] [AC1] [AC2] Add failing installer tests for zip/tar.gz success, checksum-before-extract, missing/duplicate/directory/symlink member, unsupported archive and unchanged destination on failure.
- [x] [AC1] [AC2] Implement streaming exact-member extraction and atomic placement by reusing the existing release fetch/checksum/version-floor flow.
- [x] [AC3] Preserve old-reader safety: unknown source types remain explicit skips and do not block other tools.
- [x] [AC1] [AC2] Run focused tools tests, full `internal/tools`, vet and formatting.
- [x] [AC1] [AC2] Apply independent review findings: atomic mode-correct placement, plan/refusal archive parity and asset/member platform parity.
- [x] Open PR 1 referencing ADR-058 and this spec: #3231.

### PR 2 - Role selection and Git drop-in

- [x] [AC4] Add `ts-bridge-client` to known machine roles with add/remove/idempotence tests.
- [x] [AC4] Add optional role selection to catalog tools; no-arg install and converge filter, named install remains an explicit override.
- [x] [AC1] [AC4] Add the v1.20.0 ts-bridge `github-archive` entry with Windows/POSIX assets, checksums and member paths.
- [x] [AC5] [AC6] Add the always-included optional Git drop-in path and a role-gated reconciler that creates/removes only `ts-bridge.gitconfig`.
- [x] [AC4] [AC5] [AC6] Prove non-role machines and GitHub remotes remain direct; second converge changes nothing.

### PR 3 - Doctor and end-to-end evidence

- [ ] [AC7] Add role-gated doctor tests for binary pin, listener-down, listener-ready/destination-failed and TLS-valid destination success.
- [ ] [AC7] Implement bounded health probes without printing profile/credential values.
- [ ] [AC8] Run a synthetic bridge-client converge twice and record zero changes.
- [ ] [AC7] [AC8] Record live consequence probes for health, Gitea TLS and Git remote access; finish verification and adversarial review.

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

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-034-manage-ts-bridge-client-routing/features.json`):

```json
[
  {
    "id": "TOOL-034-manage-ts-bridge-client-routing-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
