---
tags: [spec, verification, claude-plugins, removal]
created: "2026-05-19"
---

# Verification - BUG-007-remove-github-plugin-broken

## Evidence

- [x] AC1 Zero matches for `github@claude-plugins-official` in `ai/claude/settings.json`, `setup-linux.sh`, `setup-windows.ps1`. Confirmed via `grep -nE` (returns no hits).
- [x] AC2 `tests/claude-settings-template.bats` has 3 inverse assertions (BUG-007 guards): template lookup must fail, setup-linux.sh grep must miss, setup-windows.ps1 grep must miss.
- [x] AC3 Positive sample list rebalanced: `github` removed from the 5-plugin sample, replaced with `feature-dev` (also widely used). Sample size preserved at 5.
- [x] AC4 Plugin count assertion updated from 14 to 13 (the only place a count is hardcoded). Comment updated to flag the pre/post-BUG-007 state.
- [x] AC5 Full bats suite: **673/673 pass** (was 670; +3 net from inverse guards).
- [x] AC6 `jq -e . ai/claude/settings.json` clean.
- [x] AC7 `shellcheck --severity=error setup-linux.sh` clean.

## Test status

- `bats tests/claude-settings-template.bats` → all green (including new inverse asserts).
- `bats tests/*.bats` → 673/673 pass, 0 fail.
- Manual cross-check: `grep -rn "github@claude-plugins-official" .` returns only the inverse-assertion lines in the bats file and references in archived specs (`specs/archive/SDD-002-settings-portability/proposal.md`) — which are correctly left untouched.

## Decisions made during implementation

- **Sample plugin swap rather than shrink.** Removing `github` from the 5-plugin sample would have left only 4 — weaker test. Adding `feature-dev` keeps the assertion strength while reflecting the new state. Comment documents the swap.
- **Triple inverse assertion (template + 2 setup scripts) instead of just template.** Each surface can drift independently; checking all three catches any re-add path. Direct application of the SDD-006 incident → guard lesson.
- **Plugin count down-tick from 14 → 13.** The count is hardcoded in only one place (line 74). Updating it locks the new contract; a future re-add would require BOTH adding the plugin AND bumping the count — two simultaneous edits is a higher bar than one.
- **No removal from `specs/archive/SDD-002-settings-portability/proposal.md`** (which mentions "14 plugins"). Archive is frozen historical record; rewriting it would distort the audit trail. The current count assertion (13) and the prose explanation in the inverse-assert comment carry the truth forward.

## Post-merge user action

The user's existing `~/.claude/settings.json` may still have `github@claude-plugins-official` enabled (from prior setup runs). Per SDD-002 merge policy, `enabledPlugins` is dotfiles-owned → the **next `setup-linux.sh` run will overwrite the list** without the github entry, but the plugin will remain installed on disk under `~/.claude/plugins/`. For full cleanup:

```bash
claude plugin uninstall github@claude-plugins-official
```

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? Already captured — SDD-006's "Incident → guard pattern" lesson. This PR is a textbook application: bug encountered → inverse assertion added in the same diff.
- [ ] ADR-worthy? No — operational removal, not architecture.
- [ ] Pattern for `00_meta/patterns/`? Premature.

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`.
- [ ] Folder moved: `specs/BUG-007-.../` → `specs/archive/BUG-007-.../`.
- [ ] Vault `11-tasks.md` ticked with PR link.
