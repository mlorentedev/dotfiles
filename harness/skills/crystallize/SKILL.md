---
generated: true
generated_from: 00_meta/skills/crystallize/SKILL.md
generated_sha: d6d6cf0c738eab7d
id: crystallize-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: crystallize
targets: [claude]
description: Use when /insights shows unvaulted observations, stale MEMORY.md, or
  after completing a significant sprint. Addresses knowledge gaps between session
  observations and vault lessons.
keywords: [crystallize, cristalizar, promote lesson, pattern promotion, leccion aprendida]
paths: [00_meta/patterns/**, docs/lessons/**]
---
# /crystallize — Knowledge Crystallization Ritual

Full maintenance ritual for the Neural Hive knowledge loop. Run when `/insights` shows issues or after a major sprint.

## Protocol

### Step 1 — Audit State
- Read the project's `memory/MEMORY.md` (agent auto-memory store)
- Note: line count, Last Crystallized date, days elapsed since last run
- Read the repo's `docs/lessons/` and `docs/lessons/_index.md` for current **project** lesson state (project lessons live in the repo — see [[pattern-knowledge-placement]]); read `$VAULT_PATH/00_meta/patterns/` for cross-project state

### Step 2 — Mine Observations
- Run `/mem-search` filtering for 🔴 (bugfix) and ⚖️ (decision) type observations from the past 14 days
- List each unvaulted observation with its ID and title

### Step 3 — Gap Detection
- Compare mined observations against the repo's `docs/lessons/` (project lessons) and `00_meta/` (cross-project)
- Identify which bugs/decisions are NOT yet documented in the vault
- These become new lesson candidates

### Step 4 — Vault Update
- For each gap, write a **project** lesson to the repo's `docs/lessons/lesson-NNN-<slug>.md` and register in `docs/lessons/_index.md` using the standard frontmatter (a genuinely **cross-project** lesson goes to `00_meta/patterns/`):

```markdown
---
id: "lesson-NNN-<slug>"
type: lesson
status: active
created: "YYYY-MM-DD"
owner: manu
tags: [tag1, tag2]
---

# <Title>

**Context:** <what you were doing when you hit this>
**Problem:** <what went wrong or what decision was needed>
**Solution:** <what fixed it or what was decided>
**Why:** <root cause or rationale>
```

### Step 5 — Pattern Promotion
- For each new lesson, ask: "Does this appear in >1 project?"
- If YES → check `$VAULT_PATH/00_meta/patterns/`
- If no matching pattern file exists → propose creating `pattern-<topic>.md`

### Step 6 — MEMORY.md Trim
- Remove any sections from MEMORY.md that are already covered verbatim by `.claude/CLAUDE.md`
- Keep: User Preferences, CI Pipeline, Bugs Found, Backlog Status, Last Crystallized
- Update `## Last Crystallized: YYYY-MM-DD` to today

### Step 7 — Stamp
- Run: `dotf vault crystallize` (add `--all` to stamp every project)
- This updates currentDate and Last Crystallized automatically

### Step 8 — Report
Print a summary:
```
Crystallization complete:
  - X project lessons added to repo docs/lessons/ (cross-project -> 00_meta/patterns/)
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
