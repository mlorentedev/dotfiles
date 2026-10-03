---
generated: true
generated_from: 00_meta/skills/insights/SKILL.md
generated_sha: d221e6a0da05e84f
id: insights-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: insights
targets: [claude]
description: Use when checking AI workflow health, vault structural integrity, or
  knowledge pipeline status. Run weekly as maintenance habit. Triggers include stale
  MEMORY.md, unpersisted decisions, vault structural issues, or before starting a
  major sprint.
keywords: [insights, vault insights, graph insights, knowledge stats]
paths: [00_meta/**]
---
# /insights -- AI Workflow Health Audit

Quick, read-only audit of the Neural Hive knowledge loop and vault structural health. No files modified.

## Modes

- **Quick** (default): MEMORY.md health + vault structural health + backlog snapshot. ~2 minutes.
- **Full** (`/insights full`): Everything in quick + activity inventory + vault gap analysis + decision persistence + pattern coverage. ~5 minutes.

## Protocol

### Step 1 -- MEMORY.md Health

- Read the project's `memory/MEMORY.md` (agent auto-memory store)
- Report: line count, Last Crystallized date, days since last crystallization
- Flag: lines > 150 -> WARNING, Last Crystallized > 14 days -> WARNING

### Step 2 -- Vault Structural Health

Run `vault_health(include_usage=True)` via Hive MCP.

If Obsidian GUI is running, also run:
```bash
obs-cli.sh unresolved    # Broken wikilinks
obs-cli.sh orphans       # No incoming links
```

Report:
```
Vault structural health:
  Health tests: N passed, M failed
  Unresolved links: X [OK|ACTION NEEDED]
  Orphan notes: Y [OK|ACTION NEEDED]
  Frontmatter compliance: Z% [OK|WARNING]
```

If failures detected -> recommend `/vault-doctor`.

### Step 3 -- Backlog Snapshot

- Query open issues on the bitácora GitHub Project via `gh project item-list` or `gh issue list --state open`
- Report: active items count, status distribution, any blocked items

### -- Quick mode stops here --

### Step 4 -- Activity Inventory (full mode)

There is no observation store. The continuity layer is the vault itself (see [[pattern-dual-memory]]), so mine the last 14 days from its history:

```bash
V="$VAULT_PATH"
git -C "$V" log --since="14 days ago" --oneline | wc -l                       # activity volume
git -C "$V" log --since="14 days ago" --name-only --pretty=format: \
  | grep -E '/sessions/' | sort -u                                           # session journals
git -C "$V" log --since="14 days ago" --name-only --pretty=format: \
  | grep -E '90-lessons|00_meta/lessons/|/decisions/|adr-' | sort -u          # lessons + decisions written
SINCE=$(date -d '14 days ago' +%F)                                           # handoff decisions in the window:
awk -v s="$SINCE" 'FNR==1{d=""} /^> Updated: /{d=$3} /^\*\*Decisions:\*\*/ && d>=s' \
  "$V"/10_projects/*/memory/MEMORY.md                                         # only threads whose Updated date is inside it
```

- Report: commits, session journals, lessons and decision artifacts touched, and handoff decisions found.
- Any other source this step reads must first prove it is alive (its newest record is inside the window). Otherwise report it as **unavailable**, never as zero (lesson-009).

### Step 5 -- Vault Gap Analysis (full mode)

- Read the repo's `docs/lessons/` (and `docs/lessons/_index.md`) (project lessons live in the repo — see [[pattern-knowledge-placement]]); read `$VAULT_PATH/00_meta/` for cross-project
- Identify project-specific fixes and decisions named in session journals or handoff blocks that are NOT documented in the repo's lessons. Skip cross-project or methodology items already captured in `00_meta/` (patterns, lessons).
- List each gap: source (journal or thread), type, title

### Step 6 -- Decision Persistence Check (full mode)

- For each handoff decision from Step 4, verify it was written to the affected artifact (repo ADR, context, pattern), **in the scope of every consumer**. A decision about a shared component that lives in only one project's memory is NOT PERSISTED ([[pattern-decision-persistence]]).

```
Decision persistence:
  3 decisions this sprint
  - #14631 "Decision Persistence Pattern" -> PERSISTED (pattern-decision-persistence.md)
  - #15064 "AI Agent Instruction Updates" -> NOT PERSISTED -> ACTION NEEDED
```

### Step 7 -- Pattern Opportunities (full mode)

- For lessons in the repo's `docs/lessons/` that appear relevant to multiple projects:
  - Check `$VAULT_PATH/00_meta/patterns/` for existing patterns
  - Identify lessons that warrant a new global pattern
- For existing patterns: check if recent lessons should be added to them

### Step 8 -- Report

```
=== AI Workflow Insights ===

MEMORY.md health:
  Lines: X / 150 [OK|WARNING]
  Last Crystallized: YYYY-MM-DD (N days ago) [OK|WARNING]

Vault structural health:
  Health tests: N passed, M failed [OK|ACTION NEEDED]
  Unresolved links: X [OK|ACTION NEEDED]
  Orphan notes: Y [OK|WARNING]

Backlog: X active items (Progress: [====......] 40%)

[Full mode only:]
Activity inventory (last 14 days):
  Commits: N | Session journals: N | Lessons/decisions written: N | Handoff decisions: N (X unpersisted)

Project-lesson gaps (not in the repo's docs/lessons/):
  - #ID: <title>

Decision persistence:
  X/Y decisions persisted [OK|ACTION NEEDED]

Pattern opportunities:
  - <lesson title> -> consider pattern-<topic>.md

Recommendation: [No action needed | Run /crystallize | Run /vault-doctor | Run both]
```

## When to Use

Run **weekly** as a maintenance habit (quick mode). Run **full mode** before sprints or after major work.

| Trigger | Mode |
|---------|------|
| Weekly check-in | Quick |
| Before starting a major sprint | Full |
| After completing a major sprint | Full |
| SessionStart shows vault health warnings | Quick |
| Preparing for `/crystallize` | Full |

## Pipeline

- This skill is **read-only** -- it detects issues, never fixes them.
- When issues found -> `/crystallize` (knowledge gaps) or `/vault-doctor` (structural issues)
