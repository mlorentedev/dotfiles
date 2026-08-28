#!/usr/bin/env bats
# ai/copilot/config.json is deployed verbatim to ~/.copilot/config.json. Audited
# against GitHub Copilot CLI 1.0.80 on 2026-08-27 (AI-036/#1296): the file
# declared a key the CLI does not read (`defaultModel`; the key is `model`), an
# id the seat does not list, a `$schema` URL that 404s, and left
# `includeCoAuthoredBy` at its default of TRUE -- so every commit Copilot made
# carried a Co-authored-by trailer, against the no-attribution doctrine.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CFG="$DOTFILES_DIR/ai/copilot/config.json"
}

@test "config.json is valid JSON" {
    jq -e . "$CFG" >/dev/null
}

@test "config.json turns Co-authored-by trailers off (no AI attribution doctrine)" {
    [ "$(jq -r '.includeCoAuthoredBy' "$CFG")" = "false" ]
}

@test "config.json pins the model under the key the CLI reads, not defaultModel" {
    [ "$(jq -r '.defaultModel // "absent"' "$CFG")" = "absent" ]
    model="$(jq -r '.model // ""' "$CFG")"
    [ -n "$model" ]
}

@test "config.json turns autoUpdate off: the packages.json pin owns updates (AI-038, ADR-036)" {
    [ "$(jq -r '.autoUpdate' "$CFG")" = "false" ]
}

@test "copilot is a packages.json catalog tool (npm) and neither setup carries an install block (AI-038, ADR-036)" {
    jq -e '.tools[] | select(.name == "copilot" and .source.type == "npm")' "$DOTFILES_DIR/packages.json" >/dev/null
    ! grep -qE 'Id = "GitHub\.Copilot"' "$DOTFILES_DIR/setup-windows.ps1"
    ! grep -qE '(apt|snap|curl)[^\n]*copilot' "$DOTFILES_DIR/setup-linux.sh"
}

@test "config.json carries no \$schema URL (the published one 404s)" {
    [ "$(jq -r '.["$schema"] // "absent"' "$CFG")" = "absent" ]
}
