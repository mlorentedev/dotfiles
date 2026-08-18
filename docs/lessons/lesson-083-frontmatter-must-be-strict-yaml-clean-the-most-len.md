---
id: lesson-083-frontmatter-must-be-strict-yaml-clean-the-most-len
type: lesson
status: active
created: "2026-06-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 083: Frontmatter must be strict-YAML clean — the most lenient parser in the fleet is not the contract

**Context:** pi v0.79.1 flagged two deployed skills (`spec`, `architecture-session`) as **Skill conflicts** at startup: "Nested mappings are not allowed in compact mappings". Both had `description:` values containing `: ` sequences (e.g. "Four subcommands: init (...)") without quoting. Claude Code had consumed those SKILL.md files for weeks without complaint.
**Problem:** Cross-agent artifacts (skill frontmatter rendered by the harness to claude/opencode/agy/pi) are parsed by N different YAML implementations. Authoring against the *most lenient* consumer (Claude's parser) lets latent violations accumulate; the contract silently becomes "whatever the loosest parser accepts" until a strict consumer joins the fleet (pi) and surfaces them all at once.
**Solution:** Quote any frontmatter scalar containing `: `, `#`, or leading/trailing specials at the **vault SSOT** (fixed in vault `75fe67c`, propagated via `compile-harness --refresh`, PR #316). Validate with a real YAML parser (`python3 -c "yaml.safe_load(...)"`), not by eyeballing. When a registry serves N consumers, the strictest parser in the fleet defines the format contract — sweep the whole catalog when one file fails, not just the reported one (here a sweep of all 32 skills confirmed only 2 were affected).
**Tags:** `#yaml` `#frontmatter` `#skills` `#cross-agent` `#harness` `#strictest-consumer-wins` `#vault-ssot`
