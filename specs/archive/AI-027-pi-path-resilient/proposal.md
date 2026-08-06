---
id: "AI-027-pi-path-resilient"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-18"
issue: "mlorentedev/dotfiles#426"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-027-pi-path-resilient

> **Naming**: file lives at `<repo>/specs/AI-027-pi-path-resilient/proposal.md`. `AI-027-pi-path-resilient` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #426: AI-027: provision Orca (parallel-agent ADE) across all machines + doctor verification -->

`pi` launches from the default terminal but not from inside Orca (the parallel-agent ADE). Root cause: `setup-linux.sh` installs `pi` as an npm-global, which lands in nvm's **per-node-version** tree (`~/.nvm/versions/node/<v>/bin/pi`). The terminal resolves node via `nvm use default` (LTS, where pi lives); Orca propagates a PATH with a *different* node version (and never loads the interactive shell), so `pi` is absent there. The install location is coupled to whichever node version was active at setup time — fragile across node upgrades, across machines, and across any environment (GUI/ADE) that doesn't load the login shell.

## What

`setup-linux.sh` installs `pi` into the manager-independent `~/.local` prefix, so the launcher lands at `~/.local/bin/pi` — the same stable, always-on-PATH directory that `claude` and `dotf` already use, inherited by login shells AND GUI/ADE processes (Orca included). `dotf doctor` gains an incident→guard branch: when pi is **configured** (`~/.pi/` present) but unreachable on PATH, it FAILs with the root cause instead of the misleading "pi not installed" SKIP.

## Out of scope

- `setup-windows.ps1` parity — Windows npm-global goes to `%APPDATA%\npm` (usually already on PATH); whether the same trap exists is Windows-empirical. Tracked under AI-025 (#297).
- Removing the stale nvm-version `pi` (harmless duplicate; shadowed by `~/.local/bin` which precedes it on PATH).
- Generalizing the `--prefix ~/.local` install strategy to other npm-global agent CLIs (opencode already ships its own launcher).

## Risks / open questions

- `~/.local/bin/pi`'s `#!/usr/bin/env node` shebang runs under whatever node is first on PATH (v26 in Orca). Assumes pi 0.79.1 runs under node ≥ the Orca-provided version — true for current node releases; verified at smoke time.
- `npm install -g --prefix "$HOME/.local"` writes into `~/.local/lib/node_modules`; coexists with other tooling without collision (standard user-prefix pattern).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] After `setup-linux.sh`, `pi` resolves to `~/.local/bin/pi` (not the nvm per-version path).
- [ ] `pi` is launchable from an environment whose active node differs from the setup-time node (the Orca case).
- [ ] `dotf doctor` reports FAIL (not SKIP) when `~/.pi/` exists but `pi` is not on PATH, naming the root cause.
- [ ] `setup-linux.sh` passes shellcheck; the doctor package builds and its table tests pass.

## References

- Issue: `mlorentedev/dotfiles#426` (AI-027)
- Existing pattern mirrored: `cli/internal/doctor/checks_deploy.go` opencode "exists but not in PATH" branch
- Sibling deferral: AI-025 (#297) Windows pi verification
