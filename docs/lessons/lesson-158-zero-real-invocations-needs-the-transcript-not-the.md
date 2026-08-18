---
id: lesson-158-zero-real-invocations-needs-the-transcript-not-the
type: lesson
status: active
created: "2026-08-06"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 158: "Zero real invocations" needs the transcript, not the plugin listing — and a removed plugin can still be pinned by a hard-coded count

**Context**: Auditing which of 14 installed Claude Code plugins were actually used, to decide what to remove. The available-skills text shown to the agent lists every installed plugin's commands with a description, regardless of whether they have ever been invoked — that listing was the only signal checked at first.

**Problem**: Two separate mistakes, caught only because the audit was pushed further than the first plausible answer. First, judging usage from the skill *listing* text (which enumerates every plugin's capabilities on every session start) is indistinguishable from judging usage from actual *invocations* — grepping for a plugin's name matches the description prose just as well as a real call, so an initial pass wrongly flagged `security-guidance` and `gopls-lsp` as dead; both had real evidence (8 saved finding-state files, a real gopls diagnostic) once checked directly instead of inferring from the listing. Second, after correctly identifying and removing 8 zero-usage plugins from `ai/claude/settings.json`'s `enabledPlugins`, CI failed: `tests/claude-settings-template.bats` (the BUG-007 incident→guard pattern) hard-pinned `enabledPlugins` to exactly 13 entries and asserted 5 of the just-removed plugins by name. The removal itself was correct; it broke a regression guard that existed specifically to catch accidental removals, which a *deliberate* removal trips just as loudly.

**Solution**: For usage, grep the actual invocation record instead of the capability listing — `~/.claude/projects/**/*.jsonl` session transcripts contain the real `"skill":"..."` (Skill tool), `"subagent_type":"..."` (Agent/Task tool), and `<command-name>...</command-name>` (slash command) values; only those count as usage. For hook-driven or LSP-driven plugins with no slash-command surface, check their own state/output artifacts instead (`security_warnings_state_*.json`, an actual LSP diagnostic in a saved transcript) — the invocation grep is blind to them by design. For the guard: updated the count (13→5), replaced the 5-plugin *sample* assertion with an *exhaustive* one now that only 5 remain, and extended the existing single-plugin inverse assertion (`must NOT include github`) into a loop covering all 8 newly-removed plugins plus the same check across both `setup-linux.sh` and `setup-windows.ps1` install loops — mirroring the exact BUG-007 shape rather than inventing a new one.

**Rule**: "Is X used" is a question about the invocation log, not about whatever descriptive text happens to mention X — a listing, a README, a plugin catalog. When a prior audit only checked description text, don't trust its negatives without re-checking against the real record. Separately: a hard-pinned count/contents test guarding against *accidental* change will also fire on a *correct, deliberate* change — that is not a false positive to route around, it is doing its job; update the guard's expected value in the same commit or PR, and verify the test suite locally before pushing rather than discovering the mismatch in CI.
