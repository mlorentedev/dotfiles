---
id: "dotfiles-troubleshoot-claude-mem-broken-marketplace"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, claude-code, claude-mem, plugin, mcp]
created: "2026-05-08"
owner: manu
---

# Troubleshooting: claude-mem v12.7.4 / v13.0.0 broken marketplace artifacts

The `thedotmack/claude-mem` plugin has shipped two known-broken releases that prevent the `mcp-search` MCP server and worker from starting on a fresh install. Both issues are upstream packaging defects; we self-heal at every session start via `claude-mem-heal.sh`.

## Symptoms

### Symptom A — `.mcp.json` rejected by Claude Code

`/doctor` (or the session banner) reports:

```
Invalid MCP server config for "mcp-search": Missing environment variables: _R%/
  → Check MCP server configuration in .mcp.json or manifest
```

`mcp-search` is absent from the running MCP servers, so any tool from the plugin (`mem-search`, observation lookup, etc.) is unavailable.

### Symptom B — worker crashes with `Cannot find module 'zod/v3'`

After symptom A is fixed, the first call into the plugin returns:

```
Failed with non-blocking status code: error: Cannot find module 'zod/v3'
from '/home/manu/.claude/plugins/cache/thedotmack/claude-mem/13.0.0/scripts/worker-service.cjs'
```

## Root cause

**A — `${_R%/}` shell expansion misread as env-var.** Starting in v12.7.4, `.mcp.json` inlines a `sh -c` script that uses POSIX parameter expansion to trim a trailing slash:

```sh
_R="${_R%/}"
```

Claude Code's MCP-config loader scans every string in `.mcp.json` for `${...}` substitutions. It handles the `${VAR:-default}` form, but it does not recognise the `%` suffix-removal operator and treats `_R%/` as a literal env-var name. That env var is unset, so loading the MCP server is aborted. Tracked upstream: [thedotmack/claude-mem#2385](https://github.com/thedotmack/claude-mem/issues/2385).

**B — `zod` runtime dep declared but not shipped.** v13.0.0's `package.json` declares `"zod": "^4.3.6"` but the published `bun.lock` (50 lines, tree-sitter only) and `node_modules/` omit it entirely. `scripts/worker-service.cjs` does runtime `require("zod")`, `require("zod/v3")`, `require("zod/v4")` and `require("zod/v4-mini")`, all of which fail. Tracked upstream: see [open Issue B](https://github.com/thedotmack/claude-mem/issues) (filed by us alongside #2385).

## Automatic fix

`scripts/claude-mem-heal.sh` is invoked from `claude-session-start.sh` and silently repairs both issues at every session start. Idempotent — only acts when the cache is broken, exits 0 with no output otherwise.

For the `${_R%/}` regression it overwrites the cached `.mcp.json` with the simpler v10.6.3 form:

```json
{
  "mcpServers": {
    "mcp-search": {
      "type": "stdio",
      "command": "node",
      "args": ["${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs"]
    }
  }
}
```

For the missing zod dep it runs:

```sh
cd ~/.claude/plugins/cache/thedotmack/claude-mem/<version>
npm install --no-save --no-package-lock --no-audit --no-fund --silent zod@^4.3.6
```

When a heal action fires, the SessionStart hook surfaces it in `additionalContext` as `[claude-mem] self-healed plugin install: …` so it's visible in the session banner.

## Manual repro / diagnostic

```bash
# Confirm symptom A is present in a cached version
grep -F '${_R%/}' ~/.claude/plugins/cache/thedotmack/claude-mem/*/.mcp.json

# Confirm symptom B
ls ~/.claude/plugins/cache/thedotmack/claude-mem/13.0.0/node_modules/zod 2>&1

# Force-run the heal manually
~/.dotfiles/scripts/claude-mem-heal.sh --verbose
```

## When to retire this note

Both upstream issues need to close. Once that's confirmed:

1. Verify a clean install of the next claude-mem release does not reintroduce either symptom (grep `${_R%/}`, list `node_modules/zod`).
2. Remove the `claude-mem-heal.sh` invocation from `claude-session-start.sh`.
3. Optionally keep the script around for older cached versions; otherwise delete it.
4. Set this note's frontmatter `status: archived` and link the upstream resolutions.

## Related

- `~/Projects/dotfiles/scripts/claude-mem-heal.sh` — the heal script (canonical)
- `~/Projects/dotfiles/scripts/claude-session-start.sh` — invokes heal at session start
- [Troubleshooting: AI Tools](ai-tools.md)
- Upstream issue: [thedotmack/claude-mem#2385](https://github.com/thedotmack/claude-mem/issues/2385)
