#!/usr/bin/env bats
# #685: vault-health.sh aborts on a *healthy* vault. `grep -c` with zero
# matches prints "0" but exits 1; `|| echo "0"` then appends a second "0"
# instead of reassigning, so `$((COUNT * 100 / TOTAL))` becomes an arithmetic
# syntax error under `set -euo pipefail`. This drives the real script against
# a stubbed `obsidian` CLI simulating a zero-issue vault and asserts exit 0 --
# fails on the pre-fix script, passes on the fix.

setup() {
    export SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    TMP="$(mktemp -d)"
    VAULT_DIR="$TMP/vault"
    mkdir -p "$VAULT_DIR"
    cat > "$VAULT_DIR/note.md" <<'EOF'
---
id: note
type: note
status: active
tags: [test]
created: "2026-01-01"
owner: manu
---

# Note
EOF

    # Stub the `obsidian` CLI: empty orphans/dead-ends/unresolved/tags output
    # (the zero-match path that triggers the bug), non-empty vault connectivity.
    cat > "$TMP/obsidian" <<'EOF'
#!/usr/bin/env bash
case "$2" in
    vault) echo "ok" ;;
    orphans|dead-ends|unresolved|tags) : ;;
esac
exit 0
EOF
    chmod +x "$TMP/obsidian"
    export PATH="$TMP:$PATH"
}

teardown() {
    [ -z "${TMP:-}" ] || rm -rf "$TMP"
}

@test "vault-health.sh exits 0 against a healthy, zero-issue vault" {
    run env VAULT_DIR="$VAULT_DIR" VAULT_NAME="test" "$SCRIPTS_DIR/vault-health.sh"
    [ "$status" -eq 0 ]
}
