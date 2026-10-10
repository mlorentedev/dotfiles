#!/bin/bash

# setup-linux.sh: Install dotfiles and set up environment
# Usage: ./setup-linux.sh
#
# Philosophy: opportunistic configuration. The script attempts to set up
# every supported tool and warns when something is not available, rather
# than requiring opt-in flags. Heavy / disk-intensive installs (e.g.
# future Ollama) may add explicit flags case by case.

set -euo pipefail

# Load utility functions for logging
if [ -f ./scripts/utils.sh ]; then
    . ./scripts/utils.sh
else
    echo "Error: utils.sh not found"
    exit 1
fi

# Current directory and target directory
CURRENT_DIR=$(pwd)

# Create necessary directories
export DOTFILES_DIR="$HOME/.dotfiles"

# Refuse an in-place install (#695). The supported layout clones the repo to a
# SEPARATE directory (e.g. ~/dotfiles-repo) and deploys into ~/.dotfiles. When
# the repo is cloned directly into ~/.dotfiles, CURRENT_DIR == DOTFILES_DIR and
# setup's deploy copies collapse to same-file operations: `cp -f X X` aborts
# under `set -euo pipefail` (env-contract deploy), and the git-hooks clean-mirror
# (rm -rf dest + cp src/.) would EMPTY the dispatcher while still logging success.
# Fail fast HERE, before any directory is created or any file is copied.
if [ "$CURRENT_DIR" = "$DOTFILES_DIR" ]; then
    log_error "Refusing in-place install: the repo is checked out in \$DOTFILES_DIR ($DOTFILES_DIR)."
    log_error "That directory is the DEPLOY TARGET, not the checkout. Clone somewhere else and re-run:"
    log_error "    git clone https://github.com/mlorentedev/dotfiles.git ~/dotfiles-repo"
    log_error "    cd ~/dotfiles-repo && bash setup-linux.sh"
    exit 1
fi

# Single source of truth for tool versions (REFACTOR-011): source the manifest
# from the checkout so $PI_VERSION / etc. are reliable
# regardless of the parent shell. Read from $CURRENT_DIR: the copy under
# $DOTFILES_DIR does not exist yet on a first run (it is created below).
[ -f "$CURRENT_DIR/versions.conf" ] && . "$CURRENT_DIR/versions.conf"

log_info "Creating necessary directories..."
ensure_directory "$HOME/.zsh"
ensure_directory "$HOME/.bash"
ensure_directory "$DOTFILES_DIR"
ensure_directory "$DOTFILES_DIR/.zsh"
ensure_directory "$DOTFILES_DIR/scripts"

# Copy all files to the dotfiles directory
log_info "Setting up dotfiles in $DOTFILES_DIR..."
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    # Copy files to the dotfiles directory
    log_info "Copying files from $CURRENT_DIR to $DOTFILES_DIR..."
    safe_copy "$CURRENT_DIR/versions.conf" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/packages.json" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/mcp-servers.json" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.zshrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.profile" "$DOTFILES_DIR/" 2>/dev/null || true
    if [ -f "$CURRENT_DIR/.bashrc" ]; then
        safe_copy "$CURRENT_DIR/.bashrc" "$DOTFILES_DIR/" 2>/dev/null || true
    fi    
    safe_copy "$CURRENT_DIR/tmux.conf" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.inputrc" "$DOTFILES_DIR/" 2>/dev/null || true
    safe_copy "$CURRENT_DIR/.editorconfig" "$DOTFILES_DIR/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/.zsh/." "$DOTFILES_DIR/.zsh/" 2>/dev/null || true
    ensure_directory "$DOTFILES_DIR/ssh"
    cp -rf "$CURRENT_DIR/ssh/"* "$DOTFILES_DIR/ssh/" 2>/dev/null || true
    cp -rf "$CURRENT_DIR/scripts/"* "$DOTFILES_DIR/scripts/" 2>/dev/null || true
else
    log_info "Already in dotfiles directory, skipping copy..."
fi

# Deploy dotfiles via deploy_file (SDD-007 IaC strategy: atomic copy + idempotent).
# ~/.zshrc, ~/.zsh/* and ~/.tmux.conf are `dotf deploy` entries (ai/deploy.json:
# zshrc, zsh-*, tmux; #1843 B2, PLAT-001 P6), deployed by the bare `dotf deploy`
# below on every OS that has zsh or tmux, with a one-time .pre-dotf backup of a
# file the machine already had. Edit in the repo, never in $HOME.
log_info "Deploying main dotfiles..."
[ -f "$DOTFILES_DIR/.profile" ] && deploy_file "$DOTFILES_DIR/.profile" "$HOME/.profile"

# ~/.ssh/config (0600) and the public key are `dotf deploy` entries on every
# OS (ai/deploy.json ssh-config, ssh-pubkey; #2013 P7).

# ~/.gitconfig is co-owned by git, gh and the user, so nothing deploys over it
# (#2207): `dotf deploy` below writes ~/.config/git/dotfiles.gitconfig and
# `dotf converge --only git-config` after it adds the include and gh's helper.

# readline config (POLISH-004): case-insensitive completion + smart history.
deploy_file "$DOTFILES_DIR/.inputrc" "$HOME/.inputrc"
chmod +x "$DOTFILES_DIR/scripts/utils.sh"
chmod +x "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh"
chmod +x "$DOTFILES_DIR/scripts/dotfiles-sync.sh"

# Copy sensitive directory (encrypted *.secret.age files; the mapping lives in secrets/registry.yaml)
log_info "Setting up sensitive directory..."

# Preflight: warn if age identity key is missing. Without it `dotf secrets`
# can't decrypt, so $NAN_API_KEY / $OPENROUTER_API_KEY / etc. resolve empty --
# opencode + agy then 401 with no clear cause. Non-fatal: encrypted files still
# deploy so a key imported later works without re-running setup.
AGE_KEY="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"
if [ ! -f "$AGE_KEY" ]; then
    log_warning "age identity key not found at $AGE_KEY"
    log_warning "  Encrypted secrets will deploy but won't decrypt on demand."
    log_warning "  To enable: place your age identity at \$HOME/.config/age/key.txt"
    log_warning "  (or set AGE_KEY_PATH). Generate: age-keygen -o ~/.config/age/key.txt"
    log_warning "  See: docs/runbooks/guide-secrets-governance.md#protocol--first-machine-age-key--ssh"
fi

ensure_directory "$DOTFILES_DIR/sensitive"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/sensitive/"* "$DOTFILES_DIR/sensitive/" 2>/dev/null || true
fi

# Deploy the secrets registry (ADR-028 §2 mapping SSOT). dotf secrets reads it
# from $DOTFILES_DIR/secrets/registry.yaml; without it `dotf secrets {ls,show,run}`
# and the AI-CLI wrappers fail. Mirrors the sensitive/ deploy just above.
ensure_directory "$DOTFILES_DIR/secrets"
if [ "$CURRENT_DIR" != "$DOTFILES_DIR" ]; then
    cp -rf "$CURRENT_DIR/secrets/"* "$DOTFILES_DIR/secrets/" 2>/dev/null || true
fi

# Deploy-time secrets are fetched via `dotf secrets show` after dotf is installed
# (see the block right after install_dotf) -- the load-secrets eager-source was
# retired here (ADR-028 / #587).


# utils.sh is sourced declaratively from .zsh/functions.sh (loaded by both
# ~/.bashrc and ~/.zshrc), so setup no longer mutates the deployed functions.zsh
# to append a `. utils.sh` line. That deploy-time mutation made the deployed copy
# drift from the repo source, which the deploy-dir<->$HOME assertion caught. One
# shared entrypoint gives bash AND zsh the library without ever editing a
# deployed file (REFACTOR-010 / "presence is not convergence").
#
# That assertion used to be check_deployed, right below; OPS-043 moved it into
# `dotf doctor` (checkHomeDeployDrift). The reason this comment records still
# holds -- only the thing that would catch a regression changed name and file.

# Create a bash_aliases file for bash
log_info "Creating bash_aliases file..."
ensure_directory "$HOME/.bash"
cat > "$HOME/.bash/bash_aliases" << EOF
# Bash Aliases
$(grep -v '^\s*#' "$DOTFILES_DIR/.zsh/aliases.zsh" | grep "^alias ")
EOF

# Create a basic .bashrc if it doesn't exist
if [ -f "$DOTFILES_DIR/.bashrc" ]; then
    log_info "Using existing .bashrc from dotfiles..."
else
    log_info "Creating a basic .bashrc file (none found in source)..."
    cat > "$DOTFILES_DIR/.bashrc" << EOF
# ~/.bashrc: executed by bash(1) for non-login shells.

# Source aliases
if [ -f ~/.bash/bash_aliases ]; then
    . ~/.bash/bash_aliases
fi

# Source utility functions
if [ -f "$DOTFILES_DIR/scripts/utils.sh" ]; then
    . "$DOTFILES_DIR/scripts/utils.sh"
fi
EOF
fi

# Deploy .bashrc
deploy_file "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"

# ============================================================================
# DEVELOPER TOOLS (user-level installs to ~/.local/bin)
# ============================================================================

log_info "Installing developer tools..."
ensure_directory "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$PATH"

# tmux, xclip, gh, git-lfs, parallel, wget and eza are `system` entries
# in packages.json (#2013 P5b): `dotf tools install` below installs them through
# the OS manager, or says which `sudo` command to run once.

# age, jq, zoxide, direnv, shellcheck and bats are pinned in versions.conf and
# installed by `dotf tools sync` below (mise, ADR-044); setup has no installer
# of its own for them (#2013 W2). eza and gh are not in mise yet: these blocks
# fetch linux-amd64 release assets (F-030). Elsewhere they would place a binary
# the OS cannot execute ahead of a working copy on PATH, so they run on
# linux-amd64 only, and any such leftover is removed, along with those of the
# tools the deleted blocks used to place.
if host_is_linux_amd64; then
    # eza (modern ls replacement)
    if ! command -v eza >/dev/null 2>&1; then
        log_info "Installing eza..."
        curl -Lo /tmp/eza.tar.gz "https://github.com/eza-community/eza/releases/latest/download/eza_x86_64-unknown-linux-gnu.tar.gz" 2>/dev/null \
            && tar xzf /tmp/eza.tar.gz -C "$HOME/.local/bin/" \
            && chmod +x "$HOME/.local/bin/eza" \
            && rm -f /tmp/eza.tar.gz \
            && log_success "eza installed" \
            || log_warning "eza installation failed"
    else
        log_info "eza already installed"
    fi

    # gh (GitHub CLI — required by Copilot setup)
    if ! command -v gh >/dev/null 2>&1; then
        log_info "Installing GitHub CLI..."
        GH_VERSION=$(curl -sI "https://github.com/cli/cli/releases/latest" 2>/dev/null | grep -i '^location:' | sed 's|.*/v||;s/[[:space:]]*$//')
        if [ -n "$GH_VERSION" ]; then
            curl -Lo /tmp/gh.tar.gz "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" 2>/dev/null \
                && tar xzf /tmp/gh.tar.gz -C /tmp \
                && cp "/tmp/gh_${GH_VERSION}_linux_amd64/bin/gh" "$HOME/.local/bin/" \
                && rm -rf /tmp/gh.tar.gz "/tmp/gh_${GH_VERSION}_linux_amd64" \
                && log_success "GitHub CLI installed" \
                || log_warning "GitHub CLI installation failed"
        else
            log_warning "Could not determine gh version, skipping"
        fi
    else
        log_info "gh already installed"
    fi
else
    log_warning "Skipping the linux-amd64 downloads of eza and gh on $(uname -s)/$(uname -m): install them with the OS package manager (#2013 W2)"
    remove_unrunnable_tool age age-keygen eza jq gh shellcheck
fi

# dotf (the dotfiles Go CLI — ADR-020). Fetch the pinned release binary,
# checksum-verified, into ~/.local/bin. Idempotent; converges on drift.
if [ -f ./scripts/install-dotf.sh ]; then
    # shellcheck source=/dev/null
    . ./scripts/install-dotf.sh
    install_dotf || log_warning "dotf installation failed (continuing; see healthcheck)"
else
    log_warning "scripts/install-dotf.sh not found; skipping dotf install"
fi

# Catalog tools (CLI-029, packages.json) and the mise-pinned CLIs (ADR-044,
# versions.conf). Three steps, each needing the one before it: `dotf tools
# install` places mise and the release binaries, `dotf tools sync` has mise
# install the pinned CLIs (uv among them), and the second `dotf tools install`
# reaches the uv-tool entries, which skip while uv is absent. On a converged box
# every step reports nothing to do. mise's shims go on PATH ahead of ~/.local/bin
# so a pinned CLI wins over an older copy an installer left there; interactive
# shells get the same order from `mise activate`. Best-effort, like
# `install_dotf` above: an offline box must never abort setup.
if command -v dotf >/dev/null 2>&1; then
    dotf tools install || log_warning "dotf tools install failed (continuing; re-run 'dotf tools install')"
    export PATH="${MISE_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/mise}/shims:$PATH"
    dotf tools sync || log_warning "dotf tools sync failed (continuing; re-run 'dotf tools sync')"
    dotf tools install || log_warning "dotf tools install failed (continuing; re-run 'dotf tools install')"
fi

# GUARD-001 memory-sink dispatcher (#398/#418): deploy git-hooks/ into the
# ~/.dotfiles mirror and wire core.hooksPath machine-wide so the guard is active
# in every repo. Idempotent; preserves an unrelated pre-existing hooksPath.
#
# CLI-072 moved this into `dotf hooks install`. The block used to cite ADR-020 C7
# ("the deployed dotf release can't self-deploy these -- no source tree"), which
# was never what C7 says: C7 keeps the step that provisions THE TOOLING in shell,
# and by here dotf is already installed. The source tree is not needed either --
# the installer reads git-hooks/ from this checkout, which is where setup runs.
#
# Resolved by path, not by name, for the #1202 reason documented further down:
# install_dotf placed it in ~/.local/bin, which the rc files put on PATH but THIS
# process may not have -- the integration container hits exactly that.
_dotf_hooks=""
if command -v dotf >/dev/null 2>&1; then
    _dotf_hooks="dotf"
elif [ -x "$HOME/.local/bin/dotf" ]; then
    _dotf_hooks="$HOME/.local/bin/dotf"
fi
if [ -n "$_dotf_hooks" ]; then
    # CURRENT_DIR is the checkout (line 22); DOTFILES_DIR the deploy mirror
    # (line 25). Passed explicitly rather than left to the installer's defaults,
    # because when the repo IS ~/.dotfiles -- the in-place layout line 23 already
    # detects -- source and destination are the same directory, and naming both
    # is what lets the #695 self-mirror guard see it.
    "$_dotf_hooks" hooks install --source "$CURRENT_DIR/git-hooks" --dotfiles-dir "$DOTFILES_DIR" \
        || log_warning "git-hooks install failed (continuing; see 'dotf doctor')"
else
    log_warning "dotf not found; skipping memory-sink guard install (run 'dotf hooks install' after setup)"
fi
unset _dotf_hooks

# Antigravity CLI (agy) install — idempotent per pattern-setup-script-idempotence.
# Official install URL: https://antigravity.google/cli/install.sh
# If agy is not in PATH, install it automatically; otherwise skip.
if ! command -v agy >/dev/null 2>&1; then
    log_info "Installing Antigravity CLI (agy)..."
    if curl -fsSL https://antigravity.google/cli/install.sh | bash 2>/dev/null; then
        log_success "agy installed"
    else
        log_warning "agy install failed — re-run setup or install manually (https://antigravity.google)"
    fi
else
    log_info "agy already installed"
fi

# Setup Antigravity CLI configuration (fresh-install model, SDD-007).
# Strategy: write canonical config to where agy reads from (~/.gemini/config/),
# use deploy_file (atomic + idempotent), no symlinks. Closes #100.
log_info "Setting up Antigravity CLI configuration..."
export GEMINI_HOME="$HOME/.gemini"
export AGY_APP_DATA="$GEMINI_HOME/antigravity-cli"
# ANTIGRAVITY_ENDPOINT / CLOUDCODE_URL were exported here until OPS-040, kept
# "for backward compat" against an empirical finding that already said they did
# nothing (cli log 2026-05-25): agy 1.0.2 issues `loadCodeAssist` and
# `fetchAvailableModels` to `daily-cloudcode-pa.googleapis.com` regardless of
# them — hardcoded metadata-plane routing — and the chat completion goes through
# a separate gRPC channel whose endpoint is not env-overridable. Two exports
# whose own comment recorded that they had no observed effect.

ensure_directory "$GEMINI_HOME"
ensure_directory "$GEMINI_HOME/config"
ensure_directory "$GEMINI_HOME/prompts"
ensure_directory "$AGY_APP_DATA"

# 1. agy settings.json, AGY.md and .geminiignore are `dotf deploy` entries
#    (ai/deploy.json `agy-settings`, `agy-instructions`, `agy-geminiignore`;
#    #1843 B11): one implementation for every OS, run by the bare `dotf deploy`
#    this script makes. settings.json renders {HOME} per machine (AI-042/#1334).

# 2. Consolidate MCP servers — master at ~/.gemini/config/mcp_config.json (agy's canonical read path)
if [ -f "$CURRENT_DIR/mcp-servers.json" ] && command -v jq >/dev/null 2>&1; then
    log_info "Consolidating Antigravity MCP servers..."

    # NOTE: canonical agy schema uses `mcpServers` (not `servers`) per Antigravity docs.
    #
    # CLI-042 AC8: the OpenRouter key recovery cascade that stood here is gone
    # with the provider. hive's worker is NaN-only since mlorentedev/hive#384, so
    # an OPENROUTER_API_KEY handed to hive-vault buys nothing — and the cascade
    # ended in `dotf secrets show`, baking a live credential into
    # mcp_config.json in plaintext. Removing the provider removes that file as a
    # place a secret lives, which is AC7's rule applied to the same daemon from
    # the other side. opencode's own OpenRouter provider is untouched: that is a
    # different consumer, and the spec scopes this criterion to hive's worker.

    # Substitute ${VAULT_PATH} placeholder with the canonical Linux vault dir.
    # Committed JSON uses a placeholder so the same source works cross-OS;
    # agy does NOT expand env vars inside JSON values, so the substitution
    # must happen here at write time. Path is by convention: $HOME/Projects/knowledge.
    VAULT_PATH_DEFAULT="${VAULT_PATH:-$HOME/Projects/knowledge}"

    # Preflight: WARN (non-fatal) if the vault dir is missing. hive-vault MCP
    # will fail at first tool call without it, but encrypted secrets + AGY.md +
    # everything else still deploys cleanly so creating the vault later doesn't
    # require a setup re-run. Mirrors the age-key preflight in #106.
    if [ ! -d "$VAULT_PATH_DEFAULT" ]; then
        log_warning "Obsidian vault not found at $VAULT_PATH_DEFAULT"
        log_warning "  hive-vault MCP will error at runtime until this dir exists."
        log_warning "  Either clone/sync your vault to $VAULT_PATH_DEFAULT, or override via"
        log_warning "  the VAULT_PATH env var in your shell."
    fi

    NEW_MCP_CONFIG=$(jq --arg vault "$VAULT_PATH_DEFAULT" '
        .mcpServers["hive-vault"].env.VAULT_PATH = $vault
    ' "$CURRENT_DIR/ai/agy/mcp_servers.json")

    # Merge stdio servers from root mcp-servers.json into mcpServers map
    while IFS=$'\t' read -r name transport args; do
        [ "$name" = "hive" ] && continue
        [ "$transport" != "stdio" ] && continue

        cmd=$(echo "$args" | awk '{print $1}')
        cmd_args=$(echo "$args" | cut -d' ' -f2-)

        NEW_MCP_CONFIG=$(echo "$NEW_MCP_CONFIG" | jq --arg name "$name" --arg cmd "$cmd" --argjson args "$(echo "$cmd_args" | jq -R 'split(" ")')" \
            '.mcpServers[$name] = { "command": $cmd, "args": $args }')
    done < <(jq -r '.servers[] | [.name, .transport, .args] | @tsv' "$CURRENT_DIR/mcp-servers.json")

    # Write master config to agy's canonical read path. No symlinks (BUG-100).
    # Idempotent: write to tempfile then deploy_file (skips when content matches).
    MASTER_CONFIG="$GEMINI_HOME/config/mcp_config.json"
    [ -L "$MASTER_CONFIG" ] && rm -f "$MASTER_CONFIG"
    _mcp_tmp="$(mktemp)"
    echo "$NEW_MCP_CONFIG" > "$_mcp_tmp"
    deploy_file "$_mcp_tmp" "$MASTER_CONFIG"
    rm -f "$_mcp_tmp"

    # Hive plugin: discovery file at canonical plugin path
    HIVE_PLUGIN_DIR="$AGY_APP_DATA/plugins/hive-vault"
    ensure_directory "$HIVE_PLUGIN_DIR"
    echo '{"name": "hive-vault"}' > "$HIVE_PLUGIN_DIR/plugin.json"
    echo "$NEW_MCP_CONFIG" | jq '{ "mcpServers": { "hive-vault": .mcpServers["hive-vault"] } }' > "$HIVE_PLUGIN_DIR/mcp_config.json"

    log_success "Consolidated Antigravity MCP configuration and registered Hive plugin"
fi

# Drop project-local agy state cache from the workspace if agy leaked one
[ -d "$CURRENT_DIR/.antigravitycli" ] && rm -rf "$CURRENT_DIR/.antigravitycli"
[ -d "$CURRENT_DIR/.antigravitycli.bak" ] && rm -rf "$CURRENT_DIR/.antigravitycli.bak"
rm -f "$GEMINI_HOME/config/.migrated" 2>/dev/null

# Drop the old AGY_APP_DATA symlink/copy of mcp_config.json (replaced by canonical path)
[ -e "$AGY_APP_DATA/mcp_config.json" ] && rm -f "$AGY_APP_DATA/mcp_config.json"

# Agy skills (native Shared skills + flat prompts) are deployed from the vault
# skill records by `compile-harness.sh --deploy` (SDD-008), which renders each
# committed record under harness/skills/ to ~/.gemini/skills/<n>/ and a
# frontmatter-stripped flat prompt to ~/.gemini/prompts/<n>.md, honoring each
# skill's targets[]. The single --deploy call near the end of this script does
# this for every agent at once; no per-agent skill loop here.
ensure_directory "$HOME/.gemini/skills"


# Note: legacy `agy plugin import gemini` removed (SDD-007). Fresh-install model:
# we don't carry over from legacy gemini-cli on every setup run. If the user
# needs a one-time migration, they run it manually once.

log_success "Antigravity CLI configuration complete"

# Harness deploy engine (ENGINE-001 / HARNESS-001): re-render the generated
# "Overrides of Harness Defaults" blocks in AGENTS.md + ai/claude/CLAUDE.md and
# the committed harness records from the vault SSOT before deploying agent
# configs. `dotf harness refresh` fast-forwards the vault first and refuses one
# it cannot confirm current: the refresh reads whatever the local clone holds,
# so a clone that is behind reverted merged records (#2162). It names any
# change the refresh leaves in the checkout, with the commit that records it
# (OPS-003, #295). Without a vault the committed records stand (fresh machine).
# Resolve dotf by path, not only by name: install_dotf placed it in ~/.local/bin,
# which the rc files put on PATH but THIS process may not have -- the integration
# container installs dotf and then cannot see it in the same run (#1202 was the
# identical trap with jq, and this block inherited it the moment it moved to dotf).
_dotf=""
if command -v dotf >/dev/null 2>&1; then
    _dotf="dotf"
elif [ -x "$HOME/.local/bin/dotf" ]; then
    _dotf="$HOME/.local/bin/dotf"
fi
if [ -n "$_dotf" ]; then
    "$_dotf" harness refresh --repo "$CURRENT_DIR" || log_warning "dotf harness refresh failed (above); deploying the committed harness records"
else
    log_warning "dotf not found (PATH or ~/.local/bin) -- harness records not refreshed from the vault; deploying the committed ones"
fi

# Mirror the harness inputs into the deploy dir so `compile-harness.sh --check`
# (healthcheck section 12) and `dotf doctor` read a complete snapshot from
# ~/.dotfiles: harness/ plus every file harness/manifest.json declares as an
# injection target. `dotf harness mirror` is the one implementation for both
# OSes (WIN-007/#1288). It replaced the bash+jq block that lived here, which
# derived its target list from the manifest for the reason #1200 recorded (a
# hardcoded pair missed the third target) and resolved jq by path because the
# lookup raced its own install (#1202) -- both now moot in Go. Runs AFTER
# --refresh so the snapshot matches the refreshed repo state. Idempotent
# ("N updated, M unchanged"); never prunes (doctor --fix owns orphans, #802).
# A declared target the checkout lacks is named and exits non-zero after
# mirroring the rest: setup does not abort (it is long and idempotent), but the
# warning is loud and verify-setup.bats fails on the resulting gap.
if [ -n "$_dotf" ]; then
    "$_dotf" harness mirror --repo "$CURRENT_DIR" || log_warning "dotf harness mirror reported a gap (above) -- 'dotf doctor' will report harness drift"
else
    log_warning "dotf not found (PATH or ~/.local/bin) -- harness not mirrored to $DOTFILES_DIR; 'dotf doctor' will report harness drift"
fi
unset _dotf

# Claude Code
ensure_directory "$HOME/.claude"
ensure_directory "$HOME/.claude/skills"
# The Claude instruction file (~/.claude/CLAUDE.md) and its skills are deployed
# by the single `compile-harness.sh --deploy` call near the end of this script,
# which copies every harness/manifest.json agents.presence[] file and then
# injects the skill catalog and the persona presence regions into it (SDD-008,
# HARNESS-058). `dotf converge` runs the same deploy with a content-comparing
# plan and a post-condition probe (PLAT-001b). settings.json is merged by
# `dotf deploy` (claude-settings), and its hooks come from `dotf harness bind`.

# uv and poetry: uv is a mise-pinned CLI and poetry a packages.json uv-tool
# entry, both converged by the tools steps above (#2013 W2).

# Claude Code (primary AI coding agent — see ADR-009)
log_info "Setting up Claude Code CLI..."
if ! command -v claude >/dev/null 2>&1 && [ ! -x "$HOME/.local/bin/claude" ]; then
    log_info "Installing Claude Code CLI via official install script..."
    if curl -fsSL https://claude.ai/install.sh | bash; then
        log_success "Claude Code installed"
    elif command -v npm >/dev/null 2>&1; then
        log_info "Falling back to npm install for Claude Code..."
        npm install -g --prefix "$HOME/.local" @anthropic-ai/claude-code || log_warning "Claude Code install failed"
    else
        log_warning "Claude Code install failed — re-run setup or install manually"
    fi
else
    log_info "Claude Code already installed"
fi

# OpenCode (secondary AI coding agent — see ADR-009 and specs/AI-011-opencode-bootstrap)

# Idempotent per pattern-setup-script-idempotence:
#   - the binary is a packages.json catalog tool (npm on every OS, AI-034/#1294,
#     ADR-036): `dotf tools install` above converges it; nothing installs here
#   - reconcile config (not skip-if-exists) so source-of-truth wins on drift
#   - no silenced errors; failures surface as log_warning
log_info "Setting up OpenCode configuration..."
# The curl-script install into ~/.opencode/bin and its "last token of the
# first line" version parse lived here until AI-034. A leftover copy from that
# channel is reported by `dotf doctor` (a catalog tool resolving from two PATH
# directories) and is safe to delete once `command -v opencode` resolves the
# npm one; the rc files no longer put ~/.opencode/bin on PATH.

# Deploy opencode.jsonc with deploy-time {env:VAR} substitution (SDD-009).
# Source ships placeholders like {env:NAN_API_KEY}; we substitute the literal
# age-decrypted value at deploy time so the deployed config is self-contained
# (no runtime env-var propagation needed when opencode launches from a
# non-shell parent). Placeholders without a resolvable mapping are left intact
# and opencode's runtime resolver acts as fallback.
ensure_directory "$HOME/.config/opencode"
OPENCODE_CONFIG_SRC="$CURRENT_DIR/ai/opencode/opencode.jsonc"
OPENCODE_CONFIG_DST="$HOME/.config/opencode/opencode.jsonc"
if [ -f "$OPENCODE_CONFIG_SRC" ]; then
    OPENCODE_CONFIG_TMP=$(mktemp)
    cp "$OPENCODE_CONFIG_SRC" "$OPENCODE_CONFIG_TMP"
    # Deploy-time {env:VAR} materialization via the dotf CLI (over secrets/registry.yaml,
    # ADR-020/ADR-028). Gate on the subcommand SUCCEEDING, not just dotf's presence: a
    # stale dotf passes `command -v` but fails `secrets render`, and under set -e that
    # would abort setup. Running it in the `if` condition exempts it from set -e; if it
    # fails, the {env:VAR} placeholders are left intact for opencode's runtime resolver.
    if command -v dotf >/dev/null 2>&1 && dotf secrets render "$OPENCODE_CONFIG_TMP"; then
        : # materialized via dotf secrets render
    else
        log_warning "dotf secrets render unavailable; opencode.jsonc deployed with literal {env:VAR} placeholders (resolved at runtime)"
    fi
    mv "$OPENCODE_CONFIG_TMP" "$OPENCODE_CONFIG_DST"
    log_success "Deployed opencode.jsonc (deploy-time secrets) to $OPENCODE_CONFIG_DST"
else
    log_warning "opencode.jsonc source missing: $OPENCODE_CONFIG_SRC"
fi

# opencode's and pi's AGENTS.md (the canonical SSOT, read natively under that
# name) are harness/manifest.json agents.presence targets, deployed by
# `dotf harness instructions` through the `compile-harness.sh --deploy` call
# near the end of this script (#1843 B11).

# Deploy pi coding agent config (AI-025) — mirrors the opencode block so the two
# agents are interchangeable across Linux/Windows. pi reads ~/.pi/agent/.
# Install pinned via npm (guarded); models.json gets the same deploy-time secret
# substitution as opencode (SDD-009, self-contained cross-OS); AGENTS.md is the
# same canonical SSOT; settings.json is seeded only when absent (pi mutates it).
PI_AGENT_DIR="$HOME/.pi/agent"
PI_BIN="$HOME/.local/bin/pi"
if command -v npm >/dev/null 2>&1; then
    # Install into the manager-independent ~/.local prefix — the same dir that
    # already carries `claude`/`dotf` and is on PATH for login shells AND
    # GUI/ADE processes (e.g. Orca). A bare `npm i -g` lands in nvm's
    # per-node-version tree (~/.nvm/versions/node/<v>/bin), invisible to any
    # environment running a different node. Guard on the stable location, not
    # bare `command -v pi`, so a stale nvm-version copy cannot mask a missing
    # ~/.local launcher.
    if [ ! -x "$PI_BIN" ]; then
        PI_PKG="@earendil-works/pi-coding-agent${PI_VERSION:+@$PI_VERSION}"
        log_info "Installing pi ($PI_PKG) into ~/.local ..."
        if npm install -g --ignore-scripts --prefix "$HOME/.local" "$PI_PKG" >/dev/null 2>&1; then
            log_success "pi installed -> $PI_BIN"
        else
            log_warning "pi install failed — run: npm install -g --ignore-scripts --prefix \"\$HOME/.local\" $PI_PKG"
        fi
    else
        # REFACTOR-013: the pin is a MINIMUM. Presence alone left an outdated pi
        # in place forever (it was only ever WARNed by healthcheck, #474, never
        # converged here). Upgrade when installed < pin; leave a newer pi alone.
        installed_pi=$("$PI_BIN" --version 2>/dev/null | head -1 | awk '{print $NF}')
        if [ -n "$PI_VERSION" ] && ! version_gte "$installed_pi" "$PI_VERSION"; then
            log_info "pi ${installed_pi:-unknown} below pinned minimum $PI_VERSION — upgrading..."
            if npm install -g --ignore-scripts --prefix "$HOME/.local" "@earendil-works/pi-coding-agent@$PI_VERSION" >/dev/null 2>&1; then
                log_success "pi upgraded to $PI_VERSION"
            else
                log_warning "pi upgrade failed — run: npm install -g --ignore-scripts --prefix \"\$HOME/.local\" @earendil-works/pi-coding-agent@$PI_VERSION"
            fi
        else
            log_info "pi already installed ($PI_BIN, ${installed_pi:-unknown} >= ${PI_VERSION:-any})"
        fi
    fi
else
    log_warning "npm not found — skipping pi install (install Node.js, then re-run setup)"
fi

ensure_directory "$PI_AGENT_DIR"

# settings.json is SEED-IF-MISSING, unlike models.json/tui.json (dotf deploy): pi
# rewrites this file at runtime (lastChangelogVersion, theme, the model picked
# in the TUI), so it is the one deployed config the user's own tool edits.
# It previously used the same "copy unless identical" shape as its neighbours,
# which for a self-mutating file means copy ALWAYS -- tests/pi-config.bats
# forbids lastChangelogVersion in the committed copy, so `cmp` could never
# match once pi had run, and every setup run silently reset the user's theme
# and default model. Seed it, then never touch it again. The seed runs BEFORE
# `dotf deploy`: its pi-compaction entry merges into this file, and a merge
# onto a missing file would create one the seed then never fills (#1938).
PI_SETTINGS_SRC="$CURRENT_DIR/ai/pi/settings.json"
PI_SETTINGS_DST="$PI_AGENT_DIR/settings.json"
if [ -f "$PI_SETTINGS_SRC" ]; then
    if [ -f "$PI_SETTINGS_DST" ]; then
        log_info "pi settings.json present, preserving local edits"
    else
        cp "$PI_SETTINGS_SRC" "$PI_SETTINGS_DST"
        log_success "Seeded pi settings.json at $PI_SETTINGS_DST"
    fi
fi

# Agent configs are deployed by `dotf deploy` (CLI-039): one implementation for
# every OS, replacing the per-config copies that lived here and their twins in
# setup-windows.ps1. ADR-020 C7 keeps this script on the thin bootstrap; staging,
# rendering, comparing and installing a config is tooling logic and belongs in
# the CLI. The call names NO config: bare `dotf deploy` installs every entry
# ai/deploy.json declares, so a new entry is a manifest edit and not a change to
# two setup scripts -- `dotf deploy pi` left orca-keybindings declared and never
# installed (CLI-054, #1301). Resolved by path as well as by name, for the same
# reason as the harness mirror above (#1305: this process's PATH may not carry
# ~/.local/bin yet).
_dotf=""
if command -v dotf >/dev/null 2>&1; then _dotf="dotf"; elif [ -x "$HOME/.local/bin/dotf" ]; then _dotf="$HOME/.local/bin/dotf"; fi
if [ -n "$_dotf" ]; then
    "$_dotf" deploy || log_warning "dotf deploy failed -- run it again after setup, or see 'dotf doctor'"
    # After deploy, which writes the file the include names (#2207).
    "$_dotf" converge --only git-config || log_warning "dotf converge --only git-config failed -- a dotf older than the flag cannot run it; re-run setup once DOTF_VERSION carries it, or see 'dotf doctor'"
else
    log_warning "dotf not found (PATH or ~/.local/bin) -- skipping agent config deploy (run ./scripts/install-dotf.sh, then 'dotf deploy')"
fi
unset _dotf

# Field-level sync (AI-032, #1247): enabledModels is dotfiles-owned even once
# settings.json exists on the machine -- nothing pi itself writes at runtime
# touches that array, only theme/lastChangelogVersion/defaultModel are (the
# seed-if-missing guard above exists for exactly those three). Without this, a
# catalog addition in the repo (e.g. #1254) never reaches a machine whose
# settings.json already exists; the only way to pick it up was deleting the
# live file and re-running setup, which would also blow away the user's real
# theme/model choice. Runs unconditionally on a present destination -- on a
# freshly-seeded file the two are already equal, so it is a same-second no-op.
if [ -f "$PI_SETTINGS_DST" ] && [ -f "$PI_SETTINGS_SRC" ]; then
    if ! command -v jq >/dev/null 2>&1; then
        log_warning "jq not found — skipping pi enabledModels sync"
    else
        PI_MODELS_SRC_JSON=$(jq -c '.enabledModels' "$PI_SETTINGS_SRC" 2>/dev/null) || PI_MODELS_SRC_JSON=""
        if [ -z "$PI_MODELS_SRC_JSON" ] || [ "$PI_MODELS_SRC_JSON" = "null" ]; then
            log_warning "ai/pi/settings.json has no enabledModels — skipping pi enabledModels sync"
        elif jq -e --argjson m "$PI_MODELS_SRC_JSON" '.enabledModels == $m' "$PI_SETTINGS_DST" >/dev/null 2>&1; then
            log_info "pi enabledModels already in sync"
        else
            # Same directory as the destination, not $TMPDIR: a cross-filesystem
            # mv degrades to copy+delete (not atomic) and mktemp's 0600 mode
            # would silently tighten the destination's permissions on the first
            # sync. Copying the live file's mode before the swap keeps a rename
            # within one filesystem both atomic and permission-preserving. The
            # mode is read with GNU stat or, failing that, BSD stat: chmod
            # --reference is GNU-only and silently failed here on macOS.
            PI_SETTINGS_TMP=$(mktemp "$(dirname "$PI_SETTINGS_DST")/.settings.json.XXXXXX")
            if jq --argjson m "$PI_MODELS_SRC_JSON" '.enabledModels = $m' "$PI_SETTINGS_DST" > "$PI_SETTINGS_TMP" 2>/dev/null \
                && chmod "$(stat -c '%a' "$PI_SETTINGS_DST" 2>/dev/null || stat -f '%Lp' "$PI_SETTINGS_DST")" "$PI_SETTINGS_TMP" 2>/dev/null; then
                mv "$PI_SETTINGS_TMP" "$PI_SETTINGS_DST"
                log_success "Synced pi enabledModels (theme/defaultModel/lastChangelogVersion preserved)"
            else
                rm -f "$PI_SETTINGS_TMP"
                log_warning "failed to sync pi enabledModels — leaving $PI_SETTINGS_DST untouched"
            fi
        fi
    fi
fi

# pi packages (HARNESS-139, #1628): `dotf pi packages apply` converges pi on
# ai/pi/packages.json in BOTH directions -- installs what is declared, removes
# what is not, and archives the paths the manifest retires -- through pi's own
# CLI, never by writing settings.json (pi owns it, #754). It replaced a loop
# here that only ever installed, so a package dropped from the manifest stayed
# on every machine. The Go command keeps what that loop established: captured
# output, elapsed time on every outcome, the DOTFILES_SKIP_PI_PACKAGES skip
# (CI-002, #1478), and a warning rather than a failure without pi or npm.
# `--pi "$PI_BIN"`, never the `pi` shell function, which wraps `dotf secrets run`
# and fails on a locked vault.
_dotf=""
if command -v dotf >/dev/null 2>&1; then
    _dotf="dotf"
elif [ -x "$HOME/.local/bin/dotf" ]; then
    _dotf="$HOME/.local/bin/dotf"
fi
if [ -n "$_dotf" ]; then
    "$_dotf" pi packages apply --repo "$CURRENT_DIR" --pi "$PI_BIN" || log_warning "pi package reconcile reported a failure (above) -- run 'dotf pi packages check'"
else
    log_warning "dotf not found (PATH or ~/.local/bin) -- pi packages not reconciled"
fi
unset _dotf

# opencode's tui.json (theme, keybinds) carries no secrets and is the
# `opencode-tui` entry of ai/deploy.json, installed by the bare `dotf deploy`
# above (#1843 B11).

# opencode commands are deployed from the vault skill records by
# `compile-harness.sh --deploy` (SDD-008): each committed record under
# harness/skills/ whose targets[] includes opencode is rendered to a
# ~/.config/opencode/commands/<n>.md command file (name: dropped — opencode
# keys commands off the filename). This replaces the former
# ai/opencode/commands/ + skills-to-opencode.sh path; the per-skill targets[]
# now expresses what was the hard-coded Claude-only skip-list.

# Post-deploy assertion: binary reachable + version reports. Placement is
# `dotf tools install`'s (packages.json, AI-034/#1294); this only verifies it,
# through the one version probe (ADR-036) rather than a local parse.
if command -v opencode >/dev/null 2>&1; then
    _dotf=""
    if command -v dotf >/dev/null 2>&1; then _dotf="dotf"; elif [ -x "$HOME/.local/bin/dotf" ]; then _dotf="$HOME/.local/bin/dotf"; fi
    opencode_ready_version=""
    if [ -n "$_dotf" ]; then
        opencode_ready_version=$("$_dotf" tools version opencode 2>/dev/null) || opencode_ready_version=""
    fi
    unset _dotf
    log_success "opencode ready: ${opencode_ready_version:-unknown}"
else
    log_warning "opencode not reachable on PATH after 'dotf tools install' — agent unavailable (re-run 'dotf tools install opencode')"
fi

# GitHub Copilot CLI (BUG-003: standalone agentic CLI, drops legacy gh-copilot
# extension path). Since AI-038 (#1321, ADR-036) it is an npm catalog tool that
# `dotf tools install` above converges on every OS (pin as floor in
# packages.json); this block only deploys config when the binary is on PATH.
# Verification string set by AI-013 (pointer-style copilot-instructions.md).
#
# A loop stripping a stale 'eval "$(gh copilot alias -- bash)"' line from
# .zshrc/.bashrc lived here until OPS-040. Probed absent from both files on the
# only OS it ran on before removal.

# Its instruction file (~/.copilot/copilot-instructions.md) is an
# agents.presence[] entry gated on `requires_command: copilot`, deployed by the
# `compile-harness.sh --deploy` call near the end of this script. settings.json,
# config.json and mcp-config.json are `dotf deploy` entries (ai/deploy.json,
# AI-039/#1322): the first two by MERGE, because the CLI writes both files itself
# and a verbatim copy wiped the box's own keys (allowedUrls, effortLevel, ...).

# SDD-005 parity (.github/copilot-instructions.md vs ai/copilot/): NOT synced here.
# Setup deploys to $HOME only and MUST NEVER write into the checkout — a checkout
# write leaves `git status` dirty, and `dotf update` skips any dirty worktree,
# so a self-deploying machine silently stops updating after the first run
# (dotfiles#694). The two copilot-instructions files are kept in parity by the
# fail-loud test tests/docs-drift.bats (runs in CI), which is the correct
# enforcement point: drift blocks the merge instead of a deploy rewriting a
# committed file behind the user's back.

# BUG-004 + BUG-011: defense-in-depth wrapper around EVERY `claude <subcommand>`
# invocation in this script. The Claude Code CLI's deserialize-modify-serialize
# cycle drops fields outside its internal struct (organizationType,
# organizationRateLimitTier, projects map, onboarding flags) -- upstream bug
# anthropics/claude-code#59870 -- shrinking ~/.claude/.claude.json from ~75 KB
# to ~1.5 KB and forcing re-authentication. The bug fires on ANY subcommand
# (`plugin install`, `plugin list`, `mcp get`, `mcp add`), not just install.
#
# BUG-004 (PR #57) wrapped only `claude plugin install`. BUG-011 extends the
# guard to every other call site (MCP loop iterations + plugin list pre-fetch)
# after the user empirically observed truncation recurrence in an MCP-only path.
#
# snapshot_claude_json copies the file to a tempfile BEFORE the call;
# restore_claude_json_if_truncated restores it AFTER, iff the snapshot was
# >= 10 KB and the new file is < 50% of the snapshot size. Complementary to
# SDD-021 session-start canary in dotf mem session-start (same 10240-byte
# threshold, same upstream issue). See dotfiles#33 for the original incomplete
# trigger fix that motivated this layer.
snapshot_claude_json() {
    local claude_json="$HOME/.claude/.claude.json"
    [ -f "$claude_json" ] || return 0
    local backup
    backup=$(mktemp "${TMPDIR:-/tmp}/.claude.json.bug004.XXXXXX")
    cp -f "$claude_json" "$backup"
    printf '%s' "$backup"
}

restore_claude_json_if_truncated() {
    local backup="$1"
    [ -n "$backup" ] && [ -f "$backup" ] || return 0
    local claude_json="$HOME/.claude/.claude.json"
    if [ -f "$claude_json" ]; then
        local snapshot_size new_size half
        snapshot_size=$(stat -c %s "$backup" 2>/dev/null || stat -f %z "$backup" 2>/dev/null || echo 0)
        new_size=$(stat -c %s "$claude_json" 2>/dev/null || stat -f %z "$claude_json" 2>/dev/null || echo 0)
        half=$((snapshot_size / 2))
        if [ "$snapshot_size" -ge 10240 ] && [ "$new_size" -lt "$half" ]; then
            cp -f "$backup" "$claude_json"
            log_warning ".claude.json shrunk from $snapshot_size to $new_size bytes after install (upstream #59870); restored from backup"
        fi
    fi
    rm -f "$backup"
}

# Register MCP servers (requires Claude Code CLI, Node.js, jq)
# Idempotent: server list lives in mcp-servers.json (SSOT shared with Windows);
# `claude mcp get` is used to skip already-registered entries, and `add` errors
# are surfaced rather than swallowed. BUG-011: every `claude mcp {get,add}`
# invocation is wrapped with snapshot_claude_json / restore_claude_json_if_truncated
# because both subcommands hit the same #59870 truncation path as `plugin install`.
MCP_CONFIG="$DOTFILES_DIR/mcp-servers.json"
if command -v claude >/dev/null 2>&1 && command -v npx >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    if [ ! -f "$MCP_CONFIG" ]; then
        log_warning "mcp-servers.json not found at $MCP_CONFIG, skipping MCP registration"
    else
        log_info "Registering Claude Code MCP servers from $MCP_CONFIG..."
        mcp_added=0
        mcp_skipped=0
        mcp_failed=0
        # HIVE-118: migrate a stale `uvx hive-vault` entry by REMOVING it here so
        # the skip-if-present loop below re-adds the current definition
        # (`hive client`) from mcp-servers.json. Remove-only -- the loop owns every
        # `claude mcp add` so this stays SSOT-driven (no hardcoded server add).
        if claude mcp get hive 2>/dev/null | grep -qE 'uvx|hive-vault'; then
            _snap=$(snapshot_claude_json)
            if claude mcp remove hive --scope user >/dev/null 2>&1; then
                log_info "Migrating hive MCP entry: uvx hive-vault -> hive client (via mcp-servers.json)"
            fi
            restore_claude_json_if_truncated "$_snap"
        fi
        while IFS=$'\t' read -r mcp_name mcp_transport mcp_args mcp_prereq_bin mcp_prereq_cmd; do
            # Check prerequisite binary, run prerequisite command if specified
            if [ -n "$mcp_prereq_bin" ]; then
                if ! command -v "$mcp_prereq_bin" >/dev/null 2>&1; then
                    log_warning "MCP $mcp_name: prerequisite '$mcp_prereq_bin' not found, skipping"
                    mcp_failed=$((mcp_failed + 1))
                    continue
                fi
                if [ -n "$mcp_prereq_cmd" ]; then
                    # shellcheck disable=SC2086
                    if ! $mcp_prereq_cmd >/dev/null 2>&1; then
                        log_warning "MCP $mcp_name: prerequisite command failed: $mcp_prereq_cmd"
                    fi
                fi
            fi
            # BUG-011: snapshot before both `mcp get` and `mcp add` (one snapshot
            # per iteration -- legitimate `mcp add` additions are <<50% of file
            # size, so restore only fires on the real #59870 truncation).
            _snap=$(snapshot_claude_json)
            # Idempotence: skip if `claude mcp get` already knows this name
            if claude mcp get "$mcp_name" >/dev/null 2>&1; then
                log_info "MCP $mcp_name already registered, skipping"
                mcp_skipped=$((mcp_skipped + 1))
                restore_claude_json_if_truncated "$_snap"
                continue
            fi
            # Word-split args intentionally; entries in mcp-servers.json are
            # tokenized for `claude mcp add` argv after `--`.
            # shellcheck disable=SC2086
            if mcp_err=$(claude mcp add --transport "$mcp_transport" "$mcp_name" --scope user -- $mcp_args 2>&1); then
                log_success "Registered MCP $mcp_name"
                mcp_added=$((mcp_added + 1))
            else
                log_warning "Failed to register MCP $mcp_name: $mcp_err"
                mcp_failed=$((mcp_failed + 1))
            fi
            restore_claude_json_if_truncated "$_snap"
        done < <(jq -r '.servers[] | [.name, .transport, .args, (.prerequisite_binary // ""), (.prerequisite_command // "")] | @tsv' "$MCP_CONFIG")
        log_success "MCP servers: $mcp_added added, $mcp_skipped already present, $mcp_failed failed"
    fi
else
    log_warning "Claude Code CLI, npx, or jq not found, skipping MCP server registration"
fi

# Phase C daemon supervision (HIVE-118 / hive#176). Install the supervised
# `hive serve` service now that the MCP loop's prerequisite has installed/upgraded
# the tool. Gated on hive-vault >= 1.32.0 via the package version (NOT by probing
# `hive service`, which an older hive routes to the blocking stdio server). The MCP
# entry itself is already `hive client` (re-added from mcp-servers.json above).
# Non-fatal: a missing systemd --user or any failure leaves the in-process fallback.
# Probe the BINARY, not the installer registry: hive moved to its own installer
# and `uv tool list` stopped seeing a healthy install (AI-028/#791), so this
# gate skipped daemon supervision with "hive <unknown> predates 'hive service'"
# while `hive --version` answered 3.0.0. `dotf tools version` is the one semver
# extraction both OSes share (ADR-036); resolved by path as well as by name
# because ~/.local/bin may not be on this process's PATH (#1202 class).
hive_ver=""
if command -v dotf >/dev/null 2>&1; then
    hive_ver=$(dotf tools version hive 2>/dev/null || true)
elif [ -x "$HOME/.local/bin/dotf" ]; then
    hive_ver=$("$HOME/.local/bin/dotf" tools version hive 2>/dev/null || true)
fi
if command -v hive >/dev/null 2>&1 && [ -n "$hive_ver" ] \
    && [ "$(printf '1.32.0\n%s\n' "$hive_ver" | sort -V | head -1)" = "1.32.0" ]; then
    if hive service install >/dev/null 2>&1; then
        log_success "Installed hive daemon service (systemd --user, v$hive_ver)"
    else
        log_warning "hive service install failed (non-fatal; client works via fallback)"
    fi

    # CLI-042 AC7: the NaN credential reaches the daemon through
    # `dotf secrets run`, never through a file. `hive service install` above owns
    # hive.service and exposes no flag to change its ExecStart, so the override
    # is a systemd DROP-IN. That also makes it survive the next re-install --
    # which the hive-upgrade.timer can trigger every 15 minutes, and which would
    # silently clobber a competing hive.service shipped from this repo.
    if command -v systemctl >/dev/null 2>&1; then
        HIVE_DROPIN_SRC="$CURRENT_DIR/systemd/hive.service.d/10-dotf-secrets.conf"
        HIVE_DROPIN_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/hive.service.d"
        HIVE_DROPIN_DST="$HIVE_DROPIN_DIR/10-dotf-secrets.conf"
        if [ -f "$HIVE_DROPIN_SRC" ]; then
            ensure_directory "$HIVE_DROPIN_DIR"
            # Compare before writing. deploy_file writes unconditionally, and a
            # daemon restarted on every setup run is NOT idempotent -- it would
            # drop live MCP sessions on each pass. Content equality is the
            # convergence test, so only a real change earns a restart and a
            # re-run reports changed=0.
            if [ -f "$HIVE_DROPIN_DST" ] && cmp -s "$HIVE_DROPIN_SRC" "$HIVE_DROPIN_DST"; then
                log_info "hive.service credential drop-in already current (no restart)"
            else
                deploy_file "$HIVE_DROPIN_SRC" "$HIVE_DROPIN_DST"
                systemctl --user daemon-reload >/dev/null 2>&1 || true
                if systemctl --user restart hive.service >/dev/null 2>&1; then
                    log_success "hive daemon takes NAN_API_KEY from dotf secrets run (nothing on disk)"
                else
                    log_warning "hive.service restart failed; drop-in deployed but the running daemon keeps the old ExecStart"
                fi
            fi
        fi
    fi

    # ADR-025 + HARNESS-024 (#446): the hive serve daemon is a systemd --user
    # service and does NOT source the shell paths.sh, so provision its vault path
    # via environment.d (read by the user manager for all --user services). The
    # value follows the ADR-025 cascade ($VAULT_PATH -> machine.json -> contract).
    # hive#246 hardens the daemon's own resolution; this guarantees the env is set.
    HIVE_VAULT_RESOLVED="$(dotf env path HIVE_VAULT_PATH 2>/dev/null || true)"
    HIVE_VAULT_RESOLVED="${HIVE_VAULT_RESOLVED:-${HIVE_VAULT_PATH:-${VAULT_PATH:-$HOME/Projects/knowledge}}}"
    ENV_D_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/environment.d"
    if mkdir -p "$ENV_D_DIR" 2>/dev/null; then
        printf 'HIVE_VAULT_PATH=%s\nVAULT_PATH=%s\n' "$HIVE_VAULT_RESOLVED" "$HIVE_VAULT_RESOLVED" > "$ENV_D_DIR/10-hive-vault.conf"
        export HIVE_VAULT_PATH="$HIVE_VAULT_RESOLVED"
        systemctl --user import-environment HIVE_VAULT_PATH VAULT_PATH >/dev/null 2>&1 || true
        log_success "Provisioned hive daemon vault path via environment.d ($HIVE_VAULT_RESOLVED)"
    fi

    # AI-023 / hive#176: the upgrade policy that FEEDS the daemon's
    # restart-on-upgrade. Deploy the --user timer + oneshot from the repo's
    # systemd/ SSOT and enable the timer (every 15 min). Same version gate as the
    # service above -- daemon + its feeder install together or not at all.
    if command -v systemctl >/dev/null 2>&1; then
        SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
        ensure_directory "$SYSTEMD_USER_DIR"
        if cp -f "$CURRENT_DIR/systemd/hive-upgrade.service" "$SYSTEMD_USER_DIR/" \
            && cp -f "$CURRENT_DIR/systemd/hive-upgrade.timer" "$SYSTEMD_USER_DIR/"; then
            systemctl --user daemon-reload >/dev/null 2>&1 || true
            if systemctl --user enable --now hive-upgrade.timer >/dev/null 2>&1; then
                log_success "Enabled hive-upgrade.timer (every 15 min, systemd --user)"
                # A removal of the legacy 'uv tool upgrade hive-vault' crontab
                # line ran here until OPS-040, retiring it once the timer owned
                # upgrade policy. Probed absent from `crontab -l` before removal.
            else
                log_warning "hive-upgrade.timer enable failed (non-fatal; setup still upgrades hive each run)"
            fi
        else
            log_warning "Could not deploy hive-upgrade units to $SYSTEMD_USER_DIR (non-fatal)"
        fi
    else
        log_warning "systemctl not found; skipping hive-upgrade.timer (setup still upgrades hive each run)"
    fi
elif command -v hive >/dev/null 2>&1; then
    log_warning "hive ${hive_ver:-<unknown>} predates 'hive service' (need >= 1.32.0); skipping daemon supervision"
fi

# OPS-001: opt-in self-deploy timer (git pull --ff-only + idempotent setup).
# Tri-state, gated on DOTFILES_AUTODEPLOY so a normal setup never touches it:
#   1     -> deploy units + enable --now the daily --user timer
#   0     -> disable + remove the timer (clean opt-out / teardown)
#   unset -> no-op (opt-in, default OFF)
# Non-fatal: any failure leaves setup succeeding; the timer is a convenience, and
# `dotf update` itself no-ops on a dirty/diverged repo (OPS-001).
if command -v systemctl >/dev/null 2>&1; then
    SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
    case "${DOTFILES_AUTODEPLOY:-}" in
        1)
            ensure_directory "$SYSTEMD_USER_DIR"
            if cp -f "$CURRENT_DIR/systemd/dotfiles-selfupdate.service" "$SYSTEMD_USER_DIR/" \
                && cp -f "$CURRENT_DIR/systemd/dotfiles-selfupdate.timer" "$SYSTEMD_USER_DIR/"; then
                systemctl --user daemon-reload >/dev/null 2>&1 || true
                if systemctl --user enable --now dotfiles-selfupdate.timer >/dev/null 2>&1; then
                    log_success "Enabled dotfiles-selfupdate.timer (daily self-deploy, systemd --user)"
                else
                    log_warning "dotfiles-selfupdate.timer enable failed (non-fatal; re-run setup manually to update)"
                fi
            else
                log_warning "Could not deploy dotfiles-selfupdate units to $SYSTEMD_USER_DIR (non-fatal)"
            fi
            ;;
        0)
            systemctl --user disable --now dotfiles-selfupdate.timer >/dev/null 2>&1 || true
            rm -f "$SYSTEMD_USER_DIR/dotfiles-selfupdate.timer" "$SYSTEMD_USER_DIR/dotfiles-selfupdate.service"
            systemctl --user daemon-reload >/dev/null 2>&1 || true
            log_info "Disabled + removed dotfiles-selfupdate.timer (DOTFILES_AUTODEPLOY=0)"
            ;;
        *)
            : # unset -> no-op (opt-in, default OFF)
            ;;
    esac
fi

# Claude Code plugins (requires claude CLI).
# Idempotent: cache the installed-plugins list ONCE before the loop and skip
# entries already present. The wrapper above (BUG-004/011) catches the
# false-negative case where the idempotence guard misses a plugin and the
# resulting `claude plugin install` call truncates .claude.json. BUG-011: the
# pre-loop `claude plugin list` is now also wrapped because it goes through the
# same #59870 path.
if command -v claude >/dev/null 2>&1; then
    log_info "Installing Claude Code plugins..."
    # BUG-011: wrap the read-only `claude plugin list` pre-fetch with the
    # snapshot guard -- the CLI still rewrites .claude.json on any invocation.
    _snap=$(snapshot_claude_json)
    installed_plugins=$(claude plugin list 2>/dev/null || true)
    restore_claude_json_if_truncated "$_snap"
    plugins_added=0
    plugins_skipped=0
    for plugin in \
        "gopls-lsp@claude-plugins-official" \
        "security-guidance@claude-plugins-official" \
        "frontend-design@claude-plugins-official"; do
        if printf '%s' "$installed_plugins" | grep -qF "$plugin"; then
            plugins_skipped=$((plugins_skipped + 1))
        else
            # BUG-004: wrap the install with snapshot/restore so the upstream
            # truncation bug (#59870) cannot drop subscription state.
            _snap=$(snapshot_claude_json)
            if claude plugin install "$plugin" >/dev/null 2>&1; then
                plugins_added=$((plugins_added + 1))
            fi
            restore_claude_json_if_truncated "$_snap"
        fi
    done
    log_success "Claude Code plugins ready ($plugins_added added, $plugins_skipped already present)"
else
    log_warning "Claude Code CLI not found, skipping plugin installation"
fi

# ~/.claude/settings.json is the `claude-settings` entry of ai/deploy.json, merged
# by the `dotf deploy` above (CLI-063, #2000). Its per-key policy and the no-trailer
# attribution are pinned by cli/internal/deploy/claude_settings_test.go.

# HARNESS-045 AC1: hooks are emitted by `dotf harness bind`, for every harness
# declared in harness/manifest.json's `agents.bind` -- not just Claude's. It is
# the ONLY writer of the `hooks` key, because the jq assignment this replaced
# deleted whatever third-party group it did not know about.
#
# Gate on the CAPABILITY, never on the exit status (lesson 219): a dotf predating
# the subcommand rejects the flags with exit 1, which is indistinguishable from a
# genuine refusal, and a differently-stale one prints help and exits 0. The grep
# runs against a CAPTURED variable rather than through a pipe -- `cmd | grep -q`
# closes the pipe on first match, and under pipefail the SIGPIPE'd producer makes
# the pipeline exit 141, reporting "too old" for a binary that just proved it is
# current. TestHarnessHelpListsSubcommands pins the string from the Go side.
_dotf=""
if command -v dotf >/dev/null 2>&1; then
    _dotf="dotf"
elif [ -x "$HOME/.local/bin/dotf" ]; then
    _dotf="$HOME/.local/bin/dotf"
fi
if [ -z "$_dotf" ]; then
    log_warning "dotf not found (PATH or ~/.local/bin) -- harness hooks NOT emitted; sessions run without the memory hooks and the gate"
else
    _harness_help="$("$_dotf" harness --help 2>/dev/null || true)"
    if printf '%s\n' "$_harness_help" | grep -q '^[[:space:]]*bind[[:space:]]'; then
        # --repo-root is passed EXPLICITLY, not left to env.ResolveHarnessRoot.
        # That resolver walks up from the CWD for a .git, then falls back to
        # ~/.dotfiles. Measured under a controlled env: from a cwd outside any
        # checkout on a machine with no ~/.dotfiles yet -- a first run invoked by
        # absolute path -- it resolves to a directory with no manifest, and bind
        # exits 1 having emitted no hooks. setup knows the checkout it is running
        # from, so inferring it from ambient state is strictly worse.
        "$_dotf" harness bind --repo-root "$CURRENT_DIR" || log_warning "dotf harness bind reported a problem (above) -- 'dotf doctor' will report hook drift"
    else
        log_warning "installed dotf predates 'dotf harness bind' -- harness hooks NOT emitted; rebuild/reinstall dotf and re-run setup"
    fi
    unset _harness_help
fi
unset _dotf

# Claude's auto-memory dir is linked to its vault source per project by
# memlink (cli/internal/memlink): the session-start hook links the project a
# session opens, on every OS, and `dotf doctor --fix` repairs the current one,
# a dangling link included. A real memory dir is never moved aside (#1843 B13).

# Deploy all skills from the committed records to their per-agent $HOME paths
# (SDD-008, option A). Renders each harness/skills/<n> record to ~/.claude/skills,
# ~/.config/opencode/commands, ~/.gemini/skills + ~/.gemini/prompts, and injects
# the copilot catalog into ~/.copilot/copilot-instructions.md, honoring each
# skill's targets[]. Deploy is a regular copy — never a symlink — and replaces
# any pre-existing vault symlink with a copy first, ending the BUG-100 fragility
# class. Records are committed, so this runs offline (no vault required); the
# earlier --refresh re-renders the records from the vault SSOT when present.
# Runs last so the per-agent base files (e.g. ~/.copilot/copilot-instructions.md)
# are already in place for catalog injection.
if ( cd "$CURRENT_DIR" && "$CURRENT_DIR/scripts/compile-harness.sh" --deploy ) >/dev/null; then
    log_success "Skills deployed from records (claude / opencode / agy / copilot)"
else
    log_warning "compile-harness --deploy reported issues; check skill records"
fi

# Weekly vault maintenance cron (Sundays 10:00 AM)
log_info "Setting up weekly vault maintenance cron..."
if command -v crontab >/dev/null 2>&1; then
    CRON_CMD="7 10 * * 0 $CURRENT_DIR/scripts/vault-maintenance-weekly.sh"
    if crontab -l 2>/dev/null | grep -q "vault-maintenance-weekly"; then
        log_info "Weekly vault maintenance cron already installed"
    else
        # `crontab -l` exits 1 when the user has no crontab yet. Under the
        # script's `set -euo pipefail` that kills the subshell before the echo
        # runs, and pipefail then aborts the whole setup — so a fresh node never
        # gets past this line. `|| true` keeps the empty-crontab case benign.
        (crontab -l 2>/dev/null || true; echo "$CRON_CMD # dotfiles: vault-maintenance-weekly") | crontab -
        log_success "Installed weekly vault maintenance cron (Sundays 10:07)"
    fi
else
    log_warning "crontab not available, skipping weekly maintenance cron"
fi

# Test if files are correctly linked
log_success "Installation completed! Verifying file links..."

# OPS-043: the per-file byte comparison that used to live here (check_deployed
# x3) and the 14-tool check_dependencies sweep are now `dotf doctor`'s, which
# runs at the tail of this same script. Both moved rather than being dropped:
# doctor's `Deploy-dir<->$HOME drift` section carries the byte comparison AND
# check_deployed's symlink rejection, over eleven files instead of three, and
# every one of the fourteen tools has an equal-or-stronger verdict there --
# asserted by TestSetupShellParity, not by this comment. setup-windows.ps1 has
# had no such block since CLI-018; this brings Linux to the same shape.
#
# The exemptions the deleted NOTEs recorded now live beside the data they
# describe, in doctor's homeDeployMap, each with its own reason.
file_exists "$HOME/.bash/bash_aliases" && log_success "bash_aliases created" || log_error "bash_aliases issue"

# Deploy env-contract.json so `dotf doctor` can find it under $DOTFILES_DIR.
if [ -f "$CURRENT_DIR/env-contract.json" ]; then
    cp -f "$CURRENT_DIR/env-contract.json" "$DOTFILES_DIR/env-contract.json"
    log_success "Deployed env-contract.json to $DOTFILES_DIR/"
fi

# Render the per-machine path file (ADR-025) from the contract + the
# ~/.config/dotfiles/machine.json overrides, so the profiles source it and the
# `dotf doctor` run below sees no path drift. Sourcing it here makes this setup
# shell match what the next login shell will see.
if command -v dotf >/dev/null 2>&1; then
    # Seed DOTFILES_REPO_DIR into machine.json to the checkout setup runs from,
    # BEFORE generating the path file, so the cascade (and the generated paths.sh)
    # resolve the real repo instead of the phantom contract default — otherwise
    # `dotf update`/`mem` no-op on a fresh machine (BUG-029/#696). Idempotent and
    # preserves any other overrides (e.g. VAULT_PATH).
    # $CURRENT_DIR is where setup was INVOKED, which is not the repo when setup
    # runs from a linked worktree — an ordinary thing to do while testing a setup
    # change. Seeding a worktree writes a throwaway path into the machine SSOT,
    # and every check stays green until that worktree is deleted, at which point
    # the whole box cannot commit (measured 2026-08-31; lesson 247). Skipping the
    # seed instead would reopen BUG-029, so resolve the canonical checkout rather
    # than declining: --git-dir and --git-common-dir are equal in a normal
    # checkout and differ inside a linked worktree, where the common dir IS the
    # real repo's .git.
    SEED_REPO_DIR="$CURRENT_DIR"
    _seed_gd="$(git -C "$CURRENT_DIR" rev-parse --git-dir 2>/dev/null || true)"
    _seed_gcd="$(git -C "$CURRENT_DIR" rev-parse --git-common-dir 2>/dev/null || true)"
    if [ -n "$_seed_gcd" ] && [ "$_seed_gd" != "$_seed_gcd" ]; then
        SEED_REPO_DIR="$(dirname "$_seed_gcd")"
        log_warning "setup ran from a linked worktree; seeding the canonical checkout $SEED_REPO_DIR, not $CURRENT_DIR"
    fi
    if dotf env set DOTFILES_REPO_DIR "$SEED_REPO_DIR" >/dev/null; then
        log_success "Seeded DOTFILES_REPO_DIR=$SEED_REPO_DIR in machine.json"
    else
        log_warning "dotf env set DOTFILES_REPO_DIR failed (update/mem fall back to the git walk-up)"
    fi
    if dotf env generate; then
        log_success "Generated $DOTFILES_DIR/paths.sh (dotf env generate)"
        # shellcheck disable=SC1091
        [ -f "$DOTFILES_DIR/paths.sh" ] && . "$DOTFILES_DIR/paths.sh"
    else
        log_warning "dotf env generate failed (profiles fall back to inline defaults)"
    fi
fi

# Re-enforce main dotfiles at the end of setup to overwrite any third-party installer mutations.
if command -v dotf >/dev/null 2>&1; then dotf deploy zshrc; elif [ -x "$HOME/.local/bin/dotf" ]; then "$HOME/.local/bin/dotf" deploy zshrc; fi
[ -f "$DOTFILES_DIR/.bashrc" ] && deploy_file "$DOTFILES_DIR/.bashrc" "$HOME/.bashrc"
[ -f "$DOTFILES_DIR/.profile" ] && deploy_file "$DOTFILES_DIR/.profile" "$HOME/.profile"

# Final assertion against env-contract.json -- catches drift between what
# setup just deployed and what's actually in place / on PATH / in env vars.
#
# Pre-export the REFACTOR-002 path vars so `dotf doctor` sees what the deployed
# RC files WILL set on the next shell. Without this, every fresh setup run
# reports false warnings because the running shell hasn't re-sourced .zshrc/.bashrc.
# Values must match the corresponding `export` lines in .zshrc + .bashrc.
export SCRIPTS_DIR="${SCRIPTS_DIR:-$DOTFILES_DIR/scripts}"
export GEMINI_HOME="${GEMINI_HOME:-$HOME/.gemini}"
# AGY_HOME (SSOT name per env-contract.json) lives inside GEMINI_HOME, since
# agy inherits the legacy ~/.gemini/ path for backwards compat with gemini-cli.
export AGY_HOME="${AGY_HOME:-$GEMINI_HOME/antigravity-cli}"
export COPILOT_HOME="${COPILOT_HOME:-$HOME/.copilot}"
export OPENCODE_HOME="${OPENCODE_HOME:-$HOME/.config/opencode}"
# BUG-021 (2026-05-21): pre-export DOTFILES_REPO_DIR so `dotf doctor` (which
# byte-compares the repo against the deployed copy, CLI-019) sees it set even
# when the running shell's profile hasn't been re-evaluated post-deploy.
# Mirror of the setup-windows.ps1 fix.
export DOTFILES_REPO_DIR="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"

# Final consolidated diagnostics: `dotf doctor` (ADR-021, CLI-012) replaces the
# retired doctor.sh + healthcheck.sh twins in one cross-compiled sweep. Non-fatal:
# surfaces deploy gaps (env-contract drift, version/symlink/vault mismatches) but
# does NOT alter setup's exit code. Needs dotf on PATH (installed above by
# install_dotf); a clean box where that step failed skips gracefully.
if command -v dotf >/dev/null 2>&1; then
    log_info "Running post-setup diagnostics (dotf doctor)..."
    echo
    dotf doctor || log_warning "dotf doctor reported one or more FAIL items -- review output above; re-run with 'dotf doctor'"
    echo
else
    log_warning "dotf not on PATH -- skipping post-setup diagnostics (run ./scripts/install-dotf.sh, then 'dotf doctor')"
fi

log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"
