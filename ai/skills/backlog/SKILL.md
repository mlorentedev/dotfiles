---
name: backlog
description: Update and consolidate project backlog. Use when reviewing, reorganizing, or triaging BACKLOG.md, tasks/todo.md, or TODO files across the repo.
---

# Backlog Management

Consolidate and organize project backlog items across all tracking files.

## Process

1. **Discover** — Read `BACKLOG.md`, `tasks/todo.md`, and any TODO/FIXME markers in code
2. **Inventory** — List all items found, grouped by status (pending, in-progress, done, deprioritized)
3. **Confirm** — ASK the user before removing, merging, or reorganizing any existing items
4. **Update** — Apply changes. Never delete items — mark as completed or deprioritized
5. **Review** — Present proposed changes as a diff summary before writing

## Output Format

```markdown
## Backlog Summary

### Active (N items)
- [ ] Item description [source: file]

### Completed (N items)
- [x] Item description

### Deprioritized (N items)
- [~] Item description — reason
```

## Rules

- Preserve all existing items unless user explicitly approves removal
- Deduplicate items that appear in multiple files — keep the most detailed version
- Maintain original priority ordering unless user requests re-triage
