---
id: agents-spec-section
type: template
status: active
created: "2026-05-13"
---

# AGENTS.md Spec-Driven Development Section

> Snippet to include in any repo's `AGENTS.md` so agents (Copilot, Claude, Cursor, Codex) discover the spec workflow defined in the vault.

## Usage

**Recommended (automated):** run `init-repo-agents.sh` (Linux/macOS) or `init-repo-agents.ps1` (Windows) from anywhere inside the target repo. The script:
- creates `AGENTS.md` with a minimal header if missing, or
- appends the snippet to existing `AGENTS.md` (preserving prior content), or
- skips silently if the `## Spec-Driven Development` section is already present (idempotent).

Pass `--force` (`-Force` on Windows) to overwrite an existing SDD section in place.

**Manual fallback:** copy the section between BEGIN/END SNIPPET markers below into `AGENTS.md`. Placement: under a `## Workflow` heading or as its own section.

No substitutions needed — `$VAULT_PATH` resolves at agent runtime via env var.

## Bootstrap automation

`init-repo-agents.sh` / `.ps1` (delivered via dotfiles, on PATH after install) — implements the recommended flow above. SDD-013 closed 2026-05-14.

---

## --- BEGIN SNIPPET ---

## Spec-Driven Development

This repo follows the **Spec-Driven Development per feature** pattern. Canonical workflow definition at `$VAULT_PATH/00_meta/skills/spec/SKILL.md` (where `$VAULT_PATH` = `$HOME/Projects/knowledge` on Linux/macOS, `%USERPROFILE%\Projects\knowledge` on Windows).

When the user asks to **create, fill, or archive a spec**, read the canonical SKILL.md and follow it. Three subcommands:

| Trigger phrase | Subcommand |
|---|---|
| "create a spec for X", "scaffold spec X", "start working on X" | `init <feature-id>` |
| "fill in proposal for X", "help me write the proposal" | `fill <feature-id>` |
| "archive spec X", "close spec X" | `archive <feature-id>` |

Per-feature specs live at `specs/<feature-id>/` in this repo; archived at `specs/archive/<feature-id>/` (never deleted — audit trail).

**Skip SDD for**: typo fixes, comment-only edits, mechanical refactors, bug fixes <20 lines with obvious cause, doc-only changes.

**Pattern reference**: `$VAULT_PATH/00_meta/patterns/pattern-spec-driven-development.md`.

**Shell fallback for non-interactive use** (CI, batch): `init-spec` / `archive-spec` (POSIX) or `init-spec.ps1` / `archive-spec.ps1` (Windows), available on PATH via dotfiles install.

`<feature-id>` format: `^[A-Z]+-\d+(-[a-z0-9-]+)?$` (e.g., `AI-001-ollama-public`) or `^\d{4}-\d{2}-\d{2}-[a-z0-9-]+$` (e.g., `2026-05-13-cleanup`).

## --- END SNIPPET ---
