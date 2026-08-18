---
id: lesson-011-claude-code-path-encoding-linux-vs-windows-differ
type: lesson
status: active
created: "2026-02-28"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 011: Claude Code path encoding: Linux vs Windows differ

**Context**: Implementing `--all` auto-discovery in `knowledge-crystallize.sh/.ps1`. Both scripts need to decode `~/.claude/projects/<encoded>/` back to real project paths.

**Problem**: The encoding differs by OS. Linux uses `tr '/' '-'` on the full absolute path (leading `/` becomes leading `-`). Windows uses `.Replace('\', '-').Replace(':', '')` which strips the drive colon (no leading dash). A decode strategy that works on one OS breaks on the other.

**Solution**: Two-stage decode in each script: (1) simple character substitution + `Test-Path`/`-d` existence check (handles 95% of cases where dir names have no dashes); (2) filesystem walk under `$HOME`/`$env:USERPROFILE` up to depth 5, encode each directory, compare (handles project names with dashes like `kasa-provisioner`). The walk is O(dirs under HOME, depth ≤ 5) — fast enough in practice.

**Rule**: When round-tripping Claude Code project paths: Linux encodes with leading `-` (from `/`), Windows strips `:` so no leading char. Keep OS-specific encode/decode functions separate. Always test with a project whose name contains a dash.
