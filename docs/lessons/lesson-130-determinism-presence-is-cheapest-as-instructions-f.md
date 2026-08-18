---
id: lesson-130-determinism-presence-is-cheapest-as-instructions-f
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 130: Determinism "presence" is cheapest as instructions-file injection, not a provider hook

**Context**: ADR-027 cross-harness agent pipeline. The first cut of the curator dogfood (HARNESS-043) emitted a claude-only `SessionStart` hook into ~/.claude/settings.json to force an agent's skills into context.

**Problem**: The hook was claude-only (opencode/pi/copilot have no equivalent) and emitted a POSIX shell command, so it carried a Windows-vs-POSIX command-form axis -- it would not run on Windows without a separate native-command port. A presence mechanism that needs both a per-provider plugin AND a per-OS command form does not generalize; it is the silent-drift failure the pattern exists to prevent.

**Solution**: Every daily harness already loads a harness-managed instructions file (~/.claude/CLAUDE.md, ~/.config/opencode/AGENTS.md, ~/.pi/agent/AGENTS.md, ~/.copilot/copilot-instructions.md). "Presence" therefore equals injecting the forced-skills directive into that file via a marked region -- one uniform mechanism across all four harnesses, in a distinct AGENT-PRESENCE marker namespace that coexists with the patterns region. Text injection is cross-OS by nature, so the OS axis disappears.

**Rule**: For determinism, separate the LEVEL from the MECHANISM. Presence (skill in context every turn) is the cheapest level and needs no plugin -- a system-prompt hook that only ADDS TEXT is equivalent to injecting that text into an always-loaded file. Reserve the provider plugins (SessionStart / chat.system.transform / session_start / PreToolUse) for the Action level, where gating actually requires code. Default to the agnostic injection primitive; reach for a provider hook only when it buys something injection cannot.
