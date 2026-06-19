---
id: handoff-skill
type: skill
status: active
created: "2026-05-31"
name: handoff
description: Run a complete, standardized session handoff at session end (or on demand). Triggers on /handoff, "do handoff", "haz handoff", "wrap up the session", "session handoff", "close out the session". Updates the MEMORY.md continuity block, then vault hygiene (capture lessons), bitácora board-status reconciliation, repo/worktree/branch state verification, an artifact/PR summary, and a concrete next action. Cross-agent via AGENTS.md indirection.
allowed-tools: [Bash, Read, Edit, Write, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_write, mcp__hive__vault_patch]
---

# Handoff Workflow

> A complete, repeatable session handoff. Any agent, any session: "do handoff" runs the SAME checklist, so the next session (or a different agent) resumes with zero context loss.
> **Core principle:** the handoff is a *checklist*, not just a memory write. The always-on trigger ("at session end, run the handoff") lives in `AGENTS.md`; this skill is the SSOT for WHAT the handoff does.

## When to use

- `/handoff` explicitly, or "do handoff" / "haz handoff" / "wrap up the session".
- Automatically at the END of any session where meaningful work was done (per the `AGENTS.md` trigger).

## When to SKIP

Skip when the session left no durable state worth carrying forward — any of:
- A quick question or explanation; no files changed.
- A syntax / docs lookup, or a single-line / trivial edit with nothing committed.
- No new decision, no open thread, no PR or issue touched.

If in doubt, run it — a thin handoff is cheaper than a lost thread. (Mirrors the `MEMORY.md` "skip if trivial" rule.)

## The handoff checklist (run in order)

### 1. Continuity block — `## Session Handoff` in `MEMORY.md`

OVERWRITE the `## Session Handoff` section (the first section after the H1) of the project's `MEMORY.md` with these fields, in this exact order:

- `> Updated: YYYY-MM-DD`
- `**Last task:** [what was worked on, with PR/commit refs]`
- `**Decisions:** [durable decisions, or "None"]`
- `**Open threads:** [unfinished work + who owns each, or "None"]`
- `**Next action:** [concrete first step for the next session]`

Rules: OVERWRITE entirely (never append); dense but bounded (~8 lines — handoff, not journal); convert relative dates to absolute. **End each handoff field line with two trailing spaces** (markdown hard break) so the fields render as separate lines in Obsidian, not one crammed paragraph — vault-only convention (do NOT carry trailing spaces into code repos; pre-commit strips them). **Path:** the project's auto-memory `MEMORY.md`, which is junctioned into `vault/10_projects/<repo>/memory/` — so any agent can write it. This is the only place strategic continuity prose lives (the rest of `MEMORY.md` stays index-only).

**Lifecycle (prune, don't accumulate):** there is exactly ONE `## Session Handoff` block — the first section after the H1. Overwrite it in place every time; never add a second handoff block and never stack per-session blocks. Durable context that must outlive a single session goes in its OWN named section *below* the handoff (e.g. `## Older context`), not in the handoff block.

### 2. Knowledge & documentation sync (Standing Order #3 — in-session, never "later")

**Vault knowledge:**
- **Task state lives on the bitácora board, not in the vault** (ADR-018) — reconcile it in step 2b, not here. (Legacy `11-tasks.md` files are retired; the few that survive are not the source of truth, so don't tick them as if they were.)
- Capture any non-obvious **project** lesson in the **repo's `docs/lessons.md`** (Context / Problem / Solution / Tags) — NOT a vault `90-lessons.md` (see [[pattern-knowledge-placement]]: build/operate knowledge lives in the repo). A genuinely **cross-project / methodology** lesson goes to `00_meta/` (promote to a pattern). New architectural decision -> repo `docs/adr/`; recurring cross-project pattern -> `00_meta/patterns/`.

**Repo documentation (keep it reflecting the latest state):**
- If the session changed behavior, structure, commands, public contracts, or setup, update the repo docs that describe them: `README.md` and the repo's `docs/` (ADRs, runbooks, troubleshooting) for repos on the knowledge-placement model. ADRs in this repo live in `docs/adr/`.
- **Scope = this session's deltas, NOT a full audit.** A complete README/docs reconciliation is a dedicated task (e.g. an `AUDIT-*` ticket), not something every handoff redoes — the goal here is simply that docs do not drift behind the code the session just changed.

### 2b. Bitácora status reconciliation (Standing Order #8 — HARNESS-010)

Every issue you touched this session must reflect reality on the board ([Project #1](https://github.com/users/mlorentedev/projects/1)) — **none left in `Backlog` while actively worked**:

- **Worked, still open →** `In Progress` (self-assigning fires `bitacora-status.yml`; if you never assigned it, self-assign now or set the field by hand) — or `Blocked` with the blocker named in an issue comment.
- **Finished, closed →** the built-in workflow sets `Done` on close; confirm it landed.

Quick audit of this session's issue numbers (`N1,N2,…`):

```bash
gh project item-list 1 --owner mlorentedev --format json --limit 300 \
  | python3 -c "import json,sys; W={N1,N2}; [print(i.get('status'),'·',i.get('content',{}).get('title','')) \
       for i in json.load(sys.stdin)['items'] if i.get('content',{}).get('number') in W]"
```

Leaving an actively-worked issue in `Backlog` is the exact gap HARNESS-010 closes. Mechanics: `dotfiles/docs/runbooks/guide-bitacora-setup.md` §5.

**Best-effort, never a blocker:** if the `gh project` query is unavailable (auth, permissions, or a sandbox denial), do NOT stall the handoff. Instead make sure each touched issue is at least self-assigned or carries a status comment, and record the board change you couldn't apply in **Open threads** so the next session (or Manu) finishes it.

### 3. Repo / worktree / branch state

- No uncommitted work left dangling — or explicitly named in **Open threads** (never silently).
- Worktrees for MERGED PRs removed; their branches deleted; `git fetch --prune` gone refs.
- Every PR opened this session linked (number + merged/open state) in the handoff.

### 3b. Housekeeping (active cleanup — run for every repo touched this session)

Remove transient clutter before closing:

- **Merged branches (local):** `git branch --merged main | grep -vE '^\*|main|master' | xargs -r git branch -d`. Inspect output — do not force-delete (`-D`) without reading why a branch is unmerged.
- **Remote gone refs:** `git fetch --prune` on every repo touched. Removes stale remote-tracking refs for branches deleted upstream.
- **Done worktrees:** any worktree whose PR is merged must be removed (`git worktree remove <path>`). If not yet merged, name it in **Open threads** with PR number.
- **Temp / scratch files:** inspect `git status` for untracked `.bak`, `*.tmp`, scratch notes. **Never delete a file without explicit user confirmation** — list the candidates and ask; never `git clean -f`.
- **Empty vault stubs:** if any vault file (e.g. `90-lessons.md`) is frontmatter-only with no real content, flag it (with its inbound links) and confirm before removing. Scope: only paths touched this session.

Scope: only repos and vault paths actually touched in this session — not a global cleanup pass.

### 3c. Context refresh (conditional — `/context-refresh`)

If this session **wrote an ADR, closed a phase milestone, pivoted direction, or changed the active focus**, run `/context-refresh <project>` for each affected project. It patches only the `00-context.md` frontmatter (`phase`, `focus`, `blocked_by`, `recent_adrs`, `last_updated`) so the next session orients in <400 tokens, and never touches the stable body. **Skip** when the session only changed task state — that lives in the bitácora, not `00-context.md`. See [[context-refresh]] (HARNESS-006).

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
- **If the junction/symlink is missing** (fresh machine, or Windows where the link is a junction, not a symlink): write the continuity block at the auto-memory's REAL location directly — `~/.claude/projects/<project-hash>/memory/MEMORY.md` (POSIX) or `%USERPROFILE%\.claude\projects\<project-hash>\memory\MEMORY.md` (Windows) — then (re)create the junction afterward. The continuity write must never be skipped just because the link is absent.

## References

- Trigger (always-on): `AGENTS.md` — "at session end, run the handoff".
- Continuity-schema origin: the former Claude-only "Session Handoff (MANDATORY)" rule, now a pointer in `ai/claude/CLAUDE.md` (no duplication).
- Related: `MEMORY-001` (cross-agent session bridge), `00_meta/patterns/pattern-decision-persistence.md`, SDD-011 (the two-surface skill+trigger pattern this mirrors).
