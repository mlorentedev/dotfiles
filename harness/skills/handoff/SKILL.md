---
id: handoff-skill
type: skill
status: active
created: "2026-05-31"
name: handoff
description: Run a complete, standardized session handoff at session end (or on demand). Triggers on /handoff, "do handoff", "haz handoff", "wrap up the session", "session handoff", "close out the session". Updates the MEMORY.md continuity block, then vault hygiene (tick tasks, capture lessons), repo/worktree/branch state verification, an artifact/PR summary, and a concrete next action. Cross-agent via AGENTS.md indirection.
allowed-tools: [Bash, Read, Edit, Write, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__vault_patch]
---

# Handoff Workflow

> A complete, repeatable session handoff. Any agent, any session: "do handoff" runs the SAME checklist, so the next session (or a different agent) resumes with zero context loss.
> **Core principle:** the handoff is a *checklist*, not just a memory write. The always-on trigger ("at session end, run the handoff") lives in `AGENTS.md`; this skill is the SSOT for WHAT the handoff does.

## When to use

- `/handoff` explicitly, or "do handoff" / "haz handoff" / "wrap up the session".
- Automatically at the END of any session where meaningful work was done (per the `AGENTS.md` trigger).

## When to SKIP

- Trivial session: a quick question, no state change, nothing committed. A handoff would be noise. (Mirrors the `MEMORY.md` "skip if trivial" rule.)

## The handoff checklist (run in order)

### 1. Continuity block — `## Session Handoff` in `MEMORY.md`

OVERWRITE the `## Session Handoff` section (the first section after the H1) of the project's `MEMORY.md` with these fields, in this exact order:

- `> Updated: YYYY-MM-DD`
- `**Last task:** [what was worked on, with PR/commit refs]`
- `**Decisions:** [durable decisions, or "None"]`
- `**Open threads:** [unfinished work + who owns each, or "None"]`
- `**Next action:** [concrete first step for the next session]`

Rules: OVERWRITE entirely (never append); dense but bounded (~8 lines — handoff, not journal); convert relative dates to absolute. **Path:** the project's auto-memory `MEMORY.md`, which is junctioned into `vault/10_projects/<repo>/memory/` — so any agent can write it. This is the only place strategic continuity prose lives (the rest of `MEMORY.md` stays index-only).

### 2. Knowledge & documentation sync (Standing Order #3 — in-session, never "later")

**Vault knowledge:**
- Tick completed tasks in `vault/10_projects/<repo>/11-tasks.md` (`[ ]` -> `[x]` + ✓ date + PR link). Keep the file guard-green (one ticket = one entry — see SDD-012 `check-backlog-integrity.sh`).
- Capture any non-obvious lesson in `90-lessons.md` (Context / Problem / Solution / Tags). New architectural decision -> an ADR; recurring pattern -> `00_meta/patterns/`.

**Repo documentation (keep it reflecting the latest state):**
- If the session changed behavior, structure, commands, public contracts, or setup, update the repo docs that describe them: `README.md` and the repo's `docs/` (ADRs, runbooks, troubleshooting) for repos on the knowledge-placement model. ADRs in this repo live in `docs/adr/`.
- **Scope = this session's deltas, NOT a full audit.** A complete README/docs reconciliation is a dedicated task (e.g. an `AUDIT-*` ticket), not something every handoff redoes — the goal here is simply that docs do not drift behind the code the session just changed.

### 3. Repo / worktree / branch state

- No uncommitted work left dangling — or explicitly named in **Open threads** (never silently).
- Worktrees for MERGED PRs removed; their branches deleted; `git fetch --prune` gone refs.
- Every PR opened this session linked (number + merged/open state) in the handoff.

### 4. Artifact summary

List what the session produced: PRs (number + state), key commit hashes, files created/changed, vault entries added/ticked.

### 5. Next action

One concrete first step for the next session — actionable, not vague ("merge #185 then build X", not "continue").

### 6. Verification (preferred)

Tests/lints green; nothing claimed done-when-not. If something is incomplete, it belongs in **Open threads** with its owner — report outcomes faithfully.

## Cross-OS / cross-agent notes

- Step 1 targets the vault-junctioned `MEMORY.md` path, writable by every agent — so the handoff is portable today, before MEMORY-001 (the cross-agent session bridge) lands. Non-Claude agents write the same path directly.
- Steps 2-6 are agent-agnostic (vault + git).
- Path joining: never hardcode `/` or `\`; use platform-appropriate joining.

## References

- Trigger (always-on): `AGENTS.md` — "at session end, run the handoff".
- Continuity-schema origin: the former Claude-only "Session Handoff (MANDATORY)" rule, now a pointer in `ai/claude/CLAUDE.md` (no duplication).
- Related: `MEMORY-001` (cross-agent session bridge), `00_meta/patterns/pattern-decision-persistence.md`, SDD-011 (the two-surface skill+trigger pattern this mirrors).
