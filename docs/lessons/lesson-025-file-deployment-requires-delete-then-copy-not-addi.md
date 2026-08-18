---
id: lesson-025-file-deployment-requires-delete-then-copy-not-addi
type: lesson
status: active
created: "2026-03-29"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 025: File deployment requires delete-then-copy, not additive-only copy

**Context**: Skills ecosystem overhaul deleted 9 skill directories from the dotfiles repo. Setup scripts (`setup-linux.sh`, `setup-windows.ps1`) deployed skills by copying source to destination.

**Problem**: The copy loop only added new/updated files — it never removed entries at the destination that no longer existed in the source. After deleting 9 skills from the repo, those stale skill directories persisted at all deployment targets (`~/.claude/skills/`, Gemini prompts dir) indefinitely.

**Solution**: Changed both setup scripts to a three-step sync pattern: (1) enumerate destination entries, (2) delete any entry not present in the source, (3) copy all source entries. Linux uses `basename` + `[ -d ]` checks; Windows uses `$item.BaseName` + `Test-Path`.

**Rule**: Any file deployment pipeline that copies a directory must also handle deletions. Additive-only copy creates ghost artifacts that accumulate silently. The canonical pattern is: enumerate destination → diff against source → delete orphans → copy current. Apply this to skills, configs, prompts, or any mirrored directory.
