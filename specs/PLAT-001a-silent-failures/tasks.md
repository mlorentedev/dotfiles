---
tags: [spec, tasks, templates]
created: "2026-10-05"
---

# Tasks - PLAT-001a-silent-failures

> TDD order. One task = one focused commit. Each row of #2013 track W is one PR that ticks its own block here.
>
> **Inline markers**: `[P]` = no dependency on another unchecked task; `[AC<n>]` = helps satisfy acceptance criterion `<n>` in `proposal.md`.

## Setup

- [x] Branch created from main: `feat/plat-001-w1-install-exec-probe`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## W1 — post-install exec probe (`dotf tools install`)

- [x] [AC1] [AC2] Failing test: a staged release binary that does not run, prints no version, or reports below the pin fails the install and leaves Dest untouched (`TestInstall_StagedBinaryMustExecuteAtThePin`)
- [x] [AC3] Failing test: an npm or uv-tool install that exits 0 but whose tool does not run on PATH (or is shadowed below the pin) fails, naming the tool (`TestInstall_PackageManagerToolMustRunOnPathAfterward`)
- [x] [AC1] [AC2] Implement: stage under the command name, probe after the checksum gate, place only on success (`fetchVerifyPlace`, `checkProbed`)
- [x] [AC3] Implement: `verifyOnPath` after `npm install -g` / `uv tool install`
- [x] [AC4] Existing happy-path tests updated to model a binary that executes (probe seam) and a package manager that puts the tool on PATH
- [x] End-to-end on darwin/arm64 against real GitHub assets: the darwin asset installs; the linux asset is refused with nothing placed (0.64.0 placed the ELF and exited 0)

## W3 — bash 3.2 (ADR-003)

- [x] [AC5] Guard first: `tests/bash32-portable.bats` fails on any bash-4-only construct in a shell file bash runs. On `main` it reports 4 `mapfile` sites in compile-harness.sh, 1 in bitacora-rollout.sh, and `declare -A` in nan-quality-bench.sh and pin-actions.sh.
- [x] [AC5] compile-harness.sh: `read_array` replaces the four `mapfile`s; empty-array expansions are guarded for 3.2 under `set -u`
- [x] [AC5] bitacora-rollout.sh, nan-quality-bench.sh and pin-actions.sh ported (read loop; parallel indexed arrays; a tab-separated cache)
- [x] [AC6] Harness tests made BSD-sed portable (`sed -i.bak … && rm`), so they can run on macOS

## W6, W7 (with #2013 P6 and the W2b shell activation)

- [x] W7: the first `dotf deploy` that replaces a file the machine already had keeps it once as `<dst>.pre-dotf`, and never overwrites that backup (`TestDeploy_KeepsThePreviousFileOnceBeforeReplacingIt`, `TestDeploy_AFreshDestinationNeedsNoBackup`); `dotf deploy` reports it
- [x] W6: `.zshrc` works on a machine that has not run setup. oh-my-zsh is loaded only when it is installed, with compinit as the fallback; the `*_HOME` toolchain homes are exported and put on PATH only when their directory exists (F-041); brew shellenv is loaded when brew is installed; terraform completion uses the terraform on PATH (F-042); the Mac's hand-made case-insensitive completion is adopted
- [x] W2b (shells): `mise activate` in `.zshrc`, `.bashrc` and the PowerShell profile, guarded on mise being installed, before direnv and zoxide; the profile also initialises zoxide on Windows; parity test `every shell activates mise when it is installed`
- [x] P6 (#1843 B2, zsh and tmux rows): `.zshrc`, `.zsh/*` and `tmux.conf` are `ai/deploy.json` entries with `requires: zsh` / `tmux`, and their `deploy_file` lines leave `setup-linux.sh`; the end-of-setup re-enforcement of `.zshrc` calls `dotf deploy zshrc`. `.bashrc`, `.profile`, `.inputrc`, `.gitconfig` and `ssh/config` wait for #1843 B1's OS selector, which needs a manifest version the installed dotf cannot read
- [x] On the Mac: `dotf deploy` deployed `.zshrc` (keeping the hand-made one as `~/.zshrc.pre-dotf`) and `~/.zsh/*`, skipped tmux (not installed); a second run is `in sync`; a new zsh starts without warnings and resolves age, jq, direnv, zoxide, fzf and go through mise

## P7a — the rc files every OS shares (#2013 P7, no release needed)

- [x] Failing tests first: the `ssh-config` and `ssh-pubkey` entries and their removal from both setup scripts (`tests/setup-linux.bats`, a Windows parity test that replaces the one asserting a log line), `.gitconfig` resolving gh through PATH, and `TestDeploy_ADirectoryCreatedForAPrivateFileIsPrivate`
- [x] `~/.ssh/config` (0600) and `~/.ssh/id_ed25519.pub` are `ai/deploy.json` entries on every OS. They need no `platforms`, so the version 3 manifest the installed `dotf` reads carries them now; both setup scripts lose their copy blocks, and doctor's `homeDeployMap` drops `ssh/config` for `checkDeployManifest`
- [x] A directory `dotf deploy` creates for a file that grants nothing to group or others is 0700: `~/.ssh` landed 0755 on the Mac. An existing directory keeps its mode
- [x] W6 (`.gitconfig`): the github.com and gist credential helpers run `!gh auth git-credential` through PATH. `/usr/bin/gh` is Linux's path; macOS has `/opt/homebrew/bin/gh` and Windows `gh.exe`. On the Mac, `git credential fill` through the repo `.gitconfig` alone returns a password line
- [x] On the Mac: `dotf deploy ssh-config` and `ssh-pubkey` created `~/.ssh` 0700 with the config 0600 and the key 0644, `ssh -G rpi4` resolves the host, and a second run is in sync
- [ ] P7b, after B1 is released and pinned: `.bashrc`, `.profile` and `.inputrc` as entries with `platforms: [linux, darwin]`, the manifest at version 4
- [x] `.gitconfig` is co-owned: every `git config --global` writes it, which is why doctor exempts it from the content check (measured drifting on a converged box 2026-09-02), so a replace entry would fight those tools on every run. The design for its row: the repo file deploys whole to a file dotfiles owns (`~/.config/dotfiles/gitconfig`), and `~/.gitconfig` keeps one `[include] path` to it, added idempotently, so tools keep writing `~/.gitconfig` and neither side overwrites the other Done by #2207 (#2208) with the include target at `~/.config/git/dotfiles.gitconfig`, deployed by the `gitconfig` entry and included through `dotf converge --only git-config`

## W2, W4, W5, W8–W10

Tracked in #2013 track W. Each PR adds its block here when it starts.

## B1 — an OS selector on `ai/deploy.json` entries (#1843 B1, unblocks #2013 P7)

- [x] Failing tests first: `TestParseManifest_ReadsThePreviousVersionToo`, `TestParseManifest_ValidatesPlatforms`, `TestConfig_AppliesOn`, `TestDeployCmd_SkipsAnEntryForAnotherOS`, `TestCheckDeployManifest_AnEntryForAnotherOSIsNotCompared`; the version test moves its bounds to 2 and 5, and the field-set freeze gains version 4
- [x] `platforms` on a deploy entry, with `packages.json`'s meaning through the same `platform` package: absent is every OS, an OS not listed is skipped and said so (`dotf deploy`), and doctor does not compare it
- [x] Validation names the entry: `platforms` in a version 3 manifest, an empty list, and a name that is not a GOOS (`macos` would skip the entry everywhere)
- [x] The reader accepts versions 3 and 4 (`MinManifestVersion`), so it ships before the manifest uses the field. An exact version match would make the reader and the manifest move in one PR, and the installed `dotf` 0.64 would then refuse the manifest on every machine until the pin moved (#1814 class)
- [x] The shipped manifest stays at version 3 and documents the field; the installed 0.64 still reads it (`dotf deploy --dry-run` rc=0 from this checkout)
- [x] Review triage: `dotf deploy <name>` for an entry of another OS prints a `skipped` line and exits 0, not 1.
  A skip is not a failure (#1843 contract), and no script deploys a single entry by name, so a
  non-zero exit would only break `dotf deploy bashrc` typed on Windows without telling anyone more
- [ ] After the release carrying this reader is the `DOTF_VERSION` pin: P7 moves `.bashrc`, `.profile`, `.inputrc` and `ssh/config` (`.gitconfig` left this list with #2207) to entries with `platforms: [linux, darwin]` and the manifest to version 4

## #2162 — the harness refresh reads a vault clone of unknown age

- [x] Failing test first: `TestHarnessRefresh_ABehindVaultIsFastForwardedBeforeTheRefreshReadsIt`.
  The fixture is the measured shape: an upstream at v2, a clone one commit behind at v1, and a
  record that already holds v2. Without the sync the record comes back as v1
- [x] `update.Sync` holds the fast-forward logic `dotf update` had inline, and returns `ahead` as
  its own status. `update.Run` still reports an ahead checkout as a `diverged` skip, because a
  self-update deploys only what the upstream holds
- [x] `dotf harness refresh`: behind is fast-forwarded, level or ahead is refreshed from, and
  diverged, dirty, offline or no upstream keeps the committed records with a warning that names
  the reason. `VAULT_PATH` is pinned on the script, so it reads the clone the check vouched for
- [x] The drift report moves into the command with the same text, and a failed
  `compile-harness.sh` run relays its own output instead of `>/dev/null 2>&1`
- [x] `setup-linux.sh` calls it, and resolves `dotf` once for the refresh and the mirror.
  `harness-refresh-announce.bats` forbids a direct `compile-harness --refresh` call again
- [x] Windows: `setup-windows.ps1` has no refresh (ENGINE-001), so it has no call site to change.
  The command is OS-agnostic, and a Windows refresh would call it
- [x] Review triage: uncommitted work in the refreshed paths stops the refresh. It would have been
  overwritten, and then announced as the vault's change with a commit line that folds it in.
  The ordering guard anchors on the mirror invocation, not on the comment that names it
- [ ] The installed release lacks the command until the next `DOTF_VERSION` pin. Until then setup
  warns that the refresh failed and deploys the committed records, the safe side of #2162.
  #1814 owns this class (setup calling what the pinned `dotf` does not have)

## `dotf vault health` — every obsidian call was an error read as a result

- [x] Measured on obsidian 1.14.4 against the documentation (https://obsidian.md/help/cli): the
  CLI exits 0 and prints its errors on stdout. `--vault <name>` is not a parameter (the CLI
  answered for the active vault, or "Vault not found."), the macOS CLI reads `--no-sandbox` as a command,
  and `dead-ends` is not a command (`deadends` is), so its error line counted as one dead end
- [x] Failing tests first: the `vault-not-found` golden case (exit 2, the CLI's own words),
  `TestNamesVault` and `TestObsidianArgsPerOS`. The stubs answer in the real CLI's format. Each of
  the three fixes is mutation-checked: reverting it fails the suite
- [x] `vault=<name> <sub...>` and `deadends`, with `--no-sandbox` in front everywhere but darwin
  (the `linux-argv` golden case). The connection passes only on the vault's `name<TAB><vault>`
  record
- [x] Review triage: the orphan, dead-end, unresolved and tag sections refuse an answer that is
  not a list, and FAIL instead of counting it. An `Error:` first line was one listed file, and a
  CLI that exited non-zero with no output was an empty list, so a PASS at 0%. Golden cases
  `obsidian-error-answers` and `obsidian-exits-nonzero` (the stub honours `stub/<sub>.exit`),
  both mutation-checked
- [ ] Linux: `dotf vault health` against the live vault on msi, to measure the
  `--no-sandbox vault=<name>` form. Only darwin was measured
- [x] The `obsidian` alias in `.zshrc` and `.bashrc` keeps `--no-sandbox` (an AppImage GUI
  launch flag) everywhere but darwin
- [x] On the Mac, against the live vault: connected, and dead-ends now read 1918/2163 (88%),
  a FAIL kept on purpose (owner, 2026-10-08): the number is real, and it was `1` before

## #2203 — doctor passed a world-readable age root

- [x] Measured on the Mac (2026-10-08): `~/.config/age/key.txt` restored at 0644. `dotf secrets
  verify` FAILED it (`has mode 0644, expected 0600`); doctor's file-authority branch asked only
  whether the file existed and reported `file-authority on disk`
- [x] Failing tests first (`checks_secrets_mode_test.go`, unix only): a 0644 root is one FAIL naming
  the mode, the declared mode and `dotf doctor --fix`, and the mode is not touched without `--fix`;
  `--fix` sets 0600 and a second run is clean
- [x] `secrets.CheckFileAuthorityMode` and `RepairFileAuthorityMode` share `checkKeyMode` with
  `verify`, so doctor and verify cannot disagree again; on Windows both are no-ops (no POSIX bits)
- [x] Review triage: the `[Secrets tooling]` line said `age identity key present` as a PASS next to
  the new FAIL. It now says it checks presence only and points to `[Secrets integrity]` for the mode
- [x] Runbook: the DR restore step was GNU-only (`install -D`); it is now `mkdir -p` plus
  `install -m 600`, which macOS has too, and says doctor catches a wrong mode

## #2183 — the vault secret gate was off while doctor said it was active

- [x] Failing test first: `TestVaultHooks_GateViaDispatcherWithoutPreCommit_Fails`. The fixture
  is the measured shape: dispatcher wired, config present, no pre-commit on PATH. Before the fix
  it reported "gitleaks gate active"
- [x] `checkVaultHooks` FAILs in check mode when pre-commit is absent, before probing the stages.
  The real-git worktree tests pin pre-commit as present, so they stay about hook layout
- [x] Doctor resolves pre-commit the way the dispatcher does: PATH, then uv's tool bin dir
  (`TestVaultHooks_PreCommitInUvBinDirOnly_Passes`). Otherwise the two disagree, and doctor
  FAILs a gate that runs (found by the migration-debt audit)
- [x] The dispatcher fails closed on pre-commit, pre-push and commit-msg when a repo declares a
  config and pre-commit is missing. It stays a no-op on stages that cannot block, and it finds
  pre-commit in uv's tool bin dir when PATH lacks it (GUI launchers)
- [x] `pre-commit` 4.6.2 is a `uv-tool` entry in `packages.json` on every OS. Installed on the Mac
  with `dotf tools install pre-commit`; the second run skips
- [x] Review triage: the dispatcher also finds `pre-commit.exe` in uv's tool bin dir, the name uv installs on Windows and the name doctor's `preCommitPath` checks there. Before, doctor could report a gate that the hook then failed closed on (test `pre-commit.exe in uv's tool bin dir is found, as on Windows`, red before the fix)
- [x] `scripts/install-precommit.sh` deleted. `pre-commit install` refuses under `core.hooksPath`
  (measured), so the script could not work on a provisioned machine, and nothing called it. Its
  config assertions moved to `tests/precommit-config.bats`

## #2207 — git's credential helper needed the shell's PATH

The third defect of one class on the Mac: a tool a non-interactive process runs, resolved through the
interactive shell's PATH (the setup binaries, the vault's pre-commit gate #2184, and now gh as git's
credential helper). obsidian-git asked for a GitHub password; `env -i ... git ls-remote` exited 128 with
`gh: command not found`.

Phase 1 (this PR, no setup change, so the pinned `dotf` keeps working):

- [x] `cli/internal/gitconfig`: `Inspect` and `Apply`, one predicate for converge and doctor (lesson 368).
  `~/.gitconfig` includes `~/.config/git/dotfiles.gitconfig`; GitHub's helper is gh's absolute form
  (`IsAbsoluteGHHelper`: `!<abs path to an existing gh> auth git-credential`, quoted Windows paths
  included). `gh auth setup-git` writes it with its own path on every OS; the include goes in through
  `git config --global --add`
- [x] Failing tests first: the bare helper and the missing include are both reported; apply converges and
  a second run writes nothing; gh absent or logged out blocks only the helper and names the remedy; one
  run against the real git binary under `GIT_CONFIG_GLOBAL`
- [x] `git-config` converge reconciler (every OS), probed by re-inspecting; `dotf converge --only <names>`
  runs a subset in registry order and refuses an unknown name
- [x] Doctor `Git config (global)`: FAIL for what `--fix` can repair, WARN for what needs `gh auth login`;
  `--fix` runs the same `Apply`
- [x] `git/dotfiles.gitconfig` and the `gitconfig` deploy entry. Not `~/.config/git/config`: git reads that
  path itself and writes into it when `~/.gitconfig` is absent, so a deploy over it would erase git's
  writes
- [x] On the Mac: `dotf deploy gitconfig`, then `converge --only git-config` -> 1 changed, then 0 changed;
  doctor `Git config (global)` all ok; `env -i ... git ls-remote` exit 0
- [x] Runbook `docs/runbooks/guide-git-config.md`

Phase 2, moved into this PR. The doctor check made `test-windows` red: CI builds `dotf` from the PR, so
the gate reported the include missing on a box whose setup never added it. That is a real-box state, not
a runner one, so the known-failures list was not its home; the fix is setup converging it:

- [x] Deleted the `.gitconfig` `deploy_file` block in `setup-linux.sh` and its `setup-windows.ps1` twin
  (setup net -18 lines). Each calls `dotf converge --only git-config` once, right after `dotf deploy`
  writes the file the include names. Setup stops overwriting `~/.gitconfig`
- [x] Deleted the repo `.gitconfig`, its `safe_copy`, its `.gitattributes` line, its doctor home-deploy
  exemption and its `isManagedDeployPath` entry; the CI `code` filter lists `git/**` instead
- [x] Tests: setup-linux and setup-windows assert the converge call follows `dotf deploy` and nothing
  copies `~/.gitconfig` (both mutation-checked: removing the call turns them red); `verify-setup.bats`
  asserts the include and that it is effective (`git config --global --includes user.name` equals the
  deployed value; `--global` alone ignores includes). Run end to end in a throwaway HOME with this
  branch's build: deploy, converge 1 changed, `user.name` read through the include, second run 0 changed
- [x] The window: the pinned 0.65.0 rejects `--only` (`unknown flag`, exit 1) and deploys the new
  `gitconfig` entry (`in sync`, exit 0). Until a release moves `DOTF_VERSION`, setup on a fresh box warns
  and leaves `~/.gitconfig` absent; an existing box keeps its file. The warning names the condition.
  Recorded on #1814
- [x] Runner-only remainder: the setup step has no `GH_TOKEN` (CI-004 AC8) and the gate step has, so the
  helper reads as repairable only in the gate. Listed in `doctor-gate-known-failures.txt` against #2212,
  which holds the owner's choice
- [x] Review triage (`fa5f55aa`): `Inspect` reports `TargetMissing` when the deployed file is absent
  (doctor FAIL naming `dotf deploy`, not repairable by `Apply`); the converge detail says "applied" only
  for what a run changed and names what it left. Both mutation-checked

## #2197 — the dead-ends count read genres without links and attachments

`Dead-ends: 1936/2184 (88%)` failed every session start. The owner decided (2026-10-09) which notes
leave the count: `90_archive/`, `00_meta/templates/`, the `sessions/` journals and `20_certifications/`.

- [x] One table, `linkExemptions` in `cli/internal/vault/health.go`, declares the exempt zones of all
  three link-graph counts; orphans, dead-ends and unresolved links read it, and so do the report's
  "Not counted" lines. The vault's `00_meta/_ssot.md` points at it, rather than holding a list Go would
  have to parse
- [x] Attachments are set apart from both counts: the CLI lists them, the markdown population does not
  hold them (469 of 1936 dead-ends, 249 of 408 orphans). Orphans report them on their own line (lesson 375)
- [x] A dead-ends WARN or FAIL names its top three folders, so the session-start line says where the
  unlinked notes are; `--verbose` lists them
- [x] Golden case `deadends-expected-noise`: every exempt zone but `memory/` (GUARD-001 refuses that path
  outside the vault, so `TestLinkExemptPerCheck` pins it), a research note that still counts, an
  attachment in both listings. Six mutations killed (each rule, the attachment filter, the folder line)
- [x] On the Mac, against the live vault: dead-ends 671/1182 (56%), still FAIL; orphans 161/1042 (15%),
  from a 39% WARN. Thresholds kept at 30/50: what remains is products, clients, project research and
  memory, knowledge notes the FAIL is meant to name

## #2013 P2 — darwin read the env-contract as linux, silently

Doctor's `contractOS` mapped darwin to linux and its report named no OS, and `env.defaultFor` held
the same rule a second time. The rule now has one definition, and darwin is a key of its own.

- [x] `env.ForOS` resolves every per-OS contract value (a default, the PATH entries) and `env.AppliesOn`
  every `required_on` scope: darwin reads its own key, else linux's; windows inherits nothing. Doctor's
  `contractOS` and `env.defaultFor` are gone, and `checks_profile.go` reads through the same function
- [x] A declared darwin key wins even when empty, so darwin can opt out of a linux default
- [x] The contract sections name the OS they read: `(contract, darwin; undeclared keys read linux)`
- [x] `env-contract.json` states the rule in `_comment`. No darwin value is added: none differs from linux
  today, and the first is `HERDR_CONFIG_PATH` (#2013 H4)
- [x] A test reads the real contract and fails on a key that is not linux, darwin or windows (a `macos`
  key would be read by no OS) and on a darwin value that repeats the linux one. Mutation-checked, as is
  the fallback (eight tests fail without it)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
