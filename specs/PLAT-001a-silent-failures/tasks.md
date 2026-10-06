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

## W2–W10

Tracked in #2013 track W. Each PR adds its block here when it starts.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Lint passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
