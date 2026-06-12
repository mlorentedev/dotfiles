---
id: "DX-002-dot-umbrella-command"
type: spec
status: abandoned # superseded by ADR-020 (Go CLI convergence); subcommand map + GraphViz risk harvested into specs/CLI-001-dot-scaffold/
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# DX-002-dot-umbrella-command

> **Naming**: file lives at `<repo>/specs/DX-002-dot-umbrella-command/proposal.md`. `DX-002-dot-umbrella-command` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Single `dot` command (holman bin/dot pattern) wrapping `hc`/`dch`/`dotfiles-sync` under one verb. Cross-OS via `dot.sh` + `dot.ps1`. Effort: M, ~100 LOC. Anti-scope: don't deprecate existing aliases; thin dispatch layer only. -->

holman's `bin/dot` provides one verb for "refresh my environment". This repo has three separate verbs: `hc` (healthcheck), `dch` (drift-check), `dotfiles-sync` (bidirectional sync), plus `vault` and `profile-shell`. New contributors must learn five names. A single `dot` umbrella with subcommands matches the de-facto dotfiles convention (most popular dotfiles ship a `dot` or `dotfiles` entry point) and reduces mental load without removing existing aliases.

## What

New `bin/dot.sh` (Linux) + `bin/dot.ps1` (Windows) entrypoint. Subcommands dispatch to existing scripts:

- `dot doctor` → `healthcheck.{sh,ps1}`
- `dot drift` → `diff-check.{sh,ps1}`
- `dot sync [--secrets-only]` → `dotfiles-sync.{sh,ps1}`
- `dot vault <args>` → `vault.sh` (Linux only)
- `dot profile` → `shell-profile.sh`
- `dot help` → list subcommands + 1-line descriptions
- `dot version` → print `versions.conf` values

Each subcommand is a thin dispatcher (≤ 5 lines). Total ≈ 100 LOC per side. Existing aliases (`hc`, `dch`, `dotfiles-sync`) keep working.

## Out of scope

- **Removing existing aliases** — they stay as power-user shortcuts.
- **Adding new functionality** not already in the wrapped scripts.
- **Renaming the underlying scripts** — they're sourced from many places.
- **Fuzzy subcommand match** (`dot doctr` → `doctor`) — strict exit 1 + help for unknown subcommands; fuzzy is feature creep.

## Risks / open questions

- **R1**: `dot` may collide with GraphViz `dot` on PATH. Detect at install time; warn and offer to disable via `.zshrc.local`. If GraphViz is present, default to alias `dotfiles` and document.
- **R2**: PATH integration — `bin/dot.{sh,ps1}` must be added to PATH by `setup-linux.sh` / `setup-windows.ps1`. Verify drift detector accepts the new PATH entry.
- **R3**: Subcommand discoverability. `dot help` must list ALL subcommands; `dot` with no args should print the help (not error). Aligns with `gh` / `git` conventions.
- **R4**: Windows `.ps1` invocation friction — users may need to call `dot.ps1` or `pwsh -File dot.ps1`. Wrap with a `dot.bat` shim or ensure PSReadLine alias.

## Acceptance criteria

- [ ] `bin/dot.sh` exists, executable, added to PATH on Linux.
- [ ] `bin/dot.ps1` exists, added to PATH on Windows; `.bat` shim or pwsh alias resolves bare `dot`.
- [ ] `dot doctor`, `dot drift`, `dot sync`, `dot vault`, `dot profile`, `dot help`, `dot version` all work end-to-end.
- [ ] `dot` with no args prints help; `dot bogus` exits 1 with usage hint.
- [ ] Existing aliases (`hc`, `dch`, `dotfiles-sync`) still work — no regression.
- [ ] `tests/dot.bats` covers subcommand dispatch + unknown-subcommand error path.
- [ ] README "Key Commands" section documents `dot` as the recommended entrypoint.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → DX-002.
- Inspiration: holman/dotfiles `bin/dot` (Zach Holman, 2012).
- Existing aliases live in: `.zsh/aliases.zsh`, `powershell/profile.ps1`.
