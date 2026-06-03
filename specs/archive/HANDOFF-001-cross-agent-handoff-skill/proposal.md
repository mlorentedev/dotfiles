---
id: "HANDOFF-001-cross-agent-handoff-skill"
type: spec
status: archived
created: "2026-05-31"
tags: [spec, proposal, cross-agent, handoff, skill, harness-001, sdd-011-sibling]
template_version: "1.0"
---

# HANDOFF-001-cross-agent-handoff-skill

> **Naming**: file lives at `<repo>/specs/HANDOFF-001-cross-agent-handoff-skill/proposal.md`.

## Why

<!-- from 11-tasks.md: *(P2, queued 2026-05-31, GH TBD, cross-agent, SDD-required)*: Promote the session "handoff" from a Claude-only rule (a `~/.claude/CLAUDE.md` directive to overwrite `MEMORY.md`'s `## Session Handoff`) into a **cross-agent `/handoff` skill**, so "do handoff" runs the same complete checklist in any agent. **Two-surface design (mirrors [[SDD-011-agent-side-spec-trigger]]):** (1) detail = vault skill `00_meta/skills/handoff/SKILL.md`; (2) trigger = a short always-on line in `AGENTS.md`. **Done when:** `/handoff` skill exists in vault + deploys to all agents (`compile-harness.sh --check` green), AGENTS.md trigger added, CLAUDE.md rule reduced to a pointer. **Anti-scope:** don't change the `MEMORY.md` schema. Part of HARNESS-001. -->

Today the session handoff is a Claude-only rule in `ai/claude/CLAUDE.md` that does one thing: overwrite the `## Session Handoff` block in `MEMORY.md`. It is neither complete (no vault hygiene / repo-state / artifact-summary steps) nor portable (other agents — opencode, agy, copilot — have no handoff at all). The user wants to tell ANY session "do handoff" and get the SAME complete result, so continuity does not depend on which agent ran the session. This promotes the handoff into a real cross-agent skill on the existing SDD-008 pipeline — the same two-surface pattern SDD-011 used.

## What

After this PR:

1. **A `/handoff` skill is the SSOT** at `vault/00_meta/skills/handoff/SKILL.md` — a complete, ordered checklist: (1) overwrite the `## Session Handoff` continuity block in `MEMORY.md` (the 5 fields), (2) vault hygiene (tick `11-tasks.md`, capture lessons), (3) repo/worktree/branch state verification, (4) artifact/PR summary, (5) concrete next action, (6) verification.
2. **It deploys cross-agent** through the SDD-008 pipeline: `compile-harness.sh --refresh` regenerates the committed record `harness/skills/handoff/`; `--deploy` renders it to `~/.claude/skills/handoff`, `~/.config/opencode/commands/handoff.md`, `~/.gemini/{skills,prompts}`; `--check` stays green. So `/handoff` (or "do handoff") works in Claude, OpenCode, and agy.
3. **`AGENTS.md` carries the always-on trigger** (Phase 3 / Knowledge Crystallization): "at session end, run the `/handoff` skill", pointing to the skill — cross-agent, no checklist duplicated.
4. **The Claude-only rule becomes a pointer**: `ai/claude/CLAUDE.md`'s "Session Handoff" section is reduced to a short pointer to the skill (the 5-field detail is no longer duplicated there).

## Out of scope

- **The cross-agent session-memory bridge** (writing continuity for non-Claude agents to `00_meta/sessions/`) — that is MEMORY-001. Until it lands, the skill targets the vault-junctioned `MEMORY.md` path, which any agent can write.
- **Changing the `MEMORY.md` schema** (the 5 fields, the index-only rule) — unchanged; the skill consumes it as-is.
- **Auto-invoking the handoff** (a session-end hook that runs it without the agent deciding) — the trigger is an instruction, not automation; a hook is a separate follow-up.

## Risks / open questions

- **Continuity path portability.** The `MEMORY.md` is Claude's auto-memory, junctioned into the vault. **Mitigation:** the skill references the vault-junctioned path, writable by any agent; full cross-agent continuity is MEMORY-001 (out of scope, noted).
- **Two-surface drift** (AGENTS.md trigger vs skill detail). **Mitigation:** AGENTS.md + CLAUDE.md carry only a pointer; the checklist lives once, in the skill (verified by AC2).

## Acceptance criteria

- [ ] **AC1 — skill is the SSOT**: `vault/00_meta/skills/handoff/SKILL.md` exists with the ordered checklist (continuity block + vault hygiene + repo/worktree state + artifact summary + next action). **Verify:** grep the steps.
- [ ] **AC2 — trigger + pointer, no duplication**: `AGENTS.md` Phase 3 instructs running `/handoff` at session end and points to the skill; `ai/claude/CLAUDE.md` "Session Handoff" is reduced to a pointer (the 5-field list no longer duplicated there). **Verify:** grep AGENTS.md trigger + assert CLAUDE.md no longer lists the 5 fields.
- [ ] **AC3 — deploys cross-agent, no drift**: after `compile-harness.sh --refresh`, the committed record `harness/skills/handoff/SKILL.md` exists and `--check` exits 0; bats asserts `/handoff` deploys to claude + opencode + agy carrying its checklist. **Verify:** `--check` + `tests/skills-pipeline.bats`.
- [ ] **AC4 — minimal, scoped diff**: `--refresh` changes only the handoff record (plus the hand-authored AGENTS.md/CLAUDE.md edits); no other skill record churns. **Verify:** `git diff --stat`.

## References

- Sibling pattern: `specs/archive/SDD-011-agent-side-spec-trigger/` (two-surface skill + always-on trigger)
- Pipeline: `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` (SDD-008)
- Continuity origin: the former `ai/claude/CLAUDE.md` "Session Handoff (MANDATORY)" rule
- Epic: HARNESS-001 ([#162](https://github.com/mlorentedev/dotfiles/issues/162)); related MEMORY-001 (cross-agent session bridge)
