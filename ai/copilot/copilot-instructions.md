# Copilot Custom Instructions

> **First, read `AGENTS.md` at the repo root** — canonical SSOT for behaviour rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive Loop, MCP usage, Operational Rules). This file contains only Copilot-specific extensions on top.
>
> If `AGENTS.md` is missing from the current repo, default to the canonical version at `$DOTFILES_REPO_DIR/AGENTS.md` (resolved via `machine.json` per ADR-025; falls back to `~/Projects/Workspace/dotfiles/AGENTS.md`).

## Role & Execution Preferences

* **ROLE:** Expert Shell Engineer, DevOps & AI Architect.
* **Command First:** Provide concise POSIX shell commands (`bash` / `zsh`). Prefer `rg`, `fd`, `eza`, `bat`.
* **Safe:** Always warn before destructive commands (`rm`, `dd`, `git reset --hard`).
* **Dynamic Documentation:** Create or update runbooks in `docs/runbooks/` (per knowledge-placement model).

## Quick Reference (paths)

* **Vault root:** `$VAULT_PATH` (resolved via `machine.json` per ADR-025).
* **Project context:** `10_projects/<repo>/context.md` | **Active backlog:** bitácora GitHub Project.
* **Patterns & Templates:** `00_meta/patterns/*.md` / `00_meta/templates/*.md`.

## Model Tier (per AGENTS.md "Model Selection")

- **Top:** `gpt-5.6-terra` / `claude-opus-5` (deep reasoning / architecture) | **Mid:** `gpt-5.6-sol` / `claude-sonnet-5` (default / implementation / tests) | **Low:** `gpt-5.6-luna` / `claude-haiku-4.5` (syntax / quick transforms). Ids as listed by `copilot help config` on the seat; the cross-agent routing SSOT is `harness/model-map.json`.

## Skills

The cross-agent skill pipeline deploys compatible records as native personal Agent Skills under `~/.copilot/skills/`, including each complete `SKILL.md` and its resources. The generated catalog below remains an always-loaded index; both surfaces honor each skill's `targets[]`. Edit the skill in the vault and re-run setup — do NOT edit between the markers.

<!-- BEGIN HARNESS GENERATED -->
<!-- END HARNESS GENERATED -->

