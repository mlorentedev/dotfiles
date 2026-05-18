---
id: "SDD-002-settings-portability"
type: spec
status: draft
created: "2026-05-18"
tags: [spec, proposal, sdd, settings, portability]
template_version: "1.0"
---

# SDD-002-settings-portability

> Tier 3 of the 5-layer SDD enforcement stack started by SDD-001 (PR #49). Tracks a curated subset of `~/.claude/settings.json` in dotfiles as SSOT, refactors both setup scripts to read it instead of hardcoding hook entries, and bootstraps the file on fresh machines (closes the "run claude once, then re-run setup" doble-paso).

## Why

Today the hook registration logic for `~/.claude/settings.json` lives in TWO places: `setup-windows.ps1` lines 743-787 (PowerShell hashtable + ConvertTo-Json) and `setup-linux.sh` lines 626-649 (bash + jq invocation). Adding a new structural key requires editing both scripts in parallel -- exactly the duplication that caused the WIN-003 + BUG-002 + BUG-003 class of bugs. There's also no SSOT for "what does dotfiles own in settings.json" -- it's implicit in the imperative code of each script.

Second pain: on a fresh machine without `~/.claude/settings.json`, both scripts log `"settings.json not found, skipping"` and leave the hook unregistered. User has to run `claude` once (creates the default), then re-run setup. Two-step bootstrap when one would suffice.

## What

Three observable behavior changes after this PR:

1. **A canonical template `ai/claude/settings.json`** declares the curated "dotfiles-owned" subset: `model`, `effortLevel`, `hooks.SessionStart` (with a `__HOOK_COMMAND__` placeholder substituted at install time), `enabledPlugins` (the 14 universal plugins), `permissions.allow` (only the 3 portable MCP entries: `mcp__hive__vault_query`, `mcp__hive__vault_write`, `mcp__sequential-thinking__sequentialthinking`).
2. **Both setup scripts refactor** their hook-registration blocks to read the template and apply a **per-key merge policy** (see Risks section for the table). Net effect: the inline PowerShell hashtable + bash heredoc go away, replaced by a small read-template + apply-merge function each side. Adding new structural keys becomes a template edit, not a code change.
3. **Bootstrap on fresh machine**: if `~/.claude/settings.json` does not exist, setup creates it from the template (with `__HOOK_COMMAND__` substituted). Eliminates the doble-paso for new installs.

## Out of scope

- **Machine-specific keys** stay user-owned, not tracked in template: `permissions.allow` entries with absolute paths (e.g., `Read(//c/Users/mlorente/...)`); `permissions.additionalDirectories`; per-machine `enabledPlugins` additions beyond the 14 universal ones; per-machine hook entries from third-party tools (claude-mem, GitGuardian).
- **No platform variable substitution beyond `__HOOK_COMMAND__`.** Users with cross-machine differences in those keys handle it manually (this is acceptable because those are inherently per-machine state).
- **No CI gate for template drift** (e.g., asserting the template stays sorted, etc.). Pure template hygiene; add later if it becomes a pain point.
- **No migration of pre-existing user settings.json content into the template repo.** The template captures the canonical "ours" subset; user's existing customizations stay user-owned via the merge policy.
- **No platform-conditional template content beyond the placeholder.** Template content is identical across OSes except for what the install script substitutes for `__HOOK_COMMAND__`.

## Risks / open questions

**Risk: Per-key merge policy must be exact or third-party hooks get clobbered.** Below is the policy table; bats tests lock it in.

| Key | Policy | Rationale |
|-----|--------|-----------|
| `model` | TEMPLATE wins (overwrite) | Universal user pref |
| `effortLevel` | TEMPLATE wins (overwrite) | Universal user pref |
| `permissions.allow` | UNION (template entries + user entries, dedup) | Preserve user's machine-specific Read paths AND extra MCP entries; add our 3 portable MCP entries |
| `permissions.additionalDirectories` | USER preserved (not in template, no touch) | Machine-specific paths |
| `permissions.*` (other subkeys) | USER preserved (not in template, no touch) | Future-proof |
| `hooks.SessionStart` | TEMPLATE wins (replace entire array) | We own this hook; previous WIN-003 self-heal logic preserved by always rewriting |
| `hooks.PreToolUse` / `PostToolUse` / `Stop` | USER preserved (not in template, no touch) | Third-party tools (claude-mem heal, GitGuardian) register hooks here |
| `enabledPlugins` | Object merge (template + user, template wins on conflict) | Universal plugins always enabled; user can add more; if user disables a universal plugin it gets re-enabled (acceptable trade -- template wins) |
| Other top-level keys | USER preserved (not in template, no touch) | Future-proof |

**Open question (resolved): what about `enabledPlugins[X] = false`?** If a user has explicitly disabled a universal plugin in their settings (`"code-review@claude-plugins-official": false`), the merge policy currently overrides it to `true`. Decision: accept this as the trade -- the template's universal-plugins-enabled stance is intentional. If a user really doesn't want a plugin, they remove it from the template via PR (cross-machine effect). Documented as known behavior in `verification.md` Decisions section.

**Risk: bootstrap creates a settings.json the user didn't ask for.** Mitigation: the install script logs `"Bootstrapping ~/.claude/settings.json from template (file did not exist)"` so the user sees what happened. The template is identical to what would have been registered piecemeal before; just creates it all in one shot.

**Risk: __HOOK_COMMAND__ substitution requires the placeholder to be exact.** Mitigation: bats asserts the template contains the literal `__HOOK_COMMAND__` string; setup scripts use literal substitution (PowerShell `.Replace()`, jq with `--arg`).

## Acceptance criteria

- [ ] `ai/claude/settings.json` exists at repo root with frontmatter-free JSON (Claude Code requires plain JSON)
- [ ] Template contains: `model: "opus"`, `effortLevel: "xhigh"`, `hooks.SessionStart` (with `__HOOK_COMMAND__` placeholder), `enabledPlugins` (the 14 plugins), `permissions.allow` (the 3 MCP entries, no absolute paths)
- [ ] `setup-windows.ps1` hook block reads `ai\claude\settings.json` and applies per-key merge; the inline PowerShell hashtable for the hook entry is gone
- [ ] `setup-linux.sh` hook block reads `ai/claude/settings.json` and applies per-key merge via jq; the inline `HOOK_ENTRY=$(jq -n ...)` is gone
- [ ] Bootstrap: when `~/.claude/settings.json` does not exist, setup creates it from the template; log line `"Bootstrapping ~/.claude/settings.json from template"` visible
- [ ] Merge preservation: when `~/.claude/settings.json` has user customizations (test fixture: extra `permissions.allow` Read entry + `additionalDirectories` + a third-party `hooks.PreToolUse` entry), running setup preserves all of them
- [ ] Merge overwrite: when `~/.claude/settings.json` has `model: "sonnet"` (template says `"opus"`), running setup overwrites to `"opus"`
- [ ] Cross-OS parity: bats asserts both setup scripts call out to read `ai/claude/settings.json` and use the per-key merge pattern
- [ ] PSScriptAnalyzer clean on modified `setup-windows.ps1`; bash `-n` + shellcheck clean on `setup-linux.sh`
- [ ] Empirical smoke on this machine: re-run `setup-windows.ps1`, observe settings.json before/after; user customizations (the existing 6 `permissions.allow` entries including Read paths, `additionalDirectories`) survive

## References

- Vault: `10_projects/dotfiles/11-tasks.md` "SDD-002-settings-portability" backlog entry
- Pattern: `00_meta/patterns/pattern-spec-driven-development.md`
- Parent: SDD-001 (PR #49) -- established the discipline gate this spec is a continuation of
- Sibling: SDD-003 (next) -- CI spec-gate + PR template, Tier 4+5
- Related: WIN-003 (PR #21) -- the original SessionStart hook self-heal whose logic is preserved by the TEMPLATE-wins policy on `hooks.SessionStart`
