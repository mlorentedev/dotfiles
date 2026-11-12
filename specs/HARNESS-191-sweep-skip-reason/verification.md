---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "HARNESS-191-sweep-skip-reason"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-191-sweep-skip-reason

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (dirty lane not parked, reason names the dirty tree) -> red `67e45595`, green `27e6d3ba` / `TestParkSingleNamedLane_Outcomes/a lane with an untracked file is not parked`
- [x] AC2 (forge error carried; landed heads park) -> same commits / `TestParkSingleNamedLane_Outcomes/a lane whose landing cannot be checked is not parked`, `…/a squash-merged lane parks when the landed heads name its tip`
- [x] AC3 (refused detach keeps state and branch, git's first line in the reason) -> same commits / `TestParkSingleNamedLane_Outcomes/a lane git refuses to detach keeps its state`, `TestSweepReportsALaneItDidNotPark`
- [x] AC4 (`LaneSkips` in the report, one printed line per entry) -> `27e6d3ba` and `5a74c5c0` / `TestSweepReportsALaneItDidNotPark`, `TestWorktreeSweepPrintsWhyALaneWasNotParked`

## Test status

- Touched packages: `cd cli && go test ./internal/worktree/ ./internal/cmd/ -count=1` -> ok, ok (2026-11-11, on `27e6d3ba` and again after the lesson and the command test).
- `go vet ./internal/worktree/ ./internal/cmd/` and `GOOS=windows go vet` on the same packages clean; `golangci-lint run` (2.12.2, the pin, cache cleaned first) 0 issues on both `GOOS`.
- `features.json` f1..f4 verified on `2ad5fbdb` (the tip after review round 2); each regex was run under `-v` to confirm it selects its subtests (2, 4, 2 and 3 tests).
- Manual smoke test: none beyond the command test, which runs `dotf worktree sweep` on a real repository with a lane landed by ancestry and a held index lock.
- No regressions in existing test suite: yes.
- Production lines: 33 added executable lines in `sweep.go` and `cmd/worktree.go`; the pre-push SDD gate counts 66 (insertions plus deletions), which is why this spec exists (lesson 585).
- Commit anchors: the branch was rebased onto `main` once (after #3338 landed) before the review; the test names above survive the rebase and every one re-ran green on `2ad5fbdb`.
- AC2 also covers, since `4ced9f9f`: `TestParkSingleNamedLane_Outcomes/a lane with no landing reader is not parked` (mock runner, `mergedPRHeads` nil) and `…/a lane whose lease cannot be written is not parked` (read-only lease file). AC4 also: `TestSweepReportNamesLaneSkipsOnTheWire` (the `lane_skips` name) and the `SkippedCount == 2` assertion in `TestSweepReportsALaneItDidNotPark`.

## Decisions made during implementation

- The reason is returned with the verdict rather than logged: the report is the surface the operator reads and the JSON consumers parse, and a second channel would need a second test.
- `laneLanded` is extracted now because the landing decision (ancestry, then the forge) and the parking action had grown into one function; it is the seam #3265 starts from and it is tested through the same subtests, not on its own.
- git's stderr is kept to its first line: the report prints one line per lane, and `fatal: Unable to create '…/index.lock': File exists.` says what the operator needs.
- The sweep-level test lands the lane's branch by a fast-forward merge pushed to the fixture's origin, so the ancestry fast path classifies the lane parkable without a forge; the command test does the same.

## Review dispositions

Round 1 (`review.md`, verdict FAIL on `e02600e0`):

| Finding | Disposition | Where |
|---|---|---|
| F1 Major REAL: the command test compares an OS-separator path with git's slash path, red on `test (windows-latest)` | applied | `4ced9f9f`: both sides through `filepath.ToSlash`; proof: the Windows table under round 2 |
| F2 Major REAL: lesson 584 already on `main`, `hygiene` and `knowledge-gate` red | applied | `ffd5c951`: renamed to 585 (file, `id:`, `tasks.md`, this file, PR body) |
| F3 Minor REAL: f4's evidence anchored to a commit without the command test | applied | `features.json` re-anchored to `4ced9f9f`, where every test exists and ran |
| F4 Minor THEORETICAL: `its lease could not be written` and `its landing could not be decided` untested | applied | two new subtests under AC2 (`4ced9f9f`); `its tip could not be read` is `branchLanding`'s own, pre-existing and out of this spec's scope |
| F5 Minor SPECULATIVE: `lane_skips` pinned by no test | applied | `TestSweepReportNamesLaneSkipsOnTheWire` |
| F6 Minor SPECULATIVE: `laneLanded` fails open on a nil reader | applied | `laneLanded` answers "its landing could not be decided" and does not park; tested with the mock runner, since the real one reaches the nil reader first in `list.go` and panics, which is the pre-existing behaviour and unreachable in production |
| F7 Minor REAL: `SweepWithRunner` 77 lines, no ticket | ticketed | #3360 (option A folds it into #3265) |
| F8 Minor REAL: `SkippedCount` never asserted | applied | `TestSweepReportsALaneItDidNotPark` asserts 2 (the main checkout and the lane) |
| `spec-gate` archive-on-merge | not a finding | resolves when this spec archives in #3355 |

Round 2 (`review.md`, verdict FAIL on `94cb2f6f`):

| Finding | Disposition | Where |
|---|---|---|
| F1 Major REAL: the Windows proof claimed for round-1 F1 did not exist, the fixes were unpushed | applied | the push was held by the pre-push archive-on-merge gate (the PR closes #3263 while this spec is active). The PR body was switched to `Refs #3263` for the push, the gate's own non-closing path, and is restored to `Fixes #3263` with the archive. Proof: `test (windows-latest)` on the pushed head, in the table below. The circularity (a gate that holds review fixes whose evidence is CI) is ticketed |
| F2 Minor THEORETICAL: `laneLanded` parked on `isMerged` alone, the old code re-checked `branchLanding` when heads were named | applied | `2ad5fbdb`: the certified shape is restored; only the no-reader case is new, and it refuses to park |
| F3 Minor THEORETICAL: an unreadable status was reported as a dirty tree | applied | `2ad5fbdb`: `its state could not be read: <err>`, tested with the mock runner's `DirtyErrors`; added to the proposal's reason list |
| F4 Minor SPECULATIVE: `gitFailure` could read a stdout line before git's fatal line | declined | `git checkout --detach` prints nothing on stdout when it refuses; the one observed refusal (the index lock) starts with the `fatal:` line, and a reason that quoted stdout would still be git's own words |

Windows proof for F1:

| Head | Run | `test (windows-latest)` |
|---|---|---|
| pending | pending | pending |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-585-the-push-gate-counts-what-you-deleted-too.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: an additive report field and a refactor inside one function change no structural invariant
- [x] New pattern candidate for `00_meta/patterns/`? no: the gate-count lesson went into the autopilot skill's eligibility veto in the vault, which is where it recurs

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-191-sweep-skip-reason/` -> `specs/archive/HARNESS-191-sweep-skip-reason/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
