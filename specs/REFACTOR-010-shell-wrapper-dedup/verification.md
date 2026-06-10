---
tags: [spec, verification, refactor-010]
created: "2026-06-05"
---

# Verification - REFACTOR-010-shell-wrapper-dedup

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof.

- [x] AC1 (`_qq_call` defined once, in functions.sh) -> `tests/shell-wrapper-dedup.bats` "AC1: _qq_call is defined only in .zsh/functions.sh"; negative guard in `tests/aliases.bats` "aliases.zsh no longer defines _qq_call body".
- [x] AC2 (`oc`/`ocfull` functions in functions.sh; aliases removed) -> `tests/shell-wrapper-dedup.bats` "AC2: oc and ocfull are functions in functions.sh" + "no duplicate oc/ocfull aliases remain"; `tests/aliases.bats` "aliases.zsh no longer defines oc/ocfull".
- [x] AC3 (zsh keeps noglob aliases, bash keeps functions) -> `tests/shell-wrapper-dedup.bats` AC3 (both); `tests/aliases.bats` qq/qf alias tests still green.
- [x] AC4 (dbg uses PATH-resolved nan-debug.sh) -> `tests/shell-wrapper-dedup.bats` "AC4: dbg does not hardcode an absolute nan-debug.sh path".
- [x] AC5 (core resolves after sourcing functions.sh, bash + zsh) -> `tests/shell-wrapper-dedup.bats` AC5 (bash) + AC5 (zsh).
- [x] AC6 (shellcheck clean; full matrix green) -> `shellcheck .zsh/functions.sh` clean; full bats matrix green except 3 pre-existing failures (below).

## Test status

- Targeted: `bats tests/shell-wrapper-dedup.bats` -> 8/8 pass (incl. zsh source check). Affected files re-run (`aliases.bats opencode.bats powershell-profile.bats shell-wrapper-dedup.bats`) -> 99/99 pass.
- Lint: `shellcheck .zsh/functions.sh` clean; `bash -n` on `.bashrc`/`.zsh/functions.sh` and `zsh -n` on `.zshrc`/`.zsh/aliases.zsh` all pass.
- Full matrix: `bats tests/*.bats` -> all green EXCEPT 3 pre-existing failures in `tests/shell-profile.bats` ("rejects shell not in PATH", "time-only mode reports min/median/mean/max", "runs the requested number of iterations"). Verified these fail identically on the unmodified `main` checkout -> pre-existing, unrelated to this change, out of scope (candidate for a separate ticket).
- No regressions introduced by this change: confirmed (the only structural failures from the move — 7 tests asserting the old location — were updated to the new contract in this PR).

## Decisions made during implementation

- Design Option A (chosen over B): only the genuinely-portable core (`_qq_call`, `oc`, `ocfull`) moves to `.zsh/functions.sh`; each rc keeps its irreducible shell-specific `qq`/`qf`/`dbg` wrapper. `noglob` is a zsh-only precommand modifier and cannot live in the shared "portable" file, so a shell-detection branch there (Option B) was rejected as violating the file's contract.
- `oc`/`ocfull` added as functions (not aliases) to preserve functions.sh's "functions only" contract.
- Lazy-eval ordering verified safe: `qq`/`qf` (in aliases.zsh / .bashrc, defined before functions.sh is sourced) reference `_qq_call` whose body is evaluated only at call time, by which point functions.sh has loaded.
- Cross-OS asymmetry preserved: Linux `oc` = `opencode --pure` (DX-003 hang workaround, abandoned), Windows `oc` = plain `opencode`. The parity test was repointed to functions.sh, not broken.
- Test maintenance: 7 existing tests asserting the old location were updated (5 in aliases.bats consolidated into 2 negative guards; 1 in opencode.bats and 1 in powershell-profile.bats repointed to functions.sh). Positive coverage now lives in `tests/shell-wrapper-dedup.bats`.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? <pending user decision at archive>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <pending user decision at archive>
- [ ] New pattern candidate for `00_meta/patterns/`? <pending user decision at archive>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/REFACTOR-010-shell-wrapper-dedup/` -> `specs/archive/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
