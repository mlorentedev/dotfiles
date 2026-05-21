---
tags: [spec, tasks, claude-plugins, removal]
created: "2026-05-19"
---

# Tasks - BUG-007-remove-github-plugin-broken

## Setup

- [x] Branch: `fix/BUG-007-remove-github-plugin-broken` (off main).
- [x] Vault entry in `11-tasks.md`.

## Implementation

- [ ] Remove plugin entry from `ai/claude/settings.json` (line 33).
- [ ] Remove plugin from `setup-linux.sh` plugin install loop (line 646).
- [ ] Remove plugin from `setup-windows.ps1` plugin install loop (line 415).
- [ ] In `tests/claude-settings-template.bats`: replace the positive assertion at line 86 with an inverse assertion that fails CI if the plugin is re-added. Keep the sample size at 5 by adding a different plugin (e.g. `feature-dev`).
- [ ] `jq -e . ai/claude/settings.json` clean.
- [ ] Full bats suite green.
- [ ] `shellcheck --severity=error` clean on touched scripts.

## Closing

- [ ] verification.md filled.
- [ ] PR opened with note for user: existing `~/.claude/settings.json` will be overwritten by next setup run; user may want to `claude plugin uninstall github@claude-plugins-official` manually for clean state.
- [ ] Post-merge: tick BUG-007 in vault + archive spec.
