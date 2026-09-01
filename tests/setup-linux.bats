#!/usr/bin/env bats
# Tests for setup-linux.sh

load 'lib/refute'

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "setup-linux.sh valid bash syntax" {
    bash -n "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh valid zsh syntax" {
    zsh -n "$DOTFILES_DIR/setup-linux.sh"
}

# Behavioral guard for #695: cloning INTO ~/.dotfiles (CURRENT_DIR ==
# DOTFILES_DIR) must fail fast, BEFORE any destructive copy/mirror, not corrupt
# the checkout. Minimal fixture: setup-linux.sh + utils.sh under $HOME/.dotfiles.
@test "setup-linux.sh refuses an in-place install (CURRENT_DIR == DOTFILES_DIR) (#695)" {
    local tmp; tmp="$(mktemp -d)"
    mkdir -p "$tmp/.dotfiles/scripts"
    cp "$DOTFILES_DIR/setup-linux.sh" "$tmp/.dotfiles/setup-linux.sh"
    cp "$DOTFILES_DIR/scripts/utils.sh" "$tmp/.dotfiles/scripts/utils.sh"
    run env HOME="$tmp" bash -c "cd '$tmp/.dotfiles' && bash ./setup-linux.sh"
    rm -rf "$tmp"
    [ "$status" -eq 1 ]
    [[ "$output" == *"in-place"* ]]
    [[ "$output" == *"dotfiles-repo"* ]]
}

# --- Developer tools section ---

@test "setup-linux.sh creates ~/.local/bin directory" {
    grep -q 'ensure_directory.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh adds ~/.local/bin to PATH" {
    grep -q 'export PATH.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs uv if missing" {
    grep -q 'command -v uv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'astral.sh/uv/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs poetry via uv" {
    grep -q 'command -v poetry' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs age if missing" {
    grep -q 'command -v age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'FiloSottile/age' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'AGE_VERSION' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs eza if missing" {
    grep -q 'command -v eza' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza.*linux.*tar.gz' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs jq if missing" {
    grep -q 'command -v jq' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq-linux-amd64' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh deploys secrets/registry.yaml (dotf secrets mapping SSOT) [#587]" {
    grep -qF 'DOTFILES_DIR/secrets' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'secrets/registry.yaml' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh resolves the agy deploy-time secret via dotf, not the load-secrets twin [#587]" {
    # agy's OPENROUTER_API_KEY is fetched via dotf (opencode/pi materialize via
    # `dotf secrets render` over the registry, and their own runtime resolver).
    grep -qF 'dotf secrets show OPENROUTER_API_KEY' "$DOTFILES_DIR/setup-linux.sh"
    # the eager-source + the old secrets_show twin API are gone
    refute_grep 'load-secrets\.sh" >/dev/null 2>&1' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'secrets_show ' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs gh if missing" {
    grep -q 'command -v gh' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'cli/cli/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs zoxide if missing" {
    grep -q 'command -v zoxide' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide.*install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs direnv if missing" {
    grep -q 'command -v direnv' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv.net/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs shellcheck if missing" {
    grep -q 'command -v shellcheck' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'ShellCheck/releases' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs bats if missing" {
    grep -q 'command -v bats' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats-core/bats-core' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh skips tools already installed" {
    grep -q 'uv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'poetry already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'age already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'eza already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'jq already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'gh already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'zoxide already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'direnv already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'shellcheck already installed' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'bats already installed' "$DOTFILES_DIR/setup-linux.sh"
}

# --- tmux integration ---

@test "setup-linux.sh copies tmux.conf into deploy dir" {
    grep -qE 'safe_copy "\$CURRENT_DIR/tmux\.conf" "\$DOTFILES_DIR/"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh copies packages.json into deploy dir (CLI-029 tool catalog)" {
    grep -qE 'safe_copy "\$CURRENT_DIR/packages\.json" "\$DOTFILES_DIR/"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs catalog tools via dotf (best-effort)" {
    grep -q 'dotf tools install' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh deploys tmux.conf to ~/.tmux.conf via deploy_file (SDD-007)" {
    grep -qE 'deploy_file "\$DOTFILES_DIR/tmux\.conf" "\$HOME/\.tmux\.conf"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh checks for tmux presence" {
    grep -qE 'command -v tmux' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh tells user how to install tmux when missing" {
    grep -qE 'sudo apt install -y tmux' "$DOTFILES_DIR/setup-linux.sh"
}

# --- Session hook registration (issue #20 prevention) ---
#
# HARNESS-045 AC1 moved the SSOT. The hook command used to be a literal in this
# script (EXPECTED_HOOK_COMMAND) and is now declared in harness/manifest.json,
# emitted by `dotf harness bind`. These guards follow it: this file asserts the
# script still RUNS the emitter on every setup, and the manifest asserts WHAT it
# emits. Asserting a literal here again would recreate the second writer.

@test "setup-linux.sh emits the harness hooks by running dotf harness bind" {
    grep -qF 'harness bind' "$DOTFILES_DIR/setup-linux.sh"
}

@test "the manifest declares the session hooks setup used to hardcode" {
    local manifest="$DOTFILES_DIR/harness/manifest.json"
    jq -e '.agents.bind[] | select(.agent == "claude") | .emit_hooks
           | (map(select(.event == "SessionStart" and .command == "mem session-start")) | length == 1)
           and (map(select(.event == "SessionEnd" and .command == "mem session-end")) | length == 1)' \
        "$manifest"
}

# Hook registration must self-heal -- never trust "an entry exists" to mean "the
# entry is correct". bind runs unconditionally on every setup and replaces OUR
# entry in place when its command has drifted, which is the same guarantee the
# old unconditional rewrite gave, minus the collateral damage to third parties.
@test "setup-linux.sh runs bind unconditionally, not only when hooks are absent" {
    # The call sits behind a capability probe (is dotf new enough) and nothing
    # else -- notably not behind a "does the file already have hooks" test.
    refute_grep_fixed 'if.*hooks.*then.*harness bind' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF "grep -q '^[[:space:]]*bind[[:space:]]'" "$DOTFILES_DIR/setup-linux.sh"
}

# CLI-025 cutover guard: no deploy/registration file may deploy or invoke the
# retired session-start shell cluster (claude-session-start.{sh,ps1}, session-brief.sh,
# ensure-memory-symlink.sh) — the Go `dotf mem session-start` adapter replaced it and
# the scripts are git-rm'd. Path-anchored so Go provenance comments don't false-match.
@test "setup: no deploy/registration file invokes the retired session-start scripts" {
    for f in setup-linux.sh setup-windows.ps1 ai/claude/settings.json .github/workflows/ci.yml; do
        if grep -qE 'scripts[\\/](claude-session-start|session-brief|ensure-memory-symlink)' "$DOTFILES_DIR/$f"; then
            echo "retired session-start script still deployed/invoked in $f" >&2
            return 1
        fi
    done
}

# --- MCP server registration (SSOT + idempotence) ---

@test "mcp-servers.json exists and is valid JSON with at least one server" {
    [ -f "$DOTFILES_DIR/mcp-servers.json" ]
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.servers | length > 0' "$DOTFILES_DIR/mcp-servers.json"
        [ "$status" -eq 0 ]
    fi
}

# MCP server list must live in mcp-servers.json, not hardcoded in setup-linux.sh.
@test "setup-linux.sh MCP registration reads from mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'claude mcp add --transport (stdio|http) (drawio|socket|context7|sequential-thinking|hive)' "$DOTFILES_DIR/setup-linux.sh"
}

# MCP registration must check existence before adding, not blindly retry.
@test "setup-linux.sh MCP registration checks existence with claude mcp get" {
    grep -q 'claude mcp get' "$DOTFILES_DIR/setup-linux.sh"
}

# Both setup scripts must read the same SSOT.
@test "parity: both setup scripts source mcp-servers.json" {
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'mcp-servers\.json' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- BUG-020: DOTFILES_REPO_DIR cross-OS export parity ---
# .bashrc + .zshrc export it; powershell/profile.ps1 was missing it.
# Required by `dotf doctor` (CLI-019 repo/deploy drift check) to locate the
# repo root. Without it, the drift check cannot resolve the repo and SKIPs.

@test "parity: all 3 profiles export DOTFILES_REPO_DIR (BUG-020)" {
    grep -qF 'export DOTFILES_REPO_DIR=' "$DOTFILES_DIR/.bashrc"
    grep -qF 'export DOTFILES_REPO_DIR=' "$DOTFILES_DIR/.zshrc"
    grep -qF '$env:DOTFILES_REPO_DIR' "$DOTFILES_DIR/powershell/profile.ps1"
}

@test "env-contract.json declares DOTFILES_REPO_DIR (BUG-020)" {
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.env_vars | map(select(.name == "DOTFILES_REPO_DIR")) | length == 1' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
    fi
}

# --- BUG-004: defense-in-depth around claude plugin install (truncate guard) ---
# Linux mirror of the Windows guard. Every `claude plugin install` call triggers
# upstream anthropics/claude-code#59870, dropping subscription fields out of
# ~/.claude/.claude.json. The bash idempotence guard (`grep -qF` against
# `claude plugin list` output) can yield a false negative for a plugin not in
# that listing -- so a run reinstalls it, truncating .claude.json from ~75 KB to
# ~1.5 KB. Defense in depth: snapshot before the call, restore if it shrinks
# >50% from a baseline of >=10 KB.

@test "setup-linux.sh defines snapshot_claude_json + restore_claude_json_if_truncated (BUG-004)" {
    grep -q 'snapshot_claude_json()' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'restore_claude_json_if_truncated()' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh cites upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh uses 10240-byte sanity floor in the truncate guard (BUG-004)" {
    grep -qF '10240' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh wraps claude plugin install with snapshot+restore (BUG-004)" {
    # snapshot called before, restore called after, both within the foreach loop body.
    grep -B5 'claude plugin install "\$plugin"' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A10 'claude plugin install "\$plugin"' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "setup-linux.sh still preserves the upstream idempotence guard (BUG-004)" {
    # Defense in depth -- the wrapper does NOT replace the existing guard.
    grep -qF 'grep -qF "$plugin"' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'claude plugin list' "$DOTFILES_DIR/setup-linux.sh"
}

@test "parity: both setup scripts cite upstream issue 59870 in the truncate guard (BUG-004)" {
    grep -qF '#59870' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF '#59870' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "parity: both setup scripts wrap claude plugin install with a snapshot helper (BUG-004)" {
    # Windows uses the combined PS-idiomatic name; bash uses snapshot_/restore_ pair.
    grep -q 'Backup-AndRestoreClaudeJson' "$DOTFILES_DIR/setup-windows.ps1"
    grep -q 'snapshot_claude_json' "$DOTFILES_DIR/setup-linux.sh"
}

# --- BUG-011: extend the BUG-004 guard to every claude CLI call site ---
# BUG-004 covered only `claude plugin install`. The same upstream truncation
# (anthropics/claude-code#59870) fires on `claude mcp get`, `claude mcp add`, and
# `claude plugin list` because they go through the same deserialize-modify-
# serialize cycle. With ~9 MCP servers, each setup run unwrapped triggered
# ~18 chances of truncation. These asserts lock in the wrap on every call site.

@test "setup-linux.sh defines snapshot helpers BEFORE the MCP loop (BUG-011)" {
    # No forward references: helper definitions must precede first use.
    helper_line=$(grep -n '^snapshot_claude_json()' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    mcp_get_line=$(grep -n 'claude mcp get' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$helper_line" ] && [ -n "$mcp_get_line" ]
    [ "$helper_line" -lt "$mcp_get_line" ]
}

@test "setup-linux.sh wraps claude mcp add with snapshot+restore (BUG-011)" {
    # snapshot called within 15 lines before mcp add (header comments + mcp get
    # idempotence path live between); restore within 10 lines after.
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A10 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "setup-linux.sh wraps claude plugin list with snapshot+restore (BUG-011)" {
    grep -B5 'claude plugin list 2>/dev/null' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -A5 'claude plugin list 2>/dev/null' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'restore_claude_json_if_truncated'
}

@test "parity: both setup scripts wrap claude mcp add with the snapshot guard (BUG-011)" {
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -B15 'claude mcp add --transport' "$DOTFILES_DIR/setup-windows.ps1" | grep -q 'Backup-AndRestoreClaudeJson'
}

@test "parity: both setup scripts wrap claude plugin list with the snapshot guard (BUG-011)" {
    grep -B5 'claude plugin list' "$DOTFILES_DIR/setup-linux.sh" | grep -q 'snapshot_claude_json'
    grep -B5 'claude plugin list' "$DOTFILES_DIR/setup-windows.ps1" | grep -q 'Backup-AndRestoreClaudeJson'
}

# --- MEM-002: retire claude-mem — no longer installed; one-cycle cleanup runs ---
# claude-mem is no longer in the plugin install loop (ADR-016 Q2). Both setups
# instead ship an idempotent cleanup that uninstalls the plugin + prunes its
# leftover cache/marketplace dirs on the next run. These lock in BOTH the
# removal (no marketplace registration, plugin absent from the loop) and the
# presence of the cleanup block.

@test "setup scripts no longer register the thedotmack marketplace (MEM-002)" {
    refute_grep_fixed 'claude plugin marketplace add thedotmack/claude-mem' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'claude plugin marketplace add thedotmack/claude-mem' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "setup scripts ship the idempotent claude-mem cleanup block (MEM-002)" {
    grep -qF 'claude plugin uninstall claude-mem@thedotmack' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'claude plugin uninstall claude-mem@thedotmack' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'MEM-002' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'MEM-002' "$DOTFILES_DIR/setup-windows.ps1"
}

# --- doctor + env-contract.json (cross-OS parity) ---

@test "env-contract.json exists and is valid JSON with required sections" {
    [ -f "$DOTFILES_DIR/env-contract.json" ]
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.env_vars | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_path_entries.linux | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_path_entries.windows | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
        run jq -e '.required_binaries | length > 0' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
    fi
}

# CLI-012/CLI-018/ADR-021: doctor.sh and doctor.ps1 were both ported to
# `dotf doctor` (Go) and deleted; the check logic + --fix path live in
# cli/internal/doctor (go test).
@test "doctor.sh and doctor.ps1 are both retired (ported to dotf doctor)" {
    [ ! -f "$DOTFILES_DIR/scripts/doctor.sh" ]
    [ ! -f "$DOTFILES_DIR/scripts/doctor.ps1" ]
}

# Setup scripts run post-setup diagnostics before 'Setup Complete'. Both OSes
# now fold the old doctor + healthcheck blocks into one `dotf doctor` call
# (CLI-012 Linux, CLI-018 Windows).
@test "post-setup diagnostics: both linux and windows run dotf doctor" {
    grep -q 'dotf doctor' "$DOTFILES_DIR/setup-linux.sh"
    grep -q 'dotf doctor' "$DOTFILES_DIR/setup-windows.ps1"
}

# OPS-042 (#1336): obsidian-cli and yarn are packages.json tools (ADR-036),
# converged by `dotf tools install` on both OSes; neither setup script carries
# an npm block for them any more, and versions.conf no longer pins them.
@test "parity: obsidian and yarn are catalog tools, not setup-script npm blocks (OPS-042)" {
    [ "$(jq -r '.tools[] | select(.name=="obsidian") | "\(.source.type) \(.source.package) \(.version)"' "$DOTFILES_DIR/packages.json")" = "npm obsidian-cli 0.5.1" ]
    [ "$(jq -r '.tools[] | select(.name=="yarn") | "\(.source.type) \(.source.package) \(.version)"' "$DOTFILES_DIR/packages.json")" = "npm yarn 1.22.22" ]
    refute_grep 'obsidian-cli' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'obsidian-cli' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep 'npm install -g "yarn' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'npm install -g \$yarnPkg' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep '^OBSIDIAN_VERSION=' "$DOTFILES_DIR/versions.conf"
    refute_grep '^YARN_VERSION=' "$DOTFILES_DIR/versions.conf"
}

# MEM-002: the claude-mem install-state assertions that lived in
# cli/internal/doctor (checkClaudeMem / resolveClaudeMemHook) were removed with
# the rest of the claude-mem wiring (ADR-016 Q2). `dotf doctor` no longer probes
# for the plugin.

# CLI-012/CLI-018: both setups fold the old doctor + healthcheck blocks into ONE
# `dotf doctor` call (the consolidated sweep).
@test "setup-linux.sh runs dotf doctor non-fatally post-setup" {
    grep -q 'dotf doctor' "$DOTFILES_DIR/setup-linux.sh"
    # non-fatal: the failure path warns, never hard-exits.
    grep -q 'dotf doctor || log_warning' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-windows.ps1 runs dotf doctor post-setup (CLI-018)" {
    grep -qF 'dotf doctor' "$DOTFILES_DIR/setup-windows.ps1"
    grep -qF 'Running post-setup dotf doctor' "$DOTFILES_DIR/setup-windows.ps1"
}

# Required binaries in the contract must include min_version pins.
@test "env-contract.json pins min_version for required binaries" {
    if command -v jq >/dev/null 2>&1; then
        run jq -e '.required_binaries | all(has("min_version"))' "$DOTFILES_DIR/env-contract.json"
        [ "$status" -eq 0 ]
    fi
}

# CLI-018: doctor.ps1's min_version check + per-section summaries are covered by
# cli/internal/doctor (checkRequiredBinaries / report.go, go test) after the
# .ps1 was retired.

# Profiles must export the structural env vars declared in env-contract.json,
# so a fresh shell silences doctor without needing --fix every session.
@test ".zshrc exports DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q 'export DOTFILES_DIR=' "$DOTFILES_DIR/.zshrc"
    grep -q 'export CLAUDE_CONFIG_DIR=' "$DOTFILES_DIR/.zshrc"
}

@test ".bashrc exports DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q 'export DOTFILES_DIR=' "$DOTFILES_DIR/.bashrc"
    grep -q 'export CLAUDE_CONFIG_DIR=' "$DOTFILES_DIR/.bashrc"
}

@test "powershell/profile.ps1 sets DOTFILES_DIR and CLAUDE_CONFIG_DIR" {
    grep -q '\$env:DOTFILES_DIR' "$DOTFILES_DIR/powershell/profile.ps1"
    grep -q '\$env:CLAUDE_CONFIG_DIR' "$DOTFILES_DIR/powershell/profile.ps1"
}

# --- Harness deploy-dir mirror (drift false-FAIL fix) ---
# healthcheck section 12 runs `compile-harness.sh --check` from the deploy copy
# (~/.dotfiles), and the engine resolves its root from its own location. That
# offline check needs the harness inputs mirrored into the deploy dir -- exactly
# the complete non-git copy the rootresolve regression test models: scripts/
# (compile-harness.sh, already copied) + harness/ + every file
# harness/manifest.json declares as an injection target. setup copied none of
# them, so --check exited 2 (manifest not found) and section 12 reported a false
# drift FAIL. These guards lock the mirror in, AND assert it runs AFTER --refresh
# so the snapshot matches the refreshed repo state (else the repo<->deploy drift
# sub-check would see drift).

@test "setup-linux.sh mirrors harness/ into the deploy dir through dotf harness mirror (WIN-007)" {
    grep -qF 'dotf harness mirror' "$DOTFILES_DIR/setup-linux.sh"
    # The bash+jq block is gone -- one implementation for both OSes. A `run` +
    # status check, not a bare negation: a negated assertion is exempt from
    # set -e and cannot fail a test (lesson 224).
    run grep -qF 'cp -rf "$CURRENT_DIR/harness/." "$DOTFILES_DIR/harness/"' "$DOTFILES_DIR/setup-linux.sh"
    [ "$status" -ne 0 ]
}

@test "setup-linux.sh derives the mirrored target list from harness/manifest.json (in Go, via dotf harness mirror)" {
    # The derivation lives in cli/internal/harness/mirror.go (reads
    # .targets[].file), so the script must not restate the list -- #1200: a
    # third target was added to the manifest with no copy line, and the old
    # per-target assertions stayed green while `dotf doctor` failed permanently.
    # The OUTCOME -- every declared target present in $DOTFILES_DIR -- is
    # asserted by verify-setup.bats against a real deploy.
    grep -qF 'dotf harness mirror' "$DOTFILES_DIR/setup-linux.sh"
    run grep -qF 'safe_copy "$CURRENT_DIR/$_harness_target"' "$DOTFILES_DIR/setup-linux.sh"
    [ "$status" -ne 0 ]
}

@test "setup-linux.sh resolves dotf by path for the harness mirror, not only by name (#1202 class)" {
    # install_dotf places the binary in ~/.local/bin, which this process's PATH
    # may not carry: the integration container installed dotf and then skipped
    # the mirror in the same run, and verify-setup.bats caught the gap (#1305).
    grep -qF '[ -x "$HOME/.local/bin/dotf" ]' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF '"$_dotf" harness mirror' "$DOTFILES_DIR/setup-linux.sh"
}

# CLI-054 (#1301): bare `dotf deploy` installs every config ai/deploy.json
# declares. Naming one config at the call site meant a new manifest entry
# (orca-keybindings) was deployed by neither setup until two scripts were
# edited -- the manifest is the SSOT of what gets deployed, not the call site.
@test "setup-linux.sh deploys agent configs with bare dotf deploy, naming no config (CLI-054)" {
    grep -qE '"\$_dotf" deploy( \|\||$)' "$DOTFILES_DIR/setup-linux.sh"
    # Anchored to the invocation shape (a line that RUNS dotf), not to any
    # mention: the comment above the call names the old form on purpose.
    refute_grep '^[[:space:]]*("\$_dotf"|dotf) deploy [a-z]' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh resolves dotf by path for the config deploy, not only by name (#1305 class)" {
    # Same trap as the harness mirror: the integration container installs dotf
    # into ~/.local/bin and this process's PATH does not carry it yet.
    grep -qF '"$_dotf" deploy' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh installs the GUARD memory-sink git-hooks (#418 deploy + wire)" {
    grep -qF '. ./scripts/install-git-hooks.sh' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'install_git_hooks' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh harness mirror runs AFTER compile-harness --refresh (ordering guard)" {
    refresh_line=$(grep -n 'compile-harness.sh" --refresh' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    mirror_line=$(grep -n 'dotf harness mirror' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$refresh_line" ] && [ -n "$mirror_line" ]
    [ "$mirror_line" -gt "$refresh_line" ]
}
