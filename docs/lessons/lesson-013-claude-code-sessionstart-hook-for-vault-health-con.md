---
id: lesson-013-claude-code-sessionstart-hook-for-vault-health-con
type: lesson
status: active
created: "2026-02-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 013: Claude Code SessionStart hook for vault health context

**Context**: `vault-health.sh` was created to report Obsidian vault health (orphans, unresolved links, frontmatter coverage) but required manual invocation.

**Problem**: Claude had no automatic awareness of vault health state at session start. Users had to remember to run the script and paste results.

**Solution**: Created `claude-session-start.sh` as a Claude Code `SessionStart` hook. The hook detects if CWD is inside an Obsidian vault (walks up directories looking for `.obsidian/`), runs `vault-health.sh` if found, and returns health summary as `additionalContext` via the hook JSON output format. Registered in `~/.claude/settings.json` under `hooks.SessionStart`.

**Rule**: Claude Code hooks live in `~/.claude/settings.json` (global scope). Scripts they invoke live in dotfiles (`~/.dotfiles/scripts/`). On new machines: (1) deploy dotfiles (gets the script), (2) add the hook entry to `~/.claude/settings.json`. The hook must tolerate Obsidian GUI being down (exit code 2 from vault-health.sh) and non-vault directories (exit 0 silently).
