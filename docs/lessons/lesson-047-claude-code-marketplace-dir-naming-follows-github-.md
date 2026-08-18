---
id: lesson-047-claude-code-marketplace-dir-naming-follows-github-
type: lesson
status: active
created: "2026-05-20"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 047: Claude Code marketplace dir naming follows GitHub repo, NOT declared name field

**Context:** BUG-012 (PR #70, 2026-05-20). Diagnosing `UserPromptSubmit operation blocked by hook` on Windows. Hook command was claude-mem plugin's `bun-runner.js` discovery script. It searched at `~/.claude/plugins/marketplaces/thedotmack/plugin/scripts/` (the marketplace's declared `name` from `marketplace.json`), but Claude Code had cloned the marketplace under `~/.claude/plugins/marketplaces/thedotmack-claude-mem/` (GitHub repo name `thedotmack/claude-mem` flattened with `-`). When `CLAUDE_PLUGIN_ROOT` is unset/stale (Windows: cache `plugins/cache/thedotmack/claude-mem/` stays empty post-install), the hook falls through to the broken fallback → `exit 1` → blocked.</context>
<parameter name="problem">A plugin can declare `name: "thedotmack"` in `marketplace.json` but Claude Code names the install dir after the GitHub repo (`<owner>-<repo>` in some cases, just `<repo>` in others — naming logic not fully documented). Plugins that hardcode `marketplaces/<declared-name>/plugin/scripts/...` in fallback paths break silently on machines where the env-var path fails and the fallback is consulted. Symptom on Windows is `printf: write error: Permission denied` (Git Bash + Claude Code hook subprocess sandbox quirk); on Linux it's a cleaner `claude-mem: plugin scripts not found`. The cross-OS root cause is identical.
**Problem:** 
**Solution:** Defense in depth: create a junction (Windows, no admin) / symbolic link (Linux) from the declared name to the actual install dir during heal. Guard creation on `source exists AND target absent` for idempotence. Once the link exists, ALL code that hardcodes the declared name (plugin's bundled hooks.json, .mcp.json, etc.) resolves correctly without modifying upstream files. The link survives `/plugin update` operations. Pattern lives in `scripts/claude-mem-heal.{sh,ps1}` as `ensure_marketplace_compat_symlink` / `Repair-MarketplaceCompatJunction`.
**Tags:** `#claude-code` `#plugin-discovery` `#claude-mem` `#cross-os-parity` `#junction-pattern`
