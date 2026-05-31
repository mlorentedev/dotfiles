---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - HANDOFF-001-cross-agent-handoff-skill

> TDD order. One task = one focused commit.

## Setup

- [x] Vault gate entry HANDOFF-001 in `11-tasks.md` (committed 58c5bed)
- [x] Branch `feat/handoff-001-cross-agent-skill` off origin/main (no parallel session → primary checkout, no worktree)
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation

- [x] Author the SSOT skill `vault/00_meta/skills/handoff/SKILL.md` — the ordered checklist (continuity block + vault hygiene + repo/worktree state + artifact summary + next action + verification) (AC1)
- [x] Add the always-on trigger to `AGENTS.md` Phase 3 (Knowledge Crystallization) pointing to the skill (AC2)
- [x] Reduce `ai/claude/CLAUDE.md` "Session Handoff" to a pointer — drop the duplicated 5-field detail (AC2)
- [x] `compile-harness.sh --refresh` → regenerate record `harness/skills/handoff/`; `--check` green; minimal diff (AC3, AC4)
- [x] bats: `/handoff` deploys cross-agent (claude + opencode + agy) carrying its checklist (AC3)

## Closing

- [x] Every AC covered by a test/check
- [x] `features.json` present with non-vacuous verification commands
- [x] `--check` + bats green; no unrelated changes (diff = skill record + AGENTS.md + CLAUDE.md + spec + test)
- [x] `verification.md` filled
- [ ] PR opened referencing this spec folder

## Follow-up (separate)

- [ ] MEMORY-001 cross-agent session bridge (continuity for non-Claude agents → `00_meta/sessions/`).
- [ ] Optional: a session-end hook that auto-invokes `/handoff` (today the trigger is an instruction, not automation).
