---
id: "MEMORY-001-cross-agent-session-bridge"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-05-31"
tags: [spec, proposal, session-memory, continuity, cross-agent, adr-014, harness-001]
template_version: "1.0"
---

# MEMORY-001-cross-agent-session-bridge

> **Naming**: file lives at `<repo>/specs/MEMORY-001-cross-agent-session-bridge/proposal.md`. Implements **ADR-014** (`docs/adr/adr-014-cross-agent-session-memory-bridge.md`).

## Why

<!-- from 11-tasks.md: ([#117](https://github.com/mlorentedev/dotfiles/issues/117)) — Cross-Agent Session Memory Bridge dotfiles-side. **Dotfiles deliverables:** (a) `session-end` hook for opencode + AGY + copilot; (b) `dotfiles/scripts/session-handoff.sh` reads hook payload, writes to `00_meta/sessions/`; (c) cross-OS test parity. **Depends_on:** ADR (now ADR-014), SDD-008. -->

`HANDOFF-001` made the handoff a cross-agent `/handoff` skill, but it only *overwrites* the `## Session Handoff` block in `MEMORY.md` (latest-only) and fires only when an agent follows the instruction — there is no durable per-session history and no continuity for non-Claude agents. ADR-014 ratified the bridge design; this spec builds its **Linux/Claude core**: an append-only session-record store plus an automatic capture wired to Claude's confirmed `SessionEnd` hook, reusing the `/handoff` schema. Per-agent triggers for OpenCode/agy/Copilot (some Windows-empirical) are explicit follow-ups.

## What

1. **Append-only session store**: a record `vault/00_meta/sessions/<YYYY-MM-DD>-<project>-<agent>.md` per non-trivial session, with frontmatter (`session_id`, `agent`, `project`, `date`) + the `/handoff` body (`Last task` / `Decisions` / `Open threads` / `Next action`). Durable history; complements `MEMORY.md`'s latest snapshot.
2. **Bridge script `scripts/session-handoff.sh`**: reads the Claude `SessionEnd` JSON payload (`session_id`, `transcript_path`, `cwd`) from stdin, decides whether the session was non-trivial, and writes the session record (+ optionally refreshes the `MEMORY.md` block). No-ops cleanly on trivial sessions.
3. **Claude `SessionEnd` hook** wired in `ai/claude/settings.json` → `session-handoff.sh`, deployed by setup (the merge policy already manages `hooks`).
4. **Cross-OS + fixture-tested**: a `.ps1` mirror stub (parity) and a bats test driving `session-handoff.sh` with a fixture `SessionEnd` payload — no vault/Obsidian needed in CI.

## Out of scope

- **Per-agent session-end hooks for OpenCode / agy / Copilot** — their session-end surfaces are unconfirmed and partly Windows-empirical; tracked as downstream tickets (ADR-014 follow-up). Those agents keep the manual `/handoff` path meanwhile.
- **Session-store archival / rotation** — `00_meta/sessions/` grows append-only; pruning is a later hygiene task.
- **Changing the `/handoff` schema or the `MEMORY.md` format** — reused as-is.

## Risks / open questions

- **`SessionEnd` fires on `clear`/`resume` too** → noise records. **Mitigation:** the bridge gates on "was meaningful work done?" (e.g. transcript length / a marker) before writing.
- **Cross-OS payload parsing** (bash `jq` vs PowerShell `ConvertFrom-Json`). **Mitigation:** parity test with the same fixture payload on both; ADR-013 already establishes the jq/ConvertFrom-Json dual approach.
- **Writing to the vault from a hook** while the vault may be mid-sync. **Mitigation:** append a new file (not edit a shared one) to avoid races; obsidian-git commits it.

## Acceptance criteria

- [ ] **AC1 — bridge writes a record**: `scripts/session-handoff.sh` fed a fixture `SessionEnd` JSON payload writes `00_meta/sessions/<date>-<project>-claude.md` with the frontmatter + `/handoff` sections. **Verify:** bats fixture.
- [ ] **AC2 — trivial-session no-op**: a fixture payload representing a trivial session writes no record (exit 0, no file). **Verify:** bats fixture.
- [ ] **AC3 — Claude hook wired**: `ai/claude/settings.json` registers a `SessionEnd` hook invoking `session-handoff.sh`; the settings merge policy preserves it. **Verify:** grep settings template + bats parity assert.
- [ ] **AC4 — cross-OS parity**: `session-handoff.ps1` exists with the same contract; a Pester/bats parity assert covers the fixture payload. **Verify:** parity test.

## References

- **ADR-014** `docs/adr/adr-014-cross-agent-session-memory-bridge.md` (the ratified design this implements)
- `HANDOFF-001` — the `/handoff` skill + schema reused here
- ADR-013 — deploy engine + the jq/ConvertFrom-Json cross-OS pattern
- GH [#117](https://github.com/mlorentedev/dotfiles/issues/117); epic HARNESS-001 [#162](https://github.com/mlorentedev/dotfiles/issues/162)
