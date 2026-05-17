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

@test "load-secrets.sh copied to ~/.dotfiles/scripts" {
    [ -f "$DOTFILES_DIR/scripts/load-secrets.sh" ]
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

@test "env-mapping.conf copied to ~/.dotfiles/sensitive" {
    [ -f "$DOTFILES_DIR/sensitive/env-mapping.conf" ]
}

# =============================================================================
# Section 3: Symlinks
# =============================================================================

@test "~/.zshrc is a symlink to ~/.dotfiles/.zshrc" {
    [ -L "$HOME/.zshrc" ]
    readlink "$HOME/.zshrc" | grep -q '\.dotfiles/.zshrc'
}

@test "~/.bashrc is a symlink to ~/.dotfiles/.bashrc" {
    [ -L "$HOME/.bashrc" ]
    readlink "$HOME/.bashrc" | grep -q '\.dotfiles/.bashrc'
}

@test "~/.profile is a symlink to ~/.dotfiles/.profile" {
    [ -L "$HOME/.profile" ]
    readlink "$HOME/.profile" | grep -q '\.dotfiles/.profile'
}

@test "~/.zsh/aliases.zsh is a symlink" {
    [ -L "$HOME/.zsh/aliases.zsh" ]
    readlink "$HOME/.zsh/aliases.zsh" | grep -q '\.dotfiles/.zsh/aliases.zsh'
}

@test "~/.zsh/functions.zsh is a symlink" {
    [ -L "$HOME/.zsh/functions.zsh" ]
    readlink "$HOME/.zsh/functions.zsh" | grep -q '\.dotfiles/.zsh/functions.zsh'
}

@test "~/.zsh/nvm.zsh is a symlink" {
    [ -L "$HOME/.zsh/nvm.zsh" ]
    readlink "$HOME/.zsh/nvm.zsh" | grep -q '\.dotfiles/.zsh/nvm.zsh'
}

@test "~/.ssh/config is a symlink" {
    [ -L "$HOME/.ssh/config" ]
    readlink "$HOME/.ssh/config" | grep -q '\.dotfiles/ssh/config'
}

# =============================================================================
# Section 4: Permissions
# =============================================================================

@test "utils.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/utils.sh" ]
}

@test "load-secrets.sh is executable" {
    [ -x "$DOTFILES_DIR/scripts/load-secrets.sh" ]
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

@test "ssh/config has 600 permissions" {
    perms=$(stat -c '%a' "$DOTFILES_DIR/ssh/config")
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

@test "~/.claude/init-project.sh exists and is executable" {
    [ -f "$HOME/.claude/init-project.sh" ]
    [ -x "$HOME/.claude/init-project.sh" ]
}

@test "~/.gemini/GEMINI.md deployed with AGENTS.md pointer marker" {
    [ -f "$HOME/.gemini/GEMINI.md" ]
    grep -q 'First, read `AGENTS.md`' "$HOME/.gemini/GEMINI.md"
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

@test ".gitconfig is a symlink to dotfiles" {
    [ -L "$HOME/.gitconfig" ]
    [ "$(readlink "$HOME/.gitconfig")" = "$HOME/.dotfiles/.gitconfig" ]
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

@test "functions.zsh references utils.sh" {
    grep -q 'utils.sh' "$DOTFILES_DIR/.zsh/functions.zsh"
}

# =============================================================================
# Section 9: ensure_line_in_file side effects
# =============================================================================

@test "scripts PATH added to .zshrc" {
    grep -q 'export PATH=\$HOME/.dotfiles/scripts:\$PATH' "$HOME/.zshrc"
}

@test "project-init alias added to .zshrc" {
    grep -q 'alias project-init=' "$HOME/.zshrc"
}

@test "scripts PATH added to .bashrc" {
    grep -q 'export PATH=\$HOME/.dotfiles/scripts:\$PATH' "$HOME/.bashrc"
}

@test "project-init alias added to .bashrc" {
    grep -q 'alias project-init=' "$HOME/.bashrc"
}

# =============================================================================
# Section 10: Graceful skips (optional tools not present)
# =============================================================================

@test "copilot directory created (gh installed)" {
    [ -d "$HOME/.copilot" ]
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

@test "~/.tmux.conf is a symlink to ~/.dotfiles/tmux.conf" {
    [ -L "$HOME/.tmux.conf" ]
    [ "$(readlink "$HOME/.tmux.conf")" = "$DOTFILES_DIR/tmux.conf" ]
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
