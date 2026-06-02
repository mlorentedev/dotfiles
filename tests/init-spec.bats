#!/usr/bin/env bats
# Tests for scripts/init-spec.sh — spec scaffolder + vault-gate.
# Regression guard for BUG-024: the vault-gate must resolve the CANONICAL repo
# name (not the worktree dir basename), so init-spec works from a git worktree.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="/tmp/bats_initspec_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$TMP"

    # --- fake vault with the three spec templates ---
    VAULT="$TMP/vault"
    mkdir -p "$VAULT/00_meta/templates"
    printf '# proposal\n\n## Why\n' > "$VAULT/00_meta/templates/spec-proposal.md"
    printf '# tasks\n' > "$VAULT/00_meta/templates/spec-tasks.md"
    printf '# verification\n' > "$VAULT/00_meta/templates/spec-verification.md"

    # --- main repo; canonical name = "canonrepo" ---
    MAINREPO="$TMP/canonrepo"
    mkdir -p "$MAINREPO"
    cd "$MAINREPO" || exit 1
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    echo seed > seed.txt
    git add -A
    git commit -q -m seed

    # --- vault backlog for the CANONICAL project name ---
    mkdir -p "$VAULT/10_projects/canonrepo"
    printf -- '- [ ] **TEST-001-foo** demo backlog entry\n' \
        > "$VAULT/10_projects/canonrepo/11-tasks.md"
}

teardown() {
    cd / || true
    rm -rf "$TMP"
}

@test "scaffolds in a plain clone (backward compatible)" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-001-foo/proposal.md" ]
}

@test "accepts a sub-id feature-id (SDD-012b / WIN-002a convention)" {
    # Sibling of BUG-024: the regex rejected sub-ids that check-backlog-integrity.sh
    # treats as distinct tickets. A trailing single letter on the number is valid.
    printf -- '- [ ] **TEST-002a-subid** sub-id backlog entry\n' \
        >> "$VAULT/10_projects/canonrepo/11-tasks.md"
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-002a-subid
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-002a-subid/proposal.md" ]
}

@test "still rejects a malformed feature-id (no ticket number)" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" notaticket
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid feature-id"* ]]
}

@test "BUG-024: scaffolds from a linked worktree (vault-gate resolves canonical name)" {
    git -C "$MAINREPO" worktree add -q "$TMP/canonrepo-feature" -b feature
    cd "$TMP/canonrepo-feature"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo
    [ "$status" -eq 0 ]
    [[ "$output" == *"Vault context found"* ]]
    # spec files land in the WORKTREE, not the main repo
    [ -f "$TMP/canonrepo-feature/specs/TEST-001-foo/proposal.md" ]
    [ ! -e "$MAINREPO/specs/TEST-001-foo" ]
}

@test "vault-gate still fails (exit 3) when the entry is genuinely missing" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-999-missing
    [ "$status" -eq 3 ]
    [[ "$output" == *"No vault entry"* ]]
}

@test "vault-gate missing-entry from a worktree also fails (no false-positive from the fix)" {
    git -C "$MAINREPO" worktree add -q "$TMP/canonrepo-feature" -b feature
    cd "$TMP/canonrepo-feature"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-999-missing
    [ "$status" -eq 3 ]
}

@test "--force-no-vault bypasses the gate from a worktree" {
    git -C "$MAINREPO" worktree add -q "$TMP/canonrepo-feature" -b feature
    cd "$TMP/canonrepo-feature"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-777-bypass --force-no-vault
    [ "$status" -eq 0 ]
    [ -f "$TMP/canonrepo-feature/specs/TEST-777-bypass/proposal.md" ]
}
