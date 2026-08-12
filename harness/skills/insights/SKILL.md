---
generated: true
generated_from: 00_meta/skills/insights/SKILL.md
generated_sha: 3dacc3046a58a23c
id: insights-skill
type: skill
status: active
created: "2026-05-30"
owner: manu
name: insights
targets: [claude]
description: Use when checking AI workflow health, vault structural integrity, or knowledge pipeline status. Run weekly as maintenance habit. Triggers include stale MEMORY.md, unvaulted observations, vault structural issues, or before starting a major sprint.
---

# /insights -- AI Workflow Health Audit

Quick, read-only audit of the Neural Hive knowledge loop and vault structural health. No files modified.

## Modes

- **Quick** (default): MEMORY.md health + vault structural health + backlog snapshot. ~2 minutes.
- **Full** (`/insights full`): Everything in quick + observation inventory + vault gap analysis + decision persistence + pattern coverage. ~5 minutes.

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

### Step 4 -- Observation Inventory (full mode)

- Run `mem-search` to count recent observations by type (past 14 days):
  - Discovery, Change, Bugfix, Decision, Feature
- Show counts and highlight unvaulted bugfixes and decisions

### Step 5 -- Vault Gap Analysis (full mode)

- Read the repo's `docs/lessons.md` (project lessons live in the repo — see [[pattern-knowledge-placement]]); read `$VAULT_PATH/00_meta/` for cross-project
- Identify bugfix and decision observations NOT documented in the repo's lessons
- List each gap: ID, type, title

### Step 6 -- Decision Persistence Check (full mode)

- Search recent session observations for decisions
- For each decision, verify it was written to the affected vault file (ADR, context, tasks)
- Flag unpersisted decisions:

```
Decision persistence:
  3 decisions this sprint
  - #14631 "Decision Persistence Pattern" -> PERSISTED (pattern-decision-persistence.md)
  - #15064 "AI Agent Instruction Updates" -> NOT PERSISTED -> ACTION NEEDED
```

### Step 7 -- Pattern Opportunities (full mode)

- For lessons in the repo's `docs/lessons.md` that appear relevant to multiple projects:
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
Observation inventory (last 14 days):
  Discoveries: N | Changes: N | Bugfixes: N (X unvaulted) | Decisions: N (X unvaulted) | Features: N

Project-lesson gaps (not in the repo's docs/lessons.md):
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
