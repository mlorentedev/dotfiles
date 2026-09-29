#!/usr/bin/env bats
# Status-check names are the contract branch protection enforces (CI-004, #1739).
#
# `lint` was reported by two workflows, `ci.yml`'s shell lint and `cli.yml`'s Go
# lint, so the required check was satisfied by whichever finished last. And a
# required check that no job reports on a given pull request stays pending
# forever. And an aggregate gate such as `cli-gate` only sees the jobs it
# needs: `release-snapshot` was left out, so a red snapshot let it go green.
# The checker is tests/lib/check-workflow-contexts.py.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    command -v python3 >/dev/null 2>&1 || skip "python3 required to parse workflow YAML"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

@test "workflow job display names are unique across workflows" {
    run python3 "$BATS_TEST_DIRNAME/lib/check-workflow-contexts.py" "$REPO"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The gate rule on fixtures: which jobs count as a gate, and which are exempt.
gate_fixture() {
    mkdir -p "$BATS_TEST_TMPDIR/.github/workflows" "$BATS_TEST_TMPDIR/forge"
    printf '%s\n' '{"repos":{"mlorentedev/dotfiles":{"protection":{"required_status_checks":{"checks":[]}}}}}' \
        > "$BATS_TEST_TMPDIR/forge/branch-protection.json"
    cat > "$BATS_TEST_TMPDIR/.github/workflows/g.yml"
}

@test "gate rule: a gate reading needs.<job>.result must need every pull-request job" {
    gate_fixture <<'YML'
on: pull_request
jobs:
  a: {runs-on: ubuntu-latest, steps: [{run: "true"}]}
  b: {runs-on: ubuntu-latest, steps: [{run: "true"}]}
  gate: {runs-on: ubuntu-latest, needs: [a], steps: [{run: "test '${{ needs.a.result }}' = success"}]}
YML
    run python3 "$BATS_TEST_DIRNAME/lib/check-workflow-contexts.py" "$BATS_TEST_TMPDIR"
    [ "$status" -eq 1 ]
    [[ "$output" == *"g.yml:gate does not need 'b'"* ]]
}

@test "gate rule: only a positive tag-ref if: is exempt, a negated one is not" {
    gate_fixture <<'YML'
on: pull_request
jobs:
  release: {runs-on: ubuntu-latest, if: "startsWith(github.ref, 'refs/tags/v')", steps: [{run: "true"}]}
  notag: {runs-on: ubuntu-latest, if: "${{ !startsWith(github.ref, 'refs/tags/') }}", steps: [{run: "true"}]}
  gate: {runs-on: ubuntu-latest, needs: [], steps: [{run: "echo '${{ join(needs.*.result, ',') }}'"}]}
YML
    run python3 "$BATS_TEST_DIRNAME/lib/check-workflow-contexts.py" "$BATS_TEST_TMPDIR"
    [ "$status" -eq 1 ]
    [[ "$output" == *"does not need 'notag'"* ]]
    [[ "$output" != *"'release'"* ]]
}
