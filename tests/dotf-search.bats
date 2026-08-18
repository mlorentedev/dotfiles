#!/usr/bin/env bats
# Tests for `dotf search` (Eje 5 local BM25/token search)

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CLI="$REPO/cli"
    TMP="$BATS_TEST_TMPDIR/dotf-search"
    mkdir -p "$TMP/patterns" "$TMP/skills/test-skill"

    cat > "$TMP/patterns/pattern-sample.md" << 'EOF'
---
id: pattern-sample
name: sample-pattern
type: pattern
tags: [sample, testing]
keywords: [falsifiable, tree]
summary: Sample pattern summary
---
# Sample Pattern
Details on hypothesis testing.
EOF

    cat > "$TMP/skills/test-skill/SKILL.md" << 'EOF'
---
id: test-skill
name: test-skill
type: skill
tags: [test, automation]
description: Skill for running test workflows
---
# Test Skill
Run tests reliably.
EOF
}

@test "search: finds pattern by keyword in custom dir" {
    cd "$CLI"
    run go run ./cmd/dotf search --dir "$TMP" "falsifiable"
    [ "$status" -eq 0 ]
    [[ "$output" == *"pattern-sample"* ]]
    [[ "$output" == *"PATTERN"* ]]
}

@test "search: filters by type" {
    cd "$CLI"
    run go run ./cmd/dotf search --dir "$TMP" --type skill "test"
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-skill"* ]]
    [[ "$output" != *"pattern-sample"* ]]
}

@test "search: --json flag outputs valid JSON results" {
    cd "$CLI"
    run go run ./cmd/dotf search --dir "$TMP" --json "sample"
    [ "$status" -eq 0 ]
    run python3 -c "
import json, sys
data = json.loads('''$output''')
assert isinstance(data, list)
assert len(data) == 1
assert data[0]['id'] == 'pattern-sample'
assert data[0]['type'] == 'pattern'
"
    [ "$status" -eq 0 ]
}
