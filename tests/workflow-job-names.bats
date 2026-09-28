#!/usr/bin/env bats
# Status-check names are the contract branch protection enforces (CI-004, #1739).
#
# `lint` was reported by two workflows, `ci.yml`'s shell lint and `cli.yml`'s Go
# lint, so the required check was satisfied by whichever finished last. And a
# required check that no job reports on a given pull request stays pending
# forever. The checker is tests/lib/check-workflow-contexts.py.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    command -v python3 >/dev/null 2>&1 || skip "python3 required to parse workflow YAML"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

@test "workflow job display names are unique across workflows" {
    run python3 "$BATS_TEST_DIRNAME/lib/check-workflow-contexts.py" "$REPO"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}
