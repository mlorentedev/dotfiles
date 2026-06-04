#!/usr/bin/env bats
# Tests for .zsh/functions.sh — portable swiss-army shell functions (IDEAS-002).
# Functions are POSIX-portable, so each behavioural test is exercised under bash;
# cross-shell sourcing is asserted under both bash and zsh.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    FN_FILE="$DOTFILES_DIR/.zsh/functions.sh"
    # Per-test scratch dir, cleaned in teardown.
    WORK="$(mktemp -d "${BATS_TMPDIR:-/tmp}/ideas002.XXXXXX")"
}

teardown() {
    [ -n "${WORK:-}" ] && rm -rf "$WORK"
}

# --- File + static checks ---

@test "functions.sh exists" {
    [ -f "$FN_FILE" ]
}

@test "functions.sh is shellcheck-clean (AC2)" {
    if ! command -v shellcheck >/dev/null 2>&1; then skip "shellcheck not installed"; fi
    run shellcheck "$FN_FILE"
    [ "$status" -eq 0 ]
}

@test "functions.sh valid bash syntax" {
    run bash -n "$FN_FILE"
    [ "$status" -eq 0 ]
}

@test "functions.sh valid zsh syntax" {
    run zsh -n "$FN_FILE"
    [ "$status" -eq 0 ]
}

@test "functions.sh defines all six functions" {
    for fn in mkd targz dataurl server getcertnames gz; do
        grep -qE "^${fn}\(\)" "$FN_FILE"
    done
}

@test "functions.sh sources cleanly under bash and zsh (cross-shell, AC8)" {
    run bash -c ". '$FN_FILE'"
    [ "$status" -eq 0 ]
    run zsh -c ". '$FN_FILE'"
    [ "$status" -eq 0 ]
}

# --- mkd ---

@test "mkd creates a nested path and cd's into it (AC4)" {
    run bash -c ". '$FN_FILE'; mkd '$WORK/nested/deep'; printf '%s' \"\$PWD\""
    [ "$status" -eq 0 ]
    [ "$output" = "$WORK/nested/deep" ]
    [ -d "$WORK/nested/deep" ]
}

@test "mkd with no argument prints usage and fails" {
    run bash -c ". '$FN_FILE'; mkd"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage: mkd"* ]]
}

# --- gz ---

@test "gz reports both original and gzipped sizes (AC6)" {
    printf 'the quick brown fox jumps over the lazy dog\n' > "$WORK/sample.txt"
    run bash -c ". '$FN_FILE'; gz '$WORK/sample.txt'"
    [ "$status" -eq 0 ]
    [[ "$output" == *"original:"* ]]
    [[ "$output" == *"gzipped:"* ]]
    # Both lines carry a numeric byte count.
    echo "$output" | grep -qE 'original: [0-9]+ bytes'
    echo "$output" | grep -qE 'gzipped:  [0-9]+ bytes'
}

@test "gz on a missing file fails with usage" {
    run bash -c ". '$FN_FILE'; gz '$WORK/does-not-exist'"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage: gz"* ]]
}

# --- dataurl ---

@test "dataurl emits a data: URL that round-trips to the original (AC5)" {
    printf 'IDEAS-002 payload' > "$WORK/payload.txt"
    run bash -c ". '$FN_FILE'; dataurl '$WORK/payload.txt'"
    [ "$status" -eq 0 ]
    [[ "$output" == data:* ]]
    [[ "$output" == *";base64,"* ]]
    # Strip everything up to and including 'base64,' then decode.
    decoded="$(printf '%s' "$output" | sed 's/^data:[^,]*base64,//' | base64 -d)"
    [ "$decoded" = "IDEAS-002 payload" ]
}

@test "dataurl on a missing file fails with usage" {
    run bash -c ". '$FN_FILE'; dataurl '$WORK/nope'"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage: dataurl"* ]]
}

# --- targz ---

@test "targz produces a gzip-decompressible tarball (AC7, gzip-compatible path)" {
    mkdir -p "$WORK/src"
    printf 'one\n' > "$WORK/src/a.txt"
    printf 'two\n' > "$WORK/src/b.txt"
    run bash -c ". '$FN_FILE'; targz '$WORK/src'"
    [ "$status" -eq 0 ]
    [ -f "$WORK/src.tar.gz" ]
    # The defining property across zopfli/pigz/gzip: a gzip-compatible stream.
    run bash -c "gzip -dc '$WORK/src.tar.gz' | tar -t"
    [ "$status" -eq 0 ]
    [[ "$output" == *"a.txt"* ]]
    [[ "$output" == *"b.txt"* ]]
}

@test "targz on a missing path fails with usage" {
    run bash -c ". '$FN_FILE'; targz '$WORK/missing'"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage: targz"* ]]
}

# --- server / getcertnames: smoke only (network / python / interactive) ---

@test "server reports missing python3 instead of hanging (smoke)" {
    # Run with an empty PATH so python3 is absent: must fail fast, not block.
    run bash -c "PATH=/nonexistent; . '$FN_FILE'; server 8123"
    [ "$status" -ne 0 ]
}

@test "getcertnames prints usage with no host (smoke)" {
    run bash -c ". '$FN_FILE'; getcertnames"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage: getcertnames"* ]]
}

# --- Wiring: both rc files source the portable functions file (AC3) ---

@test ".zshrc sources .zsh/functions.sh" {
    grep -qF '.zsh/functions.sh' "$DOTFILES_DIR/.zshrc"
}

@test ".bashrc sources .zsh/functions.sh (bash parity)" {
    grep -qF '.zsh/functions.sh' "$DOTFILES_DIR/.bashrc"
}
