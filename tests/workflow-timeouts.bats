#!/usr/bin/env bats
# Every workflow job must declare timeout-minutes.
#
# The incident: on 2026-08-19 the `test` job hung in `apt-get update` against an
# Ubuntu mirror that had stopped answering. Last output 06:30:08, cancelled BY HAND
# at 06:38:22 — eight minutes of silence, and not the first time. `test-windows`
# already carried `timeout-minutes: 30`; the job that actually hung carried none, so
# nothing could end it but a person watching.
#
# This asserts the only mechanism that kills a hung job, on every job, so the next
# hang — which will have a different cause, because this one is now bounded — ends
# by itself. It is deliberately a presence check: here the declaration IS the
# mechanism. GitHub enforces it; there is no gap between saying it and it happening.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    command -v python3 >/dev/null 2>&1 || skip "python3 required to parse workflow YAML"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

@test "every workflow job, and the apt step, declares a timeout" {
    run python3 "$BATS_TEST_DIRNAME/lib/check-workflow-timeouts.py" "$REPO"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "apt is bounded on BOTH schemes, since setting only http is what failed" {
    grep -q 'Acquire::http::Timeout' "$REPO/.github/workflows/ci.yml"
    grep -q 'Acquire::https::Timeout' "$REPO/.github/workflows/ci.yml"
}
