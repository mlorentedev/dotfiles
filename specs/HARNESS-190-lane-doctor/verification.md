---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "HARNESS-190-lane-doctor"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-190-lane-doctor

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (corrupted marker names its field; coherent and uninitialised yield nothing) -> red `396b590a`, green `ec8d58a7` / `TestAuditLanes_ReportsACorruptedMarkerByField` (nine shapes), `TestAuditLanes_IsSilentOnACoherentRepository`, `TestAuditLanes_SeparatesNotInitialisedFromDrift`
- [x] AC2 (drift both directions with the git command that repairs it) -> `ec8d58a7`, `047ccc6b` / `TestAuditLanes_ReportsDriftInBothDirections` (`mv`, `git worktree move`, undeclared record, relative `gitdir`, unreadable record, empty root)
- [x] AC3 (one finding per broken lane) -> `ec8d58a7`, review round 1 fix `c30835af` (red) / `14647b4b` (green) / `TestAuditLanes_ReportsABrokenLane` (no lease, no record, lease naming another lane, plain directory after `git worktree remove`, plain directory git still registers)
- [x] AC4 (exit contract of `dotf lane doctor`) -> red `48ad8029`, green `e2bcd9d7`, `047ccc6b` / `TestLaneDoctor_ExitsZeroWithAtMostOneLine` (coherent, unmanaged, uninitialised), `TestLaneDoctor_AFindingExitsOneWithoutCobrasErrorLine`

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> exit 0, 55 packages ok on `047ccc6b` (2026-11-11, no pipe, `$?` recorded).
- Touched packages: `go test ./internal/lane/ -run 'TestAuditLanes_|TestIdentity_|TestWriteIdentity_|TestLaneInit' -count=1` -> ok 10.9s; `go test ./internal/cmd/ -run TestLaneDoctor -count=1` -> ok 1.3s.
- `go vet ./...` and `GOOS=windows go vet ./...` clean; `golangci-lint run ./...` (2.12.2, the pin) 0 issues on both `GOOS`; one gocyclo finding on the first shape of the command test, removed by the table refactor in `047ccc6b`.
- `features.json` f1..f4 verified on `1dc8f727` (evidence in the file); the tests they name are unchanged in name by `047ccc6b`.
- Manual smoke test: scratch repository (scratchpad `mvexp`, `relexp`), measured before writing the code: after `mv`, `git worktree list` marks the old path prunable, `git worktree add` at it refuses "missing but already registered", the admin dir keeps `gitdir` (old path) and `dotf-lane`; `git worktree repair <new>` rewrites `gitdir`; `git worktree move` rewrites it and keeps the record; under `worktree.useRelativePaths` (git 2.54) `gitdir` is `../../../../lanes/a/.git`.
- No regressions in existing test suite: yes.
- Production lines: 300 of the envelope's 300 after the review fixes (non-blank, non-comment lines added by `git diff <base>` to `audit.go`, `identity.go`, `lane_doctor.go`, `lane.go`); 13 files of 14.
- Commit hashes above are the ones before the rebase onto the fixed `feat/3261-lane-identity` (`a793ffd4`); the rebased branch carries the same nine commits in the same order (`7c5b01cc`..`14647b4b`).

## Review round 1 (agy/gemini-3.8-flash-high, random draw, FAIL)

Every finding, with its disposition; `review.md` is the reviewer's and is not edited.

| Finding | Class | Disposition |
|---|---|---|
| `auditLane` skips `assertWorktreeOf` when the one record sits at the canonical path, so a lane wiped and recreated by hand reads as a missing lease and the FIX names `dotf lane init`, which refuses the directory | Major, REAL | apply: red `c30835af` (`a plain directory at a lane path git still registers`, asserts no "lease" in the finding), green `14647b4b`. The reviewer's FIX ("move the directory aside, then dotf lane init") was measured on a scratch repository and also fails: git keeps the lane registered, `git worktree add` at the path refuses "missing but already registered"; `git worktree repair <path>` exits 1 yet rewrites the `.git` file. The FIX now names the prune: "move the directory aside and run git worktree prune, then dotf lane init", on both branches that report a stranger directory (`strangerDirFix`). |
| `AuditLanes` cyclomatic 18 (cyclop), 63 lines, over the AGENTS.md limits (< 10, < 40 lines) | Minor, REAL | apply in `14647b4b`: `auditRoots`, `anyLaneExists`, `undeclaredRecords` extracted, `AuditLanes` is 37 lines; `validateNames` split into `nameProblem` and `hasDrivePrefix`. `golangci-lint run --enable cyclop --enable funlen ./internal/lane/` reports nothing in `audit.go`; the pinned linter stays at 0 issues on both OSes. |
| `validateNames` accepts `.`, `..` and padded entries | Minor, THEORETICAL | apply in `14647b4b`, two cases in `TestAuditLanes_ReportsACorruptedMarkerByField` (`dot entry`, `padded entry`): cheap, and a padded entry would never match the directory it names. |
| `parseMarker` ignores data after the first JSON object | Minor, THEORETICAL | apply in `14647b4b`: `dec.More()` after the decode is a finding ("carries trailing data after the object"), case `trailing data after the object`. |

## Decisions made during implementation

- The check is `dotf lane doctor` plus `lane.AuditLanes`, not a `dotf doctor` sweep entry: the committed envelope of this run does not list `cli/internal/doctor/**`, the issue allows the subcommand as the home, and the sweep entry is #3327 with two options. The runbook's known limits say so.
- Disk-to-marker drift reads `<common dir>/worktrees/*/` (gitdir + record), not `git worktree list` and not the canonical path, after the `mv` measurement above: the path lookup would have said "absent, run `dotf lane init`", and that command fails at a registered path.
- A marker present with no lane on this machine is reported as "not initialised" with exit 0, not as three absences: the marker is committed and travels with every clone, lanes are per machine, so a fresh clone must not go red.
- A marker finding stops the audit: lanes cannot drift from a list that is not a list, and three cascaded "absent" lines would bury the field that is wrong.
- `readIdentityRecord` is the one reader of the record (ADR-061 §5), shared by `Identity` and the audit, and an unreadable record is a finding, as the guard fails closed on it. Found by the advisory pass, with the relative `gitdir`, the empty `WORKTREE_ROOT` and the full-path assertions that passed on macOS only because `/private/var/...` contains `/var/...`.
- The lane-local copy of the marker (the one the guard reads first) is not compared: an older branch legitimately carries an older marker. Recorded as a known limit.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-581-audit-against-the-registry-that-survives-the-rename.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-061 already decides where the identity lives and names this audit as its reader; one sentence added there, no new decision.
- [x] New pattern candidate for `00_meta/patterns/`? no: one project so far; the lesson carries the rule until it recurs elsewhere.
