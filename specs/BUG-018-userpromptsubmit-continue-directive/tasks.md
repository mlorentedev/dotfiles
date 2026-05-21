---
tags: [spec, tasks, claude-mem, heal]
created: "2026-05-21"
---

# Tasks - BUG-018-userpromptsubmit-continue-directive

## Setup
- [x] Branch off main (post BUG-017 merge).
- [x] Spec scaffolded (used -ForceNoVault; vault entry uses BUG-018-claude-mem-userpromptsubmit-continue-directive slug).

## Implementation
- [x] Regex substitution in claude-mem-heal.sh (sed -e).
- [x] Regex replace in claude-mem-heal.ps1 (-replace).
- [x] tests/setup-linux.bats parity assert.

## Lint
- [x] bash -n OK; PowerShell AST clean; PSSA clean; ASCII-only.
- [x] Empirical: 5 hooks patched on user's Windows; loop resolved.

## Closing
- [x] verification.md filled.
- [ ] CI green on PR #85 after spec scaffold.
- [ ] Post-merge: archive, tick vault, append lesson.
