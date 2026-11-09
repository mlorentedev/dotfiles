#!/usr/bin/env bats
# GUARD-001: the global pre-commit memory-sink dispatcher.
# Rejects MEMORY.md / memory/ in any repo that is NOT the vault, then chains to
# the repo-local hook so per-repo guards (gitleaks) survive. Tests wire the
# dispatcher via a *local* core.hooksPath on fixture repos — never the global
# config — so they are CI-safe and non-invasive.

# bats file_tags=os-sensitive

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    HOOKS="$REPO/git-hooks"
    VAULT="$(mktemp -d)"
    WORK="$(mktemp -d)"
    export VAULT_PATH="$VAULT"
    NONVAULT="$WORK/code-repo"
    mkdir -p "$NONVAULT"
    git -C "$NONVAULT" init -q
    git -C "$NONVAULT" config core.hooksPath "$HOOKS"
    git -C "$NONVAULT" config user.email t@t.io
    git -C "$NONVAULT" config user.name tester
}

teardown() { rm -rf "$VAULT" "$WORK"; }

@test "AC1: rejects MEMORY.md committed to a non-vault repo" {
    echo x > "$NONVAULT/MEMORY.md"
    git -C "$NONVAULT" add MEMORY.md
    run git -C "$NONVAULT" commit -m "try"
    [ "$status" -ne 0 ]
    [[ "$output" == *vault* ]] || false
}

@test "AC1: rejects a memory/ path committed to a non-vault repo" {
    mkdir -p "$NONVAULT/memory"
    echo x > "$NONVAULT/memory/MEMORY.md"
    git -C "$NONVAULT" add memory/MEMORY.md
    run git -C "$NONVAULT" commit -m "try"
    [ "$status" -ne 0 ]
}

@test "AC1: allows a normal file in a non-vault repo" {
    echo "print(1)" > "$NONVAULT/code.py"
    git -C "$NONVAULT" add code.py
    run git -C "$NONVAULT" commit -m "ok"
    [ "$status" -eq 0 ]
}

@test "AC2: allows MEMORY.md inside the vault (sentinel .obsidian/)" {
    V="$VAULT/somerepo"
    mkdir -p "$V/.obsidian"
    git -C "$V" init -q
    git -C "$V" config core.hooksPath "$HOOKS"
    git -C "$V" config user.email t@t.io
    git -C "$V" config user.name tester
    echo x > "$V/MEMORY.md"
    git -C "$V" add MEMORY.md
    run git -C "$V" commit -m "vault memory"
    [ "$status" -eq 0 ]
}

@test "AC2: allows MEMORY.md when repo root == VAULT_PATH" {
    git -C "$VAULT" init -q
    git -C "$VAULT" config core.hooksPath "$HOOKS"
    git -C "$VAULT" config user.email t@t.io
    git -C "$VAULT" config user.name tester
    echo x > "$VAULT/MEMORY.md"
    git -C "$VAULT" add MEMORY.md
    run git -C "$VAULT" commit -m "vault root"
    [ "$status" -eq 0 ]
}

@test "AC2: the vault comparison resolves git's path dialect before comparing" {
    # AC2 above passes on POSIX but silently failed on Git for Windows, where
    # `git rev-parse --show-toplevel` answers C:/... while pwd -P answers /c/...:
    # the vault's own MEMORY.md was blocked. Same mismatch reproduced portably —
    # the stub answers the toplevel probe with a redundant /./ segment that only
    # resolution collapses, and defers every other subcommand to the real binary.
    REAL_GIT="$(command -v git)"
    STUB="$(mktemp -d)"
    {
        echo '#!/usr/bin/env bash'
        echo 'if [ "${1:-}" = rev-parse ] && [ "${2:-}" = --show-toplevel ]; then'
        printf '    printf "%%s\\n" "%s/."\n' "$VAULT"
        echo '    exit 0'
        echo 'fi'
        printf 'exec "%s" "$@"\n' "$REAL_GIT"
    } > "$STUB/git"
    chmod +x "$STUB/git"

    git -C "$VAULT" init -q
    git -C "$VAULT" config core.hooksPath "$HOOKS"
    git -C "$VAULT" config user.email t@t.io
    git -C "$VAULT" config user.name tester
    echo x > "$VAULT/MEMORY.md"
    git -C "$VAULT" add MEMORY.md
    run env PATH="$STUB:$PATH" git -C "$VAULT" commit -m "vault root, resolved"
    rm -rf "$STUB"
    [ "$status" -eq 0 ]
}

@test "AC3: chains to repo-local .git/hooks/pre-commit (both run)" {
    mkdir -p "$NONVAULT/.git/hooks"
    printf '#!/usr/bin/env bash\ntouch "%s/local-ran"\n' "$WORK" > "$NONVAULT/.git/hooks/pre-commit"
    chmod +x "$NONVAULT/.git/hooks/pre-commit"
    echo "print(1)" > "$NONVAULT/code.py"
    git -C "$NONVAULT" add code.py
    run git -C "$NONVAULT" commit -m "ok"
    [ "$status" -eq 0 ]
    [ -f "$WORK/local-ran" ]
}

@test "AC3: a failing repo-local hook still blocks the commit (chain propagates)" {
    mkdir -p "$NONVAULT/.git/hooks"
    printf '#!/usr/bin/env bash\nexit 1\n' > "$NONVAULT/.git/hooks/pre-commit"
    chmod +x "$NONVAULT/.git/hooks/pre-commit"
    echo "print(1)" > "$NONVAULT/code.py"
    git -C "$NONVAULT" add code.py
    run git -C "$NONVAULT" commit -m "blocked by local"
    [ "$status" -ne 0 ]
}

@test "AC3: guard runs even when no repo-local hook exists" {
    echo x > "$NONVAULT/MEMORY.md"
    git -C "$NONVAULT" add MEMORY.md
    run git -C "$NONVAULT" commit -m "try"
    [ "$status" -ne 0 ]
}
