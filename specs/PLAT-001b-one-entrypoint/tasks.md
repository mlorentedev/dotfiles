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

### PR 2a — the engine: `dotf converge [--plan]` and the records-mirror reconciler (#1843 B6)

> Split from the original PR 2, which did not fit the ~300-line cap: the engine and one reconciler here, the instruction files and the bindings in 2b. Apply mode landed with the engine because plan and apply are one code path; the persisted report stays in PR 3.

- [x] [AC1] Failing test: `PlanMirror` counts what `Mirror` would write and creates nothing, the deploy dir included; implemented on `Mirror`'s own walk with a `dryRun` flag
- [x] [AC5] Extract `internal/platform` (`Supports`, `Unknown`) from the tool catalog, so the catalog and the reconcilers share one matcher; the catalog's tests still pass
- [x] [AC1] [AC4] [AC5] Failing tests for the runner: a plan applies and probes nothing; an unlisted OS is `skipped` naming it; a failed probe fails the run naming the reconciler and the reconcilers after it are not reached
- [x] Implement `internal/converge`: `Reconciler` (name, platforms, `Reconcile(env, dryRun)`, mandatory `Probe`), `Run`, `Report`, and the ordered `Registry`
- [x] [AC1] [AC3] Failing tests, then the `records-mirror` reconciler: the plan writes nothing, the apply passes its probe, a second plan reports 0 changes
- [x] [AC1] `cmd/converge.go`: `dotf converge [--plan] [--repo]`; a plan on an empty HOME writes nothing, a second apply reports 0 changed
- [x] On the Mac: `go run ./cmd/dotf converge --plan` reports `records-mirror` with 70 files to write and leaves `~/.dotfiles` untouched

### PR 2b — the agent instruction files, driven by `agents.presence[]` (#2016, #1843 B11)

> Pivot from the first plan: no `ai/deploy.json` entry. A new deploy field needs a manifest version the installed `dotf` cannot read (#1814 class), and a `replace` entry would rewrite the `AGENT-PRESENCE` region `dotf harness presence` owns on every run, so a second converge could never report zero changes (AC3). `harness/manifest.json` `agents.presence[]` already declares each instruction file (`source`, `file`, `requires_command`); it is the registry.

- [ ] [AC2] Failing test, then the `records-instructions` reconciler: for each presence entry, the deployed file outside the `AGENT-PRESENCE` markers must equal `source`; a change writes `source` plus the existing region byte for byte; `requires_command` absent → `skipped`. It reuses the existing marker split, not a second parser
- [ ] [AC3] Failing test: a HOME holding base plus region plans 0 changes
- [ ] [AC2] The probe is "outside-marker content equals `source`", which subsumes the shell's `grep 'First, read AGENTS.md'` check
- [ ] [AC2] `records-presence`: the presence injection in-process, after the instruction files (it skips a file that does not exist yet). It also runs on Linux, where `setup-linux.sh` never ran it
- [ ] Delete the CLAUDE.md force-copy and the bulk `ai/claude/*` copy from both twins (ADR-020 §5). The opencode, pi and copilot copy blocks follow in their own B11 rows if the cap is reached
- [ ] [AC2] Doctor FAIL when an agent binary is on PATH without its instruction file (#2016's guard; it covers copilot, whose file is deployed only once copilot exists)

### PR 2c — skills and hook bindings

- [ ] `records-skills`: `compile-harness.sh --deploy`, planned with `--check`, behind the shared seam that defaults to "not run" (lesson 335); recorded as a port target
- [ ] `records-bind`: the `harness bind` logic with its dry-run

### PR 3 — the persisted report (#1843 B7)

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
