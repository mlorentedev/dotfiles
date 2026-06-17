---
id: "ADR-023-agnostic-session-start"
type: adr
status: accepted
owner: manu
date: "2026-06-16"
issue: "dotfiles#405"   # HARNESS-026 — the consumer vertical that implements this
tags: [architecture, decision, harness, cross-agent, session-start]
created: "2026-06-16"
---

# ADR-023: Agnostic session-start — session-brief core + per-agent adapters

## Status

Accepted

## Date

2026-06-16

## Context

`scripts/claude-session-start.sh` is the Claude SessionStart hook. It is monolithic and conflates two responsibilities:

1. **Gathering vault signals** (agent-independent): vault detection, vault-health, memory-temperature scan, crystallize staleness, active/archived spec counts, and the vault→memory symlink (already extracted to the agnostic `ensure-memory-symlink.sh` in MEMORY-002, #404).
2. **Delivering them to Claude** (Claude-specific): emitting `hookSpecificOutput.additionalContext` JSON on stdout, plus `claude-mem-heal`, plus Claude's project-path encoding.

The other agents in the harness get none of this brief. A multi-reference audit (Regla del 3, ADR-015) of the four agents' session-start surfaces found a **capability divergence**, not merely a stylistic one:

| Agent | Session-start surface | Consumes context via |
|---|---|---|
| Claude | Native command hook (`hooks.SessionStart`, 30s timeout) | `hookSpecificOutput.additionalContext` JSON (per-session, dynamic) |
| OpenCode | No command hook (`opencode.jsonc` = MCP/context-prune/share only) | reads AGENTS.md / context files |
| Antigravity (agy) | No command hook; `context.autoLoadContext: true` | loads context files (AGY.md) |
| Copilot | No hook; `copilot-instructions.md` only | static instructions file |

**Only Claude can run a command at session start.** OpenCode, agy, and Copilot are file-based context readers — they cannot consume a per-session dynamic brief; they read instruction/context files that must be regenerated out-of-band (deploy or timer time). This is the same `native` reader vs `pointer`+overlay classification HARNESS-001 (#162) already owns via its cross-agent compiler.

Divergence-log: the **signal-gathering** is a shared core candidate (agent-independent vault facts); the **trigger** (command hook vs file read), the **output format** (JSON vs markdown block), and `claude-mem-heal` are per-agent and NOT core candidates.

## Options Considered

Constraints (from this session): **C1** one agnostic core (no per-agent duplication) · **C2** respect capability divergence (only Claude is runtime) · **C3** do not duplicate the HARNESS-001 compiler · **C4** no regression to Claude's session-start (18 tests incl. byte-equivalence) · **C5** cross-OS (Linux/Windows) · **C6** atomic, incremental PRs.

| Option | C1 | C2 | C3 | C4 | C6 |
|---|---|---|---|---|---|
| **A** — two-mode core that also writes each agent's files itself | ok | ok | **gap** (reinvents file injection) | ok | **gap** (one large PR) |
| **B** — agnostic core + Claude runtime adapter + file delivery via HARNESS-001 compiler | ok | ok | ok | ok | ok |
| **C** — minimal: leave Claude as-is, static snapshot to others, no shared core | **gap** (logic stays Claude-coupled) | ok | ok | ok | ok |

## Decision

**Option B.** Decompose session-start into:

- An **agent-agnostic `session-brief` core** that gathers the vault signals and emits structured output (a stdout mode for runtime consumers and a markdown-block mode for file-based ones).
- A **Claude adapter**: the existing `SessionStart` hook becomes a thin shim that calls the core and formats its output as `hookSpecificOutput` JSON (plus the Claude-only `claude-mem-heal`). `claude-session-start.sh` shrinks to that shim.
- **opencode / agy / copilot adapters**: the core's markdown brief block is injected into each agent's instruction file by the **HARNESS-001 compiler** at deploy/timer time — not per-session, and not by a new parallel mechanism. The session-brief is a *consumer* of the HARNESS-001 engine.

Capability divergence is honoured explicitly: only Claude receives a per-session dynamic brief; file-based agents receive a periodically-regenerated one. That is an inherent property of those agents, not a limitation of this design.

MEMORY-002 (the agnostic `ensure-memory-symlink.sh`) is the first brick of this core and is already merged-pending (#404).

## Consequences

### Positive

- One place owns signal-gathering; a new signal is added once and reaches every agent.
- Rides the HARNESS-001 compiler instead of duplicating file injection (C3) — consistent with the rest of the harness.
- Incremental: the strangler keeps Claude byte-equivalent (C4) while the core grows; each PR is atomic (C6).
- The core is independently testable (no agent runtime needed), like `ensure-memory-symlink.sh`.

### Negative

- File-based agents get a **stale** brief (regenerated at deploy/timer, not per-session). This is a hard capability limit, surfaced rather than hidden.
- Adds a dependency on HARNESS-001's compiler maturity for the non-Claude delivery path; until then only Claude's adapter is live.

### Neutral

- `claude-session-start.sh` remains as a (much thinner) Claude shim — "eliminate" in spirit means "reduce to an adapter", not delete.
- The split mirrors MEMORY-001-mirror's per-agent session-**end** hooks; start and end converge on the same core+adapter shape.

## References

- HARNESS-026 (#405) — the consumer vertical implementing this decision (multi-PR plan)
- HARNESS-001 (#162) — cross-agent harness epic; owns the compiler this rides
- MEMORY-002 (#402, PR #404) — first agnostic brick (`ensure-memory-symlink.sh`)
- MEMORY-001-mirror — sibling (per-agent session-end hooks)
- ADR-015 — Regla del 3 (multi-reference audit discipline applied here)
