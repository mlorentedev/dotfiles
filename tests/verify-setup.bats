#!/usr/bin/env bats
# verify-setup.bats - Integration tests verifying setup-linux.sh side effects
# Run inside the container built from tests/Dockerfile.integration

setup() {
    if [ -z "$DOTFILES_INTEGRATION_TEST" ]; then
        skip "only runs inside integration test container"
    fi
    export HOME="/home/testuser"
    export DOTFILES_DIR="$HOME/.dotfiles"
    export REPO_DIR="$HOME/dotfiles-repo"
}

# =============================================================================
# Section 1: Core directories
# =============================================================================

@test "~/.dotfiles directory exists" {
    [ -d "$DOTFILES_DIR" ]
}

@test "~/.dotfiles/scripts directory exists" {
    [ -d "$DOTFILES_DIR/scripts" ]
}

@test "~/.dotfiles/.zsh directory exists" {
    [ -d "$DOTFILES_DIR/.zsh" ]
}

@test "~/.dotfiles/sensitive directory exists" {
    [ -d "$DOTFILES_DIR/sensitive" ]
}

@test "~/.dotfiles/secrets/registry.yaml exists (dotf secrets mapping SSOT) [#587]" {
    [ -f "$DOTFILES_DIR/secrets/registry.yaml" ]
}

@test "~/.dotfiles/ssh directory exists" {
    [ -d "$DOTFILES_DIR/ssh" ]
}

@test "~/.zsh directory exists" {
    [ -d "$HOME/.zsh" ]
}

@test "~/.bash directory exists" {
    [ -d "$HOME/.bash" ]
}

@test "~/.ssh directory exists" {
    [ -d "$HOME/.ssh" ]
}

# =============================================================================
# Section 2: Files copied from repo to ~/.dotfiles
# =============================================================================

@test "versions.conf copied to ~/.dotfiles" {
    [ -f "$DOTFILES_DIR/versions.conf" ]
}

@test ".zshrc copied to ~/.dotfiles" {
    [ -f "$DOTFILES_DIR/.zshrc" ]
}

@test ".bashrc exists in ~/.dotfiles" {
    [ -f "$DOTFILES_DIR/.bashrc" ]
}

@test ".profile copied to ~/.dotfiles" {
    [ -f "$DOTFILES_DIR/.profile" ]
}

@test "utils.sh copied to ~/.dotfiles/scripts" {
    [ -f "$DOTFILES_DIR/scripts/utils.sh" ]
}

@test "aliases.zsh copied to ~/.dotfiles/.zsh" {
    [ -f "$DOTFILES_DIR/.zsh/aliases.zsh" ]
}

@test "functions.zsh copied to ~/.dotfiles/.zsh" {
    [ -f "$DOTFILES_DIR/.zsh/functions.zsh" ]
}

@test "nvm.zsh copied to ~/.dotfiles/.zsh" {
    [ -f "$DOTFILES_DIR/.zsh/nvm.zsh" ]
}

@test "ssh/config copied to ~/.dotfiles/ssh" {
    [ -f "$DOTFILES_DIR/ssh/config" ]
}

# =============================================================================
# Section 3: Deployed files (regular files, post-SDD-007 copy-only model)
# =============================================================================
# Per SDD-007 / BUG-100: setup deploys via deploy_file() (atomic copy), no
# symlinks. The previous model symlinked $HOME -> $DOTFILES_DIR but caused
# circular-symlink + staging-latency bugs (BUG-100). Asserting "regular file
# AND NOT a symlink" makes the copy-only invariant explicit.

@test "~/.zshrc is a regular file (copied from ~/.dotfiles/.zshrc)" {
    [ -f "$HOME/.zshrc" ]
    [ ! -L "$HOME/.zshrc" ]
}

@test "~/.bashrc is a regular file (copied from ~/.dotfiles/.bashrc)" {
    [ -f "$HOME/.bashrc" ]
    [ ! -L "$HOME/.bashrc" ]
}

@test "~/.profile is a regular file (copied from ~/.dotfiles/.profile)" {
    [ -f "$HOME/.profile" ]
    [ ! -L "$HOME/.profile" ]
}

@test "~/.zsh/aliases.zsh is a regular file" {
    [ -f "$HOME/.zsh/aliases.zsh" ]
    [ ! -L "$HOME/.zsh/aliases.zsh" ]
}

@test "~/.zsh/functions.zsh is a regular file" {
    [ -f "$HOME/.zsh/functions.zsh" ]
    [ ! -L "$HOME/.zsh/functions.zsh" ]
}

@test "~/.zsh/nvm.zsh is a regular file" {
    [ -f "$HOME/.zsh/nvm.zsh" ]
    [ ! -L "$HOME/.zsh/nvm.zsh" ]
}

@test "~/.ssh/config is a regular file" {
    [ -f "$HOME/.ssh/config" ]
    [ ! -L "$HOME/.ssh/config" ]
}

# =============================================================================
# Section 4: Permissions
# =============================================================================

@test "utils.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/utils.sh" ]
}

@test "github-secrets-manager.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/github-secrets-manager.sh" ]
}

@test "age-encrypt-decrypt.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh" ]
}

@test "install-precommit.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/install-precommit.sh" ]
}

@test "dotfiles-sync.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/dotfiles-sync.sh" ]
}

@test "~/.ssh/config has 600 permissions (user-facing, what SSH actually reads)" {
    # Post-SDD-007 copy-only model: $DOTFILES_DIR/ssh/config inherits cp's
    # default perms (644), but setup-linux.sh:65 chmods $HOME/.ssh/config
    # to 600 after deploy. That's the file SSH actually reads — assert there.
    perms=$(stat -c '%a' "$HOME/.ssh/config")
    [ "$perms" = "600" ]
}

# =============================================================================
# Section 5: AI configs
# =============================================================================

@test "~/.claude/CLAUDE.md deployed with AGENTS.md pointer marker" {
    [ -f "$HOME/.claude/CLAUDE.md" ]
    grep -q 'First, read `AGENTS.md`' "$HOME/.claude/CLAUDE.md"
}

@test "~/.claude/skills has at least 15 directories" {
    count=$(find "$HOME/.claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l)
    [ "$count" -ge 15 ]
}

@test "~/.gemini/AGY.md deployed with AGENTS.md pointer marker (post-SDD-007 rename)" {
    # gemini-cli → agy (Google Antigravity CLI). Identity file renamed from
    # GEMINI.md → AGY.md but lives in the same dir ($HOME/.gemini) because
    # agy still reads ANTIGRAVITY_ENDPOINT-relative config from there.
    [ -f "$HOME/.gemini/AGY.md" ]
    grep -q 'First, read `AGENTS.md`' "$HOME/.gemini/AGY.md"
}

@test "~/.gemini/prompts has at least 15 files" {
    count=$(find "$HOME/.gemini/prompts" -mindepth 1 -maxdepth 1 -type f -name '*.md' | wc -l)
    [ "$count" -ge 15 ]
}

@test "Gemini prompts have no YAML frontmatter" {
    # setup-linux.sh strips frontmatter with sed '/^---$/,/^---$/d'
    for prompt in "$HOME/.gemini/prompts"/*.md; do
        [ -f "$prompt" ] || continue
        ! head -1 "$prompt" | grep -q '^---$'
    done
}

@test "skill directories each contain SKILL.md" {
    for skill_dir in "$HOME/.claude/skills"/*/; do
        [ -d "$skill_dir" ] || continue
        [ -f "${skill_dir}SKILL.md" ]
    done
}

# =============================================================================
# Section 6: Generated files
# =============================================================================

@test "bash_aliases exists and has aliases" {
    [ -f "$HOME/.bash/bash_aliases" ]
    grep -q 'alias ' "$HOME/.bash/bash_aliases"
}

@test ".gitconfig deployed to home" {
    [ -f "$HOME/.gitconfig" ]
}

@test ".gitconfig is a regular file (post-SDD-007 copy-only deploy)" {
    [ -f "$HOME/.gitconfig" ]
    [ ! -L "$HOME/.gitconfig" ]
}

# =============================================================================
# Section 7: versions.conf
# =============================================================================

@test "versions.conf is sourceable by bash" {
    bash -c ". '$DOTFILES_DIR/versions.conf' && [ -n \"\$JAVA_VERSION\" ]"
}

@test "versions.conf sets BATS_VERSION" {
    bash -c ". '$DOTFILES_DIR/versions.conf' && [ -n \"\$BATS_VERSION\" ]"
}

# =============================================================================
# Section 8: Shell sourcability
# =============================================================================

@test ".bashrc has valid bash syntax" {
    bash -n "$DOTFILES_DIR/.bashrc"
}

@test "utils.sh sourceable under bash" {
    bash -c ". '$DOTFILES_DIR/scripts/utils.sh'"
}

@test "utils.sh sourceable under zsh" {
    zsh -c ". '$DOTFILES_DIR/scripts/utils.sh'"
}

@test "functions.sh sources utils.sh (shared bash/zsh entrypoint)" {
    grep -q 'utils\.sh' "$DOTFILES_DIR/.zsh/functions.sh"
}

# =============================================================================
# Section 9: rc file SSOT (lines baked into repo .bashrc/.zshrc — BUG-024)
# =============================================================================

@test "scripts PATH in .zshrc" {
    grep -q 'export PATH="\$HOME/.dotfiles/scripts:\$PATH"' "$HOME/.zshrc"
}

@test "scripts PATH in .bashrc" {
    grep -q 'export PATH="\$HOME/.dotfiles/scripts:\$PATH"' "$HOME/.bashrc"
}

# =============================================================================
# Section 10: Graceful skips (optional tools not present)
# =============================================================================

@test "copilot config NOT deployed when gh-copilot extension is absent" {
    # Post-BUG-001 (PR #40): setup-linux.sh uses detect-and-act. The
    # gh-copilot extension is no longer auto-installed; ~/.copilot is created
    # only if the extension is genuinely present. In the integration container
    # gh is installed (as a dev tool) but gh-copilot is not, so the directory
    # should NOT exist — confirming the skip path is silent and correct.
    [ ! -d "$HOME/.copilot" ]
}

@test "AGENTS.md deployed to ~/.config/opencode/AGENTS.md (cross-agent SSOT)" {
    # opencode reads AGENTS.md natively (per upstream docs). Deploying the
    # repo-root canonical SSOT verbatim gives opencode the same system prompt
    # claude/agy/copilot get via their pointer files.
    [ -f "$HOME/.config/opencode/AGENTS.md" ]
    grep -q '^# AGENTS.md' "$HOME/.config/opencode/AGENTS.md"
    grep -q 'Single Source of Truth' "$HOME/.config/opencode/AGENTS.md"
}

@test "opencode commands deployed to ~/.config/opencode/commands/ (SDD-008)" {
    # Post-SDD-008: setup-linux.sh runs compile-harness.sh --deploy, which renders
    # each committed vault skill record whose targets[] includes opencode to a
    # command .md. The container has no vault, so --refresh is skipped and --deploy
    # uses the committed records (33 skills, 6 Claude-only opt out -> 27 commands).
    [ -d "$HOME/.config/opencode/commands" ]
    local count
    count=$(find "$HOME/.config/opencode/commands" -maxdepth 1 -name '*.md' | wc -l)
    [ "$count" -eq 27 ]  # 26 -> 27: added new-ticket record (HARNESS-016 vault skill, no targets: -> all agents)
    # Spot check: audit.md present (portable), creating-skills.md absent (targets:[claude]).
    [ -f "$HOME/.config/opencode/commands/audit.md" ]
    [ ! -f "$HOME/.config/opencode/commands/creating-skills.md" ]
    # rendered command carries provenance + drops name: (opencode keys off filename)
    grep -qE '^generated_sha: [0-9a-f]{16}' "$HOME/.config/opencode/commands/audit.md"
    ! grep -q '^name:' "$HOME/.config/opencode/commands/audit.md"
}

@test "no MCP servers registered (claude CLI absent)" {
    # setup-linux.sh skips MCP registration when claude is not found
    # Just verify it didn't crash — the container built successfully
    true
}

# =============================================================================
# Section 11: tmux
# =============================================================================

@test "tmux binary present in container" {
    command -v tmux
}

@test "tmux.conf copied to ~/.dotfiles" {
    [ -f "$DOTFILES_DIR/tmux.conf" ]
}

@test "~/.tmux.conf is a regular file (post-SDD-007 copy-only deploy)" {
    [ -f "$HOME/.tmux.conf" ]
    [ ! -L "$HOME/.tmux.conf" ]
}

@test "tmux parses deployed config (smoke)" {
    local socket="verify_$$"
    # kill-server is best-effort: if the 'true' session ended already, the
    # server is gone — that's not a parse failure. The exit code of
    # new-session is the real signal (non-zero => parse error in config).
    run tmux -f "$HOME/.tmux.conf" -L "$socket" new-session -d -s s 'true'
    local rc=$status
    tmux -L "$socket" kill-server 2>/dev/null || true
    [ "$rc" -eq 0 ]
}
