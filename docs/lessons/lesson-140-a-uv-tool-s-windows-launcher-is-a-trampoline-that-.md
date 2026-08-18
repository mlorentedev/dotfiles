---
id: lesson-140-a-uv-tool-s-windows-launcher-is-a-trampoline-that-
type: lesson
status: active
created: "2026-06-29"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 140: A uv tool's Windows launcher is a trampoline that orphans silently — and a running daemon blocks its own repair

**Context**: A session opened with the start-of-session `[hive]` banner printed, but `ToolSearch` for `vault_query`/`vault_search` returned nothing — the Hive MCP server's tools never registered. The config in `~/.claude.json` looked correct (`hive.exe`, right `HIVE_VAULT_PATH`), so it was not a config problem.

**Problem**: Running the configured binary directly exposed the cause: `hive.exe --version` → `error: uv trampoline failed to canonicalize script path`. On Windows `uv tool install` cannot symlink, so it writes a tiny launcher `.exe` (a **trampoline**) into `~/.local/bin/` that embeds the absolute path to the real venv under `%APPDATA%\uv\tools\hive-vault\`. That venv had been pruned/half-written out from under the trampoline (the same malformed state as #574 — venv missing `rich`), so the launcher survived but pointed at nothing and the MCP process never started. Two Windows-specific forces produced and then *protected* the broken state: (1) Windows cannot replace a running `.exe`, so an out-of-band `uv` upgrade against the live daemon left the env partially written; (2) the stale `hive serve` daemon (PID + its `python.exe` child) held an open handle to `hive.exe`, so even the repair — `uv tool install --force hive-vault` — rebuilt the venv but failed the final entrypoint copy with `os error 32` (file in use). The banner had masked all of this: it is emitted by a startup hook, **not** by the live MCP process, so "banner printed" was never evidence the server connected.

**Solution**: Stop the holder before repairing. Kill the Startup-folder supervisor FIRST (else it respawns the daemon mid-copy — and exclude `$PID` from the match, since the kill script's own text contains the supervisor name), then the daemon + its python child, then `uv tool install --force hive-vault` (entrypoint copy now lands), then verify two things, not one: `hive.exe --version` resolves AND a real MCP `initialize` returns a JSON-RPC result with `serverInfo`. MCP connects at session start, so the running session can't see the fix — restart Claude Code. Captured the full recipe in `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`; the durable cross-machine guard is #574 (`dotf doctor --fix`), left open because this fix was manual.

**Rule**: On Windows, a uv-installed CLI in `~/.local/bin/*.exe` is a **trampoline, not the program** — "the launcher exists" ≠ "the tool runs"; diagnose by invoking it (`--version`) and reading the error, never by checking the file is present. When repairing a tool whose **running process holds its own executable**, stop every holder (and its supervisor, so it can't respawn) *before* the reinstall, or the lock turns `--force` into a half-install. And when testing a **stdio MCP server**, keep stdout pure JSON-RPC: the FastMCP startup banner goes to stderr, so a `2>&1` merge makes a healthy server look broken — separate the streams (`2>/dev/null`) and assert on the `initialize` result.
