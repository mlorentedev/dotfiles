---
tags: [spec, tasks, refactor-010]
created: "2026-06-05"
---

# Tasks - REFACTOR-010-shell-wrapper-dedup

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `refactor/shell-wrapper-dedup`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left (R1-R4 mitigated; design Option A chosen)

## Implementation

> TDD order: failing guard test first, then move the core, then trim each rc.

- [x] Write failing bats `tests/shell-wrapper-dedup.bats`: assert `_qq_call`/`oc`/`ocfull` defined after sourcing `.zsh/functions.sh` (bash + zsh); assert `_qq_call` appears only in `functions.sh`; assert dbg path not hardcoded. (red: 6/8 fail as expected)
- [x] AC1/AC2: add `_qq_call`, `oc`, `ocfull` to `.zsh/functions.sh` (portable section).
- [x] AC1/AC2: remove `_qq_call` body + `oc`/`ocfull` aliases from `.zsh/aliases.zsh`; keep `qq`/`qf`/`dbg` noglob aliases.
- [x] AC1/AC2: remove `_qq_call` body + `oc`/`ocfull` aliases from `.bashrc`; keep `qq`/`qf`/`dbg` functions.
- [x] AC4: replace `dbg` hardcoded path with PATH-resolved `nan-debug.sh` in both rc files.
- [x] Update existing guards that asserted the old location: `tests/aliases.bats` (5 -> 2 negative guards), `tests/opencode.bats` + `tests/powershell-profile.bats` repointed to functions.sh.
- [x] Run `shellcheck .zsh/functions.sh` + `bash -n`/`zsh -n` on both rc files. (clean)
- [x] Run full bats matrix — green except 3 pre-existing `shell-profile.bats` failures (verified pre-existing on main; out of scope).

## Closing

- [x] Every AC covered by at least one test
- [x] Every AC has a `features.json` entry with a non-vacuous verification command
- [x] Lint passes (shellcheck on functions.sh; `bash -n`/`zsh -n` on rc files)
- [x] No unrelated changes (no sourcing-order edits)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating: only the harness may set `"state": "passing"` after running `verification` with exit 0.
