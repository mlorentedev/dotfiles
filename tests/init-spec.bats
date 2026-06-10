#!/usr/bin/env bats
# Tests for scripts/init-spec.sh — spec scaffolder + work-gate.
#
# REFACTOR-012 (ADR-018): the work-gate is an OPEN GitHub issue (--issue <N>),
# not a vault 11-tasks.md entry. `gh` is stubbed on PATH — no network here.
# Keeps the BUG-024 worktree regression coverage (scaffold from a linked
# worktree must land files in the worktree).

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="/tmp/bats_initspec_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$TMP"

    # --- fake vault with the three spec templates (templates stay vault-sourced) ---
    VAULT="$TMP/vault"
    mkdir -p "$VAULT/00_meta/templates"
    printf '# proposal\n\n## Why\n' > "$VAULT/00_meta/templates/spec-proposal.md"
    printf '# tasks\n' > "$VAULT/00_meta/templates/spec-tasks.md"
    printf '# verification\n' > "$VAULT/00_meta/templates/spec-verification.md"

    # --- stub gh: issue 42 is OPEN, issue 43 is CLOSED, anything else is unknown ---
    # The script calls: gh issue view <N> --json state,title --jq <expr>
    STUBBIN="$TMP/bin"
    mkdir -p "$STUBBIN"
    cat > "$STUBBIN/gh" <<'EOF'
#!/bin/bash
num="$2"
[ "$1" = "issue" ] && num="$3"
case "$num" in
    42) printf 'OPEN\tDemo open issue for the gate\n'; exit 0 ;;
    43) printf 'CLOSED\tAlready done\n'; exit 0 ;;
    *)  echo "GraphQL: Could not resolve to an issue" >&2; exit 1 ;;
esac
EOF
    chmod +x "$STUBBIN/gh"
    export PATH="$STUBBIN:$PATH"

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
}

teardown() {
    cd / || true
    rm -rf "$TMP"
}

@test "scaffolds with an open issue gate (--issue 42)" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue 42
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-001-foo/proposal.md" ]
    [[ "$output" == *"Work-gate"* ]]
}

@test "injects issue context into proposal ## Why" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue 42
    [ "$status" -eq 0 ]
    grep -q '<!-- from issue #42: Demo open issue for the gate -->' \
        "$MAINREPO/specs/TEST-001-foo/proposal.md"
}

@test "accepts a sub-id feature-id (SDD-012b / WIN-002a convention)" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-002a-subid --issue 42
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-002a-subid/proposal.md" ]
}

@test "still rejects a malformed feature-id (no ticket number)" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" notaticket --issue 42
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid feature-id"* ]]
}

@test "rejects a non-numeric --issue value" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue abc
    [ "$status" -eq 1 ]
}

@test "gate fails (exit 3) without --issue and without bypass" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo
    [ "$status" -eq 3 ]
    [[ "$output" == *"--issue"* ]]
    [ ! -e "$MAINREPO/specs/TEST-001-foo" ]
}

@test "gate fails (exit 3) when the issue does not exist" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue 999
    [ "$status" -eq 3 ]
    [ ! -e "$MAINREPO/specs/TEST-001-foo" ]
}

@test "gate fails (exit 3) when the issue is CLOSED" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue 43
    [ "$status" -eq 3 ]
    [[ "$output" == *"not open"* ]]
    [ ! -e "$MAINREPO/specs/TEST-001-foo" ]
}

@test "--force-no-gate bypasses the gate without calling gh" {
    cd "$MAINREPO"
    # poison the stub: any gh call now explodes — bypass must never invoke it
    cat > "$STUBBIN/gh" <<'EOF'
#!/bin/bash
echo "gh must not be called on bypass" >&2
exit 99
EOF
    chmod +x "$STUBBIN/gh"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-777-bypass --force-no-gate
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-777-bypass/proposal.md" ]
}

@test "legacy --force-no-vault still works as a deprecated alias" {
    cd "$MAINREPO"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-778-legacy --force-no-vault
    [ "$status" -eq 0 ]
    [ -f "$MAINREPO/specs/TEST-778-legacy/proposal.md" ]
    [[ "$output" == *"deprecated"* ]]
}

@test "BUG-024 regression: scaffolds from a linked worktree into the worktree" {
    git -C "$MAINREPO" worktree add -q "$TMP/canonrepo-feature" -b feature
    cd "$TMP/canonrepo-feature"
    run env VAULT_PATH="$VAULT" "$SCRIPTS_DIR/init-spec.sh" TEST-001-foo --issue 42
    [ "$status" -eq 0 ]
    # spec files land in the WORKTREE, not the main repo
    [ -f "$TMP/canonrepo-feature/specs/TEST-001-foo/proposal.md" ]
    [ ! -e "$MAINREPO/specs/TEST-001-foo" ]
}

@test "ADR-018: no 11-tasks.md reference remains in init-spec.{sh,ps1}" {
    ! grep -q '11-tasks' "$SCRIPTS_DIR/init-spec.sh"
    ! grep -q '11-tasks' "$SCRIPTS_DIR/init-spec.ps1"
}
