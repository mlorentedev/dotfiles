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
# from the checkout so the pins it reads are reliable
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

# ~/.zshrc, ~/.zsh/* and ~/.tmux.conf are `dotf deploy` entries (ai/deploy.json:
# zshrc, zsh-*, tmux; #1843 B2, PLAT-001 P6), and so are ~/.bashrc, ~/.profile
# and ~/.inputrc on linux and darwin (bashrc, profile, inputrc; #2013 P7). The
# bare `dotf deploy` below installs them, with a one-time .pre-dotf backup of a
# file the machine already had. Edit in the repo, never in $HOME.

# ~/.ssh/config (0600) and the public key are `dotf deploy` entries on every
# OS (ai/deploy.json ssh-config, ssh-pubkey; #2013 P7).

# ~/.gitconfig is co-owned by git, gh and the user, so nothing deploys over it
# (#2207): `dotf deploy` below writes ~/.config/git/dotfiles.gitconfig and
# `dotf converge --only git-config` after it adds the include and gh's helper.
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
# `dotf doctor`, where checkDeployManifest compares the rc files now that they
# are ai/deploy.json entries. The reason this comment records still holds --
# only the thing that would catch a regression changed name and file.

# Create a bash_aliases file for bash
log_info "Creating bash_aliases file..."
ensure_directory "$HOME/.bash"
cat > "$HOME/.bash/bash_aliases" << EOF
# Bash Aliases
$(grep -v '^\s*#' "$DOTFILES_DIR/.zsh/aliases.zsh" | grep "^alias ")
EOF

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
# installed by `dotf tools sync` below (mise, ADR-044). setup has no installer
# of its own for any of these tools: the hard-coded linux-amd64 downloads are
# gone (#2013 W2, F-030), and tests/setup-no-arch-downloads.bats keeps them out.

# dotf (the dotfiles Go CLI — ADR-020). Fetch the pinned release binary,
# checksum-verified, into ~/.local/bin. Idempotent; converges on drift.
# The preflight of every dotf step below: without it setup cannot finish, so a
# failure stops here rather than in a box that only looks set up (#2013 W4).
if [ -f ./install.sh ]; then
    # shellcheck source=/dev/null
    . ./install.sh
    install_dotf || { log_error "dotf installation failed -- see the error above; setup cannot continue without it"; exit 1; }
else
    log_error "install.sh not found -- run setup from the root of a dotfiles checkout"
    exit 1
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

# Setup Antigravity CLI configuration (fresh-install model, SDD-007).
# Strategy: write canonical config to where agy reads from (~/.gemini/config/),
# use deploy_file (atomic + idempotent), no symlinks. Closes #100.
log_info "Setting up Antigravity CLI configuration..."
export GEMINI_HOME="$HOME/.gemini"
export AGY_HOME="${AGY_HOME:-$GEMINI_HOME/antigravity-cli}"
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
ensure_directory "$AGY_HOME"

# 1. agy settings.json, AGY.md and .geminiignore are `dotf deploy` entries
#    (ai/deploy.json `agy-settings`, `agy-instructions`, `agy-geminiignore`;
#    #1843 B11): one implementation for every OS, run by the bare `dotf deploy`
#    this script makes. settings.json renders {HOME} per machine (AI-042/#1334).

# 2. agy's MCP server configs (the master under ~/.gemini/config and the
#    hive-vault plugin) are written by the bare `dotf deploy` this script makes
#    and by `dotf converge` (CLI-105, #2384), which also drops the stale copy
#    an older layout left in $AGY_HOME.

# Drop project-local agy state cache from the workspace if agy leaked one
[ -d "$CURRENT_DIR/.antigravitycli" ] && rm -rf "$CURRENT_DIR/.antigravitycli"
[ -d "$CURRENT_DIR/.antigravitycli.bak" ] && rm -rf "$CURRENT_DIR/.antigravitycli.bak"
rm -f "$GEMINI_HOME/config/.migrated" 2>/dev/null

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
# ("N updated, M unchanged, K pruned"). It prunes .zsh/, ssh/ and scripts/ of
# files the checkout's history deleted, which the additive copy above leaves
# behind (#2266); harness/ and the secrets stay with doctor --fix (#802).
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

# ~/.config/opencode/opencode.jsonc is the `opencode` entry of ai/deploy.json,
# rendered with its {env:VAR} secrets by the bare `dotf deploy` below on every
# OS (SDD-009, #1843 B12).

# opencode's and pi's AGENTS.md (the canonical SSOT, read natively under that
# name) are harness/manifest.json agents.presence targets, deployed by
# `dotf harness instructions` through the `compile-harness.sh --deploy` call
# near the end of this script (#1843 B11).

# Deploy pi coding agent config (AI-025) — mirrors the opencode block so the two
# agents are interchangeable across Linux/Windows. pi reads ~/.pi/agent/. The
# CLI itself comes from mise (versions.conf `# mise: latest`, #2013 D9), which
# the tools step installs before anything reads a harness; models.json gets the
# same deploy-time secret substitution as opencode (SDD-009, self-contained
# cross-OS); AGENTS.md is the same canonical SSOT; settings.json belongs to
# the pi-settings converge step, which the `dotf converge --only pi-settings`
# line below runs just before the bare `dotf deploy` (deploy never runs it).

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
    # pi's settings.json before deploy, whose pi-compaction entry merges into
    # it: a merge onto a missing file creates one holding only compaction
    # (#1938). The step seeds the repo's copy when the file is absent and keeps
    # its enabledModels equal to the repo's, leaving pi's own keys alone (#1948).
    "$_dotf" converge --only pi-settings || log_warning "dotf converge --only pi-settings failed -- pi's settings.json was not seeded or synced; see 'dotf doctor'"
    "$_dotf" deploy || { log_error "dotf deploy failed -- the failed rows above name each config; fix them and run setup again"; exit 1; }
    # After deploy, which writes the file the include names (#2207).
    "$_dotf" converge --only git-config || log_warning "dotf converge --only git-config failed -- a dotf older than the flag cannot run it; re-run setup once DOTF_VERSION carries it, or see 'dotf doctor'"
    # Its own call: a dotf older than the step refuses the name, and must not
    # take git-config down with it (#2013 X1).
    "$_dotf" converge --only git-hooks || log_warning "dotf converge --only git-hooks failed -- a dotf older than the step cannot run it; re-run setup once DOTF_VERSION carries it, or see 'dotf doctor'"
else
    log_error "dotf not found (PATH or ~/.local/bin) after installing it -- no agent config can be deployed"
    exit 1
fi
unset _dotf

# pi packages (HARNESS-139, #1628): `dotf pi packages apply` converges pi on
# ai/pi/packages.json in BOTH directions -- installs what is declared, removes
# what is not, and archives the paths the manifest retires -- through pi's own
# CLI, never by writing settings.json (pi owns it, #754). It replaced a loop
# here that only ever installed, so a package dropped from the manifest stayed
# on every machine. The Go command keeps what that loop established: captured
# output, elapsed time on every outcome, the DOTFILES_SKIP_PI_PACKAGES skip
# (CI-002, #1478), and a warning rather than a failure without pi or npm.
# dotf resolves the pi binary itself (mise's, on PATH), never the `pi` shell
# function, which wraps `dotf secrets run` and fails on a locked vault.
_dotf=""
if command -v dotf >/dev/null 2>&1; then
    _dotf="dotf"
elif [ -x "$HOME/.local/bin/dotf" ]; then
    _dotf="$HOME/.local/bin/dotf"
fi
if [ -n "$_dotf" ]; then
    "$_dotf" pi packages apply --repo "$CURRENT_DIR" || log_warning "pi package reconcile reported a failure (above) -- run 'dotf pi packages check'"
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

# Claude Code's MCP servers (mcp-servers.json) are registered by the bare
# `dotf deploy` above (claude-mcp), which re-registers a server whose arguments
# drifted and guards .claude.json against the upstream truncation (#59870). The
# shell loop that did it here, with its own HIVE-118 remove and snapshot
# helpers, inherited setup's environment: with CLAUDE_CONFIG_DIR unset, claude
# kept a second .claude.json and the loop registered every server again there
# (from-zero run 38671157122: deploy `4 already present`, then the loop `4 added`).

# Phase C daemon supervision (HIVE-118 / hive#176). Install the supervised
# `hive serve` service now that `dotf tools install` has installed hive (the
# packages.json uv-tool entry, at its pin). Gated on hive-vault >= 1.32.0 via
# the package version (NOT by probing `hive service`, which an older hive
# routes to the blocking stdio server). The MCP entry itself is already
# `hive client` (registered from mcp-servers.json above).
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
    # is a systemd DROP-IN. That also makes it survive the next re-install,
    # which would silently clobber a competing hive.service shipped from this repo.
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

elif command -v hive >/dev/null 2>&1; then
    log_warning "hive ${hive_ver:-<unknown>} predates 'hive service' (need >= 1.32.0); skipping daemon supervision"
fi

# The opt-in self-deploy timer (OPS-001) is the `dotf converge` step self-deploy,
# gated on the self-deploy machine role (OPS-053).

# Claude Code plugins are installed by the bare `dotf deploy` above, from
# ai/claude/plugins.json: it counts each failure and names its cause (#2336).

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
    log_error "compile-harness --deploy failed -- see its errors above; no skill reached the agents"
    exit 1
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
# every home file is an ai/deploy.json entry, which doctor's checkDeployManifest
# byte-compares and which counts a symlink at the destination as drift (the
# check_deployed rejection), and every one of the fourteen tools has an
# equal-or-stronger verdict there -- asserted by TestSetupShellParity, not by
# this comment. setup-windows.ps1 has had no such block since CLI-018; this
# brings Linux to the same shape.
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

# Re-enforce the rc files at the end of setup to overwrite any third-party installer mutations.
_dotf=""
if command -v dotf >/dev/null 2>&1; then _dotf="dotf"; elif [ -x "$HOME/.local/bin/dotf" ]; then _dotf="$HOME/.local/bin/dotf"; fi
if [ -n "$_dotf" ]; then
    for _rc in zshrc bashrc profile; do
        # Fatal, like the single `dotf deploy zshrc` this loop replaces: setup
        # must not report success with an rc file it could not write (#2013 W4).
        "$_dotf" deploy "$_rc" || { log_error "dotf deploy $_rc failed -- see 'dotf doctor'"; exit 1; }
    done
fi

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
    log_warning "dotf not on PATH -- skipping post-setup diagnostics (run ./install.sh, then 'dotf doctor')"
fi

log_info "To apply changes immediately, run:"
log_info "  - For Bash: source ~/.bashrc"
log_info "  - For Zsh:  source ~/.zshrc"
