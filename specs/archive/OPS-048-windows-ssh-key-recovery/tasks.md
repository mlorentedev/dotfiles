---
tags: [spec, tasks, templates]
created: "2026-09-24"
---

# Tasks - OPS-048-windows-ssh-key-recovery

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/windows-ssh-key-recovery`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Add failing Pester coverage for client-key fingerprint validation and ACL reconciliation.
- [x] [AC1] Register the Bitwarden file secret and implement the Windows client-key guard.
- [x] [P] [AC2] Add failing Pester coverage for key validation, duplicate-free authorization, and preservation of unrelated entries.
- [x] [AC2] Implement idempotent Windows OpenSSH server reconciliation with test seams.
- [x] [P] [AC3] Add structural assertions for stable mesh and LAN SSH aliases.
- [x] [AC3] Add the aliases with the dedicated identity and connection-hardening options.
- [x] [P] [AC4] Add runbook assertions for bootstrap, reconcile, recovery, rotation, DR, and secret-safety sections.
- [x] [AC4] Write the operator runbook and cross-link the secrets-governance documentation.
- [x] [AC1] Provision the existing private key into its declared Bitwarden item through stdin and verify resolution without displaying it.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder: `mlorentedev/dotfiles#1659`

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Feature definitions are maintained in the sibling `features.json`.
