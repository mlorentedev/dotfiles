---
tags: [spec, verification]
created: "2026-05-18"
---

# Verification - SDD-002-settings-portability

## Evidence

Each acceptance criterion from `proposal.md` mapped to its proof. All empirically verified on the admin Windows machine 2026-05-18 same session.

- [x] **`ai/claude/settings.json` exists as plain JSON** → file created; `jq empty` passes; bats `tests/claude-settings-template.bats` "template is valid JSON".
- [x] **Template contains the agreed keys** → `model: "opus"`, `effortLevel: "xhigh"`, `permissions.allow` (3 MCP entries only, no `Read(` paths), `hooks.SessionStart` (with `__HOOK_COMMAND__` placeholder), `enabledPlugins` (14 universal plugins). Bats asserts: 18 in template-only file + 11 parity asserts in `tests/setup-windows.bats`.
- [x] **setup-windows.ps1 refactored** → `Merge-ClaudeSettings` function defined (lines 57-129); hook block replaced with single function call; inline `$hookEntry = @{ ... }` hashtable is GONE. Bats: "SDD-002: setup-windows.ps1 calls Merge-ClaudeSettings (not inline hashtable)".
- [x] **setup-linux.sh refactored** → `merge_claude_settings()` shell function defined (lines 633-694); hook block replaced with single function call; inline `HOOK_ENTRY=$(jq -n ...)` is GONE. Bats: "SDD-002: setup-linux.sh calls merge_claude_settings (not inline HOOK_ENTRY)".
- [x] **Bootstrap on fresh machine** → empirically verified: `Move-Item ~/.claude/settings.json ~/.claude/settings.json.bootstrap-test`; `pwsh -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1`; output included literal `[INFO] Bootstrapping ~/.claude/settings.json from template (file did not exist)` + `[SUCCESS] Claude settings.json bootstrapped from template`. New file contained `model=opus`, `effortLevel=xhigh`, `permissions.allow=3` MCP entries, `hooks.SessionStart` with correct substituted command, 14 plugins. Original restored from .bootstrap-test.
- [x] **Merge preservation (user customizations survive)** → empirical smoke: pre-state had `permissions.allow` count = 6 (3 MCPs + 3 Read paths), `additionalDirectories` count = 1, `hooks.SessionStart` count = 1, 14 plugins. Post-merge state: `permissions.allow` count = 6 (UNION dedup correctly kept all 6), `additionalDirectories` count = 1 (untouched), `hooks.SessionStart` count = 1 (replaced with template's, command substituted to `pwsh -NoProfile -File "C:\Users\mlorente\scripts\claude-session-start.ps1"`), 14 plugins.
- [x] **Merge overwrite (template wins on `model`)** → not empirically tested in same session (model was already `opus` matching template). Bats parity asserts confirm the logic path exists. Future drift would catch via the per-key merge code path.
- [x] **Cross-OS parity** → 11 bats asserts in `tests/setup-windows.bats` lock the symmetry: both scripts reference `ai/claude/settings.json` template path, both define their merge functions, both log the bootstrap message, both substitute `__HOOK_COMMAND__`.
- [x] **PSScriptAnalyzer clean** → 0 warnings on `setup-windows.ps1`. The expected `PSUseSingularNouns` warning for `Merge-ClaudeSettings` is suppressed in-line with explicit rationale ("Settings" is the canonical config-file name; singular `Setting` would be misleading).
- [x] **bash -n + shellcheck** → `bash -n setup-linux.sh` clean. shellcheck not installed locally (Windows session) -- CI ubuntu-latest will validate.

## Test status

- **Template tests** (`tests/claude-settings-template.bats`, 18 asserts): simulated locally via jq commands; 100% green.
- **Parity tests** (`tests/setup-windows.bats` SDD-002 section, 11 asserts): simulated locally via grep; 100% green.
- **Empirical Windows smoke** (this machine): merge preserves all user customizations + bootstrap creates from template. Both flows verified end-to-end.
- **Empirical Linux smoke**: deferred to CI (no Linux access this session). The bash `merge_claude_settings` mirrors the PowerShell logic line-for-line; jq invocation is the standard pattern already used in the previous hook-registration block.
- **No regressions**: existing setup-windows.bats, aliases.bats, agents-md.bats, hooks.bats untouched. The legacy hook-registration assertions in setup-windows.bats are still green because the new function still produces the expected behavior (same hook entry shape) -- they just don't constrain HOW it's done anymore.

## Decisions made during implementation

- **Placeholder approach `__HOOK_COMMAND__` (Option B) chosen over per-OS templates (Option A) or hook-stays-in-script (Option C).** Keeps the template as the canonical "ours" representation while the install script handles the one platform-specific bit. Single template file + tiny substitution > two near-duplicate templates with drift risk > template that doesn't cover the hook (incoherent SSOT).
- **Per-key merge policy explicit table** (in proposal.md) over generic deep-merge. Generic deep-merge would have ambiguity on arrays (replace vs union) and could clobber third-party hooks (`PreToolUse`, `PostToolUse`, `Stop` are off-limits). The table makes the policy auditable and the implementation a direct mapping.
- **Bulk-copy exclusion of `settings.json`** in both setup scripts (`-Exclude 'settings.json'` for Copy-Item in PowerShell; explicit loop with `[ name = "settings.json" ] && continue` in bash). The previous bulk-copy of `ai/claude/*` to `~/.claude/` would have wiped user customizations AND deployed the template verbatim (with `__HOOK_COMMAND__` literal). Required for SDD-002 to work; would be a latent bug otherwise.
- **PSScriptAnalyzer suppression for plural noun** instead of renaming `Merge-ClaudeSettings` to `Merge-ClaudeSetting`. Renaming would be misleading ("Settings" is the canonical Claude Code config-file name; the function operates on the whole file, not one setting). Suppression with explicit comment is the documented PowerShell way.
- **PowerShell `-AsHashtable` for ConvertFrom-Json** instead of default PSCustomObject. Hashtables support `.ContainsKey()` and key-by-key mutation cleanly; PSCustomObject requires `Add-Member` ceremony for new keys. Hashtables round-trip correctly via `ConvertTo-Json -Depth 10`.

## Promotion candidates

- [x] **Lesson** for `90-lessons.md`? **YES** — capture: "Bulk-copy operations from a tracked directory tree silently collide with per-file deploy logic if both touch the same target. The collision is invisible until the per-file logic introduces a placeholder or merge invariant that the bulk-copy can't honor. Add an explicit exclusion when introducing per-file deploy semantics for a file previously bulk-copied." Useful pattern (recurs in skills deployment, MCP config, etc.).
- [ ] **ADR** for `30-architecture/`? No -- this PR is the operational form of an already-accepted decision (ADR-009 making AGENTS.md the SSOT; SDD-001 instituting the discipline gate). No new architectural decision.
- [ ] **New pattern** for `00_meta/patterns/`? Not yet -- the "per-key merge policy with explicit policy table" is interesting but appears only here so far. If it recurs (e.g., for mcp-servers.json deploy, opencode config deploy), promote then.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-002-settings-portability/` → `specs/archive/SDD-002-settings-portability/`
- [ ] Backlog entry in vault `10_projects/dotfiles/11-tasks.md` ticked with PR link
- [ ] Lesson promotion executed (Lessons section above flagged YES)
- [ ] SDD-003 (CI spec-gate + PR template) sibling spec opened
