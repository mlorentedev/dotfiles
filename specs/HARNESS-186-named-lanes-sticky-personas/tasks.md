---
tags: [spec, tasks, templates]
created: "2026-11-07"
id: "HARNESS-186-named-lanes-sticky-personas"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-186-named-lanes-sticky-personas

> TDD order. One task = one focused commit. Tick as you go.
> Designed to be self-contained for implementation by any model tier: every task specifies exact files, function signatures, Adversarial Review guards (`B0`–`B7`, `M1`–`M6`, `m1`–`m4`), and verification commands.

## Setup

- [x] Branch created from main: `feat/3255-named-lanes-sticky-personas` in `C:\Users\mlorente\Projects\Workspace\worktrees\dotfiles\feat-lane-primary`
- [x] `proposal.md` is complete and acceptance criteria (`AC1`–`AC6`) are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### Batch 1: Layer 3 (`meeseek-<role>-<fruit>` + `pi` Fallback) & Layer 2 (Sticky Persona Gate + Windows Atomic State) — DONE

- [x] [P] [AC4] Write failing unit tests in `cli/internal/agent/fruits_test.go` for 64-fruit permutation without replacement (`meeseek-<role>-<fruit>`), `DOTF_WORKER_ID` propagation, and `piBackend` `--provider nan` + `StatusPoolUnavailable` classification (`cd cli && go test ./internal/agent/... -run "TestFruit|TestParseWorkerID|TestDispatch_PopulatesMeeseek|TestPiBackend"`)
- [x] [AC4] Implement `cli/internal/agent/fruits.go` (`Fruits`, `FruitDeck`, `NextWorkerID`, `ParseWorkerID`) and update `cli/internal/agent/agent.go`, `cli/internal/agent/dispatch.go`, `cli/internal/agent/backends.go`, and `cli/internal/agent/process.go` (fixes **B0**, **B7**, **m1**)
- [x] [P] [AC3] [AC4] Write failing unit tests in `cli/internal/harness/sticky_test.go` and `cli/internal/cmd/harness_gate_sticky_test.go` for read-only tool exemption (`Read`, `Grep`, `Glob`, `LS`), sticky primary persona per `(scope, branch)` in `$STATE_DIR/gate/<scope>.persona.json`, `DOTF_WORKER_ID` (`meeseek-<role>-<fruit>`) resolution in `effectiveRole()`/`loadGatePersona()`, and 16-goroutine concurrent Windows atomic replace (`cd cli && go test ./internal/harness/... ./internal/cmd/... -run "TestDecide_ExemptsReadOnly|TestStickyPersona|TestAtomicWriteFile|TestHarnessGateStickyPersona"`)
- [x] [AC3] [AC4] Implement `IsReadOnlyTool`, `PersonaStatePath`, `LoadStickyPersona`, `BindStickyPersona`, `atomicWriteFile` (with `atomicWriteMu` + 15-attempt Windows rename retry), and `meeseek-<role>-<fruit>` role stripping in `cli/internal/harness/dispatch.go`, `cli/internal/harness/gate.go`, and `cli/internal/cmd/harness_gate.go` (fixes **B2**, **B4**, **B7**)

---

### Batch 2: Layer 1 (Persistent Named Lanes `init` / `switch` / `park` & In-Place `worktree` / `handoff` / `mem` Lifecycle)

- [x] [P] [AC1] [AC2] **Task 5 (RED — Write failing tests for Named Lanes & In-Place Park):**
  - **Files to create:**
    1. `cli/internal/worktree/park_test.go`
    2. `cli/internal/lane/lanes_test.go`
  - **Exact behaviors to assert in `cli/internal/worktree/park_test.go` (`TestLaneParkInPlaceAndTipVerification`):**
    1. Extend `Metadata` in `cli/internal/worktree/worktree.go` with `Lane string` (`json:"lane,omitempty"`), `State string` (`json:"state,omitempty"` — `"ACTIVE"` or `"PARKED"`), and `Persona string` (`json:"persona,omitempty"`), and add `StateParked LifecycleState = "PARKED"` and `StateParkable LifecycleState = "PARKABLE"`.
    2. Verify `Classify()` in `cli/internal/worktree/worktree.go`: when `meta.Lane != ""` and `meta.State == "PARKED"`, returns `StateParked`; when `meta.Lane != ""` and `meta.State == "ACTIVE"` and `wt.Status == StatusClean` and `wt.PRMerged == true` and lease is expired, returns `StateParkable` (even though `meta.ReapOK == false`, fixing **B5**).
    3. Verify `Park(repoRoot, wtPath string, opts DoneOptions) (DoneResult, error)` in `cli/internal/worktree/done.go`:
       - Fails closed if `wt` is `StatusDirty` or `StatusIgnored`.
       - Calls `branchLanding(absRepo, absWT, opts.LandedHeads)` (fixing **M3**): if a local unpushed commit was added on `<branch>` *after* the PR squash-merged (so current `HEAD` SHA is neither an ancestor of `origin/main` nor in `LandedHeads`), `Park()` **refuses** with an error and does NOT detach or delete the branch.
       - When `branchLanding` confirms `HEAD` is landed: runs `git -C <wtPath> checkout --detach origin/main` (or `origin/master`), deletes the old local branch (`git -C <repoRoot> branch -D <oldBranch>`) and records its 40-char `BranchSHA` in `DoneResult`, removes `.dotf-lane-prompt.md` if present (fixing **M4**), updates `.dotf-worktree.json` in-place (`State: "PARKED"`, `Issue: 0`, `Persona: ""`, `ReapOK: false`), and **leaves `<wtPath>` directory intact on disk** (never calls `removeWorktree` or `GateF`, fixing **B5**).
       - Verify `Done()` in `cli/internal/worktree/done.go` delegates to `Park()` whenever `meta.Lane != ""`.
       - Verify `SessionEnd` in `cli/internal/mem/session_end.go` returns a clean no-op when `.dotf-worktree.json` in `cwd` has `State == "PARKED"` (fixing **B3**).
  - **Exact behaviors to assert in `cli/internal/lane/lanes_test.go` (`TestLaneInitIdempotent` & `TestLaneSwitchRoutingAndWIPCap`):**
    1. `InitLanes(repoRoot, worktreeRoot string) (changed int, err error)`:
       - First call creates `feat-lane-primary`, `feat-lane-secondary`, and `fast-track-lane` under `<worktreeRoot>/<repo>/` detached at `origin/main` with `.dotf-worktree.json` (`lane: <name>, state: "PARKED", reap_ok: false`) and returns `changed == 3`.
       - Second call on the same repo returns `changed == 0` (idempotent).
    2. `RouteBranchToLane(repoRoot, worktreeRoot, branch string) (laneName, lanePath string, err error)` (fixing **M5**):
       - Maps `feat/`, `refactor/`, `perf/`, `build/` -> `feat-lane-primary` if `PARKED`, else `feat-lane-secondary` if `PARKED`, else fails closed with WIP limit error (`both feat-lane-primary and feat-lane-secondary are ACTIVE`).
       - Maps `fix/`, `hotfix/`, `chore/`, `docs/`, `ci/`, `test/`, `revert/` -> `fast-track-lane` if `PARKED` (or spills over to `feat-lane-secondary` if `fast-track-lane` is `ACTIVE` and `feat-lane-secondary` is `PARKED`).
  - **Run to verify RED:**
    `cd cli && go test ./internal/worktree/... ./internal/lane/... -run "TestLaneInitIdempotent|TestLaneSwitchRoutingAndWIPCap|TestLaneParkInPlaceAndTipVerification" -count=1`

- [x] [AC1] [AC2] **Task 6 (GREEN — Implement `Park()`, `InitLanes()`, `SwitchLane()`, `handoff.go` reuse, and `SessionEnd` guard):**
  - **Files to modify/create:**
    1. `cli/internal/worktree/worktree.go`: Add `Lane`, `State`, `Persona` to `Metadata`; add `StateParked` and `StateParkable` to `LifecycleState`; update `Classify()` so named lanes (`meta.Lane != ""`) transition between `StateParked`, `StateActive`, and `StateParkable` (fixing **B5**, **M2**).
    2. `cli/internal/worktree/done.go` & `cli/internal/worktree/sweep.go`: Implement `Park(repoRoot, wtPath string, opts DoneOptions) (DoneResult, error)` using `filelock.Lock(DefaultLockPath(), time.Minute)` (fixing **M1**), `branchLanding` SHA check (fixing **M3**), `git checkout --detach`, `git branch -D <oldBranch>`, `.dotf-lane-prompt.md` removal (fixing **M4**), and `.dotf-worktree.json` reset to `PARKED`. In `Done()`, if `meta.Lane != ""`, call `Park()`. In `Sweep()`, for `StateParkable` entries, call `Park()` and release `refs/dotf/claims/<issue>`.
    3. `cli/internal/mem/session_end.go` & `cli/internal/mem/handoff.go`: In `SessionEnd`, check `.dotf-worktree.json` in `cwd`; if `State == "PARKED"`, return `nil` immediately (fixing **B3**). Add `ArchiveAndPruneThread(projectDir, branch string) error` under `LockHandoffMemory` that moves `memory/threads/<branch>.md` to `memory/archive/` and removes its active entry from `MEMORY.md`, called by `dotf lane park`.
    4. `cli/internal/lane/named.go` & `cli/internal/lane/handoff.go` & `cli/internal/cmd/lane.go`: Implement `InitLanes`, `RouteBranchToLane`, `SwitchLane`, and wire CLI subcommands `dotf lane init`, `dotf lane switch <branch>`, and `dotf lane park [lane]`. In `handoff.go:387`, when a `PARKED` named lane exists for the target repo, switch it in-place (`SwitchLane`) and call `b.Open` instead of `b.Create` (fixing **B1**).
  - **Run to verify GREEN:**
    `cd cli && go test ./internal/worktree/... ./internal/lane/... ./internal/mem/... -run "TestLaneInitIdempotent|TestLaneSwitchRoutingAndWIPCap|TestLaneParkInPlaceAndTipVerification" -v -count=1`

---

### Batch 3: Layer 4 (`pre-commit` Scoped Lane Guard, `board-pickup.sh` CWD Detach, & Forge `Environment` Provenance)

- [x] [P] [AC5] **Task 7 (RED — Write failing tests for `pre-commit` lane guard & `board-pickup.sh`):**
  - **File to create:** `cli/internal/hooks/lane_guard_test.go` (`TestPreCommitLaneGuard`)
  - **Exact behaviors to assert (fixes B6, M6, m4):**
    1. Commits in a repository WITHOUT `.dotf/lane.json` or with `.obsidian/` directory on `main`/`master` succeed (`exit 0`).
    2. In a repository WITH `.dotf/lane.json`:
       - Commits on `main` or `master` at the main repo root are blocked (`exit 1`) with message pointing to `dotf lane switch`.
       - Commits inside a worktree whose `.dotf-worktree.json` has `"state": "PARKED"` are blocked (`exit 1`) when NOT mid-rebase/cherry-pick.
       - Commits during an active `git rebase` or `cherry-pick` (`.git/rebase-merge`, `.git/rebase-apply`, or `CHERRY_PICK_HEAD` present in `git rev-parse --git-dir`) are ALLOWED (`exit 0`) even on detached HEAD (fixing **B6**).
       - Commits inside `copilot-worktrees/` for a repo with `.dotf/lane.json` are blocked (`exit 1`) with message pointing to `dotf lane switch` (fixing **m4**).
  - **Run to verify RED:**
    `cd cli && go test ./internal/hooks/... -run TestPreCommitLaneGuard -count=1`

- [x] [AC5] **Task 8 (GREEN — Implement `git-hooks/lib/lane-guard.sh`, wire into `git-hooks/pre-commit`, and detach CWD in `git-hooks/lib/board-pickup.sh`):**
  - **Files to modify/create:**
    1. Create `git-hooks/lib/lane-guard.sh` implementing the checks tested in Task 7.
    2. Source/invoke `lane-guard.sh` from `git-hooks/pre-commit`.
    3. In `git-hooks/lib/board-pickup.sh`, right after capturing `branch` and `origin_url`, add `cd "$HOME" || cd /` before any `gh` network calls so backgrounded `post-checkout` hooks never hold a Windows directory handle on the worktree (fixing **M6**).
  - **Run to verify GREEN:**
    `cd cli && go test ./internal/hooks/... -run TestPreCommitLaneGuard -v -count=1`

- [x] [P] [AC6] **Task 9 (RED — Write failing tests for Cross-OS Forge `Environment` Provenance & `no-attribution`):**
  - **File to create:** `cli/internal/forgewrite/provenance_test.go` (`TestForgePRProvenanceNoAttribution`)
  - **Exact behaviors to assert (fixes m2, m3):**
    1. `FormatEnvironmentProvenance(machineID, goos, goarch, lane string) string` returns `Environment: <machineID> (<goos>/<goarch>) · <lane>` (e.g. `Environment: EGW-LEN029 (windows/amd64) · feat-lane-primary`).
    2. Output passes the repo's `no-attribution` check and contains zero AI/agent/persona/meeseek identifiers (`claude`, `copilot`, `agy`, `pi`, `opencode`, `meeseek`, `builder`, `reviewer`, `architect`, `planner`, `shipper`, `curator`).
    3. `ValidateProvenanceNoAttribution(s string) error` rejects any provenance string containing agent, persona, or `meeseek-*` identifiers.
  - **Run to verify RED:**
    `cd cli && go test ./internal/forgewrite/... -run TestForgePRProvenanceNoAttribution -count=1`

- [x] [AC6] **Task 10 (GREEN — Implement `FormatEnvironmentProvenance` & `ValidateProvenanceNoAttribution` in `cli/internal/forgewrite/provenance.go`):**
  - **Files to create/modify:**
    1. Create `cli/internal/forgewrite/provenance.go` implementing `FormatEnvironmentProvenance` and `ValidateProvenanceNoAttribution` and wiring it into PR body / review triage comment generation.
    2. Verify `docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md` is committed.
  - **Run to verify GREEN:**
    `cd cli && go test ./internal/forgewrite/... -run TestForgePRProvenanceNoAttribution -v -count=1`

## Closing

- [x] Every acceptance criterion from `proposal.md` (`AC1`–`AC6`) is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`cd cli && go vet ./...`)
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (`specs/HARNESS-186-named-lanes-sticky-personas` -> `mlorentedev/dotfiles#3259`)
