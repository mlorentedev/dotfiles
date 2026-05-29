---
id: "dotfiles-troubleshoot-hive-mcp-rejection-disconnect"
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, claude-code, hive, mcp, transport]
created: "2026-05-15"
owner: manu
---

# Troubleshooting: Hive MCP tools disappear after a rejected first call

A single user rejection of the first `mcp__hive__*` tool call in a Claude Code conversation poisons the MCP transport for the rest of that conversation. The Hive daemon itself stays healthy at the process level; only the per-conversation handle is broken. Restarting the conversation recovers; nothing else in the same session does. Filed upstream as [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75).

## Symptoms

The first attempt after the rejection prints:

```
MCP error -32000: Connection closed
```

Every subsequent attempt prints:

```
No such tool available: mcp__hive__<anything>
```

Claude Code's `<system-reminder>` mechanism eventually surfaces:

```
The following deferred tools are no longer available (their MCP server disconnected). Do not search for them — ToolSearch will return no match: mcp__hive__capture_lesson, mcp__hive__delegate_task, mcp__hive__session_briefing, mcp__hive__vault_health, mcp__hive__vault_list, mcp__hive__vault_patch, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__worker_status
```

```
The following MCP servers have disconnected. Their instructions above no longer apply: hive
```

The daemon itself is fine — running `claude mcp list` from a separate terminal reports:

```
hive: uvx hive-vault - ✓ Connected
```

So the failure mode is "session-level transport state" vs "daemon-level health", and they have diverged.

## Root cause (hypothesis, unverified)

Two equally plausible candidates, only one of which the report at #75 can disambiguate:

(a) Claude Code's MCP transport layer tears down the stdio pipe when a tool call is rejected and does not spawn a replacement for the current conversation.

(b) FastMCP / `hive-vault` returns an unrecoverable status when its first call is cancelled mid-handshake, causing Claude Code to mark the server permanently dead for that conversation.

Distinguishing requires either the Hive server log at the moment of rejection (does the process see the cancellation or just stdio close?) or a parallel repro with a different MCP server using a different transport library.

## Workaround

**Always approve the first Hive tool call in a fresh conversation.** You can deny later ones individually with no transport impact — the breakage is specific to the very first call.

If the transport is already broken in the current conversation, the only recovery is restart. There is no in-session command, hook, or `claude mcp ...` invocation that re-attaches the transport to the running conversation.

Operationally that means:

1. Open Claude Code as usual.
2. Let the session's first Hive call go through (e.g. `mcp__hive__session_briefing` at start, vault read on demand).
3. Approve, then proceed normally.

If you accidentally reject the first call, start a fresh conversation immediately rather than trying to recover.

## Detection

A future session can confirm whether it has hit this failure mode by checking:

```
# Outside the conversation, in a terminal — daemon should be healthy:
claude mcp list | grep -E '^hive:'
# Expect: hive: uvx hive-vault - ✓ Connected
```

If the daemon is `✓ Connected` but the conversation cannot reach `mcp__hive__*` tools, this troubleshooting note applies — go to Workaround. If the daemon itself is unhealthy (anything other than `✓ Connected`), this is a different problem (most likely a `uvx` cache corruption — handle separately).

## When to retire this note

Issue [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75) needs to close with one of:

- A fix in `hive-vault` / FastMCP that makes the transport recoverable after rejection, OR
- A fix in Claude Code that auto-respawns the MCP transport per-conversation, OR
- A documented "this is by design" with a clear in-conversation recovery procedure.

Once the upstream resolution lands and a repro confirms the bug is gone:

1. Verify a deliberate first-call rejection no longer poisons the transport.
2. Set this note's frontmatter `status: archived` and link to the upstream resolution commit/PR.

## Related

- Upstream issue: [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75)
- Lesson: [`../lessons.md`](../lessons.md) — 2026-05-15 entry on MCP transport state vs daemon state
- Adjacent troubleshooting (different failure mode, same surface): [`claude-mem-broken-marketplace.md`](claude-mem-broken-marketplace.md) — also an MCP-layer issue, but server-side packaging rather than session-state transport
- Spec that explicitly watches this issue during OpenCode bootstrap: `~/Projects/dotfiles/specs/AI-011-opencode-bootstrap/proposal.md` (Risks section)
