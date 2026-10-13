#!/usr/bin/env bats
# #2013 H4: ai/herdr/config.toml is merged into the config.toml herdr reads.
# herdr ignores a key or binding it does not know and falls back to its
# default (an invalid prefix becomes ctrl+b, tmux's), so a typo here would
# deploy cleanly and silently undo D6. herdr's own validator is the authority:
# it exits non-zero and names each issue.
# shellcheck disable=SC2016  # the contract defaults are literal $HOME / $env: text

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SRC="$REPO/ai/herdr/config.toml"
}

@test "herdr config check accepts ai/herdr/config.toml" {
    command -v herdr >/dev/null 2>&1 || skip "herdr is not on PATH (mise installs it: dotf tools sync)"
    run env HERDR_CONFIG_PATH="$SRC" herdr config check
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "herdr config check rejects a binding it does not know, so the test above can fail" {
    command -v herdr >/dev/null 2>&1 || skip "herdr is not on PATH (mise installs it: dotf tools sync)"
    bad="$BATS_TEST_TMPDIR/config.toml"
    { cat "$SRC"; printf '\n[ui]\nno_such_key = 1\n'; } > "$bad"
    run env HERDR_CONFIG_PATH="$bad" herdr config check
    [ "$status" -ne 0 ] || false
    [[ "$output" == *"no_such_key"* ]] || false
}

@test "the prefix is not tmux's ctrl+b (D6)" {
    run grep -E '^prefix[[:space:]]*=' "$SRC"
    [ "$status" -eq 0 ] || false
    [[ "$output" != *'"ctrl+b"'* ]] || false
}

@test "HERDR_CONFIG_DIR defaults to the directory herdr reads on each OS" {
    contract="$REPO/env-contract.json"
    run jq -r '.env_vars[] | select(.name == "HERDR_CONFIG_DIR") | .default.linux' "$contract"
    [ "$output" = '$HOME/.config/herdr' ] || false
    run jq -r '.env_vars[] | select(.name == "HERDR_CONFIG_DIR") | .default.windows' "$contract"
    [ "$output" = '$env:USERPROFILE\AppData\Roaming\herdr' ]
}
