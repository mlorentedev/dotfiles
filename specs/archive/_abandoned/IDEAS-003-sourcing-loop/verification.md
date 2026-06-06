---
tags: [spec, verification, ideas-003]
created: "2026-05-25"
---

# Verification - IDEAS-003-sourcing-loop

> **Status: ABANDONED 2026-06-05.** Not implemented. Closed after a verify-before-act
> review found the spec's premise invalid against the current `.zshrc` / `.bashrc`.
> Superseded by REFACTOR-010-shell-wrapper-dedup (the real duplication).

## Why abandoned (the finding)

The spec assumed parallel contiguous `source` blocks in `.zshrc` and `.bashrc` over a
shared file set, collapsible into one brace-expanded loop. Ground truth at HEAD `eba4831`:

| Spec assumption | Verified reality |
|---|---|
| Contiguous `source` block in both rc files | `.zshrc`: scattered across 2 semantic sections (`aliases.zsh` at L86 in ALIASES; `functions.zsh`/`functions.sh`/`nvm.zsh` at L115-118 in SHELL ENHANCEMENTS). `.bashrc`: a single `.zsh/` source line (`functions.sh`, L157). |
| bash and zsh source the same set (duplication to dedup) | They diverge deliberately: zsh loads `aliases.zsh`+`functions.zsh`+`functions.sh`+`nvm.zsh`; bash uses `~/.bash_aliases` (native) + inline wrappers + only the shared `functions.sh`. |
| Missing-file tolerance is a new benefit | Already present — every source line is guarded by its own `[[ -f ]]`. |
| 6 files (path/exports/prompt/completions/...) | Do not exist; the real set is 4 (zsh) / 1 (bash). |

Implementing as specced would force the scattered zsh sourcing into one location,
reordering `aliases.zsh` relative to the inline AI aliases (`g`/`c`/`obsidian`, L91-93) —
a behavioral change the proposal explicitly forbids ("preserving exact current ordering").
In `.bashrc` a "loop" over one element is strictly worse than the current line.

Net: reorder risk for zero/negative benefit. Decision: abandon, mirroring the POLISH-001
WONTFIX precedent and the recorded lesson "verify-before-act on agent audits".

## What replaces it

The genuine bash<->zsh duplication is the opencode quick-question wrappers:
`_qq_call()` (identical body in `.zsh/aliases.zsh` and `.bashrc`), plus the `oc`/`ocfull`
aliases. Extracting the portable core into the already-shared `.zsh/functions.sh` removes
that duplication without touching sourcing order. Tracked as **REFACTOR-010-shell-wrapper-dedup**.

## Promotion candidates

- [x] Lesson? Covered by the existing "verify-before-act on agent audits" lesson — no new entry needed.
- [x] ADR-worthy? No — a scoping decision, not architecture.
- [x] New pattern? No.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: abandoned`
- [x] Folder moved: `specs/IDEAS-003-sourcing-loop/` -> `specs/archive/_abandoned/IDEAS-003-sourcing-loop/`
- [x] Backlog entries in vault `11-tasks.md` updated (IDEAS-003 marked abandoned/superseded)
- [x] GH #137 closed with rationale
