---
id: lesson-092-editing-a-committed-render-without-its-source-of-t
type: lesson
status: active
created: "2026-06-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 092: Editing a committed render without its source-of-truth is a half-migration that `--refresh` reverts

**Context**: CLI-005 repointed `harness/skills/spec/SKILL.md` and `harness/skills/adversarial-review/SKILL.md` to `dotf spec`. Those files are committed *renders*: `compile-harness.sh` (SDD-008) treats the vault `00_meta/skills/<name>/SKILL.md` as the edit-SSOT and regenerates `harness/skills/` from it via `--refresh`.

**Problem**: Editing only the committed render leaves the vault sources stale. The render is correct for CI `--check`/`--deploy`, but the next `compile-harness.sh --refresh` pulls the unchanged vault source and silently reverts the repoint — a green PR that re-introduces the dead references on the next harness refresh.

**Solution**: Sync the vault sources in lockstep with the render. On the interactive machine, vault edits land on `origin/master` via obsidian-git's periodic auto-commit — just edit the files, no manual git. Verify the vault sources no longer carry the old reference before declaring the migration done.

**Rule**: For any file that is a generated/committed render, a migration is only complete when its *source-of-truth* changes too. Identify the generator (here `compile-harness.sh`) and edit upstream, or the render's change is transient. Sibling of the GEMINI→AGY incomplete-migration lesson: a repoint that leaves a caller — or a generator's source — stale is a half-migration.
