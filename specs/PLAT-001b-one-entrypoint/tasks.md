---
tags: [spec, tasks, templates]
created: "2026-10-06"
---

# Tasks - PLAT-001b-one-entrypoint

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `docs/plat-001b-one-entrypoint-spec` (PR 1); each later PR gets its own branch from `main`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" (D4 and D2's reach close when the ADR-045 PR merges)

## Implementation

> One PR per block, each under ~300 executable lines, each landing with the guard that would have caught its defect. Go work verifies with `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` and the pinned `golangci-lint`; shell work with shellcheck and bats under bash 3.2 and zsh.

### PR 1 — design (docs only)

- [x] ADR-045 `proposed`, with #2013 D4 and D2's reach as its open questions
- [x] This spec: proposal, tasks, features
- [ ] Merge = acceptance of ADR-045 (owner)

### PR 2 — `dotf converge --plan` and the records reconciler (#1843 B6, #2016)

- [ ] [AC1] Failing test: `--plan` over a fake registry lists each reconciler that applies to the OS with its action, and the temp HOME is byte-identical afterwards
- [ ] [AC1] Implement `internal/converge`: the `Reconciler` interface (name, platforms, plan, apply, probe), the ordered registry, and `cmd/converge.go` with `--plan`
- [ ] [AC5] Failing test: a reconciler that does not list `runtime.GOOS` is reported `skipped` naming the OS; implement through the `platforms` vocabulary `packages.json` uses (#2001)
- [ ] [AC2] Failing test: the records reconciler on a HOME without `~/.claude/CLAUDE.md` plans `mirror`, `instructions` and `bind`
- [ ] [AC2] Implement the records reconciler over `harness.Mirror`, the instruction-file deploy and `harness bind`; the `compile-harness.sh --deploy` call is its one shell step, recorded as a port target
- [ ] Refactor: one report type shared with `dotf deploy`'s `Outcome` where the fields match; no second status enum next to doctor's

### PR 3 — apply mode and the persisted report (#1843 B7)

- [ ] [AC4] Failing test: a reconciler whose probe fails makes `converge` exit non-zero, names it, and stops the reconcilers after it
- [ ] [AC4] Implement apply: plan, apply, probe, in registry order (ADR-041 decision 4)
- [ ] [AC3] Failing test: a second run on a converged temp HOME reports zero changes and writes the report under the user state directory
- [ ] [AC3] Implement the report (JSON, one entry per reconciler, the run's exit status)
- [ ] [AC2] On the Mac: `dotf converge` deploys `~/.claude/CLAUDE.md` and the skills; `dotf doctor` no longer fails on the Claude instruction file

### PR 4 — one entrypoint: `install.sh` and `install.ps1` at the root

- [ ] [AC7] Failing test: no live file names `install-dotf.sh`, `install-dotf.ps1` or `DOTFILES_SKIP_SETUP` (allow-list: `docs/adr/` including audits, `docs/lessons/`, `specs/` — live specs describe the migration they deliver — and `CHANGELOG.md`)
- [ ] [AC6] `git mv scripts/install-dotf.{sh,ps1}` to the root `install.{sh,ps1}`, replacing the old `install.sh`; the standalone path ends in `exec dotf converge "$@"`, the sourced `install_dotf` contract is unchanged
- [ ] [AC6] bats (bash 3.2 and zsh) and Pester: a bad checksum and an unreachable release fail and place nothing; the hand-off execs `dotf converge` with the arguments
- [ ] [AC7] Move every live reference: setup twins, `checks_tools.go`, `stdout_contract_test.go`, the vault-maintenance scripts, README, `cli/README.md`, SECURITY.md, the release runbook
- [ ] Add the checkout reconciler (clone if absent, fast-forward under ADR-019 D2), first in the registry

### PR 5 — legacy reconcilers and `dotf update`

- [ ] [P] Failing test: on linux and windows the legacy reconciler runs the setup script after every native reconciler and plans `opaque`; on darwin it is absent from the registry
- [ ] Implement the legacy reconciler
- [ ] [AC8] Failing test: `dotf update` runs `dotf converge` after a fast-forward, keeps exit 0 on every non-actionable case, and exits non-zero only on a converge failure
- [ ] [AC8] Route `dotf update` through `converge`; keep `DOTFILES_SELFUPDATE_SETUP_CMD` as an override until #1843 A5 decides the scheduled path

### PR 6 — from-zero proof (#2013 X1)

- [ ] [AC9] CI job on clean `macos-latest` and `ubuntu-latest`: `install.sh`, `dotf doctor`, then a second `dotf converge` that reports zero native changes. It asserts the first run converged before it checks idempotence (lesson 328). Runs on pull requests to `main` and on push to `main`, so the default branch carries a run the verification can read; non-required until green
- [ ] Runbook `docs/runbooks/guide-new-machine.md`: the one-liner per OS, what `--plan` shows, how to read the report, what `skipped` on darwin means

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

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-001b-one-entrypoint/features.json`):

```json
[
  {
    "id": "PLAT-001b-one-entrypoint-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
