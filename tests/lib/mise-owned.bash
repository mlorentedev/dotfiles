#!/usr/bin/env bash
# Assert a CLI is an npm package that mise owns through its npm backend (#2438):
# versions.conf pins it under `# mise: cli`, cli/internal/tools/mise.go routes
# its short name to `npm:<package>`, and packages.json no longer lists it, so
# one owner holds it on every OS.
#
#   assert_mise_npm_cli <name> <npm-package>
#
# Needs DOTFILES_DIR set to the repository root.
assert_mise_npm_cli() {
    local name="$1" pkg="$2" var
    var="$(printf '%s' "$name" | tr '[:lower:]-' '[:upper:]_')_VERSION"
    grep -B1 "^${var}=" "$DOTFILES_DIR/versions.conf" | head -1 | grep -qx '# mise: cli' ||
        { echo "versions.conf does not pin $var under '# mise: cli'"; return 1; }
    grep -qE "\"$name\":[[:space:]]+\"npm:$pkg\"" "$DOTFILES_DIR/cli/internal/tools/mise.go" ||
        { echo "mise.go's npmBackend does not route $name to npm:$pkg"; return 1; }
    [ -z "$(jq -r --arg n "$name" '.tools[] | select(.name == $n) | .name' "$DOTFILES_DIR/packages.json")" ] ||
        { echo "packages.json still lists $name: two owners"; return 1; }
}
