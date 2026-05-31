#!/usr/bin/env bats
# MEMORY-001: session-handoff.sh — the Claude SessionEnd bridge (ADR-014).
# It ARCHIVES the `## Session Handoff` block that /handoff wrote into a project's
# MEMORY.md into an append-only record under 00_meta/sessions/. The agent reasons
# (via /handoff); the hook persists. Fixture-driven — no real vault, CI-safe.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/session-handoff.sh"
    VAULT="$(mktemp -d)"
    PROJ_PARENT="$(mktemp -d)"
    PROJ="$PROJ_PARENT/myproj"
    mkdir -p "$PROJ" "$VAULT/00_meta/sessions" "$VAULT/10_projects/myproj/memory"
    export VAULT_PATH="$VAULT"
    PAYLOAD="$VAULT/payload.json"
}

teardown() { rm -rf "$VAULT" "$PROJ_PARENT"; }

write_payload() {
    printf '{"session_id":"%s","cwd":"%s","hook_event_name":"SessionEnd","reason":"clear"}' "$1" "$PROJ" > "$PAYLOAD"
}

@test "AC1: archives the MEMORY.md handoff block into a session record" {
    cat > "$VAULT/10_projects/myproj/memory/MEMORY.md" <<'EOF'
# Memory
## Session Handoff
> Updated: 2026-05-31
**Last task:** built the thing
**Next action:** ship it
## Other
index-only stuff
EOF
    write_payload sess-123
    run bash -c "'$SCRIPT' < '$PAYLOAD'"
    [ "$status" -eq 0 ]
    rec="$(find "$VAULT/00_meta/sessions" -name '*-myproj-claude.md')"
    [ -n "$rec" ]
    grep -q 'session_id: sess-123' "$rec"
    grep -q 'built the thing' "$rec"
    # only the handoff block is archived, not the index-only tail
    ! grep -q 'index-only stuff' "$rec"
}

@test "AC2: no handoff block in MEMORY.md -> no record (trivial no-op)" {
    printf '# Memory\n## Index\nstuff\n' > "$VAULT/10_projects/myproj/memory/MEMORY.md"
    write_payload sess-1
    run bash -c "'$SCRIPT' < '$PAYLOAD'"
    [ "$status" -eq 0 ]
    [ -z "$(find "$VAULT/00_meta/sessions" -name '*.md')" ]
}

@test "AC2b: no MEMORY.md at all -> clean exit, no record" {
    write_payload sess-1
    run bash -c "'$SCRIPT' < '$PAYLOAD'"
    [ "$status" -eq 0 ]
    [ -z "$(find "$VAULT/00_meta/sessions" -name '*.md' 2>/dev/null)" ]
}

@test "usage: no stdin payload / empty -> clean no-op (never crashes a session)" {
    run bash -c "printf '' | '$SCRIPT'"
    [ "$status" -eq 0 ]
}
