---
name: context-refresh
targets: [claude]
description: Use after a brainstorm / ADR / phase-closing session to refresh a project's 00-context.md patchable fields (phase, focus, blocked_by, recent_adrs, last_updated) WITHOUT touching the stable body. Keeps 00-context.md cheap for agent orientation at session start (HARNESS-006). Triggers on /context-refresh, "actualiza el contexto", "refresh project context", or automatically from /handoff when an ADR was written or a phase milestone closed this session. Do NOT use for backlog/task changes — that state lives in the GitHub Project "bitácora", not here.
---

# /context-refresh — Project Context Patcher

Keeps `00-context.md` current and cheap for **agent orientation at session start** (HARNESS-006:
an agent reads it in <400 tokens and learns WHAT / WHERE / PHASE / FOCUS). This skill patches only
the five frontmatter *patchable* fields; the stable body is human-owned and is never rewritten.

## Scope boundary (read first)

- **Patchable (this skill writes):** frontmatter `phase`, `focus`, `blocked_by`, `recent_adrs`, `last_updated`.
- **Stable (this skill NEVER writes):** everything below the frontmatter — Vision, Strategic direction,
  Where things live, Stack. If those are wrong, that is a human edit, not this skill.
- **Out of layer (never here):** task backlog → GitHub Project "bitácora"; lessons → repo `docs/lessons.md`;
  ADR bodies → repo `docs/adr/`. This skill records *state and pointers*, not content. See [[pattern-knowledge-placement]].

## Protocol

### Step 1 — Locate
- Resolve the project: explicit arg (`/context-refresh kubelab`), else the repo of the current working
  directory → `10_projects/<project>/00-context.md`.
- If the file has no patchable frontmatter block (pre-HARNESS-006 format), STOP and offer to migrate it
  to `00_meta/templates/project-context.md` first — do not patch a legacy file.

### Step 2 — Read patchable state only
- Read just the frontmatter patchable keys. Do not load the stable body (it is not needed to compute deltas).

### Step 3 — Extract deltas from the session
From the work actually done this session, decide for each field (leave unchanged if no signal):
- **phase** — did the project move to a new phase/theme? (milestone closed, pivot, new stream opened)
- **focus** — what is now the active next concrete thing being worked?
- **blocked_by** — was a blocker added or cleared? (set `""` when nothing blocks)
- **recent_adrs** — was an ADR written this session? Prepend its slug; keep the last 1-3.

### Step 4 — Patch frontmatter only
- For each changed field, patch that single frontmatter key via Hive `vault_patch`. Never touch the body.
- Always set `last_updated` to today's date.

### Step 5 — Verify
- Re-read the frontmatter. Confirm: only patchable keys changed, the stable body is byte-identical,
  the file still parses, and the whole file stays ~25-30 lines / <400 tokens.

### Step 6 — Report
```
Context refreshed: <project>
  phase:        <old> -> <new>   (or "unchanged")
  focus:        <old> -> <new>
  blocked_by:   <old> -> <new>
  recent_adrs:  <list>
  last_updated: <today>
```

## When to use

| Trigger | Action |
|---------|--------|
| `/context-refresh [project]` | Run the protocol |
| "actualiza el contexto" / "refresh project context" | Run the protocol |
| ADR written this session | `/handoff` calls this automatically (optional step) |
| Phase milestone closed / pivot this session | `/handoff` calls this automatically (optional step) |
| Only backlog/task state changed | Do NOT run — that is bitácora state, not context |

## Notes
- **Idempotent:** re-running with no session deltas only bumps `last_updated`.
- **Cross-agent / headless:** patches via Hive `vault_patch`, so it works without the Obsidian GUI.
  Vault edits auto-commit to master — no branch/PR (see [[feedback_vault_commit_to_master]]).
- Pairs with [[pattern-knowledge-placement]] and the bitácora flow: this file is the cheap orientation
  layer; the forge holds the task state; the repo holds the durable docs.
