#!/usr/bin/env bats
# MEMORY-002: ensure-memory-symlink.sh — agent-agnostic vault->memory symlink.
# Extracted from claude-session-start.sh `ensure_memory_symlink`. The resolver
# (three vault-source conventions) is shared; the per-agent target is an
# argument. Fixture-driven — no real vault, CI-safe.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/ensure-memory-symlink.sh"
    VAULT="$(mktemp -d)"
    WORK="$(mktemp -d)"
    export VAULT_PATH="$VAULT"
    mkdir -p "$VAULT/00_meta" "$VAULT/10_projects" "$VAULT/50_work/45-development"
}

teardown() { rm -rf "$VAULT" "$WORK"; }

@test "AC1: creates symlink from 10_projects/<name>/memory convention" {
    proj="$WORK/myproj"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/myproj/memory"
    target="$WORK/agent/myproj/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
    [ "$(readlink "$target")" = "$VAULT/10_projects/myproj/memory" ]
}

@test "AC1: creates symlink for in-vault CWD/memory convention" {
    proj="$VAULT/somezone"; mkdir -p "$proj/memory"
    target="$WORK/agent/somezone/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
    [ "$(readlink "$target")" = "$proj/memory" ]
}

@test "AC1: creates symlink for nested 50_work/45-development convention" {
    mkdir -p "$VAULT/50_work/45-development/myfamily/mycomp/memory"
    proj="$WORK/myfamily/mycomp"; mkdir -p "$proj"
    target="$WORK/agent/mycomp/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
    [ "$(readlink "$target")" = "$VAULT/50_work/45-development/myfamily/mycomp/memory" ]
}

@test "AC1: --project overrides the basename for source resolution" {
    proj="$WORK/checkout-dir"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/realname/memory"
    target="$WORK/agent/realname/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target' --project realname"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
    [ "$(readlink "$target")" = "$VAULT/10_projects/realname/memory" ]
}

@test "AC1: prints a one-line message when a link is created" {
    proj="$WORK/myproj"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/myproj/memory"
    target="$WORK/agent/myproj/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [[ "$output" == *"myproj"* ]]
}

@test "AC2: no vault source -> no-op, exit 0, no symlink" {
    proj="$WORK/orphan"; mkdir -p "$proj"
    target="$WORK/agent/orphan/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ ! -e "$target" ]
}

@test "AC2: already-linked target -> idempotent no-op, exit 0" {
    proj="$WORK/myproj"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/myproj/memory"
    target="$WORK/agent/myproj/memory"; mkdir -p "$(dirname "$target")"
    ln -s "$VAULT/10_projects/myproj/memory" "$target"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
}

@test "AC2: non-empty real target dir -> never clobbered, exit 0" {
    proj="$WORK/myproj"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/myproj/memory"
    target="$WORK/agent/myproj/memory"; mkdir -p "$target"
    echo "precious" > "$target/data.txt"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ ! -L "$target" ]
    [ -f "$target/data.txt" ]
    grep -q precious "$target/data.txt"
}

@test "missing required args -> exit non-zero" {
    run bash -c "'$SCRIPT' --cwd '$WORK'"
    [ "$status" -ne 0 ]
}

@test "AC3: Claude-encoded deep target convention links correctly (delegation parity)" {
    proj="$WORK/Projects/myproj"; mkdir -p "$proj"
    mkdir -p "$VAULT/10_projects/myproj/memory"
    encoded="$(printf '%s' "$proj" | tr '/' '-')"   # Claude's encode_project_path
    target="$WORK/dotclaude/projects/$encoded/memory"
    run bash -c "'$SCRIPT' --cwd '$proj' --target '$target'"
    [ "$status" -eq 0 ]
    [ -L "$target" ]
    [ "$(readlink "$target")" = "$VAULT/10_projects/myproj/memory" ]
}

@test "AC5: claude-session-start.sh delegates and no longer inlines the resolver" {
    hook="$REPO/scripts/claude-session-start.sh"
    grep -q "ensure-memory-symlink.sh" "$hook"
    # the inline 3-convention resolver is gone (no direct 10_projects source build)
    ! grep -q 'vault_memory="\$KNOWLEDGE_VAULT/10_projects' "$hook"
}
