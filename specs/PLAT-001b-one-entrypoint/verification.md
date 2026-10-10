---
tags: [spec, verification, templates]
created: "2026-10-06"
---

# Verification - PLAT-001b-one-entrypoint

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (plan lists and writes nothing) -> commit `b70aa63` / tests `TestConvergePlan_ListsApplicableReconcilersAndTouchesNothing`, `TestRun_PlanReportsEveryReconcilerAndAppliesNothing`, `TestPlanMirror_CountsWhatMirrorWouldWriteAndWritesNothing`
- [x] AC2 (instruction files first) -> PR 2b / tests `TestRecordsHarness_*` (converge), `TestCheckInstructionDrift_InstalledAgentWithoutItsInstructionsFails` (doctor); on the Mac, `dotf converge` deployed `~/.claude/CLAUDE.md`, the opencode and pi instruction files and 33 skills, and a second run reported `0 changed`
- [x] AC3 (second run is a no-op, report persisted) -> no-op proven in `b70aa63`; the report in PR 3 / tests `TestConverge_SecondRunIsANoOpAndPersistsTheReport`, `TestWriteReport_RecordsTheRunAndItsOutcome`
- [x] AC4 (failed probe fails the run, naming it) -> commit `b70aa63` / tests `TestRun_FailedProbeFailsTheRunNamingTheReconciler`, `TestRecordsMirror_ProbeFailsWhileTheDeployDirDiffers`
- [x] AC5 (unlisted OS is skipped, named) -> commit `b70aa63` / test `TestRun_UnlistedPlatformIsSkippedNotPassed`
- [ ] AC6 (install, probe, hand-off) -> PR 4b / bats `tests/install.bats` (the executed-script tests run under macOS `/bin/bash` 3.2 through the shebang; zsh is covered by sourcing it, which defines `install_dotf` and fires nothing): bad checksum and unreachable release place nothing, an unrunnable binary is never placed, a raw stream hands `--plan --only tools` to `<dest>/dotf converge`, `DOTF_BIN_DIR` moves the install, a kept pinned dotf is the one that converges, a failed install never hands off; Pester `install.ps1 executed` (args and exit status), first run in Windows CI: there is no `pwsh` on the Mac. Open until the release floor: `the pinned dotf can converge from zero` fails while `versions.conf` pins 0.66.0
- [x] AC7 (no live reference to the old names) -> PR 4b / bats `tests/install-old-name-guard.bats`; a planted reference in README fails it
- [ ] AC9 -> PR 6
- [x] AC8 (`dotf update` converges, exit semantics kept) -> PR 5 / tests `TestUpdate_ConvergesAfterAFastForward` (cmd, real git: nothing to pull converges nothing; a push is fast-forwarded and converged, the setup script running once on Linux and Windows), the `internal/update` table (every skip exits 0; `converge-failed` is the only error)

## Test status

- Test suite (PR 2a, macOS arm64): `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` -> ok, every package; `golangci-lint run` (pinned 2.12.2) -> 0 issues
- Manual smoke test (PR 2a): on the real Mac, `go run ./cmd/dotf converge --plan` -> `records-mirror [CHANGE] harness/ + 3 target(s) → ~/.dotfiles (70 to write, 0 unchanged)`; `~/.dotfiles` unchanged afterwards
- No regressions in existing test suite: yes (the tool catalog's platform tests pass through `internal/platform`)
- Test suite (PR 2b, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./...` -> ok; `golangci-lint run` -> 0 issues; bats on the touched suites (copilot-config, setup-windows, guard-bats-negation) -> 145/145 ok
- Manual run (PR 2b): `dotf converge --plan` -> records-mirror 71 to write, records-harness 3 instruction files to deploy; `dotf converge` -> `2 changed`; second run -> `0 changed, 2 ok`; doctor 51 -> 59 passing
- Test suite (PR 3, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./... -count=1` -> ok; golangci-lint 2.12.2 -> 0 issues; feature f3 -> PASS
- `TestReadmeHasNoInternalReferences` caught an ADR id in the new `converge` README section before push; removed
- Manual run (PR 3): `dotf converge` on the Mac printed `report: ~/.local/state/dotfiles/converge/last.json`; the file holds `"result": "ok"`, `"changed": 0`, `"goos": "darwin"` and the `records-mirror` entry with its detail
- Review hardening (PR 3): `env.Home()` falls back to the user database when HOME and USERPROFILE are both empty (`TestHome_FallsBackToTheUserDatabase`), and `env.StateDir()` returns an error instead of a path relative to the working directory when no absolute home resolves, or when XDG_STATE_HOME is relative (`TestStateDir_RefusesWhenNoAbsoluteHomeResolves`, `TestStateDir_IgnoresARelativeXDGStateHome`). The converge apply fails loudly on it; the skill gate fails open with the reason on stderr

- Test suite (PR 2c-1, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./...` -> ok; golangci-lint -> 0 issues. Tests: `TestDeployedMatchesSource` (7 cases), `TestCatalogMarkerMatchesCompileHarness`, `TestTheCopilotSourceSlotIsDeployOnly`, `TestCheckInstructionDrift_ARefreshedSourceRegionIsDrift`, `TestRecordsHarness_ARefreshedSourceRegionIsPlannedAndFailsTheProbe`
- Windows CI (PR 2c-1, first push): doctor reported `.copilot/copilot-instructions.md has drifted`, a false positive. The source reserves an empty `<!-- BEGIN HARNESS GENERATED -->` slot that the deploy fills with the skill catalog, and the first rule ("deploy-only iff the BEGIN line is absent from the source") stripped the catalog from the deployed copy but kept the slot in the source

- Test suite (PR 2c-2, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./...` -> ok; golangci-lint -> 0 issues. `TestDeployInstructions_TheCatalogFillsTheSourceSlotInPlace` fails when the slot is not filled in place (mutation run: the catalog lands twice)
- End to end (PR 2c-2): a throwaway HOME with a stub `copilot` and this branch's `dotf` on PATH; `compile-harness.sh --deploy` run twice, both rc=0. The second run delegated (`instructions current` for claude, opencode, pi and copilot), the copilot file holds one catalog region and no empty slot, and `dotf harness instructions --dry-run` then reports every target current
- Manual run (PR 2c-2): `dotf harness instructions --dry-run` against the Mac's real HOME, whose files `compile-harness.sh` wrote, reports claude, opencode and pi current and skips copilot (not on PATH)

- Test suite (PR 4a, macOS arm64): Go build, vet, `GOOS=windows` and `GOOS=linux` vet, `go test ./...` -> ok; golangci-lint -> 0 issues. Real-git checkout tests (clone then idempotent, behind fast-forwarded, dirty left alone, foreign repo refused, no git from zero). Live: a scratch HOME planned the clone and six steps `waits for checkout`; `--only checkout` cloned from GitHub, then `already current`
- Test suite (PR 4b, macOS arm64): Go build, vet, `GOOS=windows` vet, `go test ./...` -> ok; the full bats suite -> one failure, the release floor (by design until 0.67.0). Mutations: dropping the `exec` fails the three hand-off tests; disabling the exec probe (it runs the copy staged in the install directory, so a noexec /tmp cannot refuse a good release) fails `a release binary that verifies but cannot run here is never placed`
- Test suite (PR 5, macOS arm64): Go build, vet, `GOOS=windows` and `GOOS=linux` vet, `go test ./...` -> ok; golangci-lint -> 0 issues. Mutations: a child without the guard env fails `TestExecSetup_RunsTheOverrideMarkedAndFromTheCheckout`; an opaque result reported as a change fails both `TestLegacySetup_*Opaque*` tests

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- PR 2 was split into 2a and 2b to stay under the size cap. Apply mode landed with the engine in 2a, because plan and apply are one code path (`Reconcile(env, dryRun)`); only the persisted report moves to PR 3.
- The `platforms` matcher moved from the tool catalog into `internal/platform`, so the catalog and the reconcilers cannot drift apart (ADR-045 decision 7).
- PR 2b drives the instruction files from `harness/manifest.json` `agents.presence[]`, not from new `ai/deploy.json` entries. A new deploy field needs a manifest version the installed `dotf` cannot read, and a `replace` entry would fight `harness presence` over the same file, so AC3 could never pass. Skills and bindings move to a PR 2c.
- PR 2b runs `compile-harness.sh --deploy` as the records-harness reconciler instead of porting it: the script already deploys the instruction files, the skill catalog and the presence regions in the right order on Linux and macOS, so a port in 2b would have needed a new marker parser and a `dotf` release before the twins could use it (#1814 class). The comparator moved to `harness.StripRegions` is blind to a changed enforced region (F-064), which the 2c port fixes.

- PR 2c-1: deploy-only regions are identified by kind, on both sides: the presence roster, the skill catalog (its BEGIN line names it; pinned against compile-harness.sh) and the empty slot a source reserves for that catalog. Absence from the source was the first rule. It produced a false drift on Windows CI for copilot and would have treated a stale enforced region as deploy-only; the enforced region, with its sha and provenance, is now always compared literally

- PR 4 was split (2026-10-10). From zero, the entrypoint runs the *released* dotf, so swapping `install.sh` before a release carries the checkout step would hand a new machine a converge that cannot clone. And Linux from zero ran `setup-linux.sh`, which only the legacy step (PR 5) brings back. So 4a and PR 5 land first, and 4b waits for the release carrying both.
- A plan gates the steps after a pending clone or fast-forward (`Result.Gate`, lesson 385).
- The legacy step is reported skipped on darwin rather than left out of the registry. AC5 already reads "not supported on darwin", and the report then says why no setup ran.
- PR 4b: executed, every installer argument goes to `dotf converge`; the version, install directory and release location come from `DOTF_VERSION`, `DOTF_BIN_DIR` and `DOTF_RELEASE_BASE`, so `curl … | bash -s -- --plan` needs no installer flags of its own. The hand-off uses the binary `install_dotf` vetted (`_dotf_bin`, private because CI exports `DOTF_BIN` for the suites), never a PATH lookup: a fresh machine has no `~/.local/bin` on PATH. Under `irm | iex` the ps1 never calls `exit`, which would close the caller's shell.
- PR 4b: a test pins `DOTF_VERSION` at or above 0.67.0, the first release whose converge clones and runs setup. It is the gate that keeps this PR from landing before that release.
- A script that cannot plan gets its own status, `opaque`. Counting it as a change would make every second run report one, and counting it as converged would hide what the script did.

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
