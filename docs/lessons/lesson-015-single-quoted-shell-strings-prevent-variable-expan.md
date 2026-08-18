---
id: lesson-015-single-quoted-shell-strings-prevent-variable-expan
type: lesson
status: active
created: "2026-03-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 015: Single-quoted shell strings prevent variable expansion in JSON

**Context**: `setup-linux.sh` built a JSON hook entry with `HOOK_ENTRY='{"command":"$HOME/.dotfiles/scripts/..."}'`. The literal `$HOME` was written into `settings.json`.

**Problem**: Claude Code reads the hook path as-is. The literal string `$HOME/...` is not a valid path, so the SessionStart hook silently failed on every fresh install since it was added.

**Solution**: Replaced string concatenation with `jq -n --arg cmd "$HOME/.dotfiles/scripts/claude-session-start.sh" '{"command":$cmd}'`. Shell expands `$HOME` in the argument, `jq` handles JSON escaping safely.

**Rule**: Never embed shell variables inside single-quoted JSON strings. Use `jq -n --arg` to build JSON with dynamic values — it handles both variable expansion and proper JSON escaping.
