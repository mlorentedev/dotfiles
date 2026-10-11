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
    [[ "$output" == *"in-place"* ]] || false
    [[ "$output" == *"dotfiles-repo"* ]] || false
}

# --- Developer tools section ---

@test "setup-linux.sh creates ~/.local/bin directory" {
    grep -q 'ensure_directory.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh adds ~/.local/bin to PATH" {
    grep -q 'export PATH.*\.local/bin' "$DOTFILES_DIR/setup-linux.sh"
}

# uv is a mise-pinned CLI and poetry a packages.json uv-tool entry (#2013 W2):
# no unpinned installer and no per-script poetry block survive.
@test "setup-linux.sh gets uv from mise, not from uv's curl installer" {
    grep -A1 '^# mise: cli$' "$DOTFILES_DIR/versions.conf" | grep -q '^UV_VERSION='
    refute_grep_fixed 'astral.sh/uv/install.sh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "poetry is a packages.json uv-tool entry, and setup-linux.sh has no poetry block" {
    jq -e '.tools[] | select(.name == "poetry" and .source.type == "uv-tool" and (.source.platforms == null))' "$DOTFILES_DIR/packages.json" >/dev/null
    refute_grep_fixed 'uv tool install poetry' "$DOTFILES_DIR/setup-linux.sh"
}

# install places mise, sync installs the pinned CLIs (uv), and the second
# install reaches the uv-tool entries; mise's shims must be on PATH, ahead of
# ~/.local/bin, before sync, or a copy an installer left there shadows the pin.
@test "setup-linux.sh runs tools install, shims onto PATH, tools sync, tools install, in that order" {
    install1=$(grep -n '^    dotf tools install ' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    shims=$(grep -n 'export PATH=.*mise}/shims:\$PATH' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    sync=$(grep -n '^    dotf tools sync ' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    install2=$(grep -n '^    dotf tools install ' "$DOTFILES_DIR/setup-linux.sh" | sed -n 2p | cut -d: -f1)
    [ -n "$install1" ]
    [ -n "$shims" ]
    [ -n "$sync" ]
    [ -n "$install2" ]
    [ "$install1" -lt "$shims" ]
    [ "$shims" -lt "$sync" ]
    [ "$sync" -lt "$install2" ]
}

@test "setup-linux.sh deploys secrets/registry.yaml (dotf secrets mapping SSOT) [#587]" {
    grep -qF 'DOTFILES_DIR/secrets' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF 'secrets/registry.yaml' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh carries neither load-secrets twin, and fetches no deploy-time secret [#587, OPS-040]" {
    # #587 migrated agy's OPENROUTER_API_KEY off the load-secrets eager
    # dot-source onto `dotf secrets show`. OPS-040 removed the fetch entirely:
    # the agy mcp_config cascade that consumed it was deleted by CLI-042 AC8, so
    # setup was decrypting a credential on every run for nobody. What survives
    # from #587 is the half that still binds -- neither retired twin API may come
    # back -- plus the stronger statement that no deploy-time secret is resolved
    # here at all.
    refute_grep 'load-secrets\.sh" >/dev/null 2>&1' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'secrets_show ' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'dotf secrets show OPENROUTER_API_KEY' "$DOTFILES_DIR/setup-linux.sh"
}

# The guard below reads its names from the marks, so a lost mark shrinks it
# silently: setup would have no installer and mise no pin for that tool. Each
# CLI whose installer W2 deleted must stay marked.
@test "versions.conf marks every CLI whose setup installer W2 deleted (#2013 W2)" {
    local name missing=""
    for name in age jq zoxide direnv shellcheck bats; do
        awk -v v="$(printf '%s' "$name" | tr 'a-z-' 'A-Z_')_VERSION" '
            m && index($0, v "=") == 1 { found = 1 }
            { m = ($0 == "# mise: cli") }
            END { exit !found }' "$DOTFILES_DIR/versions.conf" || missing="$missing $name"
    done
    [ -z "$missing" ] || { echo "not marked '# mise: cli' in versions.conf:$missing"; false; }
}

@test "setup-linux.sh has no installer of its own for a CLI versions.conf marks for mise (#2013 W2)" {
    # mise installs these through `dotf tools sync` (ADR-044). An installer that
    # ran before the sync put a second, unpinned copy in ~/.local/bin on every
    # fresh box (age and jq did); one that ran after it was dead code.
    local names name found=""
    names="$(awk '
        m && /^[A-Z0-9_]+_VERSION=/ {
            n = $0; sub(/_VERSION=.*/, "", n); n = tolower(n); gsub(/_/, "-", n); print n
        }
        { m = ($0 == "# mise: cli") }' "$DOTFILES_DIR/versions.conf")"
    [ -n "$names" ]
    while IFS= read -r name; do
        # Three shapes an installer takes, any one is enough: its log line, the
        # path it places the binary at, and the release asset it downloads.
        grep -qE "log_info \"Installing $name\.\.\.|\.local/bin/$name([^A-Za-z0-9_-]|\$)|/$name/releases/download/" \
            "$DOTFILES_DIR/setup-linux.sh" && found="$found $name"
    done <<< "$names"
    [ -z "$found" ] || { echo "setup-linux.sh installs mise-owned CLIs itself:$found"; false; }
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

@test "tmux.conf deploys to ~/.tmux.conf through dotf deploy, only where tmux is installed (#1843 B2)" {
    [ "$(jq -r '.configs[] | select(.name=="tmux") | "\(.src) \(.dst) \(.requires)"' "$DOTFILES_DIR/ai/deploy.json")" = "tmux.conf {HOME}/.tmux.conf tmux" ]
    refute_grep 'deploy_file "\$DOTFILES_DIR/tmux\.conf"' "$DOTFILES_DIR/setup-linux.sh"
}

@test "the zsh rc files deploy through dotf deploy, only where zsh is installed (#1843 B2)" {
    for name in zshrc zsh-aliases zsh-functions zsh-nvm; do
        [ "$(jq -r --arg n "$name" '.configs[] | select(.name==$n) | .requires' "$DOTFILES_DIR/ai/deploy.json")" = "zsh" ]
    done
    refute_grep 'deploy_file "\$DOTFILES_DIR/\.zshrc"' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'deploy_file "\$DOTFILES_DIR/\.zsh/' "$DOTFILES_DIR/setup-linux.sh"
}

@test "functions.sh is not gated on zsh: .bashrc sources it too (#1843 B2)" {
    grep -q '\.zsh/functions\.sh' "$DOTFILES_DIR/.bashrc"
    [ "$(jq -r '.configs[] | select(.name=="zsh-functions-sh") | .requires // "none"' "$DOTFILES_DIR/ai/deploy.json")" = "none" ]
}

@test "the ssh config and public key deploy through dotf deploy on every OS (#2013 P7)" {
    [ "$(jq -r '.configs[] | select(.name=="ssh-config") | "\(.src) \(.dst) \(.mode) \(.requires // "")"' "$DOTFILES_DIR/ai/deploy.json")" = "ssh/config {HOME}/.ssh/config 0600 " ]
    [ "$(jq -r '.configs[] | select(.name=="ssh-pubkey") | "\(.src) \(.dst) \(.mode)"' "$DOTFILES_DIR/ai/deploy.json")" = "ssh/id_ed25519.pub {HOME}/.ssh/id_ed25519.pub 0644" ]
    refute_grep 'deploy_file "\$DOTFILES_DIR/ssh/config"' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'id_ed25519\.pub" "\$HOME/\.ssh' "$DOTFILES_DIR/setup-linux.sh"
}

@test "~/.gitconfig is converged after dotf deploy, never deployed over (#2207)" {
    # Co-owned by git, gh and the user: a deploy over it erased their writes
    # and restored a helper that needs the shell's PATH (lesson 366).
    [ ! -e "$DOTFILES_DIR/.gitconfig" ]
    refute_grep '\.gitconfig" "\$HOME/\.gitconfig"' "$DOTFILES_DIR/setup-linux.sh"
    # gh writes the credential helper, with its absolute path; the repo's
    # settings carry none.
    run git config -f "$DOTFILES_DIR/git/dotfiles.gitconfig" --get-regexp '^credential\.'
    [ "$status" -eq 1 ]
    deploy=$(grep -n '"\$_dotf" deploy ||' "$DOTFILES_DIR/setup-linux.sh" | cut -d: -f1)
    converge=$(grep -n '"\$_dotf" converge --only git-config ||' "$DOTFILES_DIR/setup-linux.sh" | cut -d: -f1)
    [ -n "$deploy" ] && [ -n "$converge" ] && [ "$converge" -gt "$deploy" ]
}

# tmux and xclip are packages.json `system` entries (#2013 P5b): `dotf tools
# install` installs them through apt or brew, or prints the sudo command once.
@test "packages.json declares tmux and xclip as system entries" {
    run jq -r '.tools[] | select(.source.type == "system") | "\(.name)=\(.source.apt // "")"' "$DOTFILES_DIR/packages.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"tmux=tmux"* ]] || false
    [[ "$output" == *"xclip=xclip"* ]] || false
}

# The hand-written install hint is what the catalog replaced. A second copy in
# setup would drift from the catalog's declaration the first time one changes.
@test "setup-linux.sh no longer hand-checks tmux or xclip" {
    refute_grep 'command -v (tmux|xclip)' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep '(apt|apt-get|brew) install[^#]*(tmux|xclip)' "$DOTFILES_DIR/setup-linux.sh"
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

# --- MEM-002: retire claude-mem — no longer installed (ADR-016 Q2) ---
# Neither setup registers its marketplace, and neither carries a cleanup block:
# the retirement is `retired_marketplaces` in ai/claude/plugins.json, which
# `dotf deploy` converges on every OS (#1431).

@test "setup scripts no longer register the thedotmack marketplace (MEM-002)" {
    refute_grep_fixed 'claude plugin marketplace add thedotmack/claude-mem' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'claude plugin marketplace add thedotmack/claude-mem' "$DOTFILES_DIR/setup-windows.ps1"
}

@test "the claude-mem retirement is data, not a setup block (MEM-002, #1431)" {
    # `dotf deploy` removes the marketplace and checks it is gone; the setup
    # blocks it replaces stripped a settings.json key Claude Code stopped using.
    jq -e '.retired_marketplaces | index("thedotmack")' "$DOTFILES_DIR/ai/claude/plugins.json"
    refute_grep_fixed 'claude plugin uninstall claude-mem@thedotmack' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'claude plugin uninstall claude-mem@thedotmack' "$DOTFILES_DIR/setup-windows.ps1"
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

# OPS-042 (#1336): yarn is a packages.json tool (ADR-036), converged by
# `dotf tools install` on both OSes; neither setup script carries an npm block
# for it any more, and versions.conf no longer pins it.
#
# obsidian is deliberately NOT in the catalog (#1615). The `obsidian` that
# `dotf vault health` drives is the CLI built into the Obsidian desktop app. The
# npm package `obsidian-cli` is an unrelated third-party test-result importer
# that the catalog used to `npm install -g` on every machine, after a 404 on the
# scoped name was "fixed" by dropping the scope. The refute below keeps any
# obsidian npm entry from returning under either name.
@test "parity: yarn is a catalog tool and obsidian is not an npm tool (OPS-042, #1615)" {
    [ -z "$(jq -r '.tools[] | select(.name=="obsidian" or .source.package=="obsidian-cli") | .name' "$DOTFILES_DIR/packages.json")" ]
    [ "$(jq -r '.tools[] | select(.name=="yarn") | "\(.source.type) \(.source.package) \(.version)"' "$DOTFILES_DIR/packages.json")" = "npm yarn 1.22.22" ]
    refute_grep 'obsidian-cli' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep 'obsidian-cli' "$DOTFILES_DIR/setup-windows.ps1"
    refute_grep '^OBSIDIAN_VERSION=' "$DOTFILES_DIR/versions.conf"
    refute_grep '^YARN_VERSION=' "$DOTFILES_DIR/versions.conf"
}

# ADR-036: a tool packages.json installs through npm is converged by
# `dotf tools install`, so neither setup twin may `npm install -g` it as well.
# This matches the property, not a spelling: OPS-042's guard grepped for the two
# forms it had deleted, and `npm install -g yarn@1.22.22` passed it (#1864). The
# scan lives in tests/lib/npm-global-scan.py, which says what it resolves.
@test "parity: neither setup twin npm-installs a tool packages.json owns (ADR-036, #1864)" {
    run python3 "$DOTFILES_DIR/tests/lib/npm-global-scan.py" "$DOTFILES_DIR/packages.json" \
        "$DOTFILES_DIR/setup-linux.sh" "$DOTFILES_DIR/setup-windows.ps1"
    [ "$status" -eq 0 ] || { echo "a setup twin installs a catalog-owned npm tool: $output"; false; }
}

# The scan's own contract, on fixtures rather than on the twins, so it holds
# when the twins carry no global npm install at all (ADR-036's end state).
@test "npm-global-scan: every hostile spelling fails and every benign one passes (#1864)" {
    scan="$DOTFILES_DIR/tests/lib/npm-global-scan.py"
    fixture="$BATS_TEST_TMPDIR/twin"
    hostile=(
        'npm install -g yarn@1.22.22'
        'npm install -g "yarn@${YARN_VERSION}" || true'
        'npm i --global @bitwarden/cli@2025.1.0'
        'npm install -g --ignore-scripts @github/copilot;'
        $'npm install -g \\\n    yarn@1.22.22'
        $'& npm install -g `\n    opencode-ai'
        'npm -g install yarn'
        'npm --prefix /x install -g yarn'
        'npm dedupe && npm install -g yarn'
        'X=$(npm install -g yarn@1.22.22)'
        'npm.cmd install -g opencode-ai'
        $'export YARN_ALT=yarn\nnpm install -g "$YARN_ALT"'
        $'BASE=yarn\nTOOL=$BASE\nnpm install -g "$TOOL"'
        $'$yarnPkg = "yarn@1.22.22"\n& npm install -g $yarnPkg'
        $'if ($x) { $alt = "opencode-ai" }\n& npm install -g $alt'
        'npm install -g "$NEVER_ASSIGNED"'
        $'TOOL=$1\nnpm install -g "$TOOL"'
        'npm install -g $(cat pkgs)'
        'npm install -g `cat pkgs`'
        '& npm install -g $env:SOME_PKG'
        $'TOOL=yarn\nnpm install -g "${TOOL:-x}"'
        $'foo=unrelated\nnpm install -g "$FOO"'
        'npm install -g yarn${SUFFIX}'
        'npm install -g "@github/copilot$SUFFIX"'
        'npm install --location global yarn@1.22.22'
        $'# installed with: \\\nnpm install -g yarn@1.22.22'
        'true;npm install -g yarn'
        'x |npm install -g yarn'
        '& NPM install -g yarn'
        '& npm.exe install -g opencode-ai'
        '& "C:\\Program Files\\nodejs\\npm.cmd" install -g yarn'
        '/usr/bin/npm install -g yarn'
        $'TOOL=foo; TOOL=yarn\nnpm install -g "$TOOL"'
        $'if a; then TOOL=nodejs; else TOOL=yarn; fi\nnpm install -g "$TOOL"'
    )
    benign=(
        'npm install -g yarnish'
        'npm install yarn'
        '# was: npm install -g yarn'
        'npm install -g foo || log_warning "yarn failed"'
        'echo "nothing installs here"'
        'npm install -g foo  # yarn would be the other choice'
        'chmod +x s.sh # used to npm install -g yarn'
        $'npm install -g foo \\ \n    yarn'
        $'PI_PKG="@earendil-works/pi-coding-agent${PI_VERSION:+@$PI_VERSION}"\nnpm install -g --prefix "$HOME/.local" "$PI_PKG"'
    )
    # The hostile fixtures are hostile only while packages.json owns these
    # through npm. Say so here, so a catalog migration reads as one, not as a
    # broken scan.
    for pkg in yarn opencode-ai @bitwarden/cli @github/copilot; do
        run python3 -c 'import json, sys
sys.exit(0 if any(t["source"]["type"] == "npm" and t["source"].get("package") == sys.argv[2]
                  for t in json.load(open(sys.argv[1]))["tools"]) else 1)' "$DOTFILES_DIR/packages.json" "$pkg"
        [ "$status" -eq 0 ] || { echo "the fixtures assume packages.json owns $pkg through npm; pick another owned package"; false; }
    done
    # A rejection is a verdict naming the line, not just exit 1: a traceback
    # exits 1 too.
    for snippet in "${hostile[@]}"; do
        printf '%s\n' "$snippet" >"$fixture"
        run python3 "$scan" "$DOTFILES_DIR/packages.json" "$fixture"
        [ "$status" -eq 1 ] && [[ "$output" =~ twin:[0-9]+:\  ]] && [[ "$output" != *Traceback* ]] ||
            { echo "not rejected with a verdict: $snippet: $output"; false; }
    done
    for snippet in "${benign[@]}"; do
        printf '%s\n' "$snippet" >"$fixture"
        run python3 "$scan" "$DOTFILES_DIR/packages.json" "$fixture"
        [ "$status" -eq 0 ] || { echo "failed, but should pass: $snippet: $output"; false; }
    done
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

# From the harness refresh to the mirror: the two share one dotf resolution.
extract_linux_harness_mirror_block() {
    local script="${1:-$DOTFILES_DIR/setup-linux.sh}"
    awk '
        /^# Harness deploy engine \(ENGINE-001/ { capture = 1 }
        capture { print }
        capture && /^unset _dotf$/ { ended = 1; exit }
        END { if (!capture || !ended) exit 1 }
    ' "$script"
}

@test "WIN-014 extractor fails closed when the harness block terminator is missing" {
    local broken="$BATS_TEST_TMPDIR/setup-linux-no-harness-end.sh"
    grep -v '^unset _dotf$' "$DOTFILES_DIR/setup-linux.sh" > "$broken"
    run extract_linux_harness_mirror_block "$broken"
    [ "$status" -ne 0 ]
}

@test "setup-linux.sh executes harness mirror with its checkout path (WIN-014)" {
    local block command_script
    block="$(extract_linux_harness_mirror_block)"
    [ -n "$block" ]
    command_script='
log_warning() { printf "WARN:%s\n" "$*"; }
dotf() {
    printf "CALL"
    printf " <%s>" "$@"
    printf "\n"
    return 0
}
CURRENT_DIR="/checkout with spaces"
DOTFILES_DIR="/deploy"
'
    run bash -c "$command_script
$block"
    [ "$status" -eq 0 ]
    [[ "$output" == *"CALL <harness> <mirror> <--repo> </checkout with spaces>"* ]] || false
    [[ "$output" == *"CALL <harness> <refresh> <--repo> </checkout with spaces>"* ]] || false
}

@test "setup-linux.sh warns on a failed harness refresh and still mirrors (#2162)" {
    local block command_script
    block="$(extract_linux_harness_mirror_block)"
    [ -n "$block" ]
    command_script='
log_warning() { printf "WARN:%s\n" "$*"; }
dotf() {
    printf "CALL <%s>\n" "$2"
    [ "$2" != refresh ]
}
CURRENT_DIR="/checkout"
DOTFILES_DIR="/deploy"
'
    run bash -c "$command_script
$block"
    [ "$status" -eq 0 ]
    [[ "$output" == *"WARN:dotf harness refresh failed (above); deploying the committed harness records"* ]] || false
    [[ "$output" == *"CALL <mirror>"* ]] || false
}

@test "setup-linux.sh warns when dotf cannot mirror the harness (WIN-014)" {
    local block command_script
    block="$(extract_linux_harness_mirror_block)"
    [ -n "$block" ]
    command_script='
log_warning() { printf "WARN:%s\n" "$*"; }
PATH="/missing"
HOME="/home/without-dotf"
CURRENT_DIR="/checkout"
DOTFILES_DIR="/deploy"
'
    run bash -c "$command_script
$block"
    [ "$status" -eq 0 ]
    [[ "$output" == *"WARN:dotf not found (PATH or ~/.local/bin) -- harness not mirrored to /deploy"* ]] || false
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
    # CLI-072 replaced the sourced shell twin with `dotf hooks install`. Resolved
    # by path like the other dotf call sites, for the #1202 reason: the process
    # may not carry ~/.local/bin on PATH yet even though install_dotf just put it
    # there. The ordering against install_dotf, and the Windows side, are
    # asserted in tests/guard-setup-hooks-order.bats.
    grep -qF 'hooks install --source' "$DOTFILES_DIR/setup-linux.sh"
    grep -qF '"$_dotf_hooks" hooks install' "$DOTFILES_DIR/setup-linux.sh"
}

@test "setup-linux.sh harness mirror runs AFTER the harness refresh (ordering guard)" {
    refresh_line=$(grep -n 'harness refresh --repo' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    # Both anchors are invocations: a comment names `dotf harness mirror` above
    # the call, and matching it would pin the order of the prose instead.
    mirror_line=$(grep -n 'harness mirror --repo' "$DOTFILES_DIR/setup-linux.sh" | head -1 | cut -d: -f1)
    [ -n "$refresh_line" ] && [ -n "$mirror_line" ]
    [ "$mirror_line" -gt "$refresh_line" ]
}

@test "setup-linux.sh seeds the canonical checkout, not the linked worktree it ran from (lesson 247)" {
    # The mechanism, on a real repo + worktree: --git-dir and --git-common-dir are
    # equal in a normal checkout and differ inside a linked worktree, where the
    # common dir is the real repo's .git. Seeding $CURRENT_DIR from a worktree put
    # a throwaway path into machine.json and blocked every commit on the box once
    # that worktree was deleted.
    local tmp; tmp="$(mktemp -d)"
    git -C "$tmp" init -q main_repo
    git -C "$tmp/main_repo" -c user.email=t@t -c user.name=t commit -q --allow-empty -m init
    git -C "$tmp/main_repo" worktree add -q "$tmp/wt" -b side

    # canonical checkout: the two agree, so the seed is CURRENT_DIR unchanged
    local gd gcd
    gd="$(git -C "$tmp/main_repo" rev-parse --git-dir)"
    gcd="$(git -C "$tmp/main_repo" rev-parse --git-common-dir)"
    [ "$gd" = "$gcd" ]

    # linked worktree: they differ, and dirname(common) is the canonical checkout
    gd="$(git -C "$tmp/wt" rev-parse --git-dir)"
    gcd="$(git -C "$tmp/wt" rev-parse --git-common-dir)"
    [ "$gd" != "$gcd" ]
    [ "$(cd "$(dirname "$gcd")" && pwd -P)" = "$(cd "$tmp/main_repo" && pwd -P)" ]

    git -C "$tmp/main_repo" worktree remove --force "$tmp/wt"
    rm -rf "$tmp"
}

@test "setup-linux.sh no longer seeds DOTFILES_REPO_DIR from CURRENT_DIR directly (lesson 247)" {
    # Wiring, not mechanism: the resolution above is worthless if the script still
    # passes the invocation directory to `dotf env set`.
    grep -q 'dotf env set DOTFILES_REPO_DIR "\$SEED_REPO_DIR"' "$DOTFILES_DIR/setup-linux.sh"
    refute_grep_fixed 'dotf env set DOTFILES_REPO_DIR "$CURRENT_DIR"' "$DOTFILES_DIR/setup-linux.sh"
}

# #1843 B13: the auto-memory sweep linked every vault project by guessing its
# repo at ~/Projects/<name>, and moved a real memory dir aside. memlink links
# the project a session opens and never moves data, so no setup carries a sweep.
@test "neither setup sweeps auto-memory links; memlink owns them (#1843 B13)" {
    local f
    for f in "$DOTFILES_DIR/setup-linux.sh" "$DOTFILES_DIR/setup-windows.ps1"; do
        refute_grep 'Deploying auto-memory|Migrating orphan memory|Backing up existing memory' "$f"
    done
    refute_grep_fixed 'function Get-ClaudeProjectKey' "$DOTFILES_DIR/scripts/utils.ps1"
}
