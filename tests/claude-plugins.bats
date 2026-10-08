#!/usr/bin/env bats
# ai/claude/plugins.json is the plugin list `dotf deploy` installs (CLI-063, #1339).
# Until the cutover deletes them, both setup twins still carry the same list
# hardcoded. This guard fails the moment the three disagree, so the port stays a
# port and no list drifts while two of them still run.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PLUGINS_JSON="$DOTFILES_DIR/ai/claude/plugins.json"
}

ssot_plugins() {
    jq -r '.plugins[]' "$PLUGINS_JSON" | sort
}

# The ids quoted between `for plugin in \` and the `; do` that closes the list.
linux_plugins() {
    sed -n '/^    for plugin in \\$/,/; do$/p' "$DOTFILES_DIR/setup-linux.sh" |
        grep -oE '"[^"]+@[^"]+"' | tr -d '"' | sort
}

# The ids quoted inside the `$plugins = @(` array. The file is CRLF.
windows_plugins() {
    tr -d '\r' < "$DOTFILES_DIR/setup-windows.ps1" |
        sed -n '/^[[:space:]]*\$plugins = @($/,/^[[:space:]]*)$/p' |
        grep -oE '"[^"]+@[^"]+"' | tr -d '"' | sort
}

@test "plugins.json is valid JSON with a non-empty plugins array of ids" {
    jq -e '.plugins | type == "array" and length > 0 and all(type == "string" and test("^[^@]+@[^@]+$"))' "$PLUGINS_JSON"
}

@test "setup-linux.sh installs exactly the plugins in plugins.json" {
    [ -n "$(linux_plugins)" ]
    [ "$(linux_plugins)" = "$(ssot_plugins)" ]
}

@test "setup-windows.ps1 installs exactly the plugins in plugins.json" {
    [ -n "$(windows_plugins)" ]
    [ "$(windows_plugins)" = "$(ssot_plugins)" ]
}
