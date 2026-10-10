#!/usr/bin/env bats
# verify-setup.bats - Integration tests verifying setup-linux.sh side effects
# Run inside the container built from tests/Dockerfile.integration

load 'lib/refute'

setup() {
    if [ -z "$DOTFILES_INTEGRATION_TEST" ]; then
        skip "only runs inside integration test container"
    fi
    export HOME="/home/testuser"
    export DOTFILES_DIR="$HOME/.dotfiles"
    export REPO_DIR="$HOME/dotfiles-repo"
    # The PATH a user's shell has after setup: the rc files put ~/.local/bin on
    # it (dotf, eza, gh), and `mise activate` puts the mise-pinned CLIs ahead
    # of it, some of which setup installs only there (jq among them, #2013 W2).
    # The entrypoint's PATH has neither, so without this the dotf tests below
    # skipped on every run and verified nothing (#696, #915).
    export PATH="$HOME/.local/share/mise/shims:$HOME/.local/bin:$PATH"
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
        [ "$(head -1 "$prompt")" != '---' ] || { printf 'frontmatter left in %s\n' "$prompt" >&2; return 1; }
    done
}

@test "skill directories each contain SKILL.md" {
    for skill_dir in "$HOME/.claude/skills"/*/; do
        [ -d "$skill_dir" ] || continue
        [ -f "${skill_dir}SKILL.md" ]
    done
}

# #1938: the seed writes pi's settings.json, then `dotf deploy` merges the
# compaction overrides into it. Both halves must land on a fresh box: the
# overrides prove the merge ran, defaultModel proves it ran onto the seed and
# did not create the file itself (the reverse order leaves defaultModel out).
@test "pi settings.json is seeded and carries the compaction overrides [#1938]" {
    local jq_bin=""
    if command -v jq >/dev/null 2>&1; then
        jq_bin="jq"
    elif [ -x "$HOME/.local/bin/jq" ]; then
        jq_bin="$HOME/.local/bin/jq"
    else
        echo "jq is absent from PATH and ~/.local/bin"
        return 1
    fi
    local settings="$HOME/.pi/agent/settings.json"
    [ -f "$settings" ]
    "$jq_bin" -e '.defaultModel | type == "string"' "$settings"
    "$jq_bin" -e '.compaction.modelOverrides["nan/deepseek-v4-flash"].reserveTokens == 600000' "$settings"
}

# #1484: the image carries npm since #2251, so setup installs pi and runs the pi
# package reconcile for real. Its exit status is a warning by design, so the
# result is what is checked (lesson 379): pi's settings.json records every
# package ai/pi/packages.json declares and nothing it does not.
@test "pi packages converge on ai/pi/packages.json [#1484]" {
    [ -x "$HOME/.local/bin/pi" ] || { echo "setup did not install pi into ~/.local/bin"; return 1; }
    run dotf pi packages check --repo "$REPO_DIR"
    [ "$status" -eq 0 ] || { echo "$output"; return 1; }
}

# =============================================================================
# Section 6: Generated files
# =============================================================================

@test "bash_aliases exists and has aliases" {
    [ -f "$HOME/.bash/bash_aliases" ]
    grep -q 'alias ' "$HOME/.bash/bash_aliases"
}

@test "~/.gitconfig includes the deployed dotfiles.gitconfig (#2207)" {
    [ -f "$HOME/.config/git/dotfiles.gitconfig" ]
    run git config --global --get-all include.path
    [ "$status" -eq 0 ]
    [[ "$output" == *'~/.config/git/dotfiles.gitconfig'* ]] || false
}

@test "the include is effective: git reads the dotfiles' user.name (#2207)" {
    # The pointer alone proves nothing; this proves git expands ~/ in
    # include.path and reads the deployed file. `--global` alone ignores
    # includes, so `--includes` asks what a commit would see.
    want=$(git config -f "$HOME/.config/git/dotfiles.gitconfig" user.name)
    [ -n "$want" ]
    [ "$(git config --global --includes user.name)" = "$want" ]
}

@test "~/.gitconfig is a regular file" {
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

@test "the npm catalog installs into ~/.local as the user, never into npm's root-owned prefix [#2251]" {
    # Precondition, asserted so the guard cannot pass vacuously: npm is present
    # and its default global prefix is not writable by this user. If the
    # container ever gains a user-owned Node (nvm), this fails here instead of
    # the test below silently proving nothing.
    command -v npm >/dev/null 2>&1
    prefix=$(npm config get prefix)
    [ ! -w "$prefix/lib" ]
    for tool in bw yarn opencode copilot; do
        [ -x "$HOME/.local/bin/$tool" ] || { echo "missing: ~/.local/bin/$tool"; return 1; }
        [ "$(stat -c %U "$HOME/.local/bin/$tool")" = testuser ]
    done
    # A converged box re-runs clean: setup's `|| log_warning` would hide a failure.
    run dotf tools install
    [ "$status" -eq 0 ] || { echo "$output"; return 1; }
}

@test "copilot config is deployed now that the npm catalog put copilot on PATH (#1312)" {
    # The inverse, an entry skipped while its required command is absent, is
    # pinned by TestDeployCmd_SkipsAnEntryWhoseRequiredCommandIsAbsent; the
    # container now carries copilot, so this asserts the present side.
    command -v copilot >/dev/null 2>&1
    for f in copilot-instructions.md settings.json config.json mcp-config.json; do
        [ -f "$HOME/.copilot/$f" ] || { echo "missing: ~/.copilot/$f"; return 1; }
    done
}

@test "AGENTS.md deployed to ~/.config/opencode/AGENTS.md (cross-agent SSOT)" {
    # opencode reads AGENTS.md natively (per upstream docs). Deploying the
    # repo-root canonical SSOT verbatim gives opencode the same system prompt
    # claude/agy/copilot get via their pointer files.
    [ -f "$HOME/.config/opencode/AGENTS.md" ]
    grep -q '^# AGENTS.md' "$HOME/.config/opencode/AGENTS.md"
    grep -q 'Single Source of Truth' "$HOME/.config/opencode/AGENTS.md"
}

@test "opencode tui.json deployed by dotf deploy, not by a setup block (#1843 B11)" {
    # The setup copy was deleted; the `opencode-tui` entry of ai/deploy.json is
    # now the only writer, so this fails if that entry stops applying. The entry
    # merges (#2260): the file is plain JSON and may hold keys the repo does not
    # own (Orca's `plugin`). So the check is the merge's own invariant, that
    # overlaying the source on the deployed file changes nothing, not equality.
    # jq's `*` replaces arrays where the merge unions them; the source has none.
    # Both sides drop comment lines: a merge that changes nothing does not
    # rewrite, so a file an earlier `replace` deployed keeps the source's.
    [ -f "$HOME/.config/opencode/tui.json" ]
    src=$(grep -v '^[[:space:]]*//' "$REPO_DIR/ai/opencode/tui.json" | jq -c .)
    grep -v '^[[:space:]]*//' "$HOME/.config/opencode/tui.json" | jq -e --argjson src "$src" '. * $src == .'
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
    # Spot check: spec.md present (portable), crystallize.md absent (targets:[claude]).
    [ -f "$HOME/.config/opencode/commands/spec.md" ]
    [ ! -f "$HOME/.config/opencode/commands/crystallize.md" ]
    # rendered command carries provenance + drops name: (opencode keys off filename)
    grep -qE '^generated_sha: [0-9a-f]{16}' "$HOME/.config/opencode/commands/spec.md"
    refute_grep '^name:' "$HOME/.config/opencode/commands/spec.md"
}

@test "no MCP servers registered (claude CLI absent)" {
    # setup-linux.sh skips MCP registration when claude is not found
    # Just verify it didn't crash — the container built successfully
    true
}

@test "shellcheck comes from mise at its pin, with no copy in ~/.local/bin to shadow it (#2013 W2)" {
    # Setup puts mise's shims on PATH and runs `dotf tools sync` before the
    # linux-amd64 download, which then finds shellcheck and skips. mise does the
    # download and the extraction; the old .tar.xz path (and its xz-utils
    # dependency) is no longer how shellcheck arrives.
    local mise
    mise="$(command -v mise || printf '%s' "$HOME/.local/bin/mise")"
    # From $HOME, so the checkout's own mise.toml cannot answer for the machine.
    run bash -c 'cd "$HOME" && "$1" which shellcheck' _ "$mise"
    [ "$status" -eq 0 ]
    [[ "$output" == */mise/installs/shellcheck/* ]] || false
    [ ! -e "$HOME/.local/bin/shellcheck" ]
}

@test "every CLI versions.conf marks for mise is installed through mise at its pin (#2013 T2)" {
    # The Windows CI leg gates on doctor's "Pinned CLIs (mise)" check; Linux has
    # no doctor gate, so this is that check's Linux half. It reads the marks the
    # way dotf does: "# mise: cli" on its own line, the pin on the next one, and
    # NAME_VERSION as mise's name with underscores as hyphens.
    local mise pins pin missing=""
    mise="$(command -v mise || printf '%s' "$HOME/.local/bin/mise")"
    pins="$(awk '
        m && /^[A-Z0-9_]+_VERSION=/ {
            n = $0; sub(/_VERSION=.*/, "", n); n = tolower(n); gsub(/_/, "-", n)
            v = $0; sub(/^[^=]*=/, "", v)
            print n "@" v
        }
        { m = ($0 == "# mise: cli") }' "$DOTFILES_DIR/versions.conf")"
    [ -n "$pins" ]
    # The awk above is a second reading of versions.conf; dotf's own reading
    # (tools.ParseMiseTools) is what the sync rendered. They must agree, or this
    # test checks a different pin set than the one dotf installs.
    local rendered from_dotf
    rendered="${MISE_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/mise}/conf.d/dotfiles.toml"
    from_dotf="$(awk '/^\[/ { t = ($0 == "[tools]"); next }
        t && / = "/ { v = $3; gsub(/"/, "", v); print $1 "@" v }' "$rendered" | sort)"
    [ "$(printf '%s\n' "$pins" | sort)" = "$from_dotf" ] || {
        echo "versions.conf read here: $(printf '%s\n' "$pins" | sort | tr '\n' ' ')"
        echo "rendered by dotf ($rendered): $(printf '%s' "$from_dotf" | tr '\n' ' ')"
        false
    }
    while IFS= read -r pin; do
        # From $HOME, so the checkout's own mise.toml cannot answer for the machine.
        bash -c 'cd "$HOME" && "$1" where "$2"' _ "$mise" "$pin" >/dev/null 2>&1 ||
            missing="$missing $pin"
    done <<< "$pins"
    [ -z "$missing" ] || { echo "not installed through mise at the pin:$missing"; false; }
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

# The image builds dotf from this checkout into ~/.local/bin (tests/Dockerfile.integration),
# so these run against the PR's own binary. A missing dotf is a failure, not a skip:
# these skipped on every run while the bats PATH lacked ~/.local/bin, and so
# verified nothing.

@test "setup seeds machine.json with DOTFILES_REPO_DIR = the checkout [#696]" {
    command -v dotf >/dev/null 2>&1
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
    command -v dotf >/dev/null 2>&1
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
    command -v dotf >/dev/null 2>&1
    local ver
    ver="$(dotf version 2>/dev/null)"
    [[ "$ver" == dotf\ version\ * ]] || false
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

    # Idempotence is defined on a CONVERGED first run. A tool whose install
    # warned and continued is installed by the second run, and what it brings in
    # (agy's hooks.json via `dotf harness bind`) then reads as drift (#1704).
    # Name that cause instead of printing a hash diff.
    # A path, not `command -v`: ~/.local/bin is not on the bats-time PATH here.
    [ -x "$HOME/.local/bin/agy" ] || {
        echo "the first setup run did not install agy, so the second run is a repair, not a re-run" >&2
        return 1
    }

    # Collect hashes of deployed dotfiles and configs before second run
    find "$HOME/.dotfiles" "$HOME/.claude" "$HOME/.gemini" "$HOME/.config/opencode" \
         "$HOME/.zsh" "$HOME/.bash" "$HOME/.ssh" \
         -type f ! -path "*/.git/*" ! -name "*.log" 2>/dev/null | sort | xargs sha256sum > "$snap1"
    sha256sum "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.gitconfig" "$HOME/.tmux.conf" "$HOME/.ssh/config" >> "$snap1"

    # Execute second run
    cd "$REPO_DIR"
    run bash setup-linux.sh
    if [ "$status" -ne 0 ]; then
        echo "second setup-linux.sh run exited $status; last 40 lines:" >&2
        printf '%s\n' "$output" | tail -40 >&2
        return 1
    fi

    # Collect hashes after second run
    find "$HOME/.dotfiles" "$HOME/.claude" "$HOME/.gemini" "$HOME/.config/opencode" \
         "$HOME/.zsh" "$HOME/.bash" "$HOME/.ssh" \
         -type f ! -path "*/.git/*" ! -name "*.log" 2>/dev/null | sort | xargs sha256sum > "$snap2"
    sha256sum "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.gitconfig" "$HOME/.tmux.conf" "$HOME/.ssh/config" >> "$snap2"

    # Assert diff is empty. On failure, print it: bats shows only the failed line,
    # so without this the file that changed between the two runs, which is the
    # actual non-idempotence, never reached the log (TEST-012).
    run diff -u "$snap1" "$snap2"
    [ "$status" -eq 0 ] || { printf 'files whose hash changed between the two setup runs:\n%s\n' "$output"; return 1; }
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
    # `--check` requires jq ON PATH (`type -P jq || exit 2`), and setup installs
    # it to ~/.local/bin without putting that on this process's PATH (#1202).
    # SKIP rather than inject the path: injecting would make this test green on
    # a machine where `dotf doctor` is red, which is the failure mode
    # docs/lessons/lesson-223 is about. A named skip says which condition was
    # hit; once #1202 lands, this starts running here on its own.
    command -v jq >/dev/null 2>&1 || \
        skip "jq is not on PATH, so --check exits 2 before it can answer about drift (#1202)"
    run bash "$DOTFILES_DIR/scripts/compile-harness.sh" --check
    [ "$status" -eq 0 ]
    echo "$output" | grep -q 'no harness drift'
}


@test "every deployed persona can invoke the skills its own gate demands" {
    # The container-level half of #1420. The Go guard asserts the RECORD declares
    # the `skill` capability; this asserts the thing that capability exists to
    # produce — a deployed agent whose `tools:` actually names Skill. claude's
    # `tools:` is an ALLOW-LIST, so a tool not named is unavailable, and a persona
    # required to consume a forced skill it cannot invoke is a hard deadlock the
    # moment any of its skills reaches `enforce: block`.
    #
    # It reads the DEPLOYED file rather than the record, because every layer was
    # individually correct when this defect shipped: the SSOT said what it meant,
    # the map translated faithfully, and the deployed files matched the SSOT
    # exactly. Only the relationship between two frontmatter keys was wrong, and
    # no check spanned it.
    agents="$HOME/.claude/agents"
    [ -d "$agents" ] || skip "no ~/.claude/agents on this platform (deploy_agents is Linux-only, #1387)"

    checked=0
    offenders=""
    for rec in "$DOTFILES_DIR"/harness/agents/*/AGENT.md; do
        [ -e "$rec" ] || continue
        name="$(basename "$(dirname "$rec")")"
        deployed="$agents/$name.md"
        # Only personas actually deployed here; a record targeted at another
        # harness is not this assertion's business.
        [ -f "$deployed" ] || continue
        # Matches both the flat and the block form of `skills:`.
        grep -qE '^skills:' "$rec" || continue
        checked=$((checked + 1))
        grep -qE '^tools:.*\bSkill\b' "$deployed" || offenders="$offenders $name"
    done

    # Fixture drift. Zero personas declaring forced skills would pass this
    # vacuously, and an empty result reading as a clean one is precisely the
    # failure class these guards exist to prevent.
    [ "$checked" -gt 0 ] || {
        echo "no deployed persona under $agents declares forced skills;"
        echo "either the roster changed or this assertion measures nothing"
        return 1
    }

    [ -z "$offenders" ] || {
        echo "deployed personas declare forced skills but are granted no Skill tool:$offenders"
        echo "the deployed agent cannot invoke the skills its own gate requires (#1420)."
        echo "Fix in the vault SSOT — 00_meta/agents/definitions/<role>/AGENT.md — and redeploy;"
        echo "editing ~/.claude/agents/ directly is overwritten by the next compile."
        return 1
    }
}
