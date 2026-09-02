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

    # C15: a check that cannot answer must SKIP saying why, never pass. OPS-040
    # removed the last two call sites (the OPENROUTER_API_KEY export both setup
    # scripts ran for a consumer CLI-042 AC8 had already deleted), so this loop
    # now iterates an empty list. A pass in that state would be indistinguishable
    # from "every call site checks out" and would stay green through the
    # reintroduction of a typo'd one, which is the entire failure #698 describes.
    if [ -z "$ids" ]; then
        skip "no 'dotf secrets show <id>' call sites in the tree — nothing to resolve (the shell/ps1 sweep found none)"
    fi

    missing=""
    for id in $ids; do
        grep -qE "^[[:space:]]*-[[:space:]]*id:[[:space:]]*${id}([[:space:]]|#|\$)" secrets/registry.yaml \
            || missing="$missing $id"
    done
    [ -z "$missing" ] || { echo "unknown registry ids referenced:$missing" >&2; return 1; }
}
