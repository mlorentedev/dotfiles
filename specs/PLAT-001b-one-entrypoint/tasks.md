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

- [ ] [AC9] CI job on clean `macos-latest` and `ubuntu-latest`: `install.sh`, `dotf doctor --scope machine` (D10: a runner has no identity), then a second `dotf converge` that reports zero native changes. It asserts the first run converged before it checks idempotence (lesson 328). Runs on pull requests to `main` and on push to `main`, so the default branch carries a run the verification can read; non-required until green
- [x] `.github/workflows/from-zero.yml`: the job above, the PR's source build on PATH (install.sh keeps a `dev` build), `DOTFILES_REPO_DIR` at the workspace, `GITHUB_TOKEN` for mise's downloads; the owner's trigger (install path on PRs, every push to main, 2026-10-10)
- [ ] Gap, deliberate: the job never exercises install.sh's clone-when-absent path (the workspace is the checkout) nor its release download (covered by the file:// fixture in `tests/install.bats`)
- [ ] After #2013 P5c PR2 (the agents under `# mise: latest`): X1 asserts `claude`, `pi` and `agy` resolve on PATH after the first converge, on both runners
- [x] Each red cause of the first runs becomes a row on the #2013 ledger, or is fixed here when it is the job's own. Run 38115559203, classified on #2013: kubectl on darwin (PLAT-001c's `# mise: cli` pin), the hive daemon (S2), `.profile`/`.inputrc` on darwin (P7), `~/Applications` and `*_HOME` on linux (T3); the dispatcher became the git-hooks step (#2340), the plugin miscount BUG-121 (#2336), test-windows' fabricated identity the X4 row
- [x] Doctor gate (owner, 2026-10-11: a ratchet of known failures): `.github/scripts/from-zero-doctor-gate.sh` fails on a [FAIL] that `from-zero-known-failures.txt` does not list for the OS, on a listed entry no [FAIL] matches (stale: the list only shrinks), and on a non-zero doctor with no [FAIL] line. Each entry names its #2013 owner above an `# example:` line; `tests/from-zero-doctor-gate.bats` covers the verdicts under bash and zsh and checks every entry against its example. Replayed on run 38115559203: 4 known on darwin, 8 on linux, 0 new, 0 stale; dropping an entry or a FAIL turns it red
- [x] Runbook `docs/runbooks/guide-new-machine.md`: the one-liner per OS, what `--plan` shows, how to read the report, what `skipped` on darwin means; it also orders the identity restore (#2013 D10)

### PR 6a — doctor scopes its checks: machine or identity (#2013 D10)

- [x] Failing test: every entry of the doctor sweep has a kind, and none is left at the zero value; an identity entry names its section and its restore step, once (`TestSweep_EveryCheckIsClassified`)
- [x] Failing test: an identity entry's section is a title some check prints, so the SKIP lands under the heading a default run shows (`TestSweep_IdentitySectionsAreTheTitlesTheChecksPrint`)
- [x] Failing test: end to end on a machine with no identity, the default scope reports the missing vault and `--scope machine` runs no identity check, printing one SKIP per identity entry and never failing on one (`TestRun_MachineScopeSkipsEveryIdentityCheck`)
- [x] Implement the sweep table (`check{kind, section, restore, run}`), `Options.Scope`, `ParseScope` and `dotf doctor --scope all|machine`; a plain `dotf doctor` is unchanged
- [x] Split the two checks that mixed both kinds: `checkAgeIdentity` (the key) from `checkSecretsTooling` (the binaries), and `checkAgentSecretConfigs` (configs rendered from secrets) from `checkOpenCode` (the binaries)
- [x] ADR-045 amendment for D10, D11 and D12; the runbook orders the restore the SKIP lines cite
- [x] Follow-up: X1 (#2331) runs `dotf doctor --scope machine` (`from-zero.yml`, step `dotf doctor --scope machine`)
- [x] Follow-up: test-windows runs `--scope machine` and drops the identity it fabricates (age key, vault skeleton, `obsidian.cmd` stub)
  - The machine scope also skips doctor's gh checks. spec-issue-state was the one verifying something on the runner (CI-004 AC8), so `dotf spec audit` became its own step with `GH_TOKEN`; branch-protection only ever printed `not verified` (HTTP 403) under the job token, so nothing is lost there
  - `DOTFILES_DOCTOR_NO_IDENTITY` (TEST-005) had no other caller and was deleted: under the machine scope Bitwarden reach is skipped by declaration
  - Guard: `tests/ci-windows-doctor-gate.bats`, two tests, each mutation-checked (a re-added `age-keygen -o`, an audit step without the token)

### PR 6b — the git-hooks step (#2013 X1 finding)

- [x] Failing test: from zero, a plan names the mirror and the wiring and changes nothing; the apply makes both, the probe passes, and the next run reports 0 (`TestGitHooks_FromZeroDeploysAndWires`)
- [x] Failing test: a foreign `core.hooksPath` is kept and named, only the mirror counts, the probe passes, and the next run reports 0 (`TestGitHooks_KeepsAForeignHooksPath`); an unset hooksPath or a stale mirror fails the probe
- [x] Failing test: only git's exit 1 reads as an unset hooksPath; any other read failure is an error and nothing is written, in `Inspect` and the install alike (`TestAFailedHooksPathReadIsNotAnUnsetOne`, PR-Agent on #2340)
- [x] `hooks.Inspect` and `hooks.Apply`: what `dotf hooks install` would change, read without writing (the mirror compared as `copyTree` and `markExecutable` would write it, the bit ignored on Windows), and the install through a given runner
- [x] The `git-hooks` reconciler, after `git-config`, every OS
- [x] Replace the hooks blocks of `setup-linux.sh` and `setup-windows.ps1` (one twin pair) with `dotf converge --only git-hooks`, as git-config already is: a setup script run on its own (test-windows) still gets the dispatcher; `tests/guard-setup-no-hooks-install.bats` keeps the blocks out and the delegation in

### PR 6c — converge guides the identity restore (#2013 D12)

- [x] Failing test: from zero on a terminal, Enter at each prompt walks the chain in the runbook's order; the age key is waited for, never run; the chain ends restored (`TestGuide_FromZeroRunsTheChainInOrder`)
- [x] Failing test: a skipped, failed or ineffective step, or the end of input, stops the chain; no later step is offered (`TestGuide_AStepNotDoneStopsTheChain`); a failed `secrets verify` does not (`TestGuide_AFailedVerifyDoesNotStopTheChain`), and the vault step wires git-config before it clones (`TestGuide_TheVaultStepWiresTheHelperBeforeItClones`)
- [x] Failing test: without a terminal a converge ends with the identity plan and runs nothing; on a terminal it walks; `--plan` and `--only` never walk; the exit status is the converge's (`cli/internal/cmd/identity_test.go`)
- [x] `internal/identity`, a leaf package: the chain, the plan, the guide, and the Restore texts doctor's SKIP lines cite (moved from doctor); `dotf identity restore [--plan]`; converge's epilogue
- [x] Runbook, ADR-045 D12 and `cli/README.md` say what the walk runs, and why `gh auth login` is in the chain
- [x] Lesson 398: under `curl | bash` stdin is the script, so the walk reads the person from `/dev/tty` and hands that terminal to `bw` and `gh`

### PR 6d — plugin failures counted and named (#2013 X1 finding, BUG-121 #2336)

- [x] Failing test: every declared id lands in exactly one of Present, Added and Failed, and a failed install keeps the CLI's error (`TestSyncAccountsForEveryIDAndKeepsTheCause`)
- [x] Failing test: a failed install puts `1 added, 1 failed, 0 already present` under a `failed` state and names the plugin with its cause (`TestDeployCmd_AFailedPluginInstallIsNamedAndFailsTheRun`)
- [x] `PluginReport.Causes`, filled by `Syncer.Sync`; `reportCounts` counts failures for both claude steps, so the MCP summary gains the count too
- [x] Delete the plugin loops of `setup-linux.sh` and `setup-windows.ps1` (one twin pair): both already run the bare `dotf deploy`, which installs `ai/claude/plugins.json`
- [x] Guard: `tests/claude-plugins.bats` fails when either script runs `claude plugin install|list` outside a comment or quotes a declared id; the BUG-004/011 structure tests that read the deleted loops are retired, and the zero-usage list guard reads `plugins.json`
- [x] After merge: the next X1 run names why the installs fail on a fresh runner; record the cause on #2013. Run 38365444507: `Plugin "gopls-lsp" not found in marketplace "claude-plugins-official"` for all three, on both converges; filed as BUG-126 (#2378), fixed in PR 6e; the cause is recorded on #2013 (https://github.com/mlorentedev/dotfiles/issues/2013#issuecomment-6140797662, row 3)

### PR 6e — a plugin's marketplace is registered from a declared source (BUG-126 #2378)

- [x] First fix refuted before its PR: refreshing each marketplace (the CLI's hint) failed on dispatched run 38366816961 with `Marketplace 'claude-plugins-official' not found. Available marketplaces: anthropic-plugin-directory`. The `latest` Claude Code no longer registers it; upstream `anthropics/claude-plugins-official` still names itself so and carries the three plugins (lesson 404)
- [x] Failing tests first: each marketplace is prepared once before its first install, added from its source when missing and refreshed when registered, with the registry listed once and read again after each add (`TestSyncPreparesEachMarketplaceOnceBeforeItsFirstInstall`); nothing is called on a converged box or a dry run (`TestSyncPreparesNothingWithoutAnInstall`); a failed preparation, or a marketplace with no source, is named in the cause of an install that then fails (`TestAFailedPreparationIsNamedInTheCauseOfAFailedInstall`)
- [x] `ai/claude/marketplaces.json`, read by `LoadMarketplaceSources`: its own file, since every released dotf reads `plugins.json` with `DisallowUnknownFields` and a new field would fail the plugin step on a box whose binary predates it. Absent means no sources; present, it is read strictly (`TestLoadMarketplaceSources`)
- [x] Guard: `TestEveryDeclaredPluginsMarketplaceHasASource` fails when a plugin is declared from a marketplace with no source
- [x] Review fixes (PR-Agent on #2383): the deploy's read of `marketplaces.json` is tested (`TestDeployCmd_APluginsMarketplaceIsPreparedFromTheDeclaredSource`, mutation-checked), and an add is followed by a registry read, so a source that registers under another name is the named cause (`TestARegistrationUnderAnotherNameIsNamedInTheCause`)
- [x] Evidence: from-zero run 38368564873 on `66201aba`, ubuntu: `installed claude-plugins 3 added, 0 already present` on the first converge, `in sync claude-plugins 0 added, 3 already present` on the second (the report omits a zero failure count). The macOS runner prints `[SKIP] claude ... not installed` on both converges, before and after the fix: converge installs no Claude Code on macOS, which is #2317 (PLAT-002), so the defect never reached that runner and this PR cannot prove anything there

### PR 7 — the workstation SSH target (#2013 A1, SEC-011 #2356)

Owner (2026-10-12): a machine that accepts SSH is a converge step here, not a spec of its own. SSH is reachable through the mesh only: no LAN alias, no public bastion.

- [x] PR 7a, interim: `msi` and `macmini` client profiles in `ssh/config` (MagicDNS names, `ForwardAgent no`, keepalives, `IdentitiesOnly`), and `docs/runbooks/guide-workstation-ssh-target.md` pairing every server step with its check
- [x] PR 7a: `scripts/windows-openssh-authorize.ps1` narrows the OpenSSH firewall rule to the `Tailscale` adapter and `100.64.0.0/10` on every run, before sshd starts, and refuses a host without the adapter; the office LAN alias goes (lesson 401)
- [x] PR 7b: the opt-in. `machine.json` gains a `roles` section, written by `dotf env role add|rm <ROLE>` and read by `env.LoadMachineRoles`; an unknown role is an error in both directions. The writers now edit the file per top-level section: `dotf env set` used to drop `machine`, `pools` and any role on every write (lesson 402)
- [x] PR 7c: `workstation-target` on darwin (`cli/internal/sshtarget`, after `env-persist`). `Inspect` is the plan and needs no root: the authorized key, the host keys, the sshd drop-in `01-workstation.conf` rendered from `ssh/sshd-workstation.conf`, and Remote Login. The preflight refuses a node with no mesh address and a main config that does not include the drop-in dir. `Apply` asks for root once, writes the host keys before `sshd -t` and the drop-in before sshd listens, then proves the result: sshd accepts the owner's key from the mesh address and refuses it from the LAN address. A failed proof or a rejected drop-in undoes the run's changes in reverse order. Without a terminal the apply is skipped with the plan as its reason
- [x] ADR-045 amendment 2026-10-12: how converge gets root (the plan never asks; the apply asks once on the terminal; no terminal means skipped with the plan)
- [x] PR 7c acceptance on the macmini, with the owner at the sudo prompt: plan, apply, and a second run reporting 0 changes; the outputs go on #2356 and below in `verification.md`
- [x] PR 7d, doctor half: the `Workstation SSH target` section runs `Inspect` with the step's gates (darwin only, skipped until the machine opts in, skipped off the mesh). A config that shadows the drop-in is a WARN with Inspect's own message. A pending resource is a WARN naming `dotf converge --only workstation-target`. `--fix` does not call `Apply`, because doctor never prompts for root (the `env-persist` precedent). The proof stays converge's: it asks sshd over the network, and doctor is read-only and fast. `sshtarget.New` builds the target for converge and doctor alike
- [ ] PR 7d: the firewall (a pf anchor and the LaunchDaemon that reloads it at boot), the power policy (`pmset`) and the unattended tailscale service, each with its undo, each added to the plan doctor already reports
- [x] PR 7e, Linux half: `sshtarget.Systemd` drives sshd under systemd. It finds `ssh` (Debian, Ubuntu) or `sshd` (Fedora, Arch), and the socket unit when there is one. Enable and its undo act on the same unit, the socket when the distribution ships one. The reload reaches the service, and only when it runs. `sshd -t` gets `/run/sshd` when socket activation has not created it yet. The step and the doctor section run on darwin and linux. Power on Linux: a logind lid policy for external power only, proven through logind's D-Bus property where logind can reload. Code and unit tests are done; accepted on msi (the first test is `ssh msi true` from the macmini) closes it. Accepted 2026-10-13 on msi with dotf 0.68.0: two converges (1 changed, then 0), `--plan` reports converged, `ssh -o BatchMode=yes msi` from the macmini connects (evidence on #2356 and in `verification.md`)
- [ ] PR 7e, Linux firewall: the step converges the UFW rule the runbook applies by hand (port 22 `in on tailscale0` only, default deny incoming), with its undo and a plan line. Found 2026-10-13: the Linux half writes no firewall rule at all, so on a converged node only sshd's `AllowUsers` keeps the LAN out. msi has UFW active with rules not yet read (needs root)
- [x] Client agent: a last `Host *` block in `ssh/config` sets `AddKeysToAgent yes` and `UseKeychain yes` behind `IgnoreUnknown UseKeychain`, so a reboot never needs `ssh-add --apple-use-keychain` (a step of the msi acceptance). `tests/ssh-client-agent.bats`: the file parses and adds keys to the agent on the running OpenSSH, the guard order holds, and Apple's build accepts `UseKeychain`. Ubuntu 24.04's OpenSSH parses it, and rejects it with `Bad configuration option: usekeychain` once `IgnoreUnknown` is dropped
- [ ] PR 7e, Windows half: port `scripts/windows-openssh-authorize.ps1` into the step. X1 renders the plan on all three OSes
- [ ] Host keys pinned (#2356): `ssh/known_hosts` deployed to `~/.ssh/known_hosts_pinned`, and the workstation profiles and github.com refuse any other key with `StrictHostKeyChecking yes`. Each line carries the fingerprint it was verified against out of band; `tests/ssh-known-hosts.bats` guards parse, fingerprint and coverage. The SSH CA (#2373, EPIC D11 / S11) replaces the list
- [ ] The runbook's sections 2 to 4 become one `dotf converge --only workstation-target` per machine, once PR 7d lands. Linux and macOS sshd and Linux power collapsed 2026-10-13; the firewall on every OS, macOS power and Windows remain hand steps

### PR 8 — agent extensions: MCP servers and plugins (#1843 B10, BUG-127 #2387)

Converge never registered Claude Code's MCP servers or installed its plugins. Only a bare `dotf deploy` did, and only the setup scripts called it, so a Mac from zero had a Claude Code without hive or plugins. Measured on from-zero 38376819599: ubuntu prints `claude-plugins 3 added`, macOS has no row.

- [ ] `claude.ApplyMCP` and `claude.ApplyPlugins`: the orchestration moves from `cmd` to `internal/claude`, and the bare deploy and converge call the same code. A failed install's error carries the CLI's cause, because converge prints one line per step (BUG-121)
- [ ] `agent-extensions`, every OS, after `workstation-target` and before `legacy-setup`: after `tools`, whose PATH reaches the claude that mise installs, and last among the native steps, because the first failure stops a run and a plugin that cannot reach its marketplace must not stop git or the SSH target. Skipped while claude is absent. `Probe` re-plans as a dry run, which asks claude and changes nothing
- [ ] Tests: a fresh claude gets the server and plugins, the probe passes and the second run changes nothing; a plan makes no mutating call; a failed install fails with its cause; an install that exits 0 and installs nothing fails the probe by name; `convergeOptions` wires the deploy seam; the registry order
- [ ] Evidence: from-zero on macOS with claude installed (P5c, #2392) prints the step adding servers and plugins on the first converge and 0 changes on the second
- [ ] Not here: agy's `mcp_config.json` (CLI-105 #2384, the next PR, into the same step); a doctor check for a missing plugin or server (CLI-106 #2393)

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
