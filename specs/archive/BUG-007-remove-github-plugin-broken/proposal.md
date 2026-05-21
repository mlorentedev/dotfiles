---
id: "BUG-007-remove-github-plugin-broken"
type: spec
status: draft
created: "2026-05-19"
tags: [spec, proposal, claude-plugins, removal]
template_version: "1.0"
---

# BUG-007-remove-github-plugin-broken

## Why

The user reports `github@claude-plugins-official` is no longer useful and does not work. Every dotfiles setup run currently includes this plugin in the `enabledPlugins` list, triggering a `claude plugin install` call for a broken dependency. Removing it cleans the deploy contract.

## What

Remove every reference to `github@claude-plugins-official` across the repo:

- `ai/claude/settings.json` (template `enabledPlugins`)
- `setup-linux.sh` (plugin install loop)
- `setup-windows.ps1` (plugin install loop)
- `tests/claude-settings-template.bats` (positive assertion at line 86)

Add an **inverse bats assertion** that fails CI if the plugin is ever re-added to the template (incident → guard pattern from SDD-006).

## Out of scope

- Removing other plugins.
- Auditing the remaining 13 plugins.
- Touching archived spec documents that mention the plugin (`specs/archive/SDD-002-settings-portability/proposal.md`) — archive is frozen historical record.

## Risks / open questions

- **Risk: user's existing `~/.claude/settings.json` still has the plugin enabled** post-setup (since `merge_claude_settings` preserves user values for keys outside the dotfiles-owned subset, but `enabledPlugins` IS in the subset). **Mitigation**: per SDD-002 design, `enabledPlugins` is dotfiles-owned → next setup run will overwrite with the new list (sans github). User may want to manually `claude plugin uninstall` for cleanup; documented in PR body.
- **Risk: someone re-adds the plugin in a future PR.** **Mitigation**: inverse bats assertion is the guard.

## Acceptance criteria

- [ ] Zero matches for `github@claude-plugins-official` in `ai/claude/settings.json`, `setup-linux.sh`, `setup-windows.ps1`.
- [ ] `tests/claude-settings-template.bats` has an **inverse assertion**: `! jq -e '.enabledPlugins["github@claude-plugins-official"]'`.
- [ ] The previously-existing positive assertion (line 86) is removed or replaced with a different sample plugin.
- [ ] Full bats suite green (target: 670 + new = 671 or unchanged if the inverse replaces the positive).
- [ ] `jq -e . ai/claude/settings.json` clean.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` BUG-007 entry.
- Vault lesson: [[90-lessons]] "Incident → guard pattern (red-team thyself)" — applied here as the inverse assertion.
- Trigger: user statement 2026-05-19 "el plugin de github habria que quitarlo de claude code, no nos sirve mas y no funciona".
