---
id: "ADR-014-cross-agent-session-memory-bridge"
type: adr
status: accepted
owner: manu
date: "2026-05-31"
extends: [adr-013-agent-artifact-deploy-engine]
tags: [architecture, decision, session-memory, continuity, cross-agent, dotfiles, harness-001, memory-001]
created: "2026-05-31"
---

# ADR-014: Cross-agent session-memory bridge

> Drives `MEMORY-001` (GH [#117](https://github.com/mlorentedev/dotfiles/issues/117)); builds on `HANDOFF-001` (the `/handoff` skill) and the SDD-008 deploy engine (ADR-013).

> **Update (2026-06-22):** the session-record **location** specified below (`00_meta/sessions/`) was
> superseded by **HARNESS-031** (`feedback_sessions_in_project_folder`): records now live in
> `10_projects/<project>/sessions/`, enforced by `vault-validate.py` §9 and written there by the
> `/handoff` skill. The `session-handoff.{sh,ps1}` bridge was updated to match (it already resolves
> the project at `10_projects/<project>/`). The rest of this ADR — the bridge mechanism, the
> trigger model, and the record schema — stands unchanged.

## Status

Accepted

## Date

2026-05-31

## Context

`HANDOFF-001` made the session handoff a cross-agent `/handoff` skill, but two gaps remain. (1) **No durable history**: `/handoff` *overwrites* the `## Session Handoff` block in `MEMORY.md` — only the latest session survives; there is no per-session record to look back on. (2) **No auto-trigger and no non-Claude continuity**: the handoff fires only when an agent follows the AGENTS.md instruction, and `MEMORY.md` is Claude's auto-memory — other agents (OpenCode, agy, Copilot) have no continuity sink at all. `MEMORY-001` is the bridge that closes both gaps.

The enabling fact that shapes the design: **Claude Code exposes a reliable `SessionEnd` hook** (verified) — it fires once at session termination (reasons: `clear`/`resume`/`logout`/…), delivers JSON on stdin (`session_id`, `transcript_path`, `cwd`, `hook_event_name`), and can run a shell command. So an *automatic* session-end capture is viable, at least for Claude. The other agents' session-end surfaces are not yet confirmed (and partly Windows-empirical).

## Decision Drivers

- Durable, append-only continuity that survives across sessions and agents.
- Reuse `/handoff` (HANDOFF-001) — one procedure, two invocation paths (auto + manual), no second schema.
- Don't bet the architecture on session-end hooks every agent may not have.
- Cross-OS (bash + PowerShell), and CI-testable without the private vault.

## Considered Options

1. **Per-agent auto session-end hook + manual fallback** (CHOSEN). Each agent triggers `/handoff` at session end via its native session-end hook where one is confirmed reliable; agents without one fall back to the manual `/handoff` instruction already in AGENTS.md.
2. **Manual-only `/handoff`.** Rejected — it wastes Claude's confirmed `SessionEnd` capability and leaves continuity dependent on the agent remembering.
3. **A daemon / filesystem watcher.** Rejected — added moving part, no clean "session boundary" signal, cross-OS service complexity.

## Decision

Adopt a **cross-agent session-memory bridge** with three parts:

1. **Append-only session store.** Each session writes a record `vault/00_meta/sessions/<YYYY-MM-DD>-<project>-<agent>.md` using the `/handoff` schema (`Last task` / `Decisions` / `Open threads` / `Next action`) plus frontmatter (`session_id`, `agent`, `project`, `date`). This is the durable **history**; the `MEMORY.md` `## Session Handoff` block remains the **latest** snapshot. The two are complementary, not redundant.
2. **Trigger = native session-end hook where reliable, manual `/handoff` otherwise.** Claude wires its confirmed `SessionEnd` hook (in `settings.json`) to `scripts/session-handoff.sh`. Agents without a confirmed reliable session-end hook use the manual `/handoff` instruction (unchanged). Per-agent hook wiring for OpenCode / agy / Copilot is **empirical validation, tracked separately, non-blocking** on this ADR.
3. **The bridge script `scripts/session-handoff.sh` (+ `.ps1`).** Reads the hook payload (`session_id`, `transcript_path`) from stdin, decides whether the session was non-trivial, and writes/updates the session record + the `MEMORY.md` block. Cross-OS; unit-testable against a fixture payload (no vault needed in CI).

## Consequences

- **Positive.** Claude gets automatic, durable session records immediately (SessionEnd is confirmed). `/handoff` is the single shared procedure both the auto-hook and the manual path invoke — no schema fork. The session store gives true cross-agent continuity once each agent's trigger is wired. The bridge is fixture-testable, so CI covers the logic without the private vault.
- **Negative / risks.** (a) Non-Claude agents stay manual until their session-end surfaces are validated — partial coverage at first (tracked, acceptable). (b) `SessionEnd` fires on `clear`/`resume` too, so the bridge must gate on "was meaningful work done?" to avoid noise records. (c) An append-only store grows; periodic archival of `00_meta/sessions/` is a future hygiene task, not a blocker.
- **Follow-up.** `MEMORY-001` implements parts 1–3 for Claude (Linux first); per-agent hook validation (OpenCode/agy/Copilot, some Windows-empirical) and session-store archival are downstream tickets.

## References

- `HANDOFF-001` — the `/handoff` skill (the shared procedure this bridge auto-invokes). `vault/00_meta/skills/handoff/SKILL.md`.
- ADR-013 — agent-artifact deploy engine (how the hook + script deploy cross-OS).
- Claude Code `SessionEnd` hook (verified capability: once-per-session, stdin JSON payload with `session_id` + `transcript_path`).
- `claude-mem` — precedent for session-hook capture (conversation flow), complementary to this crystallized-continuity store.
- `00_meta/patterns/pattern-decision-persistence.md`; GH [#117](https://github.com/mlorentedev/dotfiles/issues/117) (MEMORY-001).
