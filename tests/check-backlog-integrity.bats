#!/usr/bin/env bats
# SDD-012: tasks-integrity guard. Detects the backlog-drift class — the same
# ticket ID on 2+ entry lines (duplication) and, as a refinement, the same ID
# marked both open and done (status contradiction). Fixture-based: no vault,
# CI-safe. Sibling of tests covering check-md-escapes.sh (SDD-006).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/check-backlog-integrity.sh"
    VAULT="$REPO/scripts/vault.sh"
    TMP="$(mktemp -d)"
}

teardown() { rm -rf "$TMP"; }

@test "AC3 clean: unique IDs incl WIN-002 vs WIN-002a (sub-id) -> exit 0" {
    cat > "$TMP/clean.md" <<'EOF'
- [ ] **AI-001-alpha**: a thing
- [x] **AI-002-beta**: done thing
- [ ] **WIN-002-sweep**: parent ticket
- [ ] **WIN-002a-subitem**: a DISTINCT sub-ticket, not a duplicate of WIN-002
EOF
    run "$SCRIPT" "$TMP/clean.md"
    [ "$status" -eq 0 ]
}

@test "AC1 duplicate: same ID on two entry lines -> exit 1, names the ID" {
    cat > "$TMP/dup.md" <<'EOF'
- [ ] **ENGINE-001-deploy**: sprint-overview view
- [x] **BUG-009-foo**: done
- [ ] **ENGINE-001-deploy**: detailed view (the drift)
EOF
    run "$SCRIPT" "$TMP/dup.md"
    [ "$status" -eq 1 ]
    echo "$output" | grep -q 'ENGINE-001'
}

@test "full-id: same NUMBER, different slug -> exit 0 + advisory NOTE (not drift)" {
    cat > "$TMP/collide.md" <<'EOF'
- [x] **BUG-020-pwsh-profile-corruption-heal**: fixed via #122
- [x] **BUG-020-splitpath-literalpath-parent**: a DIFFERENT bug, fixed via #124
EOF
    run "$SCRIPT" "$TMP/collide.md"
    [ "$status" -eq 0 ]
    echo "$output" | grep -qi 'NOTE'
    echo "$output" | grep -q 'BUG-020'
}

@test "AC2 contradiction: same ID both [ ] and [x] -> exit 1, flagged as contradiction" {
    cat > "$TMP/contra.md" <<'EOF'
- [ ] **POLISH-004-bundle**: still open in the sprint overview
- [x] **POLISH-004-bundle**: but ticked done in the detailed list
EOF
    run "$SCRIPT" "$TMP/contra.md"
    [ "$status" -eq 1 ]
    echo "$output" | grep -q 'POLISH-004'
    echo "$output" | grep -qi 'contradict'
}

@test "usage: no args -> exit 2" {
    run "$SCRIPT"
    [ "$status" -eq 2 ]
}

@test "missing file -> exit 2" {
    run "$SCRIPT" "$TMP/nope.md"
    [ "$status" -eq 2 ]
}

@test "AC4 dispatcher: 'vault check-tasks <file>' routes to the guard (exit passthrough)" {
    cat > "$TMP/dup.md" <<'EOF'
- [ ] **X-001-a**: one
- [ ] **X-001-a**: two
EOF
    run "$VAULT" check-tasks "$TMP/dup.md"
    [ "$status" -eq 1 ]
}
