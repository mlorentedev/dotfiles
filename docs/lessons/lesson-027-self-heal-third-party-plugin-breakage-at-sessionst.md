---
id: lesson-027-self-heal-third-party-plugin-breakage-at-sessionst
type: lesson
status: active
created: "2026-05-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 027: Self-heal third-party plugin breakage at SessionStart

**Context**: `thedotmack/claude-mem` shipped v12.7.4 and v13.0.0 to the marketplace with two independent bugs that prevent the `mcp-search` MCP server and worker from starting on a fresh install: (1) `.mcp.json` embeds `${_R%/}` shell parameter expansion which Claude Code's MCP loader misreads as a missing env var (upstream #2385), and (2) v13.0.0's `bun.lock` and shipped `node_modules/` omit the `zod` dep declared in `package.json`, so the worker crashes with `Cannot find module 'zod/v3'`.

**Problem**: A manual edit of the cached `.mcp.json` plus a manual `npm install zod` fixes both, but `/plugin update` (or any reinstall) wipes the workaround. Documenting the manual steps in a vault troubleshooting note relies on me remembering to re-run them, which violates the "automate, don't instruct" standing order. Pinning to v10.6.3 loses upstream fixes and eventually stops working when the marketplace stops serving the old version. Forking the plugin is heavyweight for two trivial patches.

**Solution**: Encoded both fixes into `scripts/claude-mem-heal.sh` (POSIX `sh`, idempotent: `grep -F '${_R%/}'` to detect bug 1, `[ -d node_modules/zod ]` to detect bug 2; silent on healthy installs). Wired into `claude-session-start.sh` so it runs on every session start before vault detection. Heal output (when something was actually fixed) is surfaced via `additionalContext` so the user sees what was repaired. Iterates over all cached versions plus the marketplace fallback copy. Cost on healthy installs: <50ms (just two filesystem checks per cached version).

**Rule**: When a third-party tool ships a broken artifact and the fix is small and idempotent, encode it as a self-heal script wired into the relevant lifecycle hook (SessionStart, shell init, install verification). Three properties are required: idempotent (re-running on healed state is a no-op), silent on success (don't pollute every session with status spam), and surfacing only when it acts (so the user knows the workaround fired). Document the bug, the fix, and the retire-when criteria in `50-troubleshooting/`. Promote to a `00_meta/patterns/` pattern only after the second occurrence — one instance is a workaround, two is a pattern.
