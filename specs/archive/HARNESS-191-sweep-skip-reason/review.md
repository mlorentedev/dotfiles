---
spec: "HARNESS-191-sweep-skip-reason"
verdict: "PASS"
reviewed_sha: "145411865a6dec879f18c1d68a353299e80e44a9"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-191-sweep-skip-reason (#3263 / PR #3355), whole range `git diff c94246c87fdc7c1765c3970361c69f5fc2ecfb73...HEAD` at `145411865a6dec879f18c1d68a353299e80e44a9` (11 commits on branch; origin/fix/3263-sweep-skip-reason matches HEAD).
**Sources**: `specs/HARNESS-191-sweep-skip-reason/{proposal,tasks,verification}.md`, `features.json`, `review-round1.md` (FAIL on `e02600e0`), `review-round2.md` (FAIL on `94cb2f6f`), git history, and CI run records (run 43143382318).

### Spec and task alignment

- **AC1** (dirty lane not parked, stays ACTIVE, reason names dirty tree) — **PROVEN**. `TestParkSingleNamedLane_Outcomes/a_lane_with_an_untracked_file_is_not_parked` and `.../a_lane_whose_state_cannot_be_read_is_not_parked` pass. Mutation replacing the status/dirty error string turned the test red (`sweep_park_test.go:94`, `sweep_park_test.go:169`), proving the assertion bites.
- **AC2** (forge error carried; landed heads park; unreadable/undecided landing named) — **PROVEN**. All 4 subtests in `TestParkSingleNamedLane_Outcomes` (`a_squash-merged_lane_parks_when_the_landed_heads_name_its_tip`, `a_lane_whose_landing_cannot_be_checked_is_not_parked`, `a_lane_with_no_landing_reader_is_not_parked`, `a_lane_whose_lease_cannot_be_written_is_not_parked`) pass fresh. Landing re-check with `branchLanding` restored in `2ad5fbdb` when heads are specified.
- **AC3** (refused detach keeps state and branch, reason starts `git refused to detach it: ` followed by git's first line) — **PROVEN**. `TestParkSingleNamedLane_Outcomes/a_lane_git_refuses_to_detach_keeps_its_state` asserts state remains ACTIVE, symbolic-ref remains `feat/lane-a`, and reason matches git's error token (`index.lock`).
- **AC4** (`SweepReport.LaneSkips` lists unparked lanes with path and reason, `dotf worktree sweep` prints one line per entry) — **PROVEN**. Verified by `TestSweepReportsALaneItDidNotPark`, wire serialization test `TestSweepReportNamesLaneSkipsOnTheWire` (verifying `lane_skips` tag and omitempty), and real command execution test `TestWorktreeSweepPrintsWhyALaneWasNotParked`.
- **Windows Path Normalization** (Round 1 F1 & Round 2 F1): Verified locally that `worktree_sweep_lane_skip_test.go` normalizes both `physical` and `out` with `filepath.ToSlash`. Remote branch `origin/fix/3263-sweep-skip-reason` was pushed to HEAD (`14541186`), and Windows CI proof is recorded for run 43143382318 (job 130935507892: pass).
- **Contract Digests**: `dotf spec digest HARNESS-191-sweep-skip-reason` matches `review-request.json` exactly across `proposal.md`, `tasks.md`, and `features.json`.
- **Static Analysis & Tooling**:
  - `go vet ./internal/worktree/ ./internal/cmd/` passes with 0 issues on both `darwin` and `windows` (`GOOS=windows`).
  - `golangci-lint run ./internal/worktree/... ./internal/cmd/...` passes with 0 issues on both `darwin` and `windows` (`GOOS=windows`).
  - `go run ./cmd/dotf lessons fmt --check --dir ../docs/lessons` reports `[OK] lessons are in their canonical form` (Lesson 585 is unique and collision-free).
- **Mutation Testing** (probes run and reverted clean this session):
  1. Corrupted reason strings in `Dirt(info.Path)` / `dirt.Any()` -> `TestParkSingleNamedLane_Outcomes` subtests failed with expected mismatch.
  2. Deleted `report.LaneSkips` append in `SweepWithRunner` -> `TestSweepReportsALaneItDidNotPark` failed with empty `LaneSkips`.
  3. Omitted `report.LaneSkips` in `cmd/worktree.go` printing loop -> `TestWorktreeSweepPrintsWhyALaneWasNotParked` failed.
- **Guardrails**: No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags in spec folder. Working tree outside `specs/` is completely clean.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | quality | `parkSingleNamedLane` (46 lines) and `SweepWithRunner` (77 lines) exceed the repository's 40-line guideline. `parkSingleNamedLane` improved during this feature and is tracked under issue #3265; `SweepWithRunner` is tracked under issue #3360. | Measured via `wc -l` on `cli/internal/worktree/sweep.go` | N/A (structural code quality) | code (tracked in follow-up issues #3265, #3360) |
| Minor | REAL | tasks | `tasks.md` has `- [ ] PR opened referencing this spec folder` unticked, while PR #3355 exists and tracks this work. Cosmetic only; editing tasks.md post-review would invalidate contract digests. | `specs/HARNESS-191-sweep-skip-reason/tasks.md:35` | N/A | spec (dispositioned as cosmetic) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 4 ACs verified with named tests; 3 mutation probes independently confirmed tests bite; edge cases handled. |
| Verification       | A | All verification commands in features.json reproduced green; Windows path normalization verified; vet/lint 0 issues on both GOOS. |
| Scope              | A | Diff strictly matches proposal; no unrelated changes or creep outside declared scope. |
| Reliability        | A | Refused detach restores metadata and preserves branch; lease write failures, unreadable status, and nil readers handled safely. |
| Maintainability    | B | parkSingleNamedLane (46 lines) and SweepWithRunner (77 lines) exceed 40-line threshold; tracked in #3265 and #3360. |
| Handoff-readiness  | A | Lesson 585 captured with unique ID; round 1 & 2 review dispositions recorded in verification.md; archive checklist complete. |

### Verdict
PASS

### Recommended next steps

1. Proceed with `dotf spec archive HARNESS-191-sweep-skip-reason` as part of landing PR #3355.
2. Address the two minor function length extractions in follow-up tickets #3265 and #3360 as scheduled.

**Archive advisable?**
YES. All acceptance criteria are proven, prior review findings from Rounds 1 and 2 are resolved and verified, contract digests match, tests pass across platforms, and repository hygiene gates are satisfied.
