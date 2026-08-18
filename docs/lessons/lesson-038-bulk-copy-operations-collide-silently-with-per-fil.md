---
id: lesson-038-bulk-copy-operations-collide-silently-with-per-fil
type: lesson
status: active
created: "2026-05-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 038: Bulk-copy operations collide silently with per-file deploy logic

**Context:** SDD-002 (PR #51) introduced a per-file deploy for `ai/claude/settings.json`: read template, substitute `__HOOK_COMMAND__` placeholder, merge with existing target using per-key policy. Both setup scripts also had a pre-existing bulk-copy `Copy-Item ai/claude/* ~/.claude/` (PowerShell) / `cp -rf ai/claude/* ~/.claude/` (bash) that would copy ALL files from the source dir, including the new `settings.json` template -- which contained the literal `__HOOK_COMMAND__` placeholder AND would have wiped the user's customizations.

**Problem:** The bulk-copy + per-file collision is invisible until the per-file logic introduces a placeholder or merge invariant that the bulk-copy can't honor. While both deploys produce "a file at the target", the bulk-copy version produces semantically wrong content (placeholder unsubstituted, user customizations lost). No error fires because both operations succeed at the filesystem level. The bug surfaces only when the user opens the deployed file and finds garbage, OR when downstream logic chokes on the placeholder.

**Solution:** When introducing per-file deploy semantics for a file previously covered by a bulk-copy, ALWAYS add an explicit exclusion to the bulk-copy at the same time. PowerShell: `Copy-Item ... -Exclude 'settings.json'`. POSIX bash: explicit loop with `[ "$(basename "$src")" = "settings.json" ] && continue`. Document the exclusion next to the bulk-copy with the reason ("handled by per-file logic in <function>"). A bats parity assert that grep-checks for the exclusion locks it in.

**Rule:** Per-file deploy + bulk-copy of the parent dir is a guaranteed collision. The first-PR-after-refactor often misses it because the symptom is "file exists at target, looks OK on glance". Pair every new per-file deploy with the exclusion edit in the same atomic PR. Generalizes beyond settings.json: skills, MCP configs, opencode commands, anywhere a curated subset of a directory needs per-file handling within an otherwise-bulk deploy.

**Tags:** `#setup-scripts` `#deploy` `#bulk-copy` `#collision` `#parity` `#shell` `#powershell`
