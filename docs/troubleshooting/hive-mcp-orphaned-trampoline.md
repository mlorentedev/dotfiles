---
id: "dotfiles-troubleshoot-hive-mcp-orphaned-trampoline"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, claude-code, hive, mcp, uv, windows]
created: "2026-06-29"
owner: manu
---

# Troubleshooting: Hive MCP fails to load — orphaned uv trampoline (Windows)

On Windows the Hive MCP server (`hive-vault`, invoked from `~/.claude.json` as
`~/.local/bin/hive.exe`) silently fails to register: `vault_query` / `vault_search` /
`session_briefing` and the rest of the `mcp__hive__*` tools never appear in the session,
even though the start-of-session `[hive]` banner prints (that banner comes from a startup
hook, not from the live MCP process — it is not proof the server connected).

This is a **process/install integrity** failure, distinct from the per-conversation
transport poisoning in [`hive-mcp-rejection-disconnect.md`](hive-mcp-rejection-disconnect.md).
There the daemon is healthy and only the session handle is broken; here the executable
itself cannot start.

## Symptoms

`ToolSearch` for `vault_query`/`vault_search` returns no match (the tools were never
registered). Running the configured binary directly reveals the real cause:

```
$ ~/.local/bin/hive.exe --version
error: uv trampoline failed to canonicalize script path
```

`uv tool list` shows the tool is gone or malformed:

```
$ uv tool list
warning: Ignoring malformed tool `hive-vault` (run `uv tool uninstall hive-vault` to remove)
aider-chat v0.86.2
pre-commit v4.6.0
# hive-vault absent / malformed
```

## Root cause

On Windows `uv tool install` cannot create symlinks, so it writes a tiny launcher
`.exe` — a **trampoline** — into `~/.local/bin/` that embeds the absolute path to the
real Python environment under `%APPDATA%\uv\tools\hive-vault\`. When that backing
environment is moved, pruned, or left half-written by an interrupted upgrade, the
trampoline survives but points at nothing — hence `failed to canonicalize script path`.
The MCP server process never starts, so Claude Code registers zero hive tools.

Two forces conspire to produce the orphaned state:

1. **Windows cannot replace a running executable.** The Phase C daemon (`hive serve`,
   started at logon by the Startup-folder supervisor) holds an open handle to its own
   `hive.exe`. A plain `uv tool upgrade` / `--reinstall` against a locked install fails
   (`os error 32` on the entrypoint copy, `os error 5` on the venv) and can leave the
   environment **partially written** — e.g. missing `rich`
   (`No module named 'rich.traceback'`), the exact symptom tracked in **#574**. The
   dotfiles `hive-upgrade.ps1` orchestrator sidesteps this with *defer-if-locked*, but a
   manual or out-of-band `uv` operation has no such guard.
2. **The lock blocks the repair, too.** Once broken, `uv tool install --force hive-vault`
   rebuilds the venv but still fails the final entrypoint copy while the stale daemon is
   alive:

   ```
   error: Failed to install entrypoint
     Caused by: failed to copy file from ...\hive-vault\Scripts\hive.exe
     to C:\Users\mlorente\.local\bin\hive.exe:
     The process cannot access the file because it is being used by another process. (os error 32)
   ```

   The running daemon (`hive.exe` + its `python.exe` child under
   `...\uv\tools\hive-vault\Scripts\`) must be stopped **before** the reinstall can land.

## Fix (manual recipe — until #574 automates it)

Stop everything holding `hive.exe`, force-reinstall, verify, then restart Claude Code so
the MCP subsystem reconnects.

```powershell
# 1. Stop the supervisor FIRST (else it respawns the daemon mid-copy), then the daemon.
#    Exclude $PID — the script text itself contains 'hive-serve-supervisor', which would
#    otherwise match (and kill) your own shell.
$me = $PID
Get-CimInstance Win32_Process -Filter "Name='powershell.exe' OR Name='pwsh.exe'" |
  Where-Object { $_.ProcessId -ne $me -and $_.CommandLine -like '*hive-serve-supervisor.ps1*' } |
  ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
Get-Process | Where-Object {
  $_.Id -ne $me -and ($_.Path -like '*\.local\bin\hive.exe' -or $_.Path -like '*uv\tools\hive-vault*')
} | Stop-Process -Force
```

```bash
# 2. Force-reinstall the tool (entrypoint copy now succeeds — nothing holds the exe).
uv tool install --force hive-vault
```

```bash
# 3. Verify the trampoline resolves AND the server speaks MCP (stdout JSON-RPC only;
#    the FastMCP banner is stderr — never merge it with 2>&1 when testing a stdio server).
~/.local/bin/hive.exe --version          # -> hive-vault <ver>, no canonicalize error
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"diag","version":"0"}}}' \
  | HIVE_VAULT_PATH="<vault>" "$HOME/.local/bin/hive.exe" 2>/dev/null | head -c 200
# Expect: {"jsonrpc":"2.0","id":1,"result":{...,"serverInfo":{"name":"Hive",...}}}
```

4. **Restart Claude Code.** MCP servers connect at session start; the running session
   will not pick up the repaired binary. The daemon/supervisor self-reprovision at the
   next logon (Startup folder), so no manual restart is needed — and the deployed config
   runs `hive` with no args (the per-session stdio MCP server), which does not depend on
   the daemon anyway.

## Detection

```bash
# Healthy: resolves and prints a version.  Broken: 'failed to canonicalize script path'.
~/.local/bin/hive.exe --version

# Healthy: 'hive-vault vX.Y.Z' with executables hive, hive-vault.
# Broken:  'Ignoring malformed tool `hive-vault`' or hive-vault absent.
uv tool list
```

**Since 2026-08-07 the auto-upgrade timer detects it for you.** This failure
recurred on the maintainer's box and sat unnoticed for months, because
`hive-upgrade.ps1` treated "no install found" and "already up to date" as the same
silent `exit 0`. It now exits **non-zero** with a message when no install is
resolvable, so the condition surfaces in Task Scheduler without anyone running
anything (dotfiles#796 / AI-028 PR1):

```powershell
Get-ScheduledTaskInfo -TaskName DotfilesHiveUpgrade   # LastTaskResult != 0 => broken install
```

That is detection, not repair — the recipe above is still the fix until #574 lands.

The durable, cross-machine version of this check + repair is **#574** — `dotf doctor`
should FAIL loudly when the Hive install cannot start and `dotf doctor --fix` should run
the stop-daemon → force-reinstall recipe idempotently. Until that lands, this note is the
manual fallback.

## When to retire this note

When #574 ships and `dotf doctor --fix` repairs the orphaned trampoline automatically:

1. Deliberately break the install (stop the daemon, `uv tool uninstall hive-vault`, or
   point the trampoline at a missing venv) and confirm `dotf doctor` FAILs loudly and
   `dotf doctor --fix` restores `hive.exe --version` + a clean MCP handshake.
2. Set this note's frontmatter `status: archived` and link to the #574 PR.

A complementary upstream hardening (self-detect/self-heal a malformed install inside
`hive service`, or a `hive doctor` subcommand) would close the gap at the source; if
filed, link the `mlorentedev/hive` issue here.

## Related

- Durable guard: **#574** (HARNESS-049) — `dotf doctor --fix` repairs the Hive venv.
- Upgrade orchestration that prevents the locked-exe corruption on the happy path:
  `windows/hive-upgrade.ps1` (only-if-newer → defer-if-locked → stop → upgrade → start).
- Adjacent failure mode (same surface, different layer): [`hive-mcp-rejection-disconnect.md`](hive-mcp-rejection-disconnect.md) — per-conversation transport, not install integrity.
- Lesson: [`../lessons.md`](../lessons.md) — 2026-06-29 entry on orphaned uv trampolines.
