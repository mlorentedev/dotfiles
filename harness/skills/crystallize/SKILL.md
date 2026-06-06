---
name: crystallize
targets: [claude]
description: Use when /insights shows unvaulted observations, stale MEMORY.md, or after completing a significant sprint. Addresses knowledge gaps between session observations and vault lessons.
---

# /crystallize — Knowledge Crystallization Ritual

Full maintenance ritual for the Neural Hive knowledge loop. Run when `/insights` shows issues or after a major sprint.

## Protocol

### Step 1 — Audit State
- Read `~/.claude/projects/<encoded-cwd>/memory/MEMORY.md`
- Note: line count, Last Crystallized date, days elapsed since last run
- Read the repo's `docs/lessons.md` for current **project** lesson state (project lessons live in the repo — see [[pattern-knowledge-placement]]); read `~/Projects/knowledge/00_meta/patterns/` for cross-project state

### Step 2 — Mine Observations
- Run `/mem-search` filtering for 🔴 (bugfix) and ⚖️ (decision) type observations from the past 14 days
- List each unvaulted observation with its ID and title

### Step 3 — Gap Detection
- Compare mined observations against the repo's `docs/lessons.md` (project lessons) and `00_meta/` (cross-project)
- Identify which bugs/decisions are NOT yet documented in the vault
- These become new lesson candidates

### Step 4 — Vault Update
- For each gap, append a **project** lesson to the repo's `docs/lessons.md` using the **Lesson Template** (a genuinely **cross-project** lesson goes to `00_meta/`, promoted to a pattern — never a vault per-project `90-lessons.md`):

```markdown
### [YYYY-MM-DD] <Title>
**Context:** <what you were doing when you hit this>
**Problem:** <what went wrong or what decision was needed>
**Solution:** <what fixed it or what was decided>
**Why:** <root cause or rationale>
**Tags:** `#tag1` `#tag2`
```

### Step 5 — Pattern Promotion
- For each new lesson, ask: "Does this appear in >1 project?"
- If YES → check `~/Projects/knowledge/00_meta/patterns/`
- If no matching pattern file exists → propose creating `pattern-<topic>.md`

### Step 6 — MEMORY.md Trim
- Remove any sections from MEMORY.md that are already covered verbatim by `.claude/CLAUDE.md`
- Keep: User Preferences, CI Pipeline, Bugs Found, Backlog Status, Last Crystallized
- Update `## Last Crystallized: YYYY-MM-DD` to today

### Step 7 — Stamp
- Run: `./scripts/knowledge-crystallize.sh`
- This updates currentDate and Last Crystallized automatically

### Step 8 — Report
Print a summary:
```
Crystallization complete:
  - X project lessons added to the repo's docs/lessons.md (cross-project -> 00_meta/)
  - Y pattern proposals (list titles)
  - MEMORY.md: N lines before → M lines after
  - Last Crystallized: <today>
```

## When to Use

| Trigger | Action |
|---------|--------|
| `/insights` shows unvaulted 🔴/⚖️ obs | Run `/crystallize` |
| MEMORY.md > 150 lines | Run `/crystallize` |
| Last Crystallized > 14 days ago | Run `/crystallize` |
| After completing a major sprint | Run `/crystallize` |

## Difference from /insights

`/insights` is **read-only** — it tells you what needs attention.
`/crystallize` is **write mode** — it actually fixes the issues.

Run `/insights` weekly. Run `/crystallize` when insights shows problems.
