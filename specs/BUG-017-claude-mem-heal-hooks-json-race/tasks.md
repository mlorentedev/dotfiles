---
tags: [spec, tasks, claude-mem, heal, cross-os-parity]
created: "2026-05-21"
---

# Tasks - BUG-017-claude-mem-heal-hooks-json-race

## Setup
- [x] Branch `fix/BUG-017-claude-mem-heal-hooks-json-race` (off main).
- [x] Spec scaffolded.
- [x] Empirical: 14 broken hook commands across 2 hooks.json files (7 cache + 7 marketplace-via-junction).

## Implementation
- [x] `heal_hooks_json` function in claude-mem-heal.sh + walk both `<dir>/hooks/` and `<dir>/plugin/hooks/`.
- [x] `Repair-HooksJson` function in claude-mem-heal.ps1; same logic.
- [x] 3 new bats parity asserts.

## Lint
- [x] `bash -n` OK; PowerShell AST + PSScriptAnalyzer clean; ASCII-only.
- [x] Empirical: first run patches 14 hooks across 2 files; second run silent.

## Closing
- [x] verification.md filled.
- [ ] PR opened referencing this spec.
- [ ] Post-merge: archive, tick vault, append lesson.
