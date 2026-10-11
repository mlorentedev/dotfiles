---
tags: [spec, tasks, templates]
created: "2026-10-06"
---

# Tasks - PLAT-001b-one-entrypoint

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `docs/plat-001b-one-entrypoint-spec` (PR 1); each later PR gets its own branch from `main`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" (D4 and D2's reach close when the ADR-045 PR merges)

## Implementation

> One PR per block, each under ~300 executable lines, each landing with the guard that would have caught its defect. Go work verifies with `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` and the pinned `golangci-lint`; shell work with shellcheck and bats under bash 3.2 and zsh.

### PR 1 — design (docs only)

- [x] ADR-045 `proposed`, with #2013 D4 and D2's reach as its open questions
- [x] This spec: proposal, tasks, features
- [ ] Merge = acceptance of ADR-045 (owner)

### PR 2a — the engine: `dotf converge [--plan]` and the records-mirror reconciler (#1843 B6)

> Split from the original PR 2, which did not fit the ~300-line cap: the engine and one reconciler here, the instruction files and the bindings in 2b. Apply mode landed with the engine because plan and apply are one code path; the persisted report stays in PR 3.

- [x] [AC1] Failing test: `PlanMirror` counts what `Mirror` would write and creates nothing, the deploy dir included; implemented on `Mirror`'s own walk with a `dryRun` flag
- [x] [AC5] Extract `internal/platform` (`Supports`, `Unknown`) from the tool catalog, so the catalog and the reconcilers share one matcher; the catalog's tests still pass
- [x] [AC1] [AC4] [AC5] Failing tests for the runner: a plan applies and probes nothing; an unlisted OS is `skipped` naming it; a failed probe fails the run naming the reconciler and the reconcilers after it are not reached
- [x] Implement `internal/converge`: `Reconciler` (name, platforms, `Reconcile(env, dryRun)`, mandatory `Probe`), `Run`, `Report`, and the ordered `Registry`
- [x] [AC1] [AC3] Failing tests, then the `records-mirror` reconciler: the plan writes nothing, the apply passes its probe, a second plan reports 0 changes
- [x] [AC1] `cmd/converge.go`: `dotf converge [--plan] [--repo]`; a plan on an empty HOME writes nothing, a second apply reports 0 changed
- [x] On the Mac: `go run ./cmd/dotf converge --plan` reports `records-mirror` with 70 files to write and leaves `~/.dotfiles` untouched

### PR 2b — the instructions-first records step: `records-harness` (#2016)

> Second pivot, after reading `compile-harness.sh`: its `--deploy` already deploys every `agents.presence[]` instruction file, then the skill catalog and the presence regions, in the order they need, on Linux and macOS. 2b runs it as a reconciler with a Go plan and probe instead of porting it, so there is no new marker parser and no coupling to a `dotf` release. The Go port of `deploy_instructions` is PR 2c.

- [x] [AC2] Move doctor's region stripper to `harness.StripRegions` (one comparator for doctor and converge); `harness.PresenceTarget` gains `Source`
- [x] [AC2] [AC3] Failing tests, then `records-harness` (linux, darwin): the plan counts instruction files that are missing or differ from their source outside the harness regions, honouring `requires_command`, and writes nothing; the apply runs `compile-harness.sh --deploy` through an injected runner (lesson 335), which fails loudly when none is wired; the probe requires every file current and every rendered presence region `current`
- [x] [AC2] The apply runs the deploy even when the instruction files are current, so a changed skill record is deployed (skills are not planned yet)
- [x] [AC2] Doctor: an installed agent without its instruction file is a FAIL naming both (#2016); the unconditional CLAUDE.md existence FAIL is removed, and AGY.md's is gated on `agy` being installed (#843)
- [x] Delete from `setup-linux.sh` the CLAUDE.md force copy, the bulk `ai/claude/*` copy and the copilot-instructions copy: `compile-harness.sh --deploy`, already run by the script, deploys all three (ADR-020 §5). The bats tests that pinned the copies now pin the manifest entries
- [x] On the Mac: `dotf converge` deployed `~/.claude/CLAUDE.md`, the opencode and pi instruction files and 33 skills; a second run reported `0 changed`; doctor went from 51 to 59 passing

### PR 2c — instruction files in Go, then the twins' copies go (F-064)

- [x] PR 2c-1 (F-064): `harness.DeployedMatchesSource` replaces `StripRegions`, and `records-harness` (plan and probe) and doctor's instruction-drift check share it
  - Deploy-only regions are identified by kind and stripped from both sides: the presence roster, the skill catalog (its BEGIN line names it, pinned against `compile-harness.sh`) and the empty slot a source reserves for that catalog
  - Every other region, the enforced one with its sha and provenance included, is compared literally, so a refreshed source region is drift on a stale deployed copy
  - Tests: `TestDeployedMatchesSource` (refreshed source region, text outside regions, lost source region, deploy-only regions, filled catalog slot, CRLF), `TestCatalogMarkerMatchesCompileHarness`, `TestTheCopilotSourceSlotIsDeployOnly`, `TestCheckInstructionDrift_ARefreshedSourceRegionIsDrift`, `TestRecordsHarness_ARefreshedSourceRegionIsPlannedAndFailsTheProbe`
- [x] PR 2c-2: `harness.DeployInstructions` and `dotf harness instructions [--dry-run]` write each source verbatim with the deployed file's deploy-only regions, so a refreshed source region replaces the stale one and the presence roster and skill catalog survive
  - The catalog fills the source's empty slot in place, the position `compile-harness.sh` `replace_region` writes it to, so the two writers agree byte for byte and neither duplicates it; the presence roster is appended, as `dotf harness presence` does
  - `compile-harness.sh` delegates `deploy_instructions` to it when the installed dotf carries the subcommand (the `dotf_knows_subcommand` probe) and keeps the copy as the fallback, so no release is needed to land it
- [ ] Gated on the release that ships `harness instructions` and the `DOTF_VERSION` bump (#1814 class): `setup-windows.ps1` drops its CLAUDE.md and copilot-instructions copies for `dotf harness instructions`
- [ ] [AC2] `records-skills` (feature f11 moves here once skills are planned in Go): the skill records planned from their rendered form
- [x] `records-bind` (#2232): `harness.Bind` returns one outcome per target and `dotf harness bind` renders it, so converge and doctor count changes without parsing text
  - Every OS, after `configs-deploy` (which seeds the settings files the hooks merge into); the plan writes nothing, the probe re-plans and requires zero writes, and a second run reports 0 changes; with no dotf path resolver wired the step is skipped and says so, as the other steps report a runner they were not given
  - Hooks whose markers another writer stripped are converged (`TestRecordsBind_HooksWithStrippedMarkersAreConverged`), resting on #2276's identity by command signature
  - Doctor's `Harness hook bindings` section plans the same bind: drift FAILs naming the harness and `dotf doctor --fix`, which binds. Both setup scripts' bind warnings already promised this check (#2232), and it is now true
  - A target that fails after writing (a retirement it cannot read) still counts its write: `harness.Bind` returns that outcome with the error (#2281), and the step passes both to the runner, which records changes alongside an error
  - On the Mac: `converge --plan --only records-bind` -> `2 harness(es) in sync`, pi and opencode skipped (emit:false); `doctor --verbose` -> claude and agy `hooks current`
- [x] [AC12] Catalog in the `tools` step (#2013): converge installs `packages.json` on every OS, which nothing did on darwin
  - An apply walks install, mise sync, install, the order of `setup-linux.sh`: the first pass places mise, the sync brings uv, and the second pass installs what waited on it, so one run converges a fresh machine
  - `tools.Installer.Plan` gains the two outcomes the apply already skipped, so plan, apply and probe agree: `missing-manager` now covers npm as well as uv (node is installed by nothing yet, T4b), and `needs-sudo` asks the apply's own `sudo -n true` classifier up front. Without it the probe would re-plan an apt entry as `install` and fail every unattended Linux `dotf update`
  - The probe fails on an entry still to install or refused, and on a manager this machine's tools install (mise from the catalog, uv as a mise pin) that the run's PATH cannot reach; it reports a genuine wait (npm, which nothing installs yet) or a needs-sudo instead of failing. The second pass walks only what waited, so a failure is attempted and reported once
  - The setup scripts keep their own install lines until #2275 makes `install.sh` hand off to converge
  - On the Mac: `converge --plan --only tools` lists the same ten entries as `dotf tools install --dry-run`
- [x] [AC13] Path file in converge (#2013 P6, F-005): an `env-generate` step between `records-mirror` and `records-harness`
  - The rc files render `paths.sh` only when it is missing; a contract or `machine.json` change left it stale, doctor failed it and nothing repaired it
  - Plan reads `env.Generate` in check mode, apply writes, the probe re-checks; the deploy dir it writes into exists once the mirror has run
  - `configs-deploy` stays after `tools`: entries that `require` an agent would be skipped on a fresh machine's first run and deployed on the second, which breaks AC3
  - F-052 (`CLAUDE_CONFIG_DIR` before the first `claude` run) was already closed by #1992: the deploy's one `claude` call pins it (`deploy_claude.go`)

### PR 2d — the configs reconciler (#1843 B15)

> Found while landing #2236: converge had no step for `ai/deploy.json`, so a template change (the agy model pin) still needed `dotf deploy` on every machine. ADR-045 decision 4 already orders `configs` after `tools`.

- [x] [AC10] Failing tests, then `deploy.Run`: one loop for `dotf deploy` and converge (OS selector, `requires`, deploy, private-dir tightening); `dotf deploy` prints its rows from the result
- [x] [AC1] Failing test, then a plan of a rendered entry stages in a scratch directory and creates nothing under HOME
- [x] [AC10] Failing test, then `deploy.ErrRenderIncomplete`: a strict renderer turns a placeholder the store could not resolve into a skip that keeps the installed file (lesson 378)
- [x] [AC10] [AC3] `configs-deploy` between `tools` and `git-config` (git-config's probe needs the include target a config entry deploys); plan, apply, probe and a second plan with 0 changes; a template change fails the probe until applied; no renderer wired fails loudly
- [x] On the Mac: `go run ./cmd/dotf converge --plan --only configs-deploy` reports 20 configs in sync, `agy-settings` to deploy, 1 not for this machine
- [ ] Orca's hooks and Claude Code's MCP servers and plugins stay with a bare `dotf deploy`; their reconciler is #1843 B10

### PR 2e — the macOS environment step: `env-persist` (#2013 S3)

> GUI apps on macOS read no rc file, only the user's launchd session (F-021). The owner released S3 from S1 (2026-10-09), so it lands as a converge step now and S1 absorbs the login agent later.

- [x] [AC11] Failing tests, then `env.LaunchdUserEnv`: `launchctl getenv/setenv/unsetenv` as the darwin `UserEnvStore`. An unset name reads as absent (getenv exits 0 and prints nothing), so an empty value is refused; Persist twice changes nothing
- [x] [AC11] `env.LaunchAgentPlist`: `~/.local/bin/dotf env persist` at load, absolute paths, well-formed XML
- [x] [AC11] [AC3] [AC5] Failing tests, then `env-persist` (darwin, last in the registry, closing ADR-045's configs step): the plan writes nothing; the apply sets the variables, writes the plist and bootstraps it only when `launchctl print` fails or the plist changed (bootout first); the probe requires the plist current, the agent loaded and no drift; a second run reports 0 changes; a failed bootstrap fails the run
- [x] Doctor's persisted-environment check runs on macOS through the same store, and names `dotf converge --only env-persist` there; `dotf env persist --help` describes the macOS scope
- [x] On the Mac: the plan listed 11 variables, the marker and the agent; the apply loaded the agent (`last exit code = 0`); a second run reported `0 changed`; `launchctl getenv VAULT_PATH` answers
- [x] Windows check (#2013 wave rule; the registry, doctor and `env` are shared paths): `test (windows-latest)` passed on `9babaddf`. Its first run caught the plist rendered with `filepath.Join`, now `path.Join`, because launchd reads POSIX paths whatever OS renders it
- [ ] [AC11] Owner, at the Mac: quit and relaunch an app from the Dock and confirm it sees `VAULT_PATH` (the agent's shell cannot launch a GUI app without inheriting its own environment)

### PR 3 — the persisted report (#1843 B7)

- [x] [AC3] Failing test: a second run on a converged temp HOME reports zero changes and writes the report under the user state directory
- [x] [AC3] Implement the report (JSON, one entry per reconciler, the run's result and error), written atomically under `env.StateDir()`, which the skill gate's ledger now shares; a plan writes none. `cli/README.md` documents `converge`, which shipped in PR 2a without a section
- [ ] [AC2] On the Mac: `dotf converge` deploys `~/.claude/CLAUDE.md` and the skills; `dotf doctor` no longer fails on the Claude instruction file

### PR 4a — the checkout step, so converge can run from zero

Ships first, on its own release: from zero, `install.sh` downloads the *released* dotf and execs `converge`, so the entrypoint swap in 4b only works once a release can clone the checkout.

- [x] `update.Assess`: the fast-forward decision without the merge, so a plan reports `behind` and moves nothing; `Sync` is Assess plus `merge --ff-only`
- [x] Add the checkout reconciler (clone if absent, fast-forward under ADR-019 D2), first in the registry; it refuses a directory that is not a dotfiles checkout, and names `xcode-select --install` on macOS when git is missing
- [x] `Result.Gate`: under a plan, a step that has not converged yet (a clone or a fast-forward pending) holds back the steps after it (`waits for checkout`) instead of planning against a tree the apply would change first; an apply ignores it
- [x] `dotf converge` without `--repo` resolves the checkout like `dotf deploy`: the working directory's dotfiles checkout (`env.IsDotfilesCheckout`; another project's repository does not count), else `DOTFILES_REPO_DIR`, else `env.DefaultCheckoutDir` (drift-tested against `env-contract.json`)
- [x] Real-git tests: clone then idempotent, behind fast-forwarded (plan leaves HEAD), dirty left alone, foreign repo refused, no git from zero; a cmd test plans the clone first from an empty cwd

### PR 4b — one entrypoint: `install.sh` and `install.ps1` at the root

Gated on a release that carries 4a and PR 5 (0.67.0): `tests/install.bats` fails while `versions.conf` pins an older one.

- [x] [AC7] Failing test: no live file names `install-dotf.sh`, `install-dotf.ps1` or `DOTFILES_SKIP_SETUP` (allow-list: `docs/adr/` including audits, `docs/lessons/`, `specs/` — live specs describe the migration they deliver — and `CHANGELOG.md`)
- [x] [AC6] `git mv scripts/install-dotf.{sh,ps1}` to the root `install.{sh,ps1}`, replacing the old `install.sh`; the standalone path ends in `exec dotf converge "$@"`, the sourced `install_dotf` contract is unchanged
- [x] [AC6] bats (bash 3.2 and zsh) and Pester: a bad checksum and an unreachable release fail and place nothing; the hand-off execs `dotf converge` with the arguments
- [x] [AC7] Move every live reference: setup twins, `checks_tools.go`, `stdout_contract_test.go`, the vault-maintenance scripts, README, `cli/README.md`, SECURITY.md, the release runbook

### PR 5 — legacy reconcilers and `dotf update`

- [x] [P] Failing test: on linux and windows the legacy reconciler runs the setup script after every native reconciler and plans `opaque`; on darwin it is reported skipped with the OS named (AC5), rather than absent, so the report says why no setup ran
- [x] Implement the legacy reconciler: `StatusOpaque` (never counted as changed or converged), last in the registry, and a guard env so a setup script that calls `dotf converge` cannot start itself again
- [x] [AC8] Failing test: `dotf update` runs `dotf converge` after a fast-forward, keeps exit 0 on every non-actionable case, and exits non-zero only on a converge failure
- [x] [AC8] Route `dotf update` through `converge` (one `runConverge` path for both commands); `DOTFILES_SELFUPDATE_SETUP_CMD` stays an override, now of the command the legacy step runs, until #1843 A5 decides the scheduled path

### PR 6 — from-zero proof (#2013 X1)

- [ ] [AC9] CI job on clean `macos-latest` and `ubuntu-latest`: `install.sh`, `dotf doctor`, then a second `dotf converge` that reports zero native changes. It asserts the first run converged before it checks idempotence (lesson 328). Runs on pull requests to `main` and on push to `main`, so the default branch carries a run the verification can read; non-required until green
- [x] `.github/workflows/from-zero.yml`: the job above, the PR's source build on PATH (install.sh keeps a `dev` build), `DOTFILES_REPO_DIR` at the workspace, `GITHUB_TOKEN` for mise's downloads; the owner's trigger (install path on PRs, every push to main, 2026-10-10)
- [ ] Gap, deliberate: the job never exercises install.sh's clone-when-absent path (the workspace is the checkout) nor its release download (covered by the file:// fixture in `tests/install.bats`)
- [ ] After #2013 P5b PR2 (the agents under `# mise: latest`): X1 asserts `claude`, `pi` and `agy` resolve on PATH after the first converge, on both runners
- [ ] Each red cause of the first runs becomes a row on the #2013 ledger, or is fixed here when it is the job's own
- [ ] Runbook `docs/runbooks/guide-new-machine.md`: the one-liner per OS, what `--plan` shows, how to read the report, what `skipped` on darwin means

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-001b-one-entrypoint/features.json`):

```json
[
  {
    "id": "PLAT-001b-one-entrypoint-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
