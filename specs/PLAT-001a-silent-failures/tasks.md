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

## W2, W4–W10

Tracked in #2013 track W. Each PR adds its block here when it starts.

## B1 — an OS selector on `ai/deploy.json` entries (#1843 B1, unblocks #2013 P7)

- [x] Failing tests first: `TestParseManifest_ReadsThePreviousVersionToo`, `TestParseManifest_ValidatesPlatforms`, `TestConfig_AppliesOn`, `TestDeployCmd_SkipsAnEntryForAnotherOS`, `TestCheckDeployManifest_AnEntryForAnotherOSIsNotCompared`; the version test moves its bounds to 2 and 5, and the field-set freeze gains version 4
- [x] `platforms` on a deploy entry, with `packages.json`'s meaning through the same `platform` package: absent is every OS, an OS not listed is skipped and said so (`dotf deploy`), and doctor does not compare it
- [x] Validation names the entry: `platforms` in a version 3 manifest, an empty list, and a name that is not a GOOS (`macos` would skip the entry everywhere)
- [x] The reader accepts versions 3 and 4 (`MinManifestVersion`), so it ships before the manifest uses the field. An exact version match would make the reader and the manifest move in one PR, and the installed `dotf` 0.64 would then refuse the manifest on every machine until the pin moved (#1814 class)
- [x] The shipped manifest stays at version 3 and documents the field; the installed 0.64 still reads it (`dotf deploy --dry-run` rc=0 from this checkout)
- [ ] After the release carrying this reader is the `DOTF_VERSION` pin: P7 moves `.bashrc`, `.profile`, `.inputrc`, `.gitconfig` and `ssh/config` to entries with `platforms: [linux, darwin]` and the manifest to version 4

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
