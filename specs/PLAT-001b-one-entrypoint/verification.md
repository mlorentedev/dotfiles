---
tags: [spec, verification, templates]
created: "2026-10-06"
---

# Verification - PLAT-001b-one-entrypoint

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (plan lists and writes nothing) -> commit `b70aa63` / tests `TestConvergePlan_ListsApplicableReconcilersAndTouchesNothing`, `TestRun_PlanReportsEveryReconcilerAndAppliesNothing`, `TestPlanMirror_CountsWhatMirrorWouldWriteAndWritesNothing`
- [ ] AC2 (instruction files first) -> PR 2b
- [x] AC3 (second run is a no-op, report persisted) -> no-op proven in `b70aa63`; the report in PR 3 / tests `TestConverge_SecondRunIsANoOpAndPersistsTheReport`, `TestWriteReport_RecordsTheRunAndItsOutcome`
- [x] AC4 (failed probe fails the run, naming it) -> commit `b70aa63` / tests `TestRun_FailedProbeFailsTheRunNamingTheReconciler`, `TestRecordsMirror_ProbeFailsWhileTheDeployDirDiffers`
- [x] AC5 (unlisted OS is skipped, named) -> commit `b70aa63` / test `TestRun_UnlistedPlatformIsSkippedNotPassed`
- [ ] AC6–AC9 -> PRs 4 to 6

## Test status

- Test suite (PR 2a, macOS arm64): `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` -> ok, every package; `golangci-lint run` (pinned 2.12.2) -> 0 issues
- Manual smoke test (PR 2a): on the real Mac, `go run ./cmd/dotf converge --plan` -> `records-mirror [CHANGE] harness/ + 3 target(s) → ~/.dotfiles (70 to write, 0 unchanged)`; `~/.dotfiles` unchanged afterwards
- No regressions in existing test suite: yes (the tool catalog's platform tests pass through `internal/platform`)
- Test suite (PR 3, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./... -count=1` -> ok; golangci-lint 2.12.2 -> 0 issues; feature f3 -> PASS
- `TestReadmeHasNoInternalReferences` caught an ADR id in the new `converge` README section before push; removed
- Manual run (PR 3): `dotf converge` on the Mac printed `report: ~/.local/state/dotfiles/converge/last.json`; the file holds `"result": "ok"`, `"changed": 0`, `"goos": "darwin"` and the `records-mirror` entry with its detail
- Review hardening (PR 3): `env.Home()` falls back to the user database when HOME and USERPROFILE are both empty, so the report path (and every path built on Home) is never relative to the working directory; test `TestHome_FallsBackToTheUserDatabase`

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- PR 2 was split into 2a and 2b to stay under the size cap. Apply mode landed with the engine in 2a, because plan and apply are one code path (`Reconcile(env, dryRun)`); only the persisted report moves to PR 3.
- The `platforms` matcher moved from the tool catalog into `internal/platform`, so the catalog and the reconcilers cannot drift apart (ADR-045 decision 7).
- PR 2b drives the instruction files from `harness/manifest.json` `agents.presence[]`, not from new `ai/deploy.json` entries. A new deploy field needs a manifest version the installed `dotf` cannot read, and a `replace` entry would fight `harness presence` over the same file, so AC3 could never pass. Skills and bindings move to a PR 2c.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-001b-one-entrypoint/` -> `specs/archive/PLAT-001b-one-entrypoint/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
