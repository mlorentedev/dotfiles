#!/usr/bin/env bats
# Tests for scripts/check-backlog-merged.sh — the "merged-but-open" backlog guard
# (SDD-012b). Semantic sibling of check-backlog-integrity.bats (SDD-012, structural).
# The done-signal is an archived spec keyed on the FULL ticket id.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    GUARD="$SCRIPTS_DIR/check-backlog-merged.sh"
    TMP="/tmp/bats_mergedguard_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$TMP"

    # fake repo with an archived-specs tree
    REPO="$TMP/repo"
    mkdir -p "$REPO/specs/archive/SDD-009-opencode-deploy-time-secrets"
    mkdir -p "$REPO/specs/archive/WIN-002a-vault-health-sh-inverted-check"

    TASKS="$TMP/11-tasks.md"
}

teardown() {
    cd / || true
    rm -rf "$TMP"
}

@test "flags an open entry whose full id has an archived spec" {
    printf -- '- [ ] **SDD-009-opencode-deploy-time-secrets** still open here\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 1 ]
    [[ "$output" == *"STALE-OPEN: SDD-009-opencode-deploy-time-secrets"* ]]
    [[ "$output" == *"archived spec exists"* ]]
}

@test "clean when the same id is already ticked [x]" {
    printf -- '- [x] **SDD-009-opencode-deploy-time-secrets** done\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 0 ]
}

@test "clean when an open entry has NO archived spec" {
    printf -- '- [ ] **AI-020-gemini-empirical-validation** genuinely open\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 0 ]
}

@test "full-id keyed: a different ticket sharing a number is NOT flagged" {
    # archive has SDD-009-opencode...; an open SDD-009-something-else must not match
    printf -- '- [ ] **SDD-009-some-other-ticket** distinct full id\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 0 ]
}

@test "sub-id (WIN-002a) resolves distinctly from its parent number" {
    printf -- '- [ ] **WIN-002a-vault-health-sh-inverted-check** sub-id\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 1 ]
    [[ "$output" == *"WIN-002a-vault-health-sh-inverted-check"* ]]
}

@test "open statuses [~] and [-] are also checked" {
    printf -- '- [~] **SDD-009-opencode-deploy-time-secrets** in progress\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 1 ]
}

@test "multiple stale-open entries are all reported and counted" {
    {
        printf -- '- [ ] **SDD-009-opencode-deploy-time-secrets** a\n'
        printf -- '- [ ] **WIN-002a-vault-health-sh-inverted-check** b\n'
        printf -- '- [ ] **AI-020-gemini-empirical-validation** c\n'
    } > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$REPO"
    [ "$status" -eq 1 ]
    [[ "$output" == *"SDD-009-opencode-deploy-time-secrets"* ]]
    [[ "$output" == *"WIN-002a-vault-health-sh-inverted-check"* ]]
    # AI-020 has no archived spec -> not flagged
    [[ "$output" != *"AI-020"* ]]
}

@test "no repo / no specs/archive -> advisory skip, exit 0" {
    printf -- '- [ ] **SDD-009-opencode-deploy-time-secrets** open\n' > "$TASKS"
    run "$GUARD" "$TASKS" --repo "$TMP/nonexistent-repo"
    [ "$status" -eq 0 ]
    [[ "$output" == *"SKIP"* ]]
}

@test "missing tasks file -> exit 2" {
    run "$GUARD" "$TMP/does-not-exist.md" --repo "$REPO"
    [ "$status" -eq 2 ]
}

@test "no argument -> usage, exit 2" {
    run "$GUARD"
    [ "$status" -eq 2 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "repo inference: <vault>/10_projects/<proj>/11-tasks.md -> \$HOME/Projects/<proj>" {
    # Build a path the inference will map to our fake repo via HOME override.
    FAKE_HOME="$TMP/home"
    mkdir -p "$FAKE_HOME/Projects/repo/specs/archive/SDD-009-opencode-deploy-time-secrets"
    VAULTTASKS="$TMP/vault/10_projects/repo/11-tasks.md"
    mkdir -p "$(dirname "$VAULTTASKS")"
    printf -- '- [ ] **SDD-009-opencode-deploy-time-secrets** open\n' > "$VAULTTASKS"
    run env HOME="$FAKE_HOME" "$GUARD" "$VAULTTASKS"
    [ "$status" -eq 1 ]
    [[ "$output" == *"STALE-OPEN: SDD-009-opencode-deploy-time-secrets"* ]]
}
