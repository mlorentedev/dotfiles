#!/usr/bin/env bats
# Secret-resolution contract for the nan-* helper scripts (ADR-028 Phase 1,
# spec CLI-024-secrets-retire-loadsecrets / #587).
#
# The nan-* scripts must resolve NAN_API_KEY through the `dotf secrets` facade —
# injected by `dotf secrets run`, or self-resolved by re-exec'ing through it —
# never by sourcing the retired `load-secrets` twin. Structural (grep-based) like
# the other RC/script contract tests in this suite.
#
# The facade verb changed in CLI-042 (#1190): `dotf secrets show` no longer
# resolves this secret, because it is now multi-var. See the test below.

load 'lib/refute'

setup() {
    export SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    NAN_SCRIPTS=(nan-bench.sh nan-debug.sh nan-quality-bench.sh)
}

@test "nan-* scripts exist" {
    for s in "${NAN_SCRIPTS[@]}"; do
        [[ -f "$SCRIPTS_DIR/$s" ]] || { echo "missing: $s"; return 1; }
    done
}

@test "nan-* scripts do not source the retired load-secrets twin" {
    for s in "${NAN_SCRIPTS[@]}"; do
        refute_grep 'load-secrets' "$SCRIPTS_DIR/$s"
    done
}

@test "nan-* scripts do not call secrets_refresh (the old twin API)" {
    for s in "${NAN_SCRIPTS[@]}"; do
        refute_grep_fixed 'secrets_refresh' "$SCRIPTS_DIR/$s"
    done
}

# CLI-042 (#1190) replaced the `dotf secrets show` self-fetch this test used to
# pin. Two reasons, and the first is forced:
#
# `show` REFUSES a multi-var secret -- one value on stdout is ambiguous for one --
# and nan-api-key now exposes a second name (HIVE_WORKER_API_KEY) for hive's
# worker. The second reason is that re-exec'ing through `run` is stronger anyway:
# the credential reaches the process environment instead of a shell variable that
# `set -x`, an `env` in a subshell, or a core dump can spill.
#
# The invariant this file guards is unchanged -- the scripts resolve through the
# `dotf secrets` facade and never the retired load-secrets twin. Only the facade
# verb moved.
@test "nan-* scripts self-resolve NAN_API_KEY by re-exec'ing through dotf secrets run" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qE 'exec dotf secrets run --only NAN_API_KEY -- "\$0"' "$SCRIPTS_DIR/$s" \
            || { echo "no 'exec dotf secrets run --only NAN_API_KEY -- \$0' re-exec: $s"; return 1; }
    done
}

# `show` cannot resolve this secret any more, so a script still calling it would
# fail at runtime with a message about ambiguity rather than about the thing the
# user asked for. Guard the regression rather than trusting the rewrite.
@test "nan-* scripts no longer call dotf secrets show, which cannot resolve a multi-var secret" {
    for s in "${NAN_SCRIPTS[@]}"; do
        refute_grep_fixed 'dotf secrets show NAN_API_KEY' "$SCRIPTS_DIR/$s"
    done
}

# An unbounded re-exec would spin if `run` ever returned without injecting. Each
# script carries a sentinel that bounds it to one hop.
@test "nan-* scripts bound the re-exec with a sentinel so it cannot loop" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qE '\[ -z "\$\{NAN_[A-Z]+_REEXEC:-\}" \]' "$SCRIPTS_DIR/$s" \
            || { echo "no re-exec sentinel guard: $s"; return 1; }
        grep -qE 'NAN_[A-Z]+_REEXEC=1 exec dotf' "$SCRIPTS_DIR/$s" \
            || { echo "sentinel not set on the re-exec: $s"; return 1; }
    done
}

@test "nan-* scripts guard the dotf fetch with command -v dotf" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qE 'command -v dotf' "$SCRIPTS_DIR/$s" || { echo "no 'command -v dotf' guard: $s"; return 1; }
    done
}

@test "nan-* scripts hint 'dotf secrets run' when the key is missing" {
    for s in "${NAN_SCRIPTS[@]}"; do
        grep -qF 'dotf secrets run' "$SCRIPTS_DIR/$s" || { echo "no run hint: $s"; return 1; }
    done
}

@test "nan-* scripts parse cleanly (bash -n)" {
    for s in "${NAN_SCRIPTS[@]}"; do
        bash -n "$SCRIPTS_DIR/$s" || { echo "syntax error: $s"; return 1; }
    done
}
