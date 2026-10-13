---
id: "lesson-405-a-config-written-only-by-a-setup-script-freezes-where-it-does-not-run"
type: lesson
status: active
title: "A config written only by a setup script freezes on every machine that does not run it"
created: "2026-10-13"
---

# A config written only by a setup script freezes on every machine that does not run it

## Context
agy reads its MCP servers from `~/.gemini/config/mcp_config.json`. Until CLI-105 (#2384) the only
writers were the "Consolidate MCP servers" blocks of `setup-linux.sh` and `setup-windows.ps1`: one
in jq, one in `ConvertTo-Json`. macOS runs neither (ADR-045); it converges through `dotf`.

## The Trap
Nothing failed. On the Mac the master file existed, was valid JSON and passed every doctor check,
because a setup run months earlier had written it. It was simply frozen: rendering it from the
current checkout into a scratch `GEMINI_HOME` and diffing with `jq -S` showed the Mac still
launching `uvx pdf-modifier-mcp`, a package name `mcp-servers.json` had already dropped. A
missing file is loud; a stale one passes every check that asks "is it there, is it JSON".

Porting the block also surfaced a bug both twins carried. The jq twin split a server's args with
`cut -d' ' -f2-`, and `cut` prints a line that has no delimiter **whole**, so a one-word command
would have been handed itself as its only argument. No declared stdio server had a one-word
command, so the bug never showed.

## The Solution
`cli/internal/agy.ApplyMCP` renders both files from the checkout and writes only on a byte
change. The bare `dotf deploy` and converge's `agent-extensions` step call it on every OS, both
twins lost their block, and `tests/guard-agy-mcp-in-dotf.bats` keeps it from growing back.

## Takeaways
- A file whose only writer is an OS-specific script is frozen on every OS that does not run it.
  Doctor's existence and syntax checks cannot see that; only re-rendering from the source and
  diffing can.
- When a writer moves into `dotf`, diff its output against what is already on a real machine
  before claiming the change only rewrites bytes. Here the diff found the stale package name.
- `cut -d<sep> -f2-` is not "everything after the first field". On a line with no separator it
  returns the whole line. Use a real split (`strings.Fields`, `${line#* }` after a separator
  test) and test the one-field case.
