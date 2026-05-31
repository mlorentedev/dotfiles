---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - MEMORY-001-cross-agent-session-bridge

> This PR delivers the **Linux/Claude bridge core** (the script + tests). AC3 (hook deploy wiring) + AC4 (Windows port) are the tracked next increment.

## Evidence

- [x] **AC1 — archives the handoff block** → `tests/session-handoff.bats` test 1: a fixture `SessionEnd` payload + a `MEMORY.md` with a `## Session Handoff` block → `00_meta/sessions/<date>-myproj-claude.md` written, carrying `session_id` + the block content, excluding the index-only tail.
- [x] **AC2 — trivial no-op** → tests 2/3/4: no handoff block / no `MEMORY.md` / empty stdin → no record, exit 0. A session-end hook never crashes a session.
- [ ] **AC3 — Claude SessionEnd hook wired** → NEXT increment (settings.json + cross-OS setup merge policy). Design: `__SESSION_END_COMMAND__` placeholder + `.hooks.SessionEnd` merge, mirroring the SessionStart wiring.
- [ ] **AC4 — Windows parity** → NEXT increment (`session-handoff.ps1` + Pester).

## Test status

- `~/.local/bin/bats tests/session-handoff.bats` → `1..4` all `ok`.
- `~/.local/bin/shellcheck scripts/session-handoff.sh` + `bash -n` → clean.

## Decisions made during implementation

- **The hook archives; the agent authors.** `session-handoff.sh` does NOT synthesize the handoff (that needs the agent's reasoning) — it copies the `## Session Handoff` block that `/handoff` wrote into `MEMORY.md` into a timestamped append-only record. Clean separation, and fixture-testable without a real session (ADR-014).
- **Resilience over strictness.** No `set -e`; every missing/malformed/trivial input is a clean `exit 0`. A `SessionEnd` hook that errors would surface noise at the worst moment.
- **Scoped delivery.** The deploy wiring (settings.json + setup merge, cross-OS) is SDD-002-sensitive plumbing; shipped as a separate next increment rather than rushed into the bridge PR.

## Promotion candidates

- [ ] Lesson? **no** — standard hook-bridge; ADR-014 captures the design.
- [ ] ADR-worthy? **no** — ADR-014 already covers it.
- [ ] Pattern? **no**.

## Archive checklist

- [ ] AC3 + AC4 increments landed (then this spec is complete)
- [ ] `proposal.md` -> `status: archived`; folder moved to `specs/archive/`
- [ ] Vault `11-tasks.md` MEMORY-001 ticked with PR links
