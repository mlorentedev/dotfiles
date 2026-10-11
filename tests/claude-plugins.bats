#!/usr/bin/env bats
# ai/claude/plugins.json is the plugin list `dotf deploy` installs (CLI-063, #1339)
# on every OS, counting each failure and naming its cause. The setup twins used
# to carry the same list in a loop that counted attempts and discarded every
# error, and printed SUCCESS after `dotf deploy` had just failed the same
# installs (#2336). Neither may grow one back.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PLUGINS_JSON="$DOTFILES_DIR/ai/claude/plugins.json"
}

@test "plugins.json is valid JSON with a non-empty plugins array of ids" {
    jq -e '.plugins | type == "array" and length > 0 and all(type == "string" and test("^[^@]+@[^@]+$"))' "$PLUGINS_JSON"
}

# Comments are dropped first: a pointer to `dotf deploy` may name the command.
@test "neither setup script installs a Claude Code plugin itself" {
    for f in setup-linux.sh setup-windows.ps1; do
        run bash -c "tr -d '\r' < '$DOTFILES_DIR/$f' | grep -v '^[[:space:]]*#' | grep -nE 'claude plugin (install|list)'"
        [ "$status" -eq 1 ] || { echo "$f: $output"; false; }
    done
}

@test "neither setup script carries a plugin id from plugins.json" {
    while IFS= read -r id; do
        for f in setup-linux.sh setup-windows.ps1; do
            run grep -nF "\"$id\"" "$DOTFILES_DIR/$f"
            [ "$status" -eq 1 ] || { echo "$f carries $id: $output"; false; }
        done
    done < <(jq -r '.plugins[]' "$PLUGINS_JSON")
}
