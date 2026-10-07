#!/usr/bin/env bats
# `dotf agent run` — the one property only a real binary and a real pipe can
# show (CLI-042 AC1).
#
# This file used to hold twelve cases. Eleven restated what the Go tests
# already assert in-process, and each paid a binary invocation (and four a
# python interpreter) to do it. They were deleted when each was shown to have a
# Go twin that fails when the behaviour breaks; the mapping is in the PR that
# removed them, and the twins are in cli/internal/cmd/agent_test.go,
# agent_deny_test.go, agent_wiring_test.go and agent_probe_unix_test.go.
#
# What is left is the case the Go tests cannot assert. captureRealStreams swaps
# os.Stdout for a pipe inside one process; this runs the compiled binary and
# pipes it into a separate consumer, which is how every dispatcher reaches it.
#
# The binary is DOTF_BIN when CI built it, otherwise one build per file run.
# Skips when the Go toolchain is absent locally, fails in CI and on a build
# error (tests/lib/dotf-bin.bash).

load 'lib/dotf-bin'

setup() {
    REPO_ROOT="$BATS_TEST_DIRNAME/.."
    # Slot state goes to a per-test dir, never the machine's real one. The
    # semaphore is deliberately machine-scoped (two checkouts share one
    # subscription), so a smoke that used the default would take real slots from
    # whatever else is dispatching on this box.
    SEM_DIR="$BATS_TEST_TMPDIR/slots"
    # `dotf agent run` refuses on a machine that has not declared an identity
    # (ADR-032 §8), so the smoke points machine.json at a fixture rather than
    # the real one. Pointing at the real file would make this case pass only
    # on a machine whose owner had configured it — green here, red in CI.
    export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/config"
    mkdir -p "$XDG_CONFIG_HOME/dotfiles"
    printf '%s\n' '{"machine": {"id": "bats-fixture"}, "pools": {"deny": []}}' \
        > "$XDG_CONFIG_HOME/dotfiles/machine.json"
}

@test "agent run: a dispatcher can read the status through a pipe" {
    dotf_bin_resolve "${BATS_FILE_TMPDIR:-/tmp}/dotf-agent-run"
    command -v jq >/dev/null 2>&1 || skip "jq not installed"
    # The documented consumer, verbatim from the command's own Long text.
    run bash -c "'$DOTF_BIN' agent run --role reviewer --task smoke --tier mid \
        --backend dry-run --timeout 30s --repo-root '$REPO_ROOT' --semaphore-dir '$SEM_DIR' | jq -r .status"
    [ "$status" -eq 0 ]
    [ "$output" = "dry_run" ]
}
