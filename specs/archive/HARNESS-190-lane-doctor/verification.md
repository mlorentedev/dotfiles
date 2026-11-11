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

- Test suite: `cd cli && go test ./... -count=1` -> exit 0, 55 packages ok on `047ccc6b` and again on `cb655512` after the rebase onto `main` (2026-11-11, no pipe, `$?` recorded); the round-2 fixes re-ran `./internal/lane/` and `./internal/cmd/` green on `9fbb5452` and the full suite is re-run before the PR.
- Touched packages: `go test ./internal/lane/ -run 'TestAuditLanes_|TestIdentity_|TestWriteIdentity_|TestLaneInit' -count=1` -> ok 10.9s; `go test ./internal/cmd/ -run TestLaneDoctor -count=1` -> ok 1.3s.
- `go vet ./...` and `GOOS=windows go vet ./...` clean; `golangci-lint run ./...` (2.12.2, the pin) 0 issues on both `GOOS`; one gocyclo finding on the first shape of the command test, removed by the table refactor in `047ccc6b`.
- `features.json` f1..f4 verified on `1dc8f727` (evidence in the file); the tests they name are unchanged in name by `047ccc6b`.
- Manual smoke test: scratch repository (scratchpad `mvexp`, `relexp`), measured before writing the code: after `mv`, `git worktree list` marks the old path prunable, `git worktree add` at it refuses "missing but already registered", the admin dir keeps `gitdir` (old path) and `dotf-lane`; `git worktree repair <new>` rewrites `gitdir`; `git worktree move` rewrites it and keeps the record; under `worktree.useRelativePaths` (git 2.54) `gitdir` is `../../../../lanes/a/.git`.
- No regressions in existing test suite: yes.
- Production lines: 297 of the envelope's 300 after both review rounds (non-blank, non-comment lines added by `git diff origin/main` to `audit.go`, `identity.go`, `named.go`, `lane_doctor.go`, `lane.go`); 14 files of 14.
- Commit anchors: the branch was rebased twice (onto the fixed `feat/3261-lane-identity`, then onto `main` once #3325 landed), so the hashes in the Evidence section name the commits as first written and are not ancestors of the current tip. The branch's own history is `6654c3e5..` (ten commits through `cb655512`, reviewed in round 2) plus the round-2 fixes `3ccf8228` (tests) and `9fbb5452` (fix); the test names and commands above are the anchors that survive a rebase, and every one re-ran green on `9fbb5452`.

## Review round 1 (agy/gemini-3.8-flash-high, random draw, FAIL)

Every finding, with its disposition; `review.md` is the reviewer's and is not edited.

| Finding | Class | Disposition |
|---|---|---|
| `auditLane` skips `assertWorktreeOf` when the one record sits at the canonical path, so a lane wiped and recreated by hand reads as a missing lease and the FIX names `dotf lane init`, which refuses the directory | Major, REAL | apply: red `c30835af` (`a plain directory at a lane path git still registers`, asserts no "lease" in the finding), green `14647b4b`. The reviewer's FIX ("move the directory aside, then dotf lane init") was measured on a scratch repository and also fails: git keeps the lane registered, `git worktree add` at the path refuses "missing but already registered"; `git worktree repair <path>` exits 1 yet rewrites the `.git` file. The FIX now names the prune: "move the directory aside and run git worktree prune, then dotf lane init", on both branches that report a stranger directory (`strangerDirFix`). |
| `AuditLanes` cyclomatic 18 (cyclop), 63 lines, over the AGENTS.md limits (< 10, < 40 lines) | Minor, REAL | apply in `14647b4b`: `auditRoots`, `anyLaneExists`, `undeclaredRecords` extracted, `AuditLanes` is 37 lines; `validateNames` split into `nameProblem` and `hasDrivePrefix`. `golangci-lint run --enable cyclop --enable funlen ./internal/lane/` reports nothing in `audit.go`; the pinned linter stays at 0 issues on both OSes. |
| `validateNames` accepts `.`, `..` and padded entries | Minor, THEORETICAL | apply in `14647b4b`, two cases in `TestAuditLanes_ReportsACorruptedMarkerByField` (`dot entry`, `padded entry`): cheap, and a padded entry would never match the directory it names. |
| `parseMarker` ignores data after the first JSON object | Minor, THEORETICAL | apply in `14647b4b`: `dec.More()` after the decode is a finding ("carries trailing data after the object"), case `trailing data after the object`. |

## Review round 2 (nan/mimo-v2.6-flash, FAIL, reviewed_sha cb655512)

| Finding | Class | Disposition |
|---|---|---|
| A fresh clone whose lane paths hold another repository's lanes (shared basename) reports three findings whose FIX is `strangerDirFix`, i.e. move aside and prune the other repository's healthy lanes; the spec's fresh-clone rule and AC3 contradict each other in that state | Major, REAL | apply: red `3ccf8228` (`TestAuditLanes_LeavesAnotherRepositorysWorktreeAlone`, two cases: zero records, and this repository's stale registration of a path another repository's worktree took), green `9fbb5452`. The move and prune now apply only to a plain directory (`errNotAWorktree`, wrapped in `named.go`); a foreign worktree is left to its owner and the FIX names a different directory name for this clone (#3318); with a stale registration the FIX names the admin dir to delete by hand. Measured on a scratch pair before writing the strings: `git worktree prune` keeps the stale entry while the other `.git` file exists, `remove --force` refuses ("does not point back"), and `git worktree repair <path>` rewrites the other repository's `.git` file to point here, so neither is named. The test also runs the named repair (rename the clone, `InitLanes`) and asserts both repositories audit clean afterwards. The contract now says the finding wins over the fresh-clone line, because that line's advice refuses in this state (proposal §What, AC3). |
| Three listed shapes have no committed test: an empty marker entry, a lease state the lanes do not use, a lane recorded in two worktrees | Minor, REAL | apply in `3ccf8228`: `empty entry` in the marker table, `lease in a state the lanes do not use` and `a lane recorded in more than one worktree`; all three green without a code change, as the reviewer's probes said. `tasks.md` and f1 now count thirteen shapes. |
| `newLaneDoctorCmd` is 43 lines | Minor, REAL | apply in `9fbb5452`: `runLaneDoctor` extracted (31 lines), and it no longer resolves the cwd itself since `ResolveMainRepoRoot(".")` does. |
| Evidence anchors name commits that are not ancestors of the reviewed tip | Minor, REAL | apply: the provenance line in Test status says why (two rebases) and anchors on test names and commands; `features.json` evidence re-run and re-anchored on `9fbb5452`. |

## Review round 3 (nan/glm5.3-flash, PASS, reviewed_sha b6d74f79)

The reviewer re-ran vet, the pinned linter and both touched packages on `b6d74f79`, matched the contract digests, and mutation-tested the round-2 fix (disabling the `errNotAWorktree` branch turns both subtests of `TestAuditLanes_LeavesAnotherRepositorysWorktreeAlone` red).

| Finding | Class | Disposition |
|---|---|---|
| `laneRecords` skips an admin dir whose record reads but whose `gitdir` is missing or unreadable, so the lane is then misdiagnosed from the paths that remain | Minor, THEORETICAL | ticketed as #3337: reporting it the way an unreadable record is reported costs about four production lines and this change sits at 297 of its 300-line budget; the state is reachable (git lists it as prunable) but was argued from the code, not observed. |
| AC4 is silent about the exit-2 path the implementation adds for an audit that cannot run | Minor, SPECULATIVE | applied in the runbook (7.6 names the three causes and that the reason is on stderr); the contract set is closed after the review, so AC4 is not edited. |

## Decisions made during implementation

- The check is `dotf lane doctor` plus `lane.AuditLanes`, not a `dotf doctor` sweep entry: the committed envelope of this run does not list `cli/internal/doctor/**`, the issue allows the subcommand as the home, and the sweep entry is #3327 with two options. The runbook's known limits say so.
- Disk-to-marker drift reads `<common dir>/worktrees/*/` (gitdir + record), not `git worktree list` and not the canonical path, after the `mv` measurement above: the path lookup would have said "absent, run `dotf lane init`", and that command fails at a registered path.
- A marker present with no lane on this machine is reported as "not initialised" with exit 0, not as three absences: the marker is committed and travels with every clone, lanes are per machine, so a fresh clone must not go red.
- A marker finding stops the audit: lanes cannot drift from a list that is not a list, and three cascaded "absent" lines would bury the field that is wrong.
- `readIdentityRecord` is the one reader of the record (ADR-061 §5), shared by `Identity` and the audit, and an unreadable record is a finding, as the guard fails closed on it. Found by the advisory pass, with the relative `gitdir`, the empty `WORKTREE_ROOT` and the full-path assertions that passed on macOS only because `/private/var/...` contains `/var/...`.
- The lane-local copy of the marker (the one the guard reads first) is not compared: an older branch legitimately carries an older marker. Recorded as a known limit.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-582-audit-against-the-registry-that-survives-the-rename.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-061 already decides where the identity lives and names this audit as its reader; one sentence added there, no new decision.
- [x] New pattern candidate for `00_meta/patterns/`? no: one project so far; the lesson carries the rule until it recurs elsewhere.
