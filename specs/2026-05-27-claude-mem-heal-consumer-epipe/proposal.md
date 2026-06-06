---
id: "2026-05-27-claude-mem-heal-consumer-epipe"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-06"
tags: [spec, proposal, claude-mem, windows, epipe]
template_version: "1.0"
---

# 2026-05-27-claude-mem-heal-consumer-epipe

> Regression of BUG-017. Fixes a consumer-side EPIPE in the hooks.json the
> `claude-mem-heal` scripts deploy. RESUMED 2026-06-06 with **Option A**
> (minimal `head -n1` → `sed -n 1p` drain), which supersedes the original
> 2026-05-27 canonical-template-emission plan — see "Decisions" below.

## Why

`claude-mem-heal.{sh,ps1}` patches the `thedotmack/claude-mem` plugin's
`hooks.json` at every session start. BUG-017 rewrote the path-resolution pipe
from `... }; done` (with an early `break`) to `... }; done | head -n1` to close
a **producer**-side EPIPE. That introduced a **consumer**-side EPIPE: the inner
`while` loop is the writer into `head -n1`; when ≥2 plugin candidates match
(cache version dir **and** the marketplace junction both contain the hook
scripts), the loop emits ≥2 lines. `head -n1` prints the first and closes its
stdin; if the loop is still writing, its next `printf` hits a closed pipe →
EPIPE. Because Claude Code is a Node process and **Node sets SIGPIPE to
SIG_IGN**, that disposition is inherited by the spawned hook, so the write
error is *reported* (`printf: write error: …`) instead of dying silently.
Claude Code surfaces the non-empty hook stderr as a `PreToolUse/PostToolUse
hook error` banner on tool calls. It is non-blocking (the `$(…)` capture
completes before the failing write, so `_P` is set correctly) but pollutes
every session on Windows.

## What

After this PR the `hooks.json` deployed by `claude-mem-heal.{sh,ps1}` resolves
the plugin path with `... }; done | sed -n 1p` instead of `... }; done |
head -n1`. `sed -n 1p` prints only the first line **but reads its input to
EOF** (no early `q`), so it never closes the pipe while the writer is still
running — the consumer-side EPIPE is eliminated for any number of matching
candidates. Both heal scripts also detect and normalise the pre-existing
`head -n1` form (already deployed on machines healed before this PR), so the
next session converts them with no `/plugin update` required. Output of the
discovery (the resolved path) is byte-identical to the `head -n1` form.

## Out of scope

- The `.mcp.json` heal (`heal_mcp_json` / `Repair-McpJson`), which also emits
  `head -n1`. It is loaded once at MCP-server startup, not per tool call, so it
  is not the per-call banner source. Its theoretical exposure to the same race
  is logged under "Risks / open questions" as a follow-up, not fixed here
  (atomic PR).
- The original "emit our own canonical hooks.json template" rewrite (rejected —
  see Decisions).
- The `init-spec.{sh,ps1}` vault-gate regex gap that rejects `[~]` (paused)
  entries — surfaced while resuming this task; tracked as a separate follow-up.
- BUG-019's `neuter_context_hook` behaviour (unchanged; only its sibling
  `heal_hooks_json` changes).

## Risks / open questions

- **sed availability/behaviour.** `sed -n 1p` (no quotes, no `q`) must read to
  EOF on the runtime sed (GNU/BSD/busys/MSYS). Guarded by the functional test;
  `sed` is already assumed present (the heal scripts and hooks use coreutils).
- **Upstream shape drift.** Detection covers the two known forms
  (`break; }; done` pristine, `}; done | head -n1` BUG-017-era). If upstream
  ships a third pipe shape, heal no-ops on it and a new bug is filed — accepted
  trade-off of keeping the sed-patch architecture (Option A) over a static
  self-emitted template (Option B), which would instead be fragile to upstream
  *semantic* drift.
- **Follow-up:** `.mcp.json` uses the same `head -n1`; same race is possible
  when ≥2 candidates carry `mcp-server.cjs`. Lower blast radius (one-shot at MCP
  load, not per tool call). Decide separately whether to apply the same
  `sed -n 1p` drain there.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] **AC1 (root-cause guard, functional, cross-OS):** under `trap '' PIPE`
  (Node's SIGPIPE-ignore) with a slow writer, the `head -n1` consumer reports a
  write error and the `sed -n 1p` consumer does not. (`tests/claude-mem-heal.bats` #18)
- [x] **AC2 (.sh heal):** `heal_hooks_json` on a `break; }; done` fixture and on
  a `}; done | head -n1` fixture both yield a file containing `sed -n 1p`, with
  no `head -n1` remaining, and the BUG-018 continue-directive preserved. (#19, #20)
- [x] **AC3 (.ps1 parity):** `Repair-HooksJson` produces the equivalent
  result on the same fixtures (Windows). (`tests/claude-mem-heal-ps1.bats` #22, #23)
- [x] **AC4 (idempotency):** re-running each heal on its own output is a no-op
  (file unchanged, "already healthy"/silent). (#21, #24)
- [x] **AC5 (live):** healing a copy of the actual deployed
  `~/.claude/.../13.3.0/hooks/hooks.json` converts all 7 `head -n1` to
  `sed -n 1p`; second run silent; output still valid JSON; diff is surgical.

## Decisions

- **Option A over Option B.** Option B (the paused 2026-05-27 plan: stop
  patching upstream's shape, emit our own canonical `hooks.json` template with a
  `_candidates=$(…)` materialise + here-doc `break`) is immune to upstream
  *shape* changes but **fragile to upstream *semantic* changes** (a new hook
  stage or changed `node` invocation would make our static template ship a
  broken/stale hooks.json). claude-mem is volatile (12.7.4 → 13.0.0 → 13.3.0,
  hooks evolving), so a static self-emitted template is the larger long-term
  risk. Option A keeps the resilient sed-patch (rewrites only the pipe shape,
  preserves upstream's `node` invocation) and fixes the EPIPE with a 2-line
  change: `head -n1` → `sed -n 1p`.
- **`sed -n 1p` over a `{ read; printf; cat >/dev/null; }` drain.** Equivalent
  first-line output, but `sed -n 1p` has no quotes/newlines/backslashes, so it
  needs no JSON-string escaping inside the hook command nor single-quote
  gymnastics inside the heal's own `sed` script.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (entry `2026-05-27-claude-mem-heal-consumer-epipe`)
- Predecessors: `specs/archive/BUG-016-claude-mem-heal-v13-refresh/`,
  `specs/archive/BUG-017-claude-mem-heal-hooks-json-race/`
- Upstream: thedotmack/claude-mem#2607 (EPIPE race) — heal becomes a no-op if/when upstream fixes it
- Related: TEST-001 (#128) wants bats coverage for `claude-mem-heal.{sh,ps1}`
