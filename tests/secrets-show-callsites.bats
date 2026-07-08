#!/usr/bin/env bats
# #698: dotf secrets show <id> call sites must reference a real registry id.
# Registry ids are UPPER_SNAKE_CASE; a lower-kebab-case typo resolves to
# "unknown secret", silently swallowed by `2>/dev/null || true` at the call
# site, baking an empty value into whatever consumes it (setup-linux.sh /
# setup-windows.ps1 baked an empty OPENROUTER_API_KEY into agy's MCP config
# on every fresh deploy, with zero signal).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "every 'dotf secrets show <id>' call site resolves to a real registry id" {
    cd "$DOTFILES_DIR"
    ids=$(grep -rhoE "dotf secrets show [A-Za-z0-9_-]+" --include='*.sh' --include='*.ps1' . \
        | awk '{print $NF}' | sort -u)
    missing=""
    for id in $ids; do
        grep -qE "^[[:space:]]*-[[:space:]]*id:[[:space:]]*${id}([[:space:]]|#|\$)" secrets/registry.yaml \
            || missing="$missing $id"
    done
    [ -z "$missing" ] || { echo "unknown registry ids referenced:$missing" >&2; return 1; }
}
