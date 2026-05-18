---
tags: [spec, tasks]
created: "2026-05-18"
---

# Tasks - SDD-002-settings-portability

> TDD order. One atomic PR. Per-key merge policy is the core invariant -- gets bats coverage first.

## Setup

- [x] Vault entry exists in `10_projects/dotfiles/11-tasks.md` (added 2026-05-18, auto-synced)
- [x] Branch created from main: `feat/SDD-002-settings-portability`
- [x] Spec scaffolded via `scripts/init-spec.ps1 SDD-002-settings-portability` (vault gate passed)
- [x] `proposal.md` complete; per-key merge policy table is testable
- [x] Open questions resolved (the `enabledPlugins[X] = false` decision documented in proposal -- accept template wins)

## Implementation (TDD)

### Template file

- [ ] Write failing bats `tests/claude-settings-template.bats` (new file): asserts template exists, valid JSON via jq, contains required top-level keys (`model`, `effortLevel`, `hooks`, `enabledPlugins`, `permissions`), contains literal `__HOOK_COMMAND__` placeholder, `enabledPlugins` has all 14 universal plugins, `permissions.allow` has exactly the 3 MCP entries and NOTHING starting with `Read(`
- [ ] Create `ai/claude/settings.json` with curated content (plain JSON, no Markdown frontmatter -- Claude Code parses it raw)
- [ ] Verify bats green (grep simulation locally)

### Per-key merge: setup-windows.ps1

- [ ] Write failing bats parity asserts in `tests/setup-windows.bats`: script reads `ai\claude\settings.json` template path; no inline `$hookEntry = @{ ... }` hashtable for the hook; references the `__HOOK_COMMAND__` substitution; logs the bootstrap message when target missing
- [ ] Add helper function `Merge-ClaudeSettings` to setup-windows.ps1: parameters = template path, target path, hook command. Reads both as hashtables (`ConvertFrom-Json -AsHashtable`), applies per-key policy from proposal, writes target with `ConvertTo-Json -Depth 10`. Bootstrap = create from template alone if target missing
- [ ] Replace the existing hook-registration block (lines 743-787) with: read template -> substitute __HOOK_COMMAND__ -> call Merge-ClaudeSettings -> log success
- [ ] PSScriptAnalyzer clean on `setup-windows.ps1` (Error+Warning severity, .PSScriptAnalyzerSettings.psd1)

### Per-key merge: setup-linux.sh

- [ ] Write failing bats parity asserts in `tests/setup-linux.bats` (new file if needed, or extend `tests/setup-windows.bats`): script reads `ai/claude/settings.json`; no inline `HOOK_ENTRY=$(jq -n ...)` heredoc; uses `--arg cmd` for __HOOK_COMMAND__ substitution
- [ ] Add shell function `merge_claude_settings` to setup-linux.sh that applies per-key policy via a single jq invocation. Bootstrap branch = write template directly when target missing
- [ ] Replace the existing hook-registration block (lines 626-649) with the new merge function call
- [ ] `bash -n` syntax clean; shellcheck severity=error clean

### Cross-OS parity locks

- [ ] Bats parity assert: both scripts reference path to `ai/claude/settings.json` (OS-specific separator)
- [ ] Bats parity assert: both scripts mention `__HOOK_COMMAND__` substitution
- [ ] Bats parity assert: both scripts log the bootstrap message

### Empirical smoke (Windows -- this machine)

- [ ] Capture pre-state: `Copy-Item ~/.claude/settings.json ~/.claude/settings.json.pre-sdd002`
- [ ] Run `pwsh -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1` (filtered to `Select-String 'settings\.json|hooks|Bootstrapping|permissions'`)
- [ ] Diff settings.json before/after via PowerShell. Required outcomes: (a) `model = "opus"` still set; (b) all pre-existing `permissions.allow` entries preserved (3 MCPs + 2 Reads + 1 work path = 6 entries, plus template adds the 3 MCPs deduped); (c) `additionalDirectories` still has the 1 entry untouched; (d) `hooks.SessionStart` still has the correct hook command (rewritten to match template's substituted value); (e) all 14 `enabledPlugins` still `true`
- [ ] Bootstrap smoke: `Move-Item ~/.claude/settings.json ~/.claude/settings.json.bak`, re-run setup, confirm new settings.json created from template; restore via `Move-Item -Force`
- [ ] Cross-OS smoke deferred to bats CI (no Linux access this session)

## Closing

- [ ] Every acceptance criterion from `proposal.md` covered by at least one test
- [ ] PSScriptAnalyzer + bash -n + shellcheck clean on modified files
- [ ] No unrelated changes in the diff (no scope creep into Tier 4/5 or other SDD work)
- [ ] `verification.md` filled with: smoke command outputs (.ps1 only on this machine; Linux relies on CI bats), bats simulation results, before/after settings.json snippets, decisions made during implementation
- [ ] PR opened referencing this spec folder; PR body notes "Tier 3 of SDD-001 stack; Tier 4+5 in SDD-003"
- [ ] Spec status moved `draft` -> `implementing` when first code commit lands; -> `verifying` when smoke green; -> `archived` (move to `specs/archive/`) only after PR merge
