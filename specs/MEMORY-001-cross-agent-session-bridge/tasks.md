---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - MEMORY-001-cross-agent-session-bridge

> TDD order. Implements ADR-014. This PR ships the **ADR + spec (design)**; the implementation tasks below execute in a follow-up.

## Setup

- [x] Vault gate entry (MEMORY-001-mirror) in `11-tasks.md`
- [x] **ADR-014** written + ratified (`docs/adr/adr-014-cross-agent-session-memory-bridge.md`)
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation (follow-up — TDD)

- [ ] Write failing bats: `session-handoff.sh` fed a fixture `SessionEnd` payload writes `00_meta/sessions/<date>-<project>-claude.md` (AC1)
- [ ] Implement `scripts/session-handoff.sh`: parse stdin JSON (`jq`), gate on non-trivial, render the session record from the `/handoff` schema → green (AC1)
- [ ] bats: trivial-session payload → no record written (AC2)
- [ ] Wire the Claude `SessionEnd` hook in `ai/claude/settings.json` → `session-handoff.sh`; extend the settings merge policy assert (AC3)
- [ ] `scripts/session-handoff.ps1` parity stub + cross-OS fixture parity assert (AC4)

## Closing

- [ ] Every AC covered by a test
- [ ] `features.json` present
- [ ] `verification.md` filled
- [ ] PR opened referencing this spec folder

## Follow-up tickets (downstream, per ADR-014)

- [ ] Per-agent session-end hooks: OpenCode (`opencode.jsonc`), agy (`~/.gemini/settings.json`), Copilot — each needs its session-end surface validated (some Windows-empirical).
- [ ] `00_meta/sessions/` archival / rotation hygiene.
