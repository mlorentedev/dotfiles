---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "HARNESS-193-lease-in-git-dir"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-193-lease-in-git-dir

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/3307-lease-in-git-dir`
- [ ] `proposal.md` is complete and acceptance criteria are testable (owner review of the `[AGENT-DRAFT]` sections)
- [ ] No open questions left in `proposal.md` "Risks / open questions" (the overlap policy, A mirror / B remove, decided by the owner and written into AC2)
- [x] ADR-062 drafted alongside the spec (`docs/adr/adr-062-the-lease-lives-in-the-worktree-git-dir.md`)

## Implementation

- [ ] [AC2] Failing test: `worktree.AdminDir(path)` resolves `<repo>/.git/worktrees/<id>` from a linked worktree's `.git` pointer (absolute and relative forms), and answers not-a-worktree for a main checkout, a plain directory and a `.git` pointing into `modules/`.
- [ ] [AC2] Implement `AdminDir` in `cli/internal/worktree` by reading the pointer file, no subprocess.
- [ ] [AC2] Failing test: `LoadMetadata` prefers the admin dir, falls back to the in-tree file (on a non-worktree path too), and returns `nil` with neither; `SaveMetadata` writes the admin dir, handles the in-tree copy per the owner's overlap decision (A mirror / B remove), and refuses a non-worktree path naming it.
- [ ] [AC2] Implement the new `LoadMetadata`/`SaveMetadata`: new constant `LeaseFileName = "dotf-worktree.json"` for the admin dir; `MetadataFileName` (`.dotf-worktree.json`) keeps naming the in-tree file for `excludeFile`/`ensureExclude`, `clean.go` and `audit.go`.
- [ ] [AC6] Rewrite the fixtures that call `SaveMetadata` or write the in-tree file inside a real linked worktree (29 lines in 16 files: `worktree/{add,adopt,detached,sweep,sweep_park,park,clean}_test`, `lane/{lease,handoff,lanes,audit,plan_mutation}_test`, `cmd/{worktree_sweep_lane_skip,lane_wiring,harness_gate_sticky}_test`, `hooks/lane_guard_test`) to the admin-dir lease; keep one fake-directory fixture per package on the legacy path as the fallback's coverage. `go test ./internal/worktree ./internal/lane ./internal/cmd` green.
- [ ] [AC6] Failing test: `dotf worktree list` reason text names a legacy-location lease; implement in `Classify`/`list`.
- [ ] [AC3] Failing test: `dotf lane init` on lanes with in-tree leases reports `changed=<n>` and moves them; a second run reports `changed=0`.
- [ ] [AC3] Implement the migration in `provisionLane`'s adopt and healthy-return paths (one `LoadMetadata` + `SaveMetadata` when the lease is at the legacy location).
- [ ] [AC1] Failing test in the lane package: ACTIVE lane, `git clean -fdx`, then `lane status` says ACTIVE, `worktree list` says ACTIVE, and `git commit` with the guard installed as the pre-commit hook succeeds.
- [ ] [AC4] Failing guard cases in `cli/internal/hooks/lane_guard_test.go`: PARKED lease in the admin dir blocks; ACTIVE lease in the admin dir with no in-tree file passes rules 5 and 6; in-tree-only PARKED lease still blocks (fallback).
- [ ] [AC4] Implement rules 5 and 6 in `git-hooks/lib/lane-guard.sh` (`lease="$git_dir/dotf-worktree.json"`, fallback `$top/.dotf-worktree.json`); `sh -n` and `shellcheck -s sh` clean.
- [ ] [AC1] Make the AC1 test pass (it depends on the guard and the Go seam together).
- [ ] [AC5] Failing test: `isWorktreeParked` recognises an admin-dir PARKED lease from the lane root and a subdirectory, and a legacy in-tree one.
- [ ] [AC5] Implement in `cli/internal/mem/session_end.go` through `readGitdirPointer`.
- [ ] Docs: runbook `guide-persistent-named-lanes.md` (lease location, re-init migrates it, mixed-version note), `docs/lessons/lesson-594-<slug>.md` for the "excluded is not ignored by clean" class and the migration shape.
- [ ] File the follow-up ticket: remove the legacy in-tree read and the `.dotf-worktree.json` exclude lines one release after this ships; link it from `proposal.md` Out of scope.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass (`go build ./... && go vet ./...`, `GOOS=windows go vet ./...`)
- [ ] Lint passes (pinned `golangci-lint run`, `GOOS=windows golangci-lint run ./...`, `shellcheck -s sh git-hooks/lib/lane-guard.sh`)
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
