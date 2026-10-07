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

## W2, W4, W5, W8–W10

Tracked in #2013 track W. Each PR adds its block here when it starts.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
