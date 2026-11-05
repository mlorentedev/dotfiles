---
tags: [spec, tasks, templates]
created: "2026-11-05"
id: "PLAT-008-workspace-root-contract"
type: spec-tasks
status: active
owner: manu
---

# Tasks - PLAT-008-workspace-root-contract

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from `origin/main`: `feat/3191-workspace-root-contract`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [ ] Run `/spec check PLAT-008-workspace-root-contract`; expected: `PASS`
- [ ] Set `proposal.md` status to `implementing` only after the consistency check passes

## Implementation

### PR 1 - project boundary, derived paths, persistence and trust

- [ ] [AC1] Add table cases to `cli/internal/env/env_test.go` for `derived_from`: resolved base override, compatibility fallback, unknown base, cycle, absolute child, and default-equivalence. Run `cd cli; go test ./internal/env -run 'TestResolveDerived|TestDerivedDefaults'`; expected: FAIL because the schema and resolver do not exist.
- [ ] [AC1] Add `DerivedFrom` to `cli/internal/env.contractVar` and implement dependency resolution in `cli/internal/env/resolve.go`; reject unknown bases, cycles, absolute children and unresolved tokens. Run the focused env tests; expected: PASS.
- [ ] [AC1] Add real-contract tests proving `PROJECTS_PATH`, `DOTFILES_REPO_DIR`, `VAULT_PATH`, `HIVE_VAULT_PATH`, `WORKTREE_ROOT` and `COPILOT_WORKTREE_ROOT` resolve identically from default and machine-overridden boundaries. Run `cd cli; go test ./internal/env -run 'TestRealContract.*Projects|TestDefaultCheckoutDirMatchesTheContract'`; expected: FAIL before the contract changes.
- [ ] [AC1] Update `env-contract.json` with `PROJECTS_PATH`, additive `derived_from` relationships and compatibility defaults. Keep explicit overrides higher precedence. Run the focused real-contract tests; expected: PASS.
- [ ] [AC1] Add failing tests in `cli/internal/doctor/checks_env_persist_test.go` for persisted drift being `FAIL`, not `WARN`, and for a current process retaining a stale dotf-generated value after the resolved value changes. Run `cd cli; go test ./internal/doctor -run TestCheckPersistedEnv`; expected: FAIL on status/seam behavior.
- [ ] [AC1] Extend persisted-environment diagnostics to distinguish dotf-generated provenance from unmanaged environment overrides and name the fresh-process requirement. Run the focused doctor tests; expected: PASS.
- [ ] [AC1] Add deploy-path tests proving Copilot and agy trust templates render `{PROJECTS_PATH}`, `{WORKTREE_ROOT}` and `{COPILOT_WORKTREE_ROOT}` without literal `Projects/Workspace`. Run `cd cli; go test ./internal/deploy -run 'Test.*(Copilot|Agy).*Path'`; expected: FAIL before template changes.
- [ ] [AC1] Update `ai/copilot/config.json` and `ai/agy/settings.json`; run `dotf deploy --dry-run` against redirected HOME fixtures or the existing deploy tests. Expected: managed trust entries change, unrelated user entries survive.
- [ ] [AC1] Run `gofmt -w cli/internal/env cli/internal/doctor`; then `cd cli; go test ./internal/env ./internal/doctor ./internal/deploy` and `go vet ./internal/env ./internal/doctor ./internal/deploy`; expected: PASS.
- [ ] [AC1] Commit PR-1 slice with only contract/persistence/trust behavior and update `features.json` evidence command for AC1.

### PR 2 - side-effect-free reconciliation inventory

- [ ] [P] [AC2] [AC5] [AC6] Add synthetic Git fixtures in `cli/internal/worktree/reconcile_test.go`: healthy, registered-path-missing with one moved candidate, prunable-without-content, missing admin metadata, hidden `.worktrees`, outside-boundary, locked, dirty, ahead and recent-session cases. Redirect HOME/state for every case.
- [ ] [AC2] [AC6] Add a failing test that the plan returns stable JSON classifications and performs zero filesystem or Git mutations. Run `cd cli; go test ./internal/worktree -run TestReconcilePlan`; expected: FAIL because reconciliation does not exist.
- [ ] [AC2] [AC6] Implement `cli/internal/worktree/reconcile.go` with typed classifications and deterministic ordering. Inventory Git administrative records, declared roots and linked-checkout `.git` files below `PROJECTS_PATH`; ambiguous or unreadable state returns an error.
- [ ] [AC6] Add `dotf worktree reconcile --plan [--json] --repo <path>` in `cli/internal/cmd/worktree.go` plus command tests in `cli/internal/cmd/worktree_reconcile_test.go`. Run focused command tests; expected: PASS with byte-stable records.
- [ ] [AC4] Add doctor tests for hidden dot-prefixed linked checkouts and workspace-boundary violations. Run `cd cli; go test ./internal/doctor -run TestCheckWorktreeRoots`; expected: FAIL before the check exists.
- [ ] [AC4] Implement the read-only doctor section from the same reconciliation plan; an unreadable inventory must be `FAIL`, never `SKIP` or empty success.
- [ ] [AC2] [AC5] [AC6] Run `cd cli; go test ./internal/worktree ./internal/cmd ./internal/doctor`; expected: PASS without touching any real worktree.
- [ ] [AC2] Commit PR-2 slice with plan/inventory only; no repair, move, prune or quarantine mutation is callable.

### PR 3 - repair-in-place checkpoint

- [ ] [AC2] Add a real-Git test that creates a linked worktree, moves its directory with the filesystem, observes the old path as prunable, and requires a repair plan. Run `cd cli; go test ./internal/worktree -run TestReconcileRepair`; expected: FAIL before apply support.
- [ ] [AC2] Add refusal tests for zero candidates, multiple candidates, mismatched main repository, missing admin metadata, active/recent sessions, locked, dirty and ahead worktrees. Expected: every unsafe case is skipped or rescue-required with a specific reason.
- [ ] [AC2] Implement repair apply as a separate verb/checkpoint using `git worktree repair`; verify the checkout and main repository agree on path and branch after the command. Do not expose move or prune in this checkpoint.
- [ ] [AC2] Add an orphan/rescue-required terminal classification. Re-running apply without an operator resolution must report the same non-mutating blocker, not create new changes.
- [ ] [AC2] Add an idempotence test: first apply repairs exactly one fixture; second plan and apply report zero repair changes.
- [ ] [AC2] Run `cd cli; go test ./internal/worktree ./internal/cmd`; expected: PASS. Run `git status --short` in the fixture parent to prove no test escaped its temp root.
- [ ] [AC2] Commit PR-3 slice; document that production `.worktrees` repair is a later attended operation after the released command is installed.

### PR 4 - safe relocation, leases and Windows path budget

- [ ] [P] [AC4] [AC5] Add `cli/internal/worktree/pathbudget_test.go` cases for Windows and POSIX separators, exact 260 refusal, 259 acceptance, less-than-20 warning, custom paths, and repositories with no tracked files. Expected: FAIL before budget evaluation exists.
- [ ] [AC4] Implement projected-path budget calculation from target root, repo, slug and longest tracked relative path; wire it into `AddWithRunner` before any directory or branch is created.
- [ ] [AC3] Add a real-Git relocation test: repair a moved worktree, make it clean and idle, move it with Git into `WORKTREE_ROOT`, preserve branch/content/registration, write or retain the lease, and require a clean second plan.
- [ ] [AC3] Add refusal tests proving active, locked, dirty, ahead, current-directory and target-exists worktrees never move.
- [ ] [AC3] Implement relocation apply with `git worktree move`, post-move verification and lease adoption. Pruning is allowed only when the inventory proves no recoverable directory points at the stale registration.
- [ ] [AC3] [AC4] Add command JSON/prose tests for moved, skipped, warning and rescue-required outcomes on Windows and POSIX paths.
- [ ] [AC3] [AC4] Run `cd cli; go test ./internal/worktree ./internal/cmd ./internal/doctor`; expected: PASS.
- [ ] [AC1] [AC2] [AC3] [AC4] Run `cd cli; go test ./...` and `go vet ./...`; expected: PASS. Do not run golangci-lint locally on this workstation.
- [ ] [AC1] [AC2] [AC3] [AC4] Run a redirected-home convergence smoke twice; expected: first run reports declared changes, second run reports `changed=0`.
- [ ] [AC3] Commit PR-4 slice. Only after all four slices are released may an attended migration plan be run against the real `.worktrees` directories.

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

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-008-workspace-root-contract/features.json`):

```json
[
  {
    "id": "PLAT-008-workspace-root-contract-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
