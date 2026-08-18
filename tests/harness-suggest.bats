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
