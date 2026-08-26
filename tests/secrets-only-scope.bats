#!/usr/bin/env bats
#
# Every token in a `dotf secrets run --only <list>` must be a live registry id or
# exposed env var.
#
# `--only` fails LOUD on an unknown token -- it refuses the whole launch rather
# than dropping the one name it cannot resolve. That is right for a secrets
# facade, and it means a stale token in a wrapper is not a degradation but a total
# outage of whatever the wrapper launches.
#
# Measured 2026-08-25: the opencode wrapper in .bashrc and .zshrc named
# OLLAMA_API_KEY for a provider slot whose registry entry was never created. Both
# shells produced:
#
#     Error: --only: unknown id or env var "OLLAMA_API_KEY"
#
# so opencode -- the primary daily agent -- did not start at all. `dotf doctor`
# reported 152 passed / 0 failed with its [OpenCode + pi] section green
# throughout: nothing checked that the wrapper's list and the registry agreed.
#
# The deeper fault is duplication: the registry already declares which secrets
# each consumer takes (`consumers: [agent:opencode]`), and the wrapper restates
# it. Two sources of truth, and this is what their drift costs. Until the wrapper
# derives its scope from the registry, this keeps them honest.
#
# The scan lives in tests/lib/only-scope-scan.py so it is independently runnable
# and this file stays an assertion rather than an 85-line test body.

load 'lib/refute'

setup() {
    REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
}

@test "every --only token names a live registry id or exposed env var" {
    run python3 "$REPO_ROOT/tests/lib/only-scope-scan.py" "$REPO_ROOT"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output"; false; }
}

# The specific name that caused the outage, asserted so a revert is loud.
#
# Comment lines are excluded on purpose: the removal is DOCUMENTED at each site it
# was removed from, and a naive whole-file refute fails on the explanation of the
# very thing it checks. What must not come back is the executable reference, not
# the memory of it.
@test "no executable line names OLLAMA_API_KEY (the provider is gone)" {
    local f
    for f in .bashrc .zshrc ai/opencode/opencode.jsonc; do
        run bash -c "grep -vE '^[[:space:]]*(#|//)' '$REPO_ROOT/$f' | grep -n 'OLLAMA_API_KEY' || true"
        [ -z "$output" ] || { echo "$f still names OLLAMA_API_KEY in code: $output"; false; }
    done
}
