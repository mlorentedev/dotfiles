---
id: lesson-018-uninitialized-variable-under-set-u-in-conditional-
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 018: Uninitialized variable under set -u in conditional-only assignment

**Context**: `claude-session-start.sh` used `VAULT_NAME` which was only assigned inside a vault-detection `if` block.

**Problem**: Under `set -euo pipefail`, referencing `VAULT_NAME` after the conditional would abort the script with "unbound variable" when no vault was detected — the variable was never defined in that code path.

**Solution**: Initialize `VAULT_NAME=""` at the top of the script, before any conditionals. The variable is always bound regardless of which branch executes.

**Rule**: When using `set -u`, any variable assigned inside a conditional block must also be initialized at a wider scope (script top or function top). If only one branch of an `if/else` assigns the variable, the other branch leaves it unbound.
