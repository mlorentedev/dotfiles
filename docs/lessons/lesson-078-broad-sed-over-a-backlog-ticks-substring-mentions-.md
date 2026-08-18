---
id: lesson-078-broad-sed-over-a-backlog-ticks-substring-mentions-
type: lesson
status: active
created: "2026-06-02"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 078: Broad sed over a backlog ticks substring mentions, not just the entry — anchor to the line-start id

**Context:** Reconciling the vault backlog (11-tasks.md) during a drift cleanup; used sed to tick the shipped IDEAS-007 entry: sed 's/^- \[ \] \(.*IDEAS-007\)/- [x] \1/'.
**Problem:** The pattern matched ANY '- [ ]' line CONTAINING "IDEAS-007", not just its own entry. It also ticked the HARNESS-001 entry (its body references IDEAS-007) and — wrongly — REFACTOR-008, a deferred-with-trigger YAGNI item whose body merely cites the IDEAS-007 reconciliation doc. One factually-incorrect tick (REFACTOR-008 is NOT done). The SDD-012b merged-but-open guard caught the IDEAS-007 case but not the over-ticks.
**Solution:** Anchor backlog seds to the entry's own id at line start — '^- \[ \] \*\*<ID>' — never a bare substring that can appear in prose. ALWAYS verify with `git diff | grep '^[+-]- \['` to see every checkbox the command flipped BEFORE committing, and re-run the integrity/merged-open guard after. Broad find/replace on a human-curated list that mentions ids in prose is a footgun.
**Tags:** `#backlog` `#sed` `#reconciliation` `#footgun` `#verify-the-diff` `#red-team-thyself`
