#!/usr/bin/env bats
# Tests for setup-windows.ps1 (structural + PSScriptAnalyzer)

load 'winpath'
load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PS1_SCRIPT="$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup-windows.ps1 exists" {
    [[ -f "$PS1_SCRIPT" ]]
}

# --- Structural checks ---

@test "setup-windows.ps1 has .SYNOPSIS block" {
    grep -q '\.SYNOPSIS' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has .DESCRIPTION block" {
    grep -q '\.DESCRIPTION' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has CmdletBinding" {
    grep -q 'CmdletBinding' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Claude configuration" {
    grep -q 'Deploying Claude configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Antigravity (agy) configuration (SDD-007)" {
    grep -q 'Deploying Antigravity (agy) configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys PowerShell profile" {
    grep -q 'Setting up PowerShell profile' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys Git configuration" {
    grep -q 'Setting up Git configuration' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys versions.conf" {
    grep -q 'Deploying versions.conf' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys packages.json (CLI-029 tool catalog)" {
    grep -q 'packages.json deployed to' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs catalog tools via dotf (best-effort)" {
    grep -q 'dotf tools install' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 no longer deploys Aider configuration (sunset)" {
    refute_grep_fixed 'Deploying Aider configuration' "$PS1_SCRIPT"
    refute_grep_fixed 'Installing aider-chat via uv' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 registers MCP servers" {
    grep -q 'Registering Claude Code MCP servers' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 MCP loop is StrictMode-safe (probes PSObject.Properties)" {
    # Under Set-StrictMode -Version Latest, $srv.prerequisite_binary throws when
    # the field is absent. Only the hive MCP declares this property; the other
    # servers (sequential-thinking, context7) don't. The loop must probe via
    # PSObject.Properties to treat missing fields as $null, matching Linux which
    # uses jq's `(.prerequisite_binary // "")` null-coalesce.
    grep -qF "PSObject.Properties['prerequisite_binary']" "$PS1_SCRIPT"
    grep -qF "PSObject.Properties['prerequisite_command']" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 does NOT eager-source load-secrets; resolves the agy secret via dotf [#587]" {
    # Migrated off the load-secrets eager dot-source (ADR-028). agy's
    # OPENROUTER_API_KEY is baked into mcp_config.json via `dotf secrets show`
    # after Install-Dotf, before the agy block (opencode/pi self-resolve).
    refute_grep '\. \$loadSecretsDeployed' "$PS1_SCRIPT"
    refute_grep_fixed 'Eager-load secrets' "$PS1_SCRIPT"
    grep -qF 'dotf secrets show OPENROUTER_API_KEY' "$PS1_SCRIPT"
}

@test "parity: both setups resolve the agy deploy-time secret via dotf secrets show [#587]" {
    for s in "$PS1_SCRIPT" "$DOTFILES_DIR/setup-linux.sh"; do
        grep -qF 'dotf secrets show OPENROUTER_API_KEY' "$s"
    done
}

@test "setup-windows.ps1 deploys secrets/registry.yaml (dotf secrets mapping SSOT) [#587]" {
    grep -qF 'Deployed secrets/registry.yaml' "$PS1_SCRIPT"
    grep -qF 'registry.yaml' "$PS1_SCRIPT"
}

@test "parity: both setups deploy the secrets registry [#587]" {
    grep -qF 'registry.yaml' "$PS1_SCRIPT"
    grep -qF 'registry.yaml' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 preflights the age identity key (warn, not fail)" {
    # Without ~/.config/age/key.txt, dotf secrets can't decrypt and opencode/agy
    # 401 with no clue. Preflight WARNs (must not abort -- encrypted files still
    # deploy so a key imported later works).
    grep -qF 'age identity key not found' "$PS1_SCRIPT"
    grep -qF 'AGE_KEY_PATH' "$PS1_SCRIPT"
    grep -qF 'docs/runbooks/secrets-management.md' "$PS1_SCRIPT"
    # Must use Write-Warn (non-fatal), not Write-Err.
    grep -B2 'age identity key not found' "$PS1_SCRIPT" | grep -q 'Test-Path'
}

@test "parity: age key preflight present in both setup-windows.ps1 and setup-linux.sh" {
    grep -qF 'age identity key not found' "$PS1_SCRIPT"
    grep -qF 'age identity key not found' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 removes legacy GEMINI.md (SDD-007 migration)" {
    # Pre-SDD-007 setups deployed ~/.gemini/GEMINI.md. After agy replaced
    # gemini-cli, AGY.md is the new identity file. Without explicit cleanup the
    # orphan GEMINI.md lingers and confuses any tool that picks up either file.
    grep -qF 'Removed legacy GEMINI.md' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 sets up GitHub Copilot CLI" {
    grep -q 'Setting up GitHub Copilot CLI' "$PS1_SCRIPT"
}

# --- BUG-003: drop gh-copilot extension path, detect new copilot CLI v2 ---
# The legacy 'gh extension install github/gh-copilot' product was the v1 wrapper
# around suggest/explain. The new 'copilot' standalone CLI (winget GitHub.Copilot,
# binary `copilot`) is agentic (closer to Claude Code). Setup detects the new one
# via 'Get-Command copilot' and stops referencing the old extension entirely.

@test "setup-windows.ps1 detects copilot via Get-Command, not gh extension list" {
    grep -qF "Get-Command copilot" "$PS1_SCRIPT"
    refute_grep_fixed "gh extension list" "$PS1_SCRIPT"
    refute_grep_fixed "github/gh-copilot" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 no longer installs GitHub.Copilot via winget: copilot is a packages.json npm tool (AI-038, ADR-036)" {
    refute_grep 'Id = "GitHub\.Copilot"' "$PS1_SCRIPT"
    jq -e '.tools[] | select(.name == "copilot" and .source.type == "npm" and .source.package == "@github/copilot")' "$DOTFILES_DIR/packages.json" >/dev/null
}

@test "setup-windows.ps1 removes retired init .ps1 orphans, does not deploy them (CLI-020)" {
    # init-project.ps1 et al. were retired -> dotf init. setup must clean up old
    # copies, never Copy-Item them.
    grep -q 'init-repo-github-defaults.ps1' "$PS1_SCRIPT"
    refute_grep 'Copy-Item .*init-project\.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 removes every retired script and the legacy ~\scripts copies of the live ones (WIN-013)" {
    # Measured on a real box 2026-08-27: seven retired scripts still sat in
    # ~\scripts because nothing ever removed them. The list is the guard.
    for name in claude-mem-heal.ps1 claude-session-start.ps1 diff-check.ps1 doctor.ps1 healthcheck.ps1 knowledge-crystallize.ps1 session-handoff.ps1; do
        grep -qF "\"$name\"" "$PS1_SCRIPT"
    done
    grep -qF '$LegacyScriptsDir = "$env:USERPROFILE\scripts"' "$PS1_SCRIPT"
    grep -qF 'foreach ($dir in @($ScriptsDir, $LegacyScriptsDir)' "$PS1_SCRIPT"
    # The legacy PATH entry is deliberately not pruned (2048-char hazard, #148).
    refute_grep 'SetEnvironmentVariable\("PATH".*LegacyScriptsDir' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 no longer deploys knowledge-crystallize.ps1 (CLI-050)" {
    # dotf vault crystallize replaced it (#1269); no per-machine deploy step needed.
    refute_grep 'Copy-Item .*knowledge-crystallize\.ps1' "$PS1_SCRIPT"
    [ ! -f "$DOTFILES_DIR/scripts/knowledge-crystallize.ps1" ]
}

@test "setup-windows.ps1 deploys claude-session-start.ps1" {
    grep -q 'claude-session-start.ps1' "$PS1_SCRIPT"
}

# MEM-002: claude-mem-heal.ps1 is retired (ADR-016 Q2). setup-windows.ps1 must
# NOT deploy it any more.
@test "setup-windows.ps1 no longer deploys claude-mem-heal.ps1 (MEM-002)" {
    # The name may appear in WIN-013's removal list; never in a deploy or a call.
    refute_grep 'Copy-Item.*claude-mem-heal\.ps1' "$PS1_SCRIPT"
    refute_grep '&.*claude-mem-heal\.ps1' "$PS1_SCRIPT"
    [ ! -f "$DOTFILES_DIR/scripts/claude-mem-heal.ps1" ]
}

@test "setup-windows.ps1 deploys dotfiles-sync.ps1" {
    grep -q 'dotfiles-sync.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys obs-cli.ps1" {
    grep -q 'obs-cli.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 sets up secrets system" {
    grep -q 'Setting up secrets system' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 registers SessionStart hook" {
    grep -q 'SessionStart' "$PS1_SCRIPT"
}

# CLI-025: the SessionStart hook now invokes the `dotf mem session-start` binary
# directly (no pwsh -File shim), via the absolute ~/.local/bin path (#531).
@test "setup-windows.ps1 SessionStart hook command invokes dotf mem session-start" {
    grep -qE '\$expectedHookCommand\s*=.*\$dotfBin.*mem session-start' "$PS1_SCRIPT"
}

# Hook registration must self-heal -- never trust "an entry exists" to mean
# "the entry is correct". Post-SDD-002 (PR #51): Merge-ClaudeSettings ALWAYS
# rewrites hooks.SessionStart from the template (with __HOOK_COMMAND__
# substituted), which is a stronger guarantee than the previous compare-then-
# rewrite -- self-heal runs unconditionally on every setup invocation.
@test "setup-windows.ps1 SessionStart hook self-heals on path drift" {
    grep -qF '$expectedHookCommand' "$PS1_SCRIPT"
    grep -qF 'Merge-ClaudeSettings -TemplatePath' "$PS1_SCRIPT"
}

# --- Scheduled task self-heal (same class as #20) ---

# DotfilesVaultMaintenance task must reconcile arguments, not skip when present;
# otherwise a stale task pointing at a moved/renamed script stays broken forever.
@test "setup-windows.ps1 vault maintenance task self-heals on argument drift" {
    grep -q '\$expectedTaskArgument' "$PS1_SCRIPT"
    grep -Eq '\$existingTaskArgument\s+-eq\s+\$expectedTaskArgument' "$PS1_SCRIPT"
}

# --- MCP server registration (SSOT + idempotence) ---

# MCP server list must live in mcp-servers.json, not hardcoded in the setup script.
@test "setup-windows.ps1 MCP registration reads from mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$PS1_SCRIPT"
    refute_grep 'claude mcp add --transport (stdio|http) (drawio|socket|context7|sequential-thinking|hive)' "$PS1_SCRIPT"
}

# MCP registration must check existence before adding, not blindly retry.
@test "setup-windows.ps1 installs Claude Code and agy via their official installers when absent (AI-041, #1325)" {
    # setup-linux.sh provisions both; on Windows every block gating on `claude`
    # and the agy config deploy were dead on a clean box. Child pwsh, skip when
    # on PATH, before the MCP block that needs the binary.
    grep -qF 'function Install-AgentBinary' "$PS1_SCRIPT"
    grep -qF 'https://claude.ai/install.ps1' "$PS1_SCRIPT"
    grep -qF 'https://antigravity.google/cli/install.ps1' "$PS1_SCRIPT"
    grep -qF -- '-NoProfile -ExecutionPolicy Bypass -File $installer' "$PS1_SCRIPT"
    install_line=$(grep -n 'Install-AgentBinary -Name "Claude Code"' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    mcp_line=$(grep -n 'claude mcp get' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [ "$install_line" -lt "$mcp_line" ]
    # parity: the Linux script installs the same two through the same vendors
    grep -qF 'https://claude.ai/install.sh' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'https://antigravity.google/cli/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 persists the contract variables at User scope after generating paths.ps1 (CLI-058, #1324)" {
    # paths.ps1 only reaches profile-loading shells; Copilot's -NoProfile tool
    # calls and Scheduled Tasks read HKCU\Environment, which setup now fills.
    grep -qF 'dotf env persist' "$PS1_SCRIPT"
    gen_line=$(grep -n 'dotf env generate | Out-Null' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    persist_line=$(grep -n 'dotf env persist | Out-Null' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [ "$gen_line" -lt "$persist_line" ]
}

@test "setup-windows.ps1 MCP registration checks existence with claude mcp get" {
    grep -q 'claude mcp get' "$PS1_SCRIPT"
}

# --- BUG-004: defense-in-depth around claude plugin install (truncate guard) ---
# Every `claude plugin install` call triggers upstream anthropics/claude-code#59870:
# the CLI's deserialize-modify-serialize cycle drops fields outside its internal
# struct (organizationType, organizationRateLimitTier, projects map, onboarding
# flags), shrinking ~/.claude/.claude.json from ~75 KB to ~1.5 KB and forcing
# re-authentication. The existing idempotence guard (`-match [regex]::Escape`
# against `claude plugin list` output) can yield a false negative for a plugin
# not in that listing -- so a run reinstalls it, triggering the truncation. The
# fix is defense in depth:
# snapshot .claude.json before the call, restore if it shrinks >50% from a
# baseline of >=10 KB. Complementary to SDD-021's session-start canary at
# claude-session-start.ps1 (same 10240 threshold, same upstream issue).

@test "setup-windows.ps1 defines Backup-AndRestoreClaudeJson helper (BUG-004)" {
    grep -q 'function Backup-AndRestoreClaudeJson' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 cites upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 uses 10240-byte sanity floor in the truncate guard (BUG-004)" {
    grep -qF '10240' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 wraps claude plugin install with Backup-AndRestoreClaudeJson (BUG-004)" {
    # Structural check: the helper invocation appears within 5 lines preceding
    # the `claude plugin install` call in the foreach loop.
    grep -B5 'claude plugin install' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 still preserves the upstream idempotence guard (BUG-004)" {
    # Defense in depth -- the wrapper does NOT replace the existing guard.
    grep -qF 'installedPlugins -match' "$PS1_SCRIPT"
    grep -qF 'claude plugin list' "$PS1_SCRIPT"
}

# --- BUG-011: extend the BUG-004 guard to every claude CLI call site ---
# BUG-004 covered only `claude plugin install`. The same upstream truncation
# (anthropics/claude-code#59870) fires on `claude mcp get`, `claude mcp add`, and
# `claude plugin list` because they go through the same deserialize-modify-
# serialize cycle. With ~9 MCP servers, each unwrapped setup run triggered
# ~18 chances of truncation. These asserts lock in the wrap on every call site.

@test "setup-windows.ps1 wraps claude mcp add with Backup-AndRestoreClaudeJson (BUG-011)" {
    grep -B15 'claude mcp add --transport' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 wraps claude mcp get with Backup-AndRestoreClaudeJson (BUG-011)" {
    # Same scriptblock typically wraps both mcp get and mcp add inside one iteration.
    grep -B15 'claude mcp get' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "setup-windows.ps1 wraps claude plugin list with Backup-AndRestoreClaudeJson (BUG-011)" {
    grep -B5 'claude plugin list' "$PS1_SCRIPT" | grep -q 'Backup-AndRestoreClaudeJson'
}

# --- MEM-002: retire claude-mem — plugin no longer installed on Windows ---
# claude-mem is no longer in the $plugins array and the thedotmack marketplace is
# no longer registered (ADR-016 Q2). setup-windows.ps1 instead ships an
# idempotent cleanup that uninstalls the plugin + prunes leftover dirs.

@test "setup-windows.ps1 no longer registers the thedotmack marketplace (MEM-002)" {
    refute_grep_fixed 'claude plugin marketplace add thedotmack/claude-mem' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 ships the idempotent claude-mem cleanup block (MEM-002)" {
    grep -qF 'claude plugin uninstall claude-mem@thedotmack' "$PS1_SCRIPT"
    grep -qF 'MEM-002' "$PS1_SCRIPT"
}

# --- BUG-013: install Obsidian CLI on Windows ---
# `obsidian-cli` (npm) provides the `obsidian` binary used by
# obs-cli.ps1 and the vault-health workflow. setup-windows.ps1 must install
# it idempotently via npm global (user scope, no admin), gated on npm
# availability so machines without Node.js gracefully skip.
# Note: package was previously @vorillaz/obsidian-cli (404), fixed to obsidian-cli.

@test "setup-windows.ps1 installs Obsidian CLI via npm (BUG-013)" {
    grep -qF "npm install -g 'obsidian-cli'" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 Obsidian CLI install is gated on npm + idempotent (BUG-013)" {
    # idempotence: skip if `obsidian` already on PATH
    grep -B10 "npm install -g 'obsidian-cli'" "$PS1_SCRIPT" | grep -q "Get-Command obsidian"
    # gating: only run if npm is available
    grep -B10 "npm install -g 'obsidian-cli'" "$PS1_SCRIPT" | grep -q "Get-Command npm"
}

# --- AI-014: OpenCode Windows bootstrap (mirror of AI-011 Linux side) ---
# setup-windows.ps1 must install OpenCode via winget SST.opencode (cleaner
# than the Linux curl-bash equivalent), deploy ai/opencode/opencode.jsonc
# reconcile-not-skip, and sync ai/opencode/commands/*.md to the user's
# OpenCode commands directory.

@test "setup-windows.ps1 installs OpenCode through packages.json (npm), not winget (AI-034, ADR-036)" {
    # Two channels coexisted on the work box (npm-global first on PATH + winget),
    # and the winget loop's version parse accepted "locked." as a version.
    refute_grep_fixed 'SST.opencode' "$PS1_SCRIPT"
    refute_grep_fixed 'OPENCODE_VERSION' "$PS1_SCRIPT"
    refute_grep_fixed "ContainsKey('Version')" "$PS1_SCRIPT"
    jq -e '.tools[] | select(.name=="opencode" and .source.type=="npm")' "$DOTFILES_DIR/packages.json" >/dev/null
    grep -qF 'dotf tools install' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys opencode.jsonc via Deploy-File helper (SDD-007)" {
    # Post-SDD-007: the inline Copy-Item + Get-FileHash block was extracted
    # into Deploy-File in scripts/utils.ps1 (atomic + idempotent by SHA256).
    # Post-SDD-009: Deploy-File now operates on the staged-substituted tmp
    # file ($opencodeConfigTmp) rather than the raw source.
    grep -qF 'opencode.jsonc' "$PS1_SCRIPT"
    grep -qE 'Deploy-File.*opencodeConfigTmp' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 opencode deploy renders via dotf secrets render (SDD-009/#587)" {
    grep -qE 'dotf secrets render \$opencodeConfigTmp' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 deploys skills from records via Deploy-SkillRecord (SDD-008, AI-014 successor)" {
    # opencode commands (and claude/agy skills) are now rendered from the committed
    # vault skill records by Deploy-SkillRecord, honoring per-skill targets[], with
    # stale outputs pruned -- replacing the former ai\opencode\commands copy loop.
    grep -qF 'function Deploy-SkillRecord' "$PS1_SCRIPT"
    grep -qF 'Deploy-SkillRecord -DotfilesDir' "$PS1_SCRIPT"
    grep -qF 'Test-SkillTargetsAgent' "$PS1_SCRIPT"
    grep -qF 'harness\manifest.json' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 no longer deploys skills from the removed ai\\skills tree (SDD-008)" {
    # the ai\skills / ai\opencode\commands references that remain are historical
    # comments in the Deploy-SkillRecord block, never active Test-Path sources.
    refute_grep '\$skillsSource\s*=\s*"\$DotfilesDir\\ai\\skills"' "$PS1_SCRIPT"
    refute_grep '\$opencodeCmdsSrc\s*=' "$PS1_SCRIPT"
}

@test "Convert-SkillRecord does not stack a second set of generated_* fields on a record that already carries its own (HARNESS-069)" {
    # A previous adversarial review of HARNESS-069 (bash side) found this bug
    # live on this exact function: it never got the strip-rule fix, so once a
    # committed record carries its own generated_* provenance, every Windows
    # skill/command deploy produced duplicate YAML keys. A source-text grep
    # cannot catch this class -- it has to run the real function.
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi

    local scratch="$BATS_TEST_TMPDIR/convert-skill-record"
    mkdir -p "$scratch"
    local record="$scratch/SKILL.md"
    # Shaped like a record HARNESS-069's refresh already stamped: its own
    # generated_* fields already present, pointing at the vault.
    printf -- '---\ngenerated: true\ngenerated_from: 00_meta/skills/demo/SKILL.md\ngenerated_sha: aaaaaaaaaaaaaaaa\nname: demo\ndescription: fixture skill.\n---\n\nBody.\n' \
        > "$record"

    # Extract just this function's body (CRLF-stripped) so the test exercises
    # a script excerpt rather than dot-sourcing (and running) the whole
    # multi-hundred-line deploy script.
    local fn_body
    fn_body="$(sed 's/\r$//' "$PS1_SCRIPT" | awk '/^function Convert-SkillRecord \{/,/^\}$/')"
    [ -n "$fn_body" ]

    local out="$scratch/out.md"
    run pwsh -NonInteractive -Command "
        $fn_body
        \$result = Convert-SkillRecord -Kind 'skill' -RecordMd '$(_winpath "$record")' -SrcPath '.claude/skills/demo/SKILL.md'
        Set-Content -LiteralPath '$(_winpath "$out")' -Value \$result -Encoding UTF8
    "
    [ "$status" -eq 0 ]

    [ "$(grep -c '^generated:' "$out")" -eq 1 ]
    [ "$(grep -c '^generated_from:' "$out")" -eq 1 ]
    [ "$(grep -c '^generated_sha:' "$out")" -eq 1 ]
    # The fresh set must win -- it must point at the deploy source, not the
    # stale vault path the record's own (stripped) fields carried.
    grep -q '^generated_from: .claude/skills/demo/SKILL.md' "$out"
    grep -q '^name: demo' "$out"
}

# --- BUG-005: Windows PowerShell 5.1 auto-reexec under pwsh ---
# SDD-002 (PR #51) introduced Merge-ClaudeSettings which uses
# `ConvertFrom-Json -AsHashtable` -- a parameter added in PowerShell 7.0 that
# does NOT exist in Windows PowerShell 5.1. When the user invokes
# `PowerShell -ExecutionPolicy Bypass -File .\setup-windows.ps1` (Windows
# default `PowerShell` resolves to 5.1.x), the inner try/catch swallows the
# ParameterBindingException as if it were a JSON parse error and the per-key
# merge from ai/claude/settings.json is silently skipped. Preamble at the top
# of the script detects PSVersion.Major < 7 and re-execs under pwsh, or fails
# loud with winget install hint if pwsh is missing.

@test "setup-windows.ps1 detects PowerShell version major (BUG-005)" {
    grep -qF 'PSVersionTable.PSVersion.Major' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 re-execs under pwsh when PS < 7 (BUG-005)" {
    grep -qF 'Get-Command pwsh' "$PS1_SCRIPT"
    grep -qF '$PSCommandPath' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 fails loud with winget hint when pwsh missing (BUG-005)" {
    grep -qF 'winget install Microsoft.PowerShell' "$PS1_SCRIPT"
    grep -qE 'exit\s+1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 cites BUG-005 in the preamble inline comment" {
    grep -qF 'BUG-005' "$PS1_SCRIPT"
}

@test "negative parity: setup-linux.sh does not reference PSVersion (BUG-005 is Windows-only)" {
    refute_grep_fixed 'PSVersion' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 deploys SSH config" {
    grep -q 'Setting up SSH config' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 configures PATH" {
    grep -q 'Configuring PATH' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has summary section" {
    grep -q 'Setup Complete' "$PS1_SCRIPT"
}

# --- Developer tools ---

@test "setup-windows.ps1 has developer tools section" {
    grep -q 'DEVELOPER TOOLS' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 checks for winget" {
    grep -q 'Get-Command winget' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs age via winget" {
    grep -q 'FiloSottile.age' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs eza via winget" {
    grep -q 'eza-community.eza' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs jq via winget" {
    grep -q 'jqlang.jq' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs GitHub CLI via winget" {
    grep -q 'GitHub.cli' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs zoxide via winget" {
    grep -q 'ajeetdsouza.zoxide' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs uv if missing" {
    grep -q 'Installing uv' "$PS1_SCRIPT"
    grep -q 'astral.sh/uv/install.ps1' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 installs poetry via uv" {
    grep -q 'Installing poetry' "$PS1_SCRIPT"
    grep -q 'uv tool install poetry' "$PS1_SCRIPT"
}

# --- Cross-platform parity: developer tools ---

@test "parity: both scripts install uv" {
    grep -q 'Installing uv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'Installing uv' "$PS1_SCRIPT"
}

@test "parity: both scripts install poetry via uv" {
    grep -q 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'uv tool install poetry' "$PS1_SCRIPT"
}

@test "parity: both scripts install age" {
    grep -q 'command -v age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'FiloSottile.age' "$PS1_SCRIPT"
}

@test "parity: both scripts install eza" {
    grep -q 'command -v eza' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza-community.eza' "$PS1_SCRIPT"
}

@test "parity: both scripts install jq" {
    grep -q 'command -v jq' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jqlang.jq' "$PS1_SCRIPT"
}

@test "parity: both scripts install gh" {
    grep -q 'command -v gh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'GitHub.cli' "$PS1_SCRIPT"
}

@test "parity: both scripts install zoxide" {
    grep -q 'command -v zoxide' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'ajeetdsouza.zoxide' "$PS1_SCRIPT"
}

# --- Cross-platform parity: AI config deploy verification ---
# AI-013 refactored CLAUDE.md/GEMINI.md/copilot-instructions.md to pointer-style
# files starting with 'First, read `AGENTS.md`'. The old marker 'CORE PRINCIPLE'
# no longer exists in any deployed file. setup-linux.sh was updated; this guard
# keeps setup-windows.ps1 in lockstep (regression class: BUG-001 + BUG-002).

@test "parity: both scripts verify CLAUDE.md deploy with AGENTS.md pointer marker" {
    grep -qF "grep -q 'First, read \`AGENTS.md\`' \"\$HOME/.claude/CLAUDE.md\"" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "Select-String -Path \"\$ClaudeHome\\CLAUDE.md\" -Pattern 'First, read \`AGENTS.md\`'" "$PS1_SCRIPT"
}

@test "parity: both scripts verify AGY.md deploy with AGENTS.md pointer marker (SDD-007)" {
    # Post-SDD-007: legacy GEMINI.md compat write removed; AGY.md is the
    # canonical pointer file for the agy CLI. Both setup scripts deploy it
    # and assert the AGENTS.md pointer marker on every run.
    grep -qF 'First, read `AGENTS.md`' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "AGY.md" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'First, read `AGENTS.md`' "$PS1_SCRIPT"
    grep -qF "AGY.md" "$PS1_SCRIPT"
}

@test "setup-windows.ps1 has no stale 'CORE PRINCIPLE' verify-pattern references" {
    # AGENTS.md itself may mention 'CORE PRINCIPLE' as documentation text -- that
    # is fine. The bug is using it as a Select-String -Pattern, which post-AI-013
    # never matches the deployed file and emits a spurious [ERROR] every run.
    refute_grep_fixed '-Pattern "CORE PRINCIPLE"' "$PS1_SCRIPT"
}

@test "parity: both setup scripts detect copilot via binary, not gh extension" {
    grep -qF "command -v copilot" "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "Get-Command copilot" "$PS1_SCRIPT"
    # Neither should still reference the legacy extension path
    refute_grep_fixed "github/gh-copilot" "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed "github/gh-copilot" "$PS1_SCRIPT"
}

# --- SDD-002: settings.json template merge ---
# Both setup scripts read ai/claude/settings.json as SSOT for the dotfiles-owned
# subset of ~/.claude/settings.json. Per-key merge policy documented in
# specs/SDD-002-settings-portability/proposal.md.

@test "SDD-002: setup-windows.ps1 defines Merge-ClaudeSettings function" {
    grep -qE '^function Merge-ClaudeSettings' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 calls Merge-ClaudeSettings (not inline hashtable)" {
    grep -qF 'Merge-ClaudeSettings -TemplatePath' "$PS1_SCRIPT"
    # The legacy inline hook hashtable must be gone
    refute_grep '\$hookEntry\s*=\s*@\{' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 references the template path ai\\claude\\settings.json" {
    grep -qF 'ai\claude\settings.json' "$PS1_SCRIPT"
}

@test "SDD-002: setup-windows.ps1 bulk copy of ai/claude/* excludes settings.json" {
    # -- separator before pattern starting with dash so grep does not parse it
    # as a flag (same fix pattern as the BUG-002 CORE PRINCIPLE assert).
    grep -qF -- "-Exclude 'settings.json'" "$PS1_SCRIPT"
}

@test "SDD-002: setup-linux.sh defines merge_claude_settings function" {
    grep -qE '^merge_claude_settings\(\)' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh calls merge_claude_settings (not inline HOOK_ENTRY)" {
    grep -qF 'merge_claude_settings "$CLAUDE_SETTINGS_TEMPLATE"' "$DOTFILES_DIR/setup-linux.sh"
    # The legacy inline HOOK_ENTRY jq -n heredoc must be gone
    refute_grep_fixed 'HOOK_ENTRY=$(jq -n' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh references the template path ai/claude/settings.json" {
    grep -qF 'ai/claude/settings.json' "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: setup-linux.sh bulk copy of ai/claude/* skips settings.json" {
    grep -qF "= \"settings.json\" ] && continue" "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: parity -- both scripts log the bootstrap message" {
    grep -qF "Bootstrapping ~/.claude/settings.json from template" "$PS1_SCRIPT"
    grep -qF "Bootstrapping ~/.claude/settings.json from template" "$DOTFILES_DIR/setup-linux.sh"
}

@test "SDD-002: parity -- both scripts substitute __HOOK_COMMAND__ placeholder" {
    grep -qF '__HOOK_COMMAND__' "$PS1_SCRIPT"
    grep -qF 'SessionStart[0].hooks[0].command) = $cmd' "$DOTFILES_DIR/setup-linux.sh"
}

# --- CLI-019: diff-check retired; `dotf doctor` owns repo/deploy drift ---
# The diff-check.{sh,ps1} twins were deleted once `dotf doctor` (PR-A, #513)
# absorbed the byte-compare. setup must no longer deploy diff-check.ps1, the
# `dch` convenience now wraps `dotf doctor`, and no production caller may
# reference diff-check.

@test "CLI-019: setup-windows.ps1 no longer deploys diff-check.ps1" {
    # The name may appear in WIN-013's removal list; never in a deploy or a call.
    refute_grep 'Copy-Item.*diff-check' "$PS1_SCRIPT"
    refute_grep '&.*diff-check' "$PS1_SCRIPT"
}

@test "CLI-019: powershell/profile.ps1 dch wraps dotf doctor" {
    grep -qF 'function dch' "$DOTFILES_DIR/powershell/profile.ps1"
    grep -A6 'function dch' "$DOTFILES_DIR/powershell/profile.ps1" | grep -qF 'dotf doctor'
    if grep -A6 'function dch' "$DOTFILES_DIR/powershell/profile.ps1" | grep -qF 'diff-check'; then
        echo "dch still references diff-check" >&2
        return 1
    fi
}

@test "CLI-019: production callers no longer reference diff-check" {
    [ ! -f "$DOTFILES_DIR/scripts/diff-check.sh" ]
    [ ! -f "$DOTFILES_DIR/scripts/diff-check.ps1" ]
    for f in setup-linux.sh powershell/profile.ps1 \
             .zsh/aliases.zsh .github/workflows/ci.yml; do
        if grep -qF 'diff-check' "$DOTFILES_DIR/$f"; then
            echo "diff-check still referenced in $f" >&2
            return 1
        fi
    done
    # setup-windows.ps1 names it only in WIN-013's removal list (guarded above).
    refute_grep 'Copy-Item.*diff-check' "$PS1_SCRIPT"
}

# --- CLI-018 (#509): healthcheck.ps1 + doctor.ps1 retired; dotf doctor owns it ---
# Both Windows .ps1 diagnostics were deleted once dotf doctor reached parity
# (the §4 residual was ported by #522). The files must be gone and no production
# caller may reference them -- setup, CI, the profile hc alias, and the
# SessionStart hook all repoint to dotf doctor.

@test "CLI-018: healthcheck.ps1 and doctor.ps1 are deleted" {
    [ ! -f "$DOTFILES_DIR/scripts/healthcheck.ps1" ]
    [ ! -f "$DOTFILES_DIR/scripts/doctor.ps1" ]
}

@test "CLI-018: no production caller references healthcheck.ps1 or doctor.ps1" {
    for f in setup-linux.sh powershell/profile.ps1 \
             .github/workflows/ci.yml; do
        if grep -qE '(healthcheck|doctor)\.ps1' "$DOTFILES_DIR/$f"; then
            echo "(healthcheck|doctor).ps1 still referenced in $f" >&2
            return 1
        fi
    done
    # setup-windows.ps1 names them only in WIN-013's removal list: never deployed or called.
    refute_grep 'Copy-Item.*(healthcheck|doctor)\.ps1' "$PS1_SCRIPT"
    refute_grep '&.*(healthcheck|doctor)\.ps1' "$PS1_SCRIPT"
}

# --- CLI-018: dotf doctor as the post-setup diagnostic (replaced WIN-001) ---

@test "CLI-018: setup-windows.ps1 runs dotf doctor post-setup, non-fatal" {
    grep -qF 'dotf doctor' "$PS1_SCRIPT"
    grep -qF 'Running post-setup dotf doctor' "$PS1_SCRIPT"
    # non-fatal: the failure path warns, never hard-exits.
    grep -qF 'dotf doctor reported one or more issues' "$PS1_SCRIPT"
}

# --- BUG-021: profile-section fail-fast (root cause of BUG-020 accumulation) ---
# Section 4 "DEPLOY POWERSHELL PROFILE" used to swallow OOM/-replace failures and
# print "[SUCCESS] Updated dotfiles section in PowerShell profile" regardless. On
# a corrupted 26 MB profile this allowed silent partial writes to compound until
# the deployed file hit 689K lines and every shell startup hit a parser error.
# The contract these tests pin: the profile-section block must (a) fail-fast on
# I/O error, (b) preflight known corruption signals (size, marker count), and
# (c) point users at the recovery script (profile-heal.ps1 from BUG-020).

@test "BUG-021: profile-section block is wrapped in try/catch with exit 1 on failure" {
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\btry\s*\{'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\bcatch\s*\{'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'exit\s+1'
}

@test "BUG-021: profile-section I/O uses -ErrorAction Stop (terminating errors only)" {
    # Without -ErrorAction Stop, Get-Content/Set-Content emit non-terminating
    # errors that bypass try/catch. Stop promotes them so the catch fires.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Get-Content[^|]*-ErrorAction Stop'
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Set-Content[^|]*-ErrorAction Stop'
}

@test "BUG-021: profile preflight aborts when existing profile size > 1 MB" {
    # Healthy profile is ~7 KB. 1 MB is ~140x — a reliable corruption signal.
    # Abort BEFORE Get-Content -Raw OOMs on the 26 MB nightmare scenario.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\.Length\s+-gt\s+1MB'
}

@test "BUG-021: profile preflight aborts when marker count > 2 (BUG-020 corruption signal)" {
    # A healthy profile has exactly 1 start + 1 end marker pair. Anything > 2
    # of either marker means a previous setup run left orphan blocks. Refuse to
    # touch it — defer to profile-heal.ps1 (BUG-020).
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'Select-String|-AllMatches|Matches\('
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE 'markerCount|MarkerCount'
}

@test "BUG-021: preflight error message points at profile-heal.ps1 (BUG-020 recovery handoff)" {
    # When preflight aborts, the user needs a one-line escape hatch. The
    # recovery script (BUG-020) is named profile-heal.ps1 in scripts/.
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qF 'profile-heal.ps1'
}

@test "parity: both setups exit non-zero on profile-write failure" {
    # Linux: setup-linux.sh:11 already has 'set -euo pipefail' which propagates
    # deploy_file's non-zero exits. Asserts the safety net is still in place.
    grep -qE '^set -euo pipefail' "$DOTFILES_DIR/setup-linux.sh"
    # Windows: try/catch + exit 1 in the profile-section block (covered structurally
    # by BUG-021 test #1; this parity test asserts both legs of the contract exist).
    awk '/4\. DEPLOY POWERSHELL PROFILE/,/5\. DEPLOY GIT CONFIGURATION/' "$PS1_SCRIPT" \
        | grep -qE '\bcatch\s*\{'
}

# --- PSScriptAnalyzer ---

@test "setup-windows.ps1 passes PSScriptAnalyzer (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$ErrorActionPreference = 'Stop'
        try {
            Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
            \$results = Invoke-ScriptAnalyzer -Path '$(_winpath "$PS1_SCRIPT")' -Settings '$(_winpath "$DOTFILES_DIR/.PSScriptAnalyzerSettings.psd1")' -Severity Error,Warning
            if (\$results) {
                \$results | Format-Table -AutoSize
                exit 1
            }
            Write-Host 'PSScriptAnalyzer: OK'
        } catch {
            Write-Warning \"PSScriptAnalyzer not available: \$_\"
            exit 0
        }
    "
    [[ "$status" -eq 0 ]]
}

@test "setup-windows.ps1 valid PowerShell syntax (if pwsh available)" {
    if ! command -v pwsh >/dev/null 2>&1; then
        skip "pwsh not available"
    fi
    run pwsh -NonInteractive -Command "
        \$errors = \$null
        [System.Management.Automation.Language.Parser]::ParseFile(
            '$(_winpath "$PS1_SCRIPT")', [ref]\$null, [ref]\$errors
        ) | Out-Null
        if (\$errors) {
            \$errors | ForEach-Object { Write-Error \$_.Message }
            exit 1
        }
        Write-Host 'Syntax OK'
    "
    [[ "$status" -eq 0 ]]
}

# --- DX-004: opencode tui.json deploy (Linux parity) ---

@test "setup-windows.ps1 deploys opencode tui.json (DX-004 AC4)" {
    grep -qF 'ai\opencode\tui.json' "$PS1_SCRIPT"
    grep -qF '.config\opencode\tui.json' "$PS1_SCRIPT"
}

@test "setup-windows.ps1 mirrors the harness inputs through dotf harness mirror (WIN-007)" {
    # ~/.dotfiles/harness never existed on Windows, so `dotf doctor` failed the
    # routing registry and the model-pin checks after every setup, with a
    # remedy ("re-run setup") that could not clear them (#1288).
    grep -qF 'dotf harness mirror' "$PS1_SCRIPT"
}

@test "parity: both setups mirror the harness through the same dotf command (WIN-007)" {
    grep -qF 'dotf harness mirror' "$PS1_SCRIPT"
    grep -qF 'dotf harness mirror' "$DOTFILES_DIR/setup-linux.sh"
}

# CLI-054 (#1301): bare `dotf deploy` installs every config ai/deploy.json
# declares. `& dotf deploy pi` left orca-keybindings declared and never
# installed on Windows -- the manifest is the SSOT of what gets deployed, and a
# call site that names one entry silently narrows it.
@test "setup-windows.ps1 deploys agent configs with bare dotf deploy, naming no config (CLI-054)" {
    # [[:space:]]*$ rather than \s*$: the .ps1 checks out CRLF everywhere
    # (.gitattributes), and this grep has no \r escape.
    grep -qE '^[[:space:]]*& dotf deploy[[:space:]]*$' "$PS1_SCRIPT"
    # Anchored to the invocation shape (a line that RUNS dotf), not to any
    # mention: the comment above the call names the old form on purpose.
    refute_grep '^[[:space:]]*& dotf deploy [a-z]' "$PS1_SCRIPT"
}

@test "parity: neither setup names a config at its dotf deploy call site (CLI-054)" {
    refute_grep '^[[:space:]]*& dotf deploy [a-z]' "$PS1_SCRIPT"
    refute_grep '^[[:space:]]*("\$_dotf"|dotf) deploy [a-z]' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 writes rendered skill files and the copilot catalog through Write-Utf8LfFile, never Set-Content (WIN-008)" {
    # Set-Content joins with CRLF and rewrites the whole file, so the deployed
    # copilot-instructions.md drifted from its LF source on every run (#1289).
    grep -qF 'Write-Utf8LfFile -Path $catFile' "$PS1_SCRIPT"
    run grep -qF 'Set-Content -LiteralPath $catFile' "$PS1_SCRIPT"
    [ "$status" -ne 0 ]
    run grep -cF 'Write-Utf8LfFile -Path' "$PS1_SCRIPT"
    [ "$output" -ge 3 ]
}

@test "setup-windows.ps1 mirrors sensitive/ recursively so dr/ reaches the deploy dir (WIN-011)" {
    grep -qF "Copy-Item -Path (Join-Path \$sensitiveSource '*') -Destination \$sensitiveDest -Recurse -Force" "$PS1_SCRIPT"
}

@test "utils.ps1 and profile.ps1 set a UTF-8 console encoding (WIN-009)" {
    grep -qF '[Console]::OutputEncoding = ' "$DOTFILES_DIR/scripts/utils.ps1"
    grep -qF '[Console]::OutputEncoding = ' "$DOTFILES_DIR/powershell/profile.ps1"
}

@test "setup-windows.ps1 provisions Node.js before dotf tools install, the npm channel's prerequisite (ADR-036)" {
    grep -qF 'Id = "OpenJS.NodeJS.LTS"' "$PS1_SCRIPT"
    # The .ps1 is CRLF (.gitattributes): [[:space:]]* absorbs the CR before the
    # end anchor on every OS, and GNU grep reads no CR escape anyway.
    node_line=$(grep -n 'Id = "OpenJS.NodeJS.LTS"' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    tools_line=$(grep -nE '^    dotf tools install[[:space:]]*$' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [ -n "$node_line" ] && [ -n "$tools_line" ]
    [ "$node_line" -lt "$tools_line" ]
}

@test "every registry PATH rebuild in setup-windows.ps1 goes through Sync-SessionPath (TEST-003)" {
    # A registry-only rebuild drops entries that exist only in this process (a
    # GITHUB_PATH addition; the runner's toolcache node), and the plain
    # Machine+User+process concatenation that first fixed that doubled PATH on
    # every call (153 entries, 72 duplicates, measured on #1308). One helper,
    # deduplicated, unit-tested in tests/windows-path-sync.Tests.ps1.
    run grep -cF 'GetEnvironmentVariable("PATH", "Machine")' "$PS1_SCRIPT"
    [ "$output" = "0" ]
    calls=$(grep -cE '^\s*Sync-SessionPath\s*$' "$PS1_SCRIPT")
    [ "$calls" -ge 2 ]
    grep -qE '^function Sync-SessionPath' "$BATS_TEST_DIRNAME/../scripts/utils.ps1"
}

@test "setup-windows.ps1 runs the uv and Bun installers in a child pwsh, never in its own session (TEST-003)" {
    # A downloaded installer executed in-session can rewrite the process
    # environment for the rest of setup; the runner lost npm after these two.
    grep -qF -- '-File $uvInstaller' "$PS1_SCRIPT"
    grep -qF -- '-File $bunInstaller' "$PS1_SCRIPT"
    run grep -F '& $uvInstaller' "$PS1_SCRIPT"
    [ "$status" -ne 0 ]
    run grep -F '& $bunInstaller' "$PS1_SCRIPT"
    [ "$status" -ne 0 ]
}

# OPS-044 (#1361): hive-vault and pdf-modifier are `uvx` MCP servers, and the
# registration loop skips a server whose prerequisite binary is absent. With the
# uv installer in section 4 a clean box's first run registered neither and only
# the second run converged -- measured on the CI runner the first time Claude
# Code installed there (#1360). Ordering is asserted by line number of the
# invocation, on both OSes.
@test "setup-windows.ps1 installs uv before registering the Claude MCP servers (OPS-044, #1361)" {
    uv_line=$(grep -n 'astral.sh/uv/install.ps1' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    # The executable invocation, never a comment naming it: the block's own
    # comments mention `claude mcp add` 40 lines above the call.
    mcp_line=$(grep -nE '^[[:space:]]*[^#[:space:]].*claude mcp add --transport' "$PS1_SCRIPT" | head -1 | cut -d: -f1)
    [ -n "$uv_line" ] && [ -n "$mcp_line" ] && [ "$uv_line" -lt "$mcp_line" ]
}

@test "parity: both setups install uv before registering the Claude MCP servers (OPS-044)" {
    uv_line=$(grep -n 'astral.sh/uv/install.sh' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    mcp_line=$(grep -nE '^[[:space:]]*[^#[:space:]].*claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$uv_line" ] && [ -n "$mcp_line" ] && [ "$uv_line" -lt "$mcp_line" ]
}
