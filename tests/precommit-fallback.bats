#!/usr/bin/env bats
# BUG-036: pre-commit-managed gates under a global core.hooksPath.
#
# GUARD-001's global core.hooksPath makes `pre-commit install` refuse ("Cowardly
# refusing to install hooks with core.hooksPath set"), so a repo managed by
# pre-commit never gets a .git/hooks/<type> for the dispatcher to chain to — and
# its gates (the vault's gitleaks pre-push secret scan) silently never run.
#
# These tests drive a STUB `pre-commit` first on PATH. CI installs only age, zsh
# and bats, and a stub is also what makes the *invocation* assertable: the real
# tool would only report pass/fail, never what it was asked to do. What is under
# test is the dispatcher's decision and the command it builds — pre-commit's own
# behaviour is upstream's business.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CHAIN="$REPO/git-hooks/lib/chain-local-hook.sh"
    WORK="$(mktemp -d)"
    STUB="$WORK/stub"
    ARGS_LOG="$WORK/precommit-args"
    STDIN_LOG="$WORK/precommit-stdin"
    FIXTURE="$WORK/repo"
    mkdir -p "$STUB" "$FIXTURE"
    git -C "$FIXTURE" init -q
}

teardown() { [ -z "${WORK:-}" ] || rm -rf "$WORK"; }

# Put a fake `pre-commit` first on PATH. It records its arguments and stdin, then
# exits with $1, so a test can assert both how it was called and that its status
# is propagated.
stub_precommit() {
    {
        echo '#!/usr/bin/env bash'
        printf 'printf "%%s\\n" "$*" > %s\n' "$ARGS_LOG"
        printf 'cat > %s\n' "$STDIN_LOG"
        printf 'exit %s\n' "${1:-0}"
    } > "$STUB/pre-commit"
    chmod +x "$STUB/pre-commit"
    PATH="$STUB:$PATH"
}

# A repo whose hooks pre-commit manages, but where `pre-commit install` was
# refused — so there is a config and no .git/hooks/<type>.
add_precommit_config() {
    cat > "$FIXTURE/.pre-commit-config.yaml" <<'YAML'
repos:
  - repo: local
    hooks:
      - id: demo
        name: demo
        entry: true
        language: system
        stages: [pre-push]
YAML
}

# An executable repo-local hook of the given type, which must keep winning.
add_local_hook() {
    mkdir -p "$FIXTURE/.git/hooks"
    printf '#!/usr/bin/env bash\ntouch "%s/local-ran"\nexit %s\n' "$WORK" "${2:-0}" \
        > "$FIXTURE/.git/hooks/$1"
    chmod +x "$FIXTURE/.git/hooks/$1"
}

@test "AC1: with no local hook, a pre-commit config routes the stage to hook-impl" {
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "printf 'refs/heads/m aaa refs/heads/m bbb\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 0 ]

    # `hook-impl` rather than `run --hook-stage`: only hook-impl parses the
    # pre-push ref list on stdin into --from-ref/--to-ref, so only it scans the
    # commits actually being pushed.
    [[ "$(cat "$ARGS_LOG")" == *"hook-impl"* ]]
    [[ "$(cat "$ARGS_LOG")" == *"--hook-type pre-push"* ]]
}

@test "AC2: a failing pre-commit blocks the operation (exit status propagates)" {
    # The entire point of the gate. A hook that runs but swallows the status
    # blocks nothing.
    stub_precommit 1
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "printf 'refs/heads/m aaa refs/heads/m bbb\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 1 ]
}

@test "AC3: the pre-push ref list on stdin reaches pre-commit unmodified" {
    # Without stdin, pre-commit cannot tell which commits are being pushed and
    # falls back to the staged file set -- a green report on the wrong input.
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    refline='refs/heads/main 1111111111111111111111111111111111111111 refs/heads/main 2222222222222222222222222222222222222222'
    run bash -c "printf '%s\n' '$refline' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 0 ]
    # cmp, not a `$(...)` string comparison -- command substitution strips
    # trailing newlines, so a dispatcher bug that dropped the final \n would
    # pass silently under `[ "$(cat ...)" = "$refline" ]`.
    printf '%s\n' "$refline" > "$WORK/expected-stdin"
    cmp -s "$STDIN_LOG" "$WORK/expected-stdin"
}

@test "AC1: the remote name and url are forwarded to pre-commit" {
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "printf '\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 0 ]
    [[ "$(cat "$ARGS_LOG")" == *"origin git@example.com:o/r.git"* ]]
}

@test "AC4: an executable repo-local hook still wins and pre-commit is not also run" {
    # Preserves today's precedence. Running both would execute every gate twice.
    stub_precommit 0
    add_precommit_config
    add_local_hook pre-push
    cd "$FIXTURE"

    run bash -c "printf '\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 0 ]
    [ -f "$WORK/local-ran" ]
    [ ! -f "$ARGS_LOG" ]
}

@test "AC4: a failing repo-local hook still blocks, with no pre-commit fallback" {
    stub_precommit 0
    add_precommit_config
    add_local_hook pre-push 1
    cd "$FIXTURE"

    run bash -c "printf '\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 1 ]
    [ ! -f "$ARGS_LOG" ]
}

@test "AC5: no pre-commit config is a clean no-op" {
    stub_precommit 1
    cd "$FIXTURE"

    run bash -c "printf '\n' | '$CHAIN' pre-push origin git@example.com:o/r.git"
    [ "$status" -eq 0 ]
    [ ! -f "$ARGS_LOG" ]
}

@test "AC5: a missing pre-commit binary is a clean no-op, not a broken commit" {
    # This dispatcher is wired machine-wide. Failing closed here would break
    # `git commit` in every repo on the box that has a config but no pre-commit.
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "PATH='$STUB:/usr/bin:/bin' '$CHAIN' pre-commit < /dev/null"
    [ "$status" -eq 0 ]
}

@test "AC6: the fallback is stage-generic, not pre-push-only" {
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "'$CHAIN' pre-commit < /dev/null"
    [ "$status" -eq 0 ]
    [[ "$(cat "$ARGS_LOG")" == *"--hook-type pre-commit"* ]]
}

@test "AC6: commit-msg forwards the message file argument" {
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "'$CHAIN' commit-msg .git/COMMIT_EDITMSG < /dev/null"
    [ "$status" -eq 0 ]
    [[ "$(cat "$ARGS_LOG")" == *"--hook-type commit-msg"* ]]
    [[ "$(cat "$ARGS_LOG")" == *".git/COMMIT_EDITMSG"* ]]
}

@test "AC1: the config is passed explicitly, not left to the caller's cwd" {
    stub_precommit 0
    add_precommit_config
    cd "$FIXTURE"

    run bash -c "'$CHAIN' pre-commit < /dev/null"
    [ "$status" -eq 0 ]
    [[ "$(cat "$ARGS_LOG")" == *".pre-commit-config.yaml"* ]]
}

@test "outside a git repo the dispatcher stays a no-op" {
    stub_precommit 1
    cd "$WORK"

    run bash -c "'$CHAIN' pre-commit < /dev/null"
    [ "$status" -eq 0 ]
}
