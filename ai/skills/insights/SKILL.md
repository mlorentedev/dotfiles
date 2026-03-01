---
name: insights
description: Read-only weekly audit of the AI knowledge workflow. Checks MEMORY.md health, counts unvaulted observations by type, identifies gaps vs vault lessons, and reports what needs attention. Run weekly; use /crystallize when this shows issues.
---

# /insights — AI Workflow Health Audit

Quick, read-only audit of the Neural Hive knowledge loop state. No files modified.

## Protocol

### Step 1 — MEMORY.md Health
- Read `~/.claude/projects/<encoded-cwd>/memory/MEMORY.md`
- Report: line count, Last Crystallized date, days since last crystallization
- Flag: lines > 150 → WARNING, Last Crystallized > 14 days → WARNING

### Step 2 — Observation Inventory
- Run `/mem-search` to count recent observations by type:
  - 🔵 Discovery (informational, low urgency)
  - ✅ Change (recorded, low urgency)
  - 🔴 Bugfix (should be vaulted)
  - ⚖️ Decision (should be vaulted)
  - 🟣 Feature (notable, may need vaulting)
- Show counts for the past 14 days

### Step 3 — Vault Gap Analysis
- Read `~/Projects/knowledge/10_projects/<repo>/90-lessons.md`
- Identify 🔴 and ⚖️ observations NOT yet documented in vault lessons
- List each gap: ID, type, title

### Step 4 — Pattern Opportunities
- For lessons in `90-lessons.md` that appear relevant to multiple projects:
  - Check `~/Projects/knowledge/00_meta/patterns/` for existing patterns
  - Identify lessons that warrant a new global pattern

### Step 5 — Backlog Snapshot
- Read `~/Projects/knowledge/10_projects/<repo>/11-tasks.md`
- Report: active items count, progress bar, any overdue P0/P1 items

### Step 6 — Report

Print a structured summary:

```
=== AI Workflow Insights ===

MEMORY.md health:
  Lines: X / 150 [OK|WARNING]
  Last Crystallized: YYYY-MM-DD (N days ago) [OK|WARNING]

Observation inventory (last 14 days):
  🔵 Discoveries: N
  ✅ Changes: N
  🔴 Bugfixes: N (X unvaulted → ACTION NEEDED)
  ⚖️ Decisions: N (X unvaulted → ACTION NEEDED)
  🟣 Features: N

Vault gaps (🔴/⚖️ not in 90-lessons.md):
  - #ID: <title>
  - ...

Pattern opportunities:
  - <lesson title> → consider pattern-<topic>.md

Backlog: X active items (Progress: [====......] 40%)

Recommendation: [No action needed | Run /crystallize]
```

## When to Use

Run **weekly** as a maintenance habit. Takes ~2 minutes.

If the report shows:
- 0 unvaulted 🔴/⚖️ obs AND MEMORY.md healthy → no action needed
- Any warnings → run `/crystallize` to resolve

## Difference from /crystallize

`/insights` is **read-only** — audit and report only, nothing written.
`/crystallize` is **write mode** — executes the full maintenance ritual.
