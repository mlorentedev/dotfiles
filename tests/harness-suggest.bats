#!/usr/bin/env bats
# Tests for `dotf harness suggest` (Eje 1 router / dynamic suggestion)

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CLI="$REPO/cli"
    TMP="$BATS_TEST_TMPDIR/harness-suggest"
    mkdir -p "$TMP"
}

@test "harness suggest: suggests python pattern and skill on prompt" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest --prompt "memory leak in python script"
    [ "$status" -eq 0 ]
    [[ "$output" == *"pattern-python-cli"* ]]
    [[ "$output" == *"async-python-patterns"* ]]
}

@test "harness suggest: matches path arguments" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest Dockerfile
    [ "$status" -eq 0 ]
    [[ "$output" == *"pattern-container-workflow"* ]]
    [[ "$output" == *"docker"* ]]
}

@test "harness suggest: --json outputs valid JSON object with patterns and skills" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest --json --prompt "create docker container"
    [ "$status" -eq 0 ]
    run python3 -c "
import json, sys
data = json.loads('''$output''')
assert 'patterns' in data
assert 'skills' in data
assert 'pattern-container-workflow' in data['patterns']
assert 'docker' in data['skills']
"
    [ "$status" -eq 0 ]
}

@test "harness suggest: resolves terraform trigger on iac prompt" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest --prompt "create terraform module for EKS cluster"
    [ "$status" -eq 0 ]
    [[ "$output" == *"pattern-terraform-standards"* ]]
    [[ "$output" == *"terraform"* ]]
}

@test "harness suggest: resolves helm trigger on Chart.yaml path" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest Chart.yaml
    [ "$status" -eq 0 ]
    [[ "$output" == *"pattern-kubernetes-packaging"* ]]
    [[ "$output" == *"helm"* ]]
}

@test "harness suggest: resolves transitive skill dependencies" {
    cd "$CLI"
    run go run ./cmd/dotf harness suggest specs/FEATURE-1/proposal.md
    [ "$status" -eq 0 ]
    [[ "$output" == *"spec"* ]]
    [[ "$output" == *"adversarial-review"* ]]
    [[ "$output" == *"verification-before-completion"* ]]
}

