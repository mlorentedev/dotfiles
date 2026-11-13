---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "HARNESS-193-lease-in-git-dir"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-193-lease-in-git-dir

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (git clean -fdx leaves an ACTIVE lane ACTIVE for status, list and a guarded commit) -> commit `bc12b7d2` / `TestActiveLaneSurvivesGitClean` (`cli/internal/lane/clean_survival_test.go`); probe: with the guard at the parent commit the same test fails on "missing its lease metadata"
- [x] AC2 (Load prefers the admin dir, falls back in-tree; Save writes both, refuses a non-worktree path) -> commits `32aaeb5f`, `6d7fa429` / `TestAdminDir`, `TestLeaseLivesInTheWorktreeGitDir` (`cli/internal/worktree/lease_location_test.go`)
- [x] AC3 (lane init copies legacy-only leases, changed keyed on the admin dir) -> commit `d38101a1` / `TestLaneInitMigratesInTreeLeases` (`cli/internal/lane/lanes_test.go`)
- [x] AC4 (guard reads the admin-dir lease first in rules 5 and 6, in-tree as fallback) -> commit `bc12b7d2` / `TestPreCommitLaneGuard` case 6c (`cli/internal/hooks/lane_guard_test.go`); `sh -n`, `shellcheck -s sh` clean
- [x] AC5 (session-end reads the admin-dir PARKED lease from the root and a subdirectory, in-tree as fallback) -> commit `bc12b7d2` / `TestIsWorktreeParkedReadsTheAdminDirLease` (`cli/internal/mem/session_end_lease_test.go`)
- [x] AC6 (list names a legacy-only lease and not one the admin dir holds; existing suites pass) -> commits `fc286c4f`, `d38101a1` / `TestListNamesALegacyLocationLease` (`cli/internal/worktree/lease_location_test.go`), `go test ./internal/worktree ./internal/lane ./internal/cmd ./internal/mem ./internal/hooks` green at `128dfebd`

## Test status

- Test suite: `cd cli && go test ./...` -> no FAIL at `128dfebd` (see the PR's Verification section for the run); every `features.json` command exit 0 at `128dfebd`
- Static: `go build ./... && go vet ./... && GOOS=windows go vet ./...`, pinned `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...` -> 0 issues; `sh -n` and `shellcheck -s sh git-hooks/lib/lane-guard.sh` clean
- Manual smoke test: none on a live seat; `TestActiveLaneSurvivesGitClean` runs the defect end to end (init, switch, `git clean -fdx`, status, list, commit through the installed guard) on a scratch repository
- No regressions in existing test suite: yes. 44 fixtures failed when `SaveMetadata` started refusing plain directories and were rewritten to linked fixtures (`gittest.LinkWorktree`); none was weakened

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Overlap policy A (mirror for one release) taken on the owner's delegation; the owner can overrule before the release ships.
- `SaveMetadata` writes the mirror first and the admin dir last, and puts the mirror back when the admin-dir write fails (review finding on #3421, `TestSaveMetadataPutsTheMirrorBackWhenTheAdminDirRefuses`), so an error always leaves both copies unchanged; the two "lease cannot be written" fixtures break the admin-dir copy so they survive the mirror's removal (#3443).
- The guard's missing-lease message keeps the phrase "lease metadata" (an existing hook test asserts it) and adds where the lease lives.
- `worktree list` appends the legacy note after `Classify` rather than inside it: `Classify` stays a pure function of `Info`, and the note is a location fact, not a lifecycle state.
- Independent review (PASS, `review.md`): the session-end reader prefers the admin-dir copy, so a lane parked by a binary older than this change (in-tree PARKED, admin-dir still ACTIVE) gets a detached journal once. That is the same mixed-version window the runbook names, it costs one stray journal and no data, and it ends when the old binary is gone; accepted as noise, no code change. The review's other findings were applied: Save refusals for the main checkout and a submodule, a source pin for the two spellings outside the worktree package (`TestLeaseFileNameSpellingsAgree`), and the duplicate test label.
- `features.json` entries stay `pending` with the exit-0 evidence line: no harness runs those commands, and the archived specs that carry `features.json` (SEC-002 among them) close the same way; the independent review, not a runner, is the archive gate.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-595-excluded-from-git-is-not-safe-from-git-clean.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-062-the-lease-lives-in-the-worktree-git-dir.md
- [x] New pattern candidate for `00_meta/patterns/`? no: one repository's lease; the clean-vs-exclude point is lesson 595 and would join a pattern only when a second project hits it

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-193-lease-in-git-dir/` -> `specs/archive/HARNESS-193-lease-in-git-dir/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
