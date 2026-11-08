---
tags: [spec, verification, templates]
created: "2026-11-07"
id: "HARNESS-186-named-lanes-sticky-personas"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-186-named-lanes-sticky-personas

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 (`AC1` — Persistent Named Lanes & Idempotent Provisioning) -> `TestLaneInitIdempotent`, `TestLaneSwitchRoutingAndWIPCap` (`cli/internal/lane/lanes_test.go`)
- [x] Criterion 2 (`AC2` — Safe In-Place Lane Park & Housekeeping) -> `TestLaneParkInPlaceAndTipVerification` (`cli/internal/worktree/park_test.go`)
- [x] Criterion 3 (`AC3` — Sticky Persona Gate & Read-Only Exemption) -> commit `7027fa36` / `TestDecide_ExemptsReadOnlyDiscoveryTools`, `TestStickyPersona_PersistsAcrossKnowledgeTouchAndResetsOnBranchChange`, `TestAtomicWriteFile_ConcurrentWritesOnWindowsDoNotFail`, `TestHarnessGateStickyPersonaAndMeeseekWorkerID`
- [x] Criterion 4 (`AC4` — `meeseek-<role>-<fruit>` Ephemeral Worker Identity & `pi` Pool Fallback) -> commit `7027fa36` / `TestFruitsDictionaryHas64UniqueLowercaseEntries`, `TestFruitDeckDraws64UniqueBeforeReshuffle`, `TestParseWorkerID`, `TestDispatch_PopulatesMeeseekWorkerIDAndEnv`, `TestPiBackend_IncludesProviderFlagAndClassifiesAuthError`
- [x] Criterion 5 (`AC5` — Scoped `pre-commit` Lane Guard & Rebase Exemption) -> `TestPreCommitLaneGuard` (`cli/internal/hooks/lane_guard_test.go`)
- [x] Criterion 6 (`AC6` — Cross-OS Forge Provenance (`no-attribution` Compliant)) -> `TestForgePRProvenanceNoAttribution` (`cli/internal/forgewrite/provenance_test.go`)

## Test status

- Test suite: `cd cli && go test ./internal/agent/... ./internal/harness/... ./internal/worktree/... ./internal/lane/... ./internal/hooks/... ./internal/forgewrite/... -> PASS`
- Manual smoke test: Provisioned `feat-lane-primary` for `dotfiles`, cleaned 19 safe worktrees on `EGW-LEN029` (`35 -> 16`), verified `dotf agent auto --role reviewer --timeout 5m` dispatches `pi --provider nan`.
- No regressions in existing test suite: yes

## Decisions made during implementation

- Added `.dotf-lane-prompt.md` to `isDisposableExactPath` in `cli/internal/worktree/clean.go` so `Dirt()` never marks a named lane dirty solely due to a handoff prompt file before `Park()` removes it.
- Used `--no-track` when `SwitchLane` checks out a new branch from `origin/main` so `git` does not set `origin/main` as the upstream tracking branch `@{u}`, allowing `branchLanding` to verify squash-merged PR heads accurately.
- Guarded `atomicWriteFile` in `cli/internal/harness/dispatch.go` with `atomicWriteMu sync.Mutex` + 15-attempt `os.Rename` retry to eliminate concurrent `Access is denied` errors on Windows.
- Executed independent code-level adversarial reviews via subagent (`meeseek-reviewer-avocado`) and `dotf secrets run --only NAN_API_KEY -- dotf agent run --tier mid --role reviewer` (`meeseek-reviewer-durian` on `nan:deepseek-v4-flash`) and applied all 8 findings:
  - `M-CODE-1`: Released `refs/dotf/claims/<issue>`, `refs/dotf/seen/<issue>`, and remote `:refs/dotf/claims/<issue>` across `done.go:Park()`, `sweep.go:parkSingleNamedLane()`, and `named.go:ParkLane()`.
  - `M-CODE-2`: Ordered `State="PARKED"` write before `git checkout --detach` / `git branch -D` in `done.go` and `sweep.go`, and checked `branchLanding` tip SHA containment in `sweep.go:parkSingleNamedLane()`.
  - `M-CODE-3`: Wired active worktree branch, `.dotf-worktree.json` `persona`, and skill-consumption persona inference into `cli/internal/cmd/harness_gate.go`.
  - `M-CODE-4`: Wired `maybeAppendLaneProvenance` into `cli/internal/cmd/forge_write.go` (`forge pr create` and `forge pr/issue comment`) and added `machine`, `os`, and `lane` fields to `cli/internal/lane/claim.go`.
  - `m-CODE-1`..`m-CODE-4`: Forwarded `DOTF_WORKER_ID`/`DOTF_ROLE` to `Hive.Dispatch`, emitted `dotf lane switch` in `handoff.go:manualCommands`, ran best-effort `git fetch --quiet origin` before `Park()`, and exempted `REVERT_HEAD` and `sequencer` in `git-hooks/lib/lane-guard.sh`.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-567-in-place-worktree-parking-and-windows-atomic-rename-mutex.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: documented in repo-scoped ADR-060 and lesson-567 first until validated across multiple repositories.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-186-named-lanes-sticky-personas/` -> `specs/archive/HARNESS-186-named-lanes-sticky-personas/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
