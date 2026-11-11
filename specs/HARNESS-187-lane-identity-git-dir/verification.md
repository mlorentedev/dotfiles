---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "HARNESS-187-lane-identity-git-dir"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-187-lane-identity-git-dir

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (record per lane, `changed=0` on re-run) -> commit `3ab5bf01` / `TestLaneInitRecordsTheIdentityInEachLanesGitDir`, `TestLaneInitRestoresADeletedIdentityRecord` (1 then 0), `TestLaneInitIdempotent` unchanged.
- [x] AC2 (guard blocks by record whatever the leaf; never by name) -> commit `5f63936f` / `TestPreCommitLaneGuard` case 10: an unleased worktree named `spike-lane` passes; with `lane.LaneIdentityFile` written into its git admin dir a real `git commit` with the guard installed as `pre-commit` (through `core.hooksPath`) is refused naming `lease metadata` and `spike-lane`; after `git worktree move` to `not-a-lane-name` it is still refused.
- [x] AC3 (second repository with the same basename refused, first untouched) -> commit `3ab5bf01` / `TestLaneInitRefusesALaneDirectoryOfAnotherRepository` (error names the lane path and the first repository; its metadata byte-identical; `HasNamedLanes(second)` false), `TestLaneInitRefusesAPlainDirectoryAtALanePath`.
- [x] AC4 (`lane.Identity` contract) -> commit `47df0b33` / `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir` (lane, after move, no record, main checkout, outside git), `TestWriteIdentity_IsIdempotent` (main checkout refused).
- [x] AC5 (other cases pass, guard reads no `canonical_lanes`) -> `go test ./internal/hooks/ ./internal/lane/` ok on `5f63936f`; `grep -c canonical_lanes git-hooks/lib/lane-guard.sh` prints `0`.

## Test status

- Test suite: `cd cli && go test ./internal/hooks/ ./internal/lane/ -count=1` -> `ok` both packages (hooks 4.9s, lane 30.3s, 2026-11-11).
- `features.json`: f1..f5 all PASS, run one by one from `cli/` on `437eb1ed` and again on the final head; f1's test now also asserts the second `InitLanes` returns `0`.
- Static: `go build ./... && go vet ./... && GOOS=windows go vet ./...` clean; `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` at the pinned 2.12.2 report `0 issues` (cache clean refused, another session held the cache; no finding to dismiss); `shellcheck -s sh` and `sh -n` on the guard clean; `dotf lessons fmt` formatted 0 files.
- Manual smoke test: the record's lifecycle (`worktree move`, `gc`, `repair`, `remove`, `rm -rf` + `prune`, re-create, `clean -fdx`) was measured on a scratch repository before the spec was written; the Go tests repeat the `move` leg. Not re-run on a live lane: this box's lanes are provisioned by the release binary and get their record when `dotf lane init` runs after the release.
- Full suite: `cd cli && go test ./... -count=1` -> every package `ok` on `5dca2592` (no FAIL line; 2026-11-11), run twice: before and after the preflight change.
- No regressions in existing test suite: yes. Case 10 of the guard test changed meaning by design (proposal, Risks).

## Decisions made during implementation

- `HasNamedLanes` gained the same membership check as `provisionLane`. The AC3 test asserted the second repository must not believe it has lanes, and the only caller (`handoff.go`) routes work into the primary lane on a true answer, so a path-only answer there was the same defect one call later. Cost: two git invocations on the handoff path.
- `Identity` treats a git exit error as "not a repository" and only a failure to run git as an error. `rev-parse` exits 128 both for a path outside any repository and for a nonexistent one; the spec wants `("", false, nil)` for the first, and a nonexistent path is not a lane either.
- The guard absolutises `git_dir` with the same `case` the script already used for `common_dir`, rather than switching to `--absolute-git-dir`: one idiom in the file, and `--path-format` needs git 2.31 while the existing pattern needs nothing. On the Go side `identity.go` uses `rev-parse --path-format=absolute --git-dir --git-common-dir` (one call answers both), the form `resolveGitCommonDir` already required, so no new git floor; `tasks.md` names `--absolute-git-dir` for both and was not rewritten.
- The membership check runs as a preflight over the three lane paths before `lane init` writes the exclude entries or the marker (second advisory pass): a refusal that came after the marker would leave the guard armed in a repository with no lanes, and the guard's own FIX line would send the user to `dotf lane switch`. `lane switch` and `lane park` resolve the lane by the same path convention, so both now run the same check and refuse a lane of another repository; the AC3 test asserts the refused repository has no marker, that switch and park from it are refused naming the owner, and that the first repository's lane stays detached.
- The AC3 test compares against `filepath.EvalSymlinks(firstRepo)`: git answers physical paths, and the macOS temp dir is a symlink (the logical form matched only as a substring; a Windows 8.3 temp path would not).

## Review round 1 (nan/glm5.3-flash, FAIL on `594d1764`; `review-round1.md`)

| Finding | Class | Disposition | Evidence |
|---|---|---|---|
| `Identity()` on a main checkout carrying `.git/dotf-lane/` (the `lane next` prompt directory) returned an error, not `("", false, nil)` | REAL Major | applied | Reproduced red: `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir` now creates `.git/dotf-lane/` in the main checkout and failed with `read …/.git/dotf-lane: is a directory`; `identity.go` short-circuits `gitDir == commonDir` and treats a directory at the record path as no record; green. |
| Guard blocks on an empty or whitespace-only record while Go reads it as no record | THEORETICAL Minor | applied | Rule 5b trims the first line and blocks only when non-empty; case 10 writes a whitespace record and asserts the pass. |
| `samePath` falls back to `filepath.Clean` per side when `EvalSymlinks` fails | SPECULATIVE Minor | declined | Every call site resolves the lane's git dirs first, so both sides exist; the reviewer marked it surface only, do not gate. |
| Two artifacts named `dotf-lane` in the git-dir namespace (prompt directory, identity record) | THEORETICAL Minor | ticketed | #3320 (rename the prompt directory; the `Identity` special case stays as defence in depth). |
| `tasks.md` names `--absolute-git-dir`, the guard keeps its `case` idiom | THEORETICAL Minor | declined | Already recorded under Decisions; the reviewer asked for no action. |
| Windows unverified on the authoring box | Question | declined | Declared in the proposal's Risks; CI's Windows leg is the authoritative run. |

## Review round 2 (nan/mimo-v2.6-flash, PASS on `c79d475d`; `review.md`)

| Finding | Class | Disposition | Evidence |
|---|---|---|---|
| An unreadable record (mode 000) fails open in the guard: `sed … 2>/dev/null` yields an empty name | THEORETICAL Minor | applied | Rule 5b now fails closed on `[ ! -r ]` with its own message; case 10 chmods the record to 000 and asserts the block (skipped on Windows and as root, where the mode has no effect). |
| Multi-line record: the guard reads the first line, `Identity` returned the whole trimmed content | THEORETICAL Minor | applied | `Identity` reads the first line; `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir` writes residue after the name and asserts the name alone. |
| `writeIdentity` writes in place, not temp-plus-rename | SPECULATIVE Minor | declined | One small write under the init lock; a truncated record is restored by the next `lane init`, which rewrites on any difference. The reviewer marked it surface only. |
| `tasks.md` names `--absolute-git-dir` for the guard | THEORETICAL Minor | declined | Round 1's disposition stands. |
| Rule 6b blocks any leaseless detached worktree with a message naming the main checkout | Question (pre-existing) | ticketed | #3324. The rule and message predate this spec; AC2's "not blocked by name" is asserted on a checked-out branch, which is the lane's working state. |
| Windows unverified on the authoring box | Question | declined | Declared in the proposal's Risks; CI's Windows leg is the authoritative run. |

The two applied rows change code outside the contract set after the reviewed sha; `go test ./internal/lane/ ./internal/hooks/`, vet and the pinned lint on both OSes are green on the resulting head.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-579-a-directory-name-is-not-an-identity.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-061-lane-identity-in-the-worktree-git-dir.md
- [x] New pattern candidate for `00_meta/patterns/`? no: one project's lane system; the lesson carries the transferable rule and no second project has lanes yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-187-lane-identity-git-dir/` -> `specs/archive/HARNESS-187-lane-identity-git-dir/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
