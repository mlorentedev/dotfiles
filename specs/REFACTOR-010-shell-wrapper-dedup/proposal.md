---
id: "REFACTOR-010-shell-wrapper-dedup"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-05"
tags: [spec, proposal, refactor, shell, dedup]
template_version: "1.0"
---

# REFACTOR-010: Shell wrapper dedup

> **Naming**: file lives at `<repo>/specs/REFACTOR-010-shell-wrapper-dedup/proposal.md`.
> Origin: IDEAS-003 verify-before-act pivot (2026-06-05). IDEAS-003 chased a non-existent
> duplication (parallel `source` blocks); this targets the real one.

## Why

The opencode quick-question wrappers are genuinely duplicated between `.zsh/aliases.zsh`
(zsh) and the inline block in `.bashrc` (bash): the `_qq_call()` function body is
byte-identical in both, and the `oc` / `ocfull` aliases are identical. Adding or changing a
wrapper means editing both files in lockstep — the maintenance hazard IDEAS-003 set out to
fix but mislocated. The portable core can live once in `.zsh/functions.sh`, which is already
sourced by both shells.

## What

After this PR:

- `.zsh/functions.sh` (already sourced by both `.zshrc` and `.bashrc`) defines the portable
  core: `_qq_call()`, plus `oc` / `ocfull` as portable **functions** (not aliases — keeps the
  file's "functions only" contract).
- `.zsh/aliases.zsh` no longer defines `_qq_call` or the `oc` / `ocfull` aliases. It keeps
  only the zsh-specific `qq` / `qf` / `dbg` `noglob` aliases (which already call `_qq_call` /
  `nan-debug.sh`).
- `.bashrc` no longer defines `_qq_call` or the `oc` / `ocfull` aliases. It keeps only the
  bash-specific `qq` / `qf` / `dbg` function wrappers.
- `dbg`'s hardcoded absolute path (`/home/manu/Projects/dotfiles/scripts/nan-debug.sh`)
  becomes the PATH-resolved `nan-debug.sh` (the `scripts/` dir is already on PATH).
- Net: `_qq_call` is defined exactly once across the whole config; the only per-shell code
  left is the irreducible `noglob`-vs-function wrapper.

## Out of scope

- The sourcing-loop refactor (IDEAS-003 — abandoned).
- Touching sourcing order or the `aliases.zsh` / `functions.zsh` / `nvm.zsh` source lines.
- Collapsing the `noglob`-vs-function divergence — it is irreducible (`noglob` is zsh-only);
  each shell keeps its idiomatic thin wrapper (design Option A, chosen over a shell-detection
  branch in the portable file).
- zsh-only wrappers not duplicated in bash (`oclog`, `cop`, `cops`) — left untouched.
- PowerShell `profile.ps1` wrappers (separate effort if ever needed).

## Risks / open questions

- **R1 — lazy-eval ordering.** zsh `qq`/`qf` aliases (in `aliases.zsh`, sourced first) and
  bash `qq`/`qf` functions (defined before `functions.sh` is sourced) both reference
  `_qq_call`, which now lives in `functions.sh` (sourced later). Alias/function bodies are
  evaluated at *call* time, not definition time, so the later definition is fine. Verified by
  the parity test sourcing `functions.sh` and asserting the core resolves.
- **R2 — functions.sh contract.** Its header declares "portable functions" (no aliases).
  `oc`/`ocfull` are added as functions, not aliases, preserving that contract.
- **R3 — `noglob` is zsh-only.** Confirmed; the zsh wrappers must stay `noglob` aliases and
  cannot move to the shared file. This is the whole reason only the core moves.
- **R4 — regressions.** Existing bats matrix (bash + zsh) must stay green; shellcheck clean on
  `.zsh/functions.sh`.

## Acceptance criteria

- [ ] AC1: `_qq_call` is defined exactly once in the repo — in `.zsh/functions.sh`; removed from `.zsh/aliases.zsh` and `.bashrc`.
- [ ] AC2: `oc` and `ocfull` are defined as portable functions in `.zsh/functions.sh`; the duplicate aliases are removed from `.zsh/aliases.zsh` and `.bashrc`.
- [ ] AC3: zsh keeps `qq`/`qf`/`dbg` as `noglob` aliases in `.zsh/aliases.zsh`; bash keeps `qq`/`qf`/`dbg` as functions in `.bashrc`; both call the shared core.
- [ ] AC4: `dbg` no longer hardcodes the absolute path — it invokes PATH-resolved `nan-debug.sh`.
- [ ] AC5: bats parity test: after sourcing `.zsh/functions.sh`, `_qq_call`, `oc`, `ocfull` are defined in both bash and zsh.
- [ ] AC6: shellcheck clean on `.zsh/functions.sh`; full bats matrix green (no regressions).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (REFACTOR-010 entry)
- Supersedes: IDEAS-003-sourcing-loop (`specs/archive/_abandoned/IDEAS-003-sourcing-loop/`), GH #137
- Related: IDEAS-002 (created `.zsh/functions.sh`)
- Project rule: `.claude/CLAUDE.md` Prohibited Patterns (bash+zsh portability; `.` over `source`)
