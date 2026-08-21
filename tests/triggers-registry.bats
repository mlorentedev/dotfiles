#!/usr/bin/env bats
# GUARD (#1137): triggers.json schema and drift guard.
#
# Asserts harness/triggers.json parses, conforms to the TriggerConfig shape,
# has unique IDs, and does not drift from cli/internal/harness/triggers.json.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TRIGGERS="$REPO/harness/triggers.json"
    EMBEDDED="$REPO/cli/internal/harness/triggers.json"
}

@test "triggers.json: file exists and is valid JSON" {
    [ -f "$TRIGGERS" ]
    jq empty "$TRIGGERS"
}

@test "triggers.json: version is a positive integer" {
    local ver
    ver="$(jq -r '.version' "$TRIGGERS")"
    [ "$ver" -ge 1 ]
}

@test "triggers.json: triggers array is non-empty" {
    local count
    count="$(jq -r '.triggers | length' "$TRIGGERS")"
    [ "$count" -ge 1 ]
}

@test "triggers.json: every trigger entry has non-empty id, pattern, and globs array" {
    local invalid
    invalid="$(jq -r '
        .triggers[] |
        select(
            (.id | type != "string" or length == 0) or
            (.pattern | type != "string" or length == 0) or
            (.globs | type != "array" or length == 0)
        ) | .id // "<missing-id>"
    ' "$TRIGGERS")"
    [ -z "$invalid" ]
}

@test "triggers.json: all trigger IDs are unique (no duplicates)" {
    local total unique
    total="$(jq -r '.triggers | length' "$TRIGGERS")"
    unique="$(jq -r '.triggers[].id' "$TRIGGERS" | sort -u | wc -l)"
    [ "$total" -eq "$unique" ]
}

@test "triggers.json: embedded Go copy matches root harness/triggers.json byte-for-byte (no drift)" {
    [ -f "$EMBEDDED" ]
    cmp -s "$TRIGGERS" "$EMBEDDED"
}
