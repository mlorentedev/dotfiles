---
id: lesson-068-a-whole-file-transform-must-inspect-the-data-shape
type: lesson
status: active
created: "2026-05-30"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 068: A whole-file transform must inspect the data shape before assuming a uniform model

**Context:** SDD-008: migrating skill deploy to render-at-deploy. The render kind 'skill' was first implemented to render only SKILL.md to the agent path.
**Problem:** Two vault skills carry auxiliary files (systematic-debugging has 5: scripts + reference .md; test-driven-development has 1). The old claude deploy did `cp -rf "$skill_dir"*` (whole directory), so a SKILL.md-only render would have silently dropped those reference files — a functional regression invisible to a test that only checks SKILL.md.
**Solution:** Before committing to a transform model, enumerate the inputs (here: `find vault/00_meta/skills -type f ! -name SKILL.md`). For dir-based renders, copy the whole record dir then overlay the rendered SKILL.md; single-file renders (opencode command, agy prompt) legitimately take only SKILL.md. Add a test that asserts an auxiliary file lands at the dir-based target and does NOT at the single-file targets.
**Tags:** `#sdd` `#skills` `#deploy` `#testing`
