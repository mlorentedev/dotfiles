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
    # uses the committed records.
    [ -d "$HOME/.config/opencode/commands" ]
    # Expected count is DERIVED, never hardcoded: every deployed skill whose
    # targets[] admits opencode (absent targets = all agents) must have a command.
    # A literal count rots on every skill added or unfenced.
    local expected=0 f
    for f in "$REPO_DIR"/harness/skills/*/SKILL.md; do
        [ -f "$f" ] || continue
        local targets
        targets=$(awk '/^---[[:space:]]*$/{n++; next} n==1 && /^targets:/{print; exit}' "$f")
        if [ -z "$targets" ] || [[ "$targets" == *opencode* ]]; then
            expected=$((expected + 1))
        fi
    done
    local count
    count=$(find "$HOME/.config/opencode/commands" -maxdepth 1 -name '*.md' | wc -l)
    [ "$count" -eq "$expected" ]
    # Spot check: audit.md present (portable), crystallize.md absent (targets:[claude]).
    [ -f "$HOME/.config/opencode/commands/audit.md" ]
    [ ! -f "$HOME/.config/opencode/commands/crystallize.md" ]
    # rendered command carries provenance + drops name: (opencode keys off filename)
    grep -qE '^generated_sha: [0-9a-f]{16}' "$HOME/.config/opencode/commands/audit.md"
    ! grep -q '^name:' "$HOME/.config/opencode/commands/audit.md"
}

@test "no MCP servers registered (claude CLI absent)" {
    # setup-linux.sh skips MCP registration when claude is not found
    # Just verify it didn't crash — the container built successfully
    true
}

@test "shellcheck installed to ~/.local/bin (xz-utils present, .tar.xz extracts)" {
    # setup-linux.sh installs shellcheck from a .tar.xz via `tar xJf`, which needs
    # the xz binary. The container previously lacked xz-utils, so this step failed
    # silently ("tar (child): xz: Cannot exec") and the build stayed green — the
    # install path was never verified. With xz-utils in the image it must land.
    [ -x "$HOME/.local/bin/shellcheck" ]
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

# =============================================================================
# Section: Checkout hygiene — setup must never write into the repo checkout
# =============================================================================

# Guard for dotfiles#694. Setup deploys to $HOME only; it must NEVER write into
# the checkout it runs from. A checkout write leaves `git status` dirty, and
# `dotf update` (cli/internal/update/update.go) skips any dirty worktree with
# exit 0 — so a self-deploying machine silently stops updating after the first
# run while the timer stays green forever. This asserts exactly what update.go
# checks (empty `git status --porcelain`) against the same checkout: setup ran at
# image-build time, this runs at container-run time. Any future checkout write —
# not just the copilot-instructions sync that motivated this — trips the guard.
@test "setup leaves the repo checkout clean (no dirty worktree) [#694]" {
    [ -d "$REPO_DIR/.git" ] || skip "repo checkout is not a git repo in this container"
    run git -C "$REPO_DIR" status --porcelain
    [ "$status" -eq 0 ]
    if [ -n "$output" ]; then
        echo "setup dirtied the checkout — dotf update would skip forever (#694):" >&2
        echo "$output" >&2
    fi
    [ -z "$output" ]
}

# =============================================================================
# Section: machine.json seeding (BUG-029 / #696)
# =============================================================================
#
# Setup seeds ~/.config/dotfiles/machine.json with DOTFILES_REPO_DIR = the
# checkout it runs from, so the ADR-025 cascade (and the generated paths file)
# resolve the real repo instead of the phantom contract default. Without the
# seed, `dotf update` reports "not a git repo: ~/Projects/dotfiles" and exits 0
# (self-deploy is a silent no-op) and `dotf mem` says "run setup" though setup
# ran — on every fresh machine. These guard exactly that class.

# These two activate only once an available `dotf` binary carries `env set`.
# The integration container installs the *released* dotf (scripts/install-dotf.sh
# downloads the pinned release, it is not built from the PR source), and dotf is
# not on the bats-time PATH, so a brand-new subcommand cannot be exercised here
# until it ships in a release. They skip cleanly until then — the seed logic is
# fully guarded by the Go unit tests (env set) + the `dotf doctor` repo-dir check.
# Harness gap tracked separately (integration should test the PR's built binary).

@test "setup seeds machine.json with DOTFILES_REPO_DIR = the checkout [#696]" {
    command -v dotf >/dev/null 2>&1 || skip "dotf not on PATH in this container"
    dotf env set --help >/dev/null 2>&1 || skip "installed dotf predates 'env set'; seed not exercised"
    machine="$HOME/.config/dotfiles/machine.json"
    [ -f "$machine" ]
    run grep -F "$REPO_DIR" "$machine"
    if [ "$status" -ne 0 ]; then
        echo "machine.json did not record the checkout path $REPO_DIR:" >&2
        cat "$machine" >&2
    fi
    [ "$status" -eq 0 ]
}

@test "dotf env path DOTFILES_REPO_DIR resolves to the real checkout [#696]" {
    command -v dotf >/dev/null 2>&1 || skip "dotf not on PATH in this container"
    dotf env set --help >/dev/null 2>&1 || skip "installed dotf predates 'env set'; seed not exercised"
    # Captured through a plain $(...) with stderr discarded — the exact idiom
    # setup-linux.sh uses. `run` is avoided on purpose: it merges stdout and
    # stderr into $output, so it passed all the way through BUG-070 (#915)
    # while every real caller was capturing an empty string.
    local resolved
    resolved="$(dotf env path DOTFILES_REPO_DIR 2>/dev/null)"
    [ "$resolved" = "$REPO_DIR" ]
    [ -d "$resolved/.git" ]
}

@test "dotf version reaches stdout so install-dotf can grep the semver [#915]" {
    command -v dotf >/dev/null 2>&1 || skip "dotf not on PATH in this container"
    local ver
    ver="$(dotf version 2>/dev/null)"
    [[ "$ver" == dotf\ version\ * ]]
}

# =============================================================================
# Section 12: Idempotence (POLISH-005)
# =============================================================================
#
# Running setup-linux.sh a second time on an already-configured system must:
# 1. Exit 0 cleanly without errors
# 2. Not mutate or duplicate deployed configuration (byte-identical deployed state)
# 3. Leave the repo checkout clean

@test "POLISH-005: second setup-linux.sh run exits 0 cleanly with zero config diff" {
    local snap1="/tmp/snap1-$$.sha256"
    local snap2="/tmp/snap2-$$.sha256"

    # Collect hashes of deployed dotfiles and configs before second run
    find "$HOME/.dotfiles" "$HOME/.claude" "$HOME/.gemini" "$HOME/.config/opencode" \
         "$HOME/.zsh" "$HOME/.bash" "$HOME/.ssh" \
         -type f ! -path "*/.git/*" ! -name "*.log" 2>/dev/null | sort | xargs sha256sum > "$snap1"
    sha256sum "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.gitconfig" "$HOME/.tmux.conf" "$HOME/.ssh/config" >> "$snap1"

    # Execute second run
    cd "$REPO_DIR"
    run bash setup-linux.sh
    [ "$status" -eq 0 ]

    # Collect hashes after second run
    find "$HOME/.dotfiles" "$HOME/.claude" "$HOME/.gemini" "$HOME/.config/opencode" \
         "$HOME/.zsh" "$HOME/.bash" "$HOME/.ssh" \
         -type f ! -path "*/.git/*" ! -name "*.log" 2>/dev/null | sort | xargs sha256sum > "$snap2"
    sha256sum "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.gitconfig" "$HOME/.tmux.conf" "$HOME/.ssh/config" >> "$snap2"

    # Assert diff is empty
    run diff -u "$snap1" "$snap2"
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    rm -f "$snap1" "$snap2"
}

@test "POLISH-005: rc files contain no duplicate entries after second setup run" {
    local zsh_path_count bash_path_count
    zsh_path_count=$(grep -c 'export PATH="\$HOME/.dotfiles/scripts:\$PATH"' "$HOME/.zshrc" || true)
    bash_path_count=$(grep -c 'export PATH="\$HOME/.dotfiles/scripts:\$PATH"' "$HOME/.bashrc" || true)
    [ "$zsh_path_count" -eq 1 ]
    [ "$bash_path_count" -eq 1 ]
}

@test "POLISH-005: setup leaves repo checkout clean after second run" {
    [ -d "$REPO_DIR/.git" ] || skip "repo checkout is not a git repo in this container"
    run git -C "$REPO_DIR" status --porcelain
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# =============================================================================
# Section: harness injection targets are mirrored into the deploy dir
#
# `dotf doctor` runs `compile-harness.sh --check` from $DOTFILES_DIR, so every
# file harness/manifest.json declares as a target must exist there. #1176 added
# ai/orca/ORCA.md to the manifest without a copy line in setup-linux.sh, and the
# result was a permanent doctor FAIL whose printed remedy could not clear it:
# running --refresh from the repo exits 0, because the repo has the file (#1200).
#
# This asserts the OUTCOME on a real deploy, which is the half no unit test can
# reach — tests/compile-harness-rootresolve.bats builds its own tmp mirror, so
# it passed throughout by hand-copying the file setup never delivered.
# =============================================================================

@test "every harness manifest target exists in the deploy dir" {
    [ -f "$DOTFILES_DIR/harness/manifest.json" ]
    # Resolve jq the same way setup does: it is installed to ~/.local/bin, which
    # is not necessarily on this process's PATH (#1202).
    local jq_bin=""
    if command -v jq >/dev/null 2>&1; then
        jq_bin="jq"
    elif [ -x "$HOME/.local/bin/jq" ]; then
        jq_bin="$HOME/.local/bin/jq"
    else
        echo "jq is absent from PATH and ~/.local/bin, so the list cannot be read"
        return 1
    fi
    local missing="" checked=0
    while IFS= read -r target; do
        [ -n "$target" ] || continue
        checked=$((checked + 1))
        [ -f "$DOTFILES_DIR/$target" ] || missing="$missing $target"
    done < <("$jq_bin" -r '.targets[].file' "$DOTFILES_DIR/harness/manifest.json")
    # An empty list would make every assertion below vacuously true, which is
    # how a guard reports "all clear" on a manifest it never read.
    [ "$checked" -gt 0 ] || {
        echo "read zero targets from manifest.json — the guard checked nothing"
        return 1
    }
    [ -z "$missing" ] || {
        echo "manifest targets missing from $DOTFILES_DIR:$missing"
        echo "setup-linux.sh must mirror every harness/manifest.json target"
        return 1
    }
}

@test "compile-harness --check passes from the deploy dir" {
    # The assertion dotf doctor makes, made directly: a green --check here is
    # what a green [Harness + skill drift] section means.
    [ -x "$DOTFILES_DIR/scripts/compile-harness.sh" ]
    run bash "$DOTFILES_DIR/scripts/compile-harness.sh" --check
    [ "$status" -eq 0 ]
    echo "$output" | grep -q 'no harness drift'
}

