---
id: lesson-106-an-env-var-seam-is-inert-until-something-sets-it-a
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 106: An env-var seam is inert until something sets it — a hardcoded fallback that matches reality hides the broken seam

**Context**: The vault and dotfiles repo were relocated from `~/Projects/` to `~/Projects/Workspace/`. Vault MCP, hive, selfupdate and the session hooks all broke silently.

**Problem**: The code already had the seams (`VAULT_PATH`, `DOTFILES_REPO_DIR`, `HIVE_VAULT_PATH`) and a Go resolver that honored them — but nothing on the deploy path ever *set* them: the shell profiles hardcoded the value, `VAULT_PATH` was never exported, and the session hooks read a literal `~/Projects/knowledge` instead of the seam. It "worked" only because each hardcoded fallback coincided with the real path. The seam was decorative. The day the assumption (path == default) shifted, every consumer broke at once, and silently — the resolver degraded to "not found" instead of erroring. The same concept even carried three names across the code (`VAULT_PATH` / `VAULT_DIR` / the literal).

**Solution**: ADR-025 — a real cascade (`env → ~/.config/dotfiles/machine.json → env-contract default[OS]`) rendered into `paths.{sh,ps1}` that shells source and that `dotf env generate` writes for every consumer; `dotf doctor` asserts drift. Collapse the three names onto `VAULT_PATH`.

**Rule**: A seam (env var, config key, DI point) is only real if something on the deploy path *sets* it AND every consumer reads it through one resolver. A fallback equal to today's reality is a latent bug, not a safety net — it postpones the failure to the first time the assumption moves and makes it silent. Audit seams by grepping who *sets* them, not just who reads them; collapse synonyms to one name.
