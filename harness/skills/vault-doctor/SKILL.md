---
generated: true
generated_from: 00_meta/skills/vault-doctor/SKILL.md
generated_sha: e0780773a756d819
id: vault-doctor-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: vault-doctor
description: Use when the Obsidian vault needs structural maintenance — unresolved
  links, missing frontmatter, orphan notes, or stale content. Triggers include vault_health
  reporting failures, /insights showing structural warnings, or periodic vault cleanup
  sessions.
keywords: [vault doctor, vault health, unresolved links, orphan notes, reparar vault]
paths: [00_meta/**, .obsidian/**]
requires: [insights]
---
# Vault Doctor

Structural maintenance for the Obsidian knowledge vault. Diagnoses issues, prioritizes by severity, and fixes them systematically.

## Prerequisites

- **Hive MCP** — `vault_health`, `vault_query`, `vault_search`, `vault_patch`, `vault_write`
- **`dotf vault health`** (optional) — 7-section CLI diagnostics including `orphans`, `dead-ends`, `unresolved`, `frontmatter`, `tag hygiene` (requires Obsidian GUI for graph checks)

If Obsidian GUI is not running, use Hive MCP only (headless, always available). `dotf vault health` degrades gracefully when the GUI is unavailable.

## Protocol

### Step 1 — Diagnose

Run diagnostics in parallel:

```bash
# Via Hive MCP (always available)
vault_health(include_usage=True)

# Via dotf CLI (if Obsidian GUI running)
dotf vault health
python3 "$VAULT_PATH/00_meta/scripts/vault-validate.py" --strict
```

Collect:
- Unresolved links (broken wikilinks pointing to non-existent notes)
- Frontmatter violations (files missing `id`, `type`, or `status` fields)
- Orphan notes (no incoming links — isolated content)
- Dead-end notes (no outgoing links — terminal nodes)
- Vault health test failures from Hive MCP

### Step 2 — Prioritize

| Severity | Issue type | Why |
|----------|-----------|-----|
| **HIGH** | Unresolved links | Broken navigation, knowledge gaps |
| **HIGH** | Frontmatter missing `type` or `status` | Breaks vault queries and templates |
| **MEDIUM** | Orphan notes | Lost knowledge, may need connecting or archiving |
| **MEDIUM** | Frontmatter missing `id` | Inconsistency but doesn't break functionality |
| **LOW** | Dead-end notes | May be intentional (leaf content) |

Present a severity report:

```
=== Vault Doctor Diagnosis ===

HIGH:
  - 22 unresolved links (list top 10)
  - 3 files missing type/status frontmatter

MEDIUM:
  - 5 orphan notes
  - 8 files missing id field

LOW:
  - 12 dead-end notes

Recommended action: Fix HIGH first, then MEDIUM. Review LOW manually.
```

### Step 3 — Fix Unresolved Links

For each unresolved link:

1. **Search for candidates** — Use `vault_search` to find notes with similar names
2. **If match found** — Suggest the correct target, apply via `vault_patch`
3. **If ambiguous** — Present options to user, wait for selection
4. **If no match** — The link target was never created:
   - If it's a common concept → suggest creating a stub note
   - If it's stale → suggest removing the link
5. **Never auto-remove** links without user confirmation

### Step 4 — Fix Frontmatter

For files missing required frontmatter fields (`id`, `type`, `status`, `owner`, `created`):

1. **Infer values** from file location and content:
   - `id`: derive from filename slug (e.g., `my-note.md` → `my-note`)
   - `type`: derive from directory (e.g., `30-architecture/` → `adr`, `00_meta/patterns/` → `pattern`). Note: per [[pattern-knowledge-placement]], project ADRs/lessons now live in the repo's `docs/`; these vault types apply only to residual or meta-project files, not new project knowledge.
   - `status`: default to `active` unless content suggests otherwise
   - `owner`: default to `manu`
   - `created`: derive from git history or file date (`"YYYY-MM-DD"`)
2. **Apply in bulk** using `vault_patch` — inject frontmatter at file top
3. **Preserve existing fields** — only add missing ones, never overwrite

**Frontmatter template:**
```yaml
---
id: "<inferred-slug>"
type: <inferred-type>
status: active
owner: manu
created: "YYYY-MM-DD"
tags: []
---
```

### Step 5 — Handle Orphans

For each orphan note (no incoming links):

1. **Check content value** — Is this useful knowledge or stale?
2. **If valuable** — Find related notes via `vault_search` and suggest adding a link from the most relevant note
3. **If stale** — Suggest moving to `90_archive/`
4. **Present list** to user for batch decision: connect, archive, or skip

### Step 6 — Report

```
=== Vault Doctor Report ===

Fixed:
  - 18/22 unresolved links resolved
  - 8 frontmatter fields added across 5 files
  - 3 orphans connected, 2 archived

Remaining (needs manual review):
  - 4 unresolved links (ambiguous targets)
  - 12 dead-end notes (may be intentional)

Vault health: PASS (was: 1 failure)
```

## Rules

- **Never auto-delete content.** Always confirm with user before removing links, archiving notes, or deleting anything.
- **Preserve existing data.** When fixing frontmatter, only add missing fields. Never overwrite existing values.
- **Batch similar fixes.** Group frontmatter fixes by directory for efficiency.
- **Degrade gracefully.** If obs-cli is unavailable, work with Hive MCP only. Report what couldn't be checked.
- **One severity tier at a time.** Fix all HIGH issues before moving to MEDIUM.

## Pipeline

- Previous: `/insights` detects structural issues in its Vault Structural Health section
- Next: Run `vault_health` to verify fixes, then `/insights` to confirm clean state
