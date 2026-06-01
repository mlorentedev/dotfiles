---
name: pattern-loader
description: Use when a task matches a known workflow pattern in vault/00_meta/patterns/. Searches the pattern catalog, loads the most relevant pattern, and applies it. Ensures agents always use the latest version of patterns without caching stale copies in memory.
---

# Pattern Loader

## Overview

When a task matches a known workflow pattern, load it from the vault pattern catalog instead of guessing. This ensures agents always use the latest version of patterns without caching stale copies in memory.

**Core principle:** Patterns live in the vault. Agents load them on demand. Never cache pattern content in long-term memory — always re-read from vault to stay current.

## When to use

- A task matches a known pattern name or category (e.g., "backup", "debug", "spec", "handoff")
- Manu says "use the [pattern-name] pattern"
- A workflow feels familiar but you're not sure of the exact steps
- Starting a session and you want to check which patterns are relevant

## When to skip

- The task is trivial and doesn't need a pattern
- You already know the exact steps from recent memory (e.g., just did this task 2 hours ago)
- No matching pattern exists in the catalog

## The protocol

### Step 1 — Search the pattern catalog

Prefer Hive MCP — it is path-agnostic and always reads the live vault:

```
vault_search(project="_meta", query="<keyword>", scope="patterns")
```

If Hive is unavailable, fall back to a file search against your local vault
clone. The clone path is environment-specific, so resolve it from your
environment instead of hardcoding it (`$VAULT_PATH` on Manu's workstations,
defaulting to `~/Projects/knowledge`; `$HERMES_VAULT_PATH` on the Hermes box):

```bash
grep -rli "<keyword>" "${VAULT_PATH:-$HOME/Projects/knowledge}/00_meta/patterns/" 2>/dev/null
```

### Step 2 — Match and load

Read the most relevant pattern file(s). Each pattern has frontmatter:

```yaml
---
id: pattern-backup
type: pattern
status: active
created: "2026-05-14"
tags: [backup, ops, infrastructure]
---
```

Use the `tags` and `id` to determine relevance. Read the full content.

### Step 3 — Apply the pattern

Follow the pattern's steps exactly. If the pattern has sections like:

- **Overview** — context and intent
- **When to use** — trigger conditions
- **Steps** — numbered procedure
- **Pitfalls** — common mistakes to avoid
- **Verification** — how to confirm success

Follow all sections in order. Do not skip pitfalls.

### Step 4 — Log which pattern was used

Add a brief note to the current session record:

```
Pattern applied: pattern-<name>.md
```

This is for audit trail — not for long-term memory.

### Step 5 — Cache only if frequent

If the same pattern is used >1x/week, cache it in memory (not in vault). Otherwise, always re-read from vault.

## Pattern categories

The pattern catalog is organized by category. Use these as search hints:

| Category | What it covers | Examples |
|----------|---------------|----------|
| `pattern-backup` | Backup procedures, restore, verification | Full backup, selective restore, test-restore |
| `pattern-debug-*` | Debugging methodologies | Systematic debugging, hardware debug, session debug |
| `pattern-spec-*` | Spec-Driven Development | SDD flow, spec init, spec archive, adversarial review |
| `pattern-git-*` | Git workflows | Worktrees, clean history, safe force |
| `pattern-security-*` | Security procedures | Credential handling, audit, threat modeling |
| `pattern-decision-*` | Decision persistence | ADR format, decision logging, crystallization |
| `pattern-agent-*` | Agent workflows | Handoff, delegation, session start, pattern loader |
| `pattern-dev-*` | Development workflows | TDD, test-driven, planning, execution |
| `pattern-ops-*` | Operations | Server setup, monitoring, incident response |

## Pitfalls

- **Don't guess the pattern name** — search first, match second
- **Don't cache patterns in long-term memory** — they change, cached versions become stale
- **Don't skip pitfalls** — they exist for a reason
- **Don't apply a pattern that doesn't fully match** — partial application causes bugs
- **Don't assume a pattern exists** — if no match, say so and proceed without it

## Verification

After applying a pattern:

1. Re-read the pattern's **Verification** section
2. Execute the verification steps
3. Report results to Manu
4. If verification fails, log it as a lesson in `clients/` or `sessions/`

## References

- Pattern catalog: `vault/00_meta/patterns/`
- Pattern format: see any file in `00_meta/patterns/pattern-*.md`
- Related: `00_meta/skills/verification-before-completion/SKILL.md` (always verify after applying)
