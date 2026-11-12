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

- [x] AC1 (dirty lane not parked, reason names the dirty tree) -> red `9a52fd06`, green `6c186810` / `TestParkSingleNamedLane_Outcomes/a lane with an untracked file is not parked`
- [x] AC2 (forge error carried; landed heads park) -> same commits / `TestParkSingleNamedLane_Outcomes/a lane whose landing cannot be checked is not parked`, `…/a squash-merged lane parks when the landed heads name its tip`
- [x] AC3 (refused detach keeps state and branch, git's first line in the reason) -> same commits / `TestParkSingleNamedLane_Outcomes/a lane git refuses to detach keeps its state`, `TestSweepReportsALaneItDidNotPark`
- [x] AC4 (`LaneSkips` in the report, one printed line per entry) -> `6c186810` and the command test commit / `TestSweepReportsALaneItDidNotPark`, `TestWorktreeSweepPrintsWhyALaneWasNotParked`

## Test status

- Touched packages: `cd cli && go test ./internal/worktree/ ./internal/cmd/ -count=1` -> ok, ok (2026-11-11, on `6c186810` and again after the lesson and the command test).
- `go vet ./internal/worktree/ ./internal/cmd/` and `GOOS=windows go vet` on the same packages clean; `golangci-lint run` (2.12.2, the pin, cache cleaned first) 0 issues on both `GOOS`.
- `features.json` f1..f4 verified on `babfb894` (the rebased tip); each regex was run under `-v` to confirm it selects its subtests (1, 2, 2 and 2 tests).
- Manual smoke test: none beyond the command test, which runs `dotf worktree sweep` on a real repository with a lane landed by ancestry and a held index lock.
- No regressions in existing test suite: yes.
- Production lines: 33 added executable lines in `sweep.go` and `cmd/worktree.go`; the pre-push SDD gate counts 66 (insertions plus deletions), which is why this spec exists (lesson 584).
- Commit anchors: the branch was rebased onto `main` once (after #3338 landed) before the review; the test names above survive the rebase and every one re-ran green on `babfb894`.

## Decisions made during implementation

- The reason is returned with the verdict rather than logged: the report is the surface the operator reads and the JSON consumers parse, and a second channel would need a second test.
- `laneLanded` is extracted now because the landing decision (ancestry, then the forge) and the parking action had grown into one function; it is the seam #3265 starts from and it is tested through the same subtests, not on its own.
- git's stderr is kept to its first line: the report prints one line per lane, and `fatal: Unable to create '…/index.lock': File exists.` says what the operator needs.
- The sweep-level test lands the lane's branch by a fast-forward merge pushed to the fixture's origin, so the ancestry fast path classifies the lane parkable without a forge; the command test does the same.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-584-the-push-gate-counts-what-you-deleted-too.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: an additive report field and a refactor inside one function change no structural invariant
- [x] New pattern candidate for `00_meta/patterns/`? no: the gate-count lesson went into the autopilot skill's eligibility veto in the vault, which is where it recurs

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-191-sweep-skip-reason/` -> `specs/archive/HARNESS-191-sweep-skip-reason/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
