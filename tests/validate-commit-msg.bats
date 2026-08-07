#!/usr/bin/env bats
# Tests for .github/hooks/validate-commit-msg.sh (POSIX sh — parses under sh,
# bash and zsh; the syntax tests below are the contract that keeps it that way).
#
# The scoped-subject cases guard #794: the hook used to match `^[a-z]+: .+`
# against the WHOLE message, which rejected every scoped Conventional Commit on
# main and could be rescued by a conforming body line.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export HOOK="$DOTFILES_DIR/.github/hooks/validate-commit-msg.sh"
}

# zsh is not installed on Windows, where this suite must still be green. Skip
# rather than fail, so the cross-shell contract keeps its teeth wherever zsh
# exists (Linux/macOS, CI) without making the suite unrunnable elsewhere (#794).
require_zsh() {
    command -v zsh > /dev/null 2>&1 || skip "zsh not installed on this host"
}

@test "validate-commit-msg.sh valid sh syntax" {
    sh -n "$HOOK"
}

@test "validate-commit-msg.sh valid bash syntax" {
    bash -n "$HOOK"
}

@test "validate-commit-msg.sh valid zsh syntax" {
    require_zsh
    zsh -n "$HOOK"
}

@test "accepts valid commit message" {
    local msg_file="/tmp/bats_commit_msg_$$"
    echo "feat: add new feature" > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "rejects invalid commit message" {
    local msg_file="/tmp/bats_commit_msg_$$"
    echo "Invalid commit message" > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    [[ "$output" == *"ERROR"* ]]
    rm -f "$msg_file"
}

@test "accepts valid message under zsh" {
    require_zsh
    local msg_file="/tmp/bats_commit_msg_zsh_$$"
    echo "fix: resolve bug" > "$msg_file"
    run zsh "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "rejects invalid message under zsh" {
    require_zsh
    local msg_file="/tmp/bats_commit_msg_zsh_$$"
    echo "Bad message" > "$msg_file"
    run zsh "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    rm -f "$msg_file"
}

# ── #794: scoped Conventional Commits ────────────────────────────────────

@test "accepts a scoped Conventional Commit" {
    local msg_file="/tmp/bats_commit_msg_scope_$$"
    echo 'feat(tmux): add a ~/.tmux.conf.local override seam' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "accepts every scoped subject currently on main" {
    # Real subjects from main — all four were rejected by the pre-#794 pattern.
    local msg_file="/tmp/bats_commit_msg_main_$$"
    local s
    for s in \
        'feat(tmux): add a ~/.tmux.conf.local override seam (#788)' \
        'docs(spec): PR-Agent on NaN inference as a uniform review workflow (#789)' \
        'chore(specs): archive HARNESS-051-copilot-native-skills (#787)' \
        'fix(setup): survive an empty crontab on a fresh node (#783)'
    do
        echo "$s" > "$msg_file"
        run bash "$HOOK" "$msg_file"
        [[ $status -eq 0 ]] || {
            echo "rejected: $s"
            rm -f "$msg_file"
            return 1
        }
    done
    rm -f "$msg_file"
}

@test "accepts a breaking-change marker, scoped and unscoped" {
    local msg_file="/tmp/bats_commit_msg_bang_$$"
    echo 'feat(api)!: drop the v1 endpoint' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    echo 'feat!: drop the v1 endpoint' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "accepts a scope containing dots, slashes and dashes" {
    local msg_file="/tmp/bats_commit_msg_sep_$$"
    echo 'chore(ai/hermes): re-pin the runtime' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "rejects a missing space after the colon" {
    local msg_file="/tmp/bats_commit_msg_nospace_$$"
    echo 'feat:no space' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    rm -f "$msg_file"
}

@test "rejects an empty subject after the colon" {
    local msg_file="/tmp/bats_commit_msg_empty_$$"
    echo 'feat: ' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    rm -f "$msg_file"
}

@test "validates the subject only: a conforming body cannot rescue it" {
    local msg_file="/tmp/bats_commit_msg_body_$$"
    printf '%s\n\n%s\n' 'THIS SUBJECT IS WRONG' 'feat: this body line conforms' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 1 ]]
    rm -f "$msg_file"
}

@test "exempts git-generated merge and revert subjects" {
    local msg_file="/tmp/bats_commit_msg_merge_$$"
    echo 'Merge branch main into feat/x' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    echo 'Revert "feat(tmux): add a seam"' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "exempts rebase fixup and squash subjects" {
    local msg_file="/tmp/bats_commit_msg_fixup_$$"
    echo 'fixup! feat(tmux): add a seam' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    echo 'squash! feat(tmux): add a seam' > "$msg_file"
    run bash "$HOOK" "$msg_file"
    [[ $status -eq 0 ]]
    rm -f "$msg_file"
}

@test "fails loudly when no message-file argument is passed" {
    run bash "$HOOK"
    [[ $status -eq 1 ]]
    [[ "$output" == *"no commit message file"* ]]
}

@test "the shebang resolves via env so the hook can run on Windows" {
    run head -n 1 "$HOOK"
    [[ $status -eq 0 ]]
    [[ "$output" == '#!/usr/bin/env sh' ]]
}
