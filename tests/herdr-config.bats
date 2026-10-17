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

@test "every harness that trusts folders trusts the worktrees herdr creates (#2503)" {
    # herdr puts a worktree at <directory>/<repo>/<branch-slug>. A harness that
    # does not trust that path asks, or refuses, in every worktree herdr opens.
    # Both trust lists render the root from WORKTREE_ROOT, the root dotf
    # worktree add uses; the next test keeps herdr's directory equal to it.
    want='{WORKTREE_ROOT}/*/*'
    # agy 1.3.5 does not read `<dir>/*/*` as a glob (measured 2026-10-14, #2533):
    # a fresh worktree under it still asked. The plain directory is trusted with
    # everything beneath it, so agy is given the root, not the pattern.
    agy_want="${want%/\*/\*}"
    run jq -e --arg w "$agy_want" '.trustedWorkspaces | index($w)' "$REPO/ai/agy/settings.json"
    [ "$status" -eq 0 ] || { echo "agy trustedWorkspaces lacks $agy_want"; false; }
    run jq -e --arg w "$want" '.trustedFolders | index($w)' "$REPO/ai/copilot/config.json"
    [ "$status" -eq 0 ] || { echo "copilot trustedFolders lacks $want"; false; }
}

@test "herdr's worktrees.directory is rendered from WORKTREE_ROOT (#2503)" {
    # dotf worktree add places checkouts under WORKTREE_ROOT; herdr under its
    # [worktrees] directory. Deploy renders the token, so the two tools cannot
    # create worktrees in different trees; without paths the token would reach
    # herdr verbatim.
    run grep -E '^directory[[:space:]]*=[[:space:]]*"\{WORKTREE_ROOT\}"$' "$SRC"
    [ "$status" -eq 0 ] || { echo "herdr directory is not \"{WORKTREE_ROOT}\""; false; }
    run jq -r '.configs[] | select(.name == "herdr") | .paths' "$REPO/ai/deploy.json"
    [ "$output" = "native" ] || { echo "herdr entry renders no paths: $output"; false; }
}

@test "pi's sidebar entry carries the session title row (#2646)" {
    # pi 1.2.0 sets the terminal title itself, so herdr's terminal_title_stripped
    # is the session name; without this row a pi pane shows the default rows only.
    # Static on purpose: it needs no herdr binary, so it runs on every runner.
    run sed -n '/^\[ui\.sidebar\.agents\.rows_by_agent\]/,/^\[worktrees\]/p' "$SRC"
    pi_block=$(printf '%s\n' "$output" | sed -n '/^pi[[:space:]]*=[[:space:]]*\[/,/^\]/p')
    [ -n "$pi_block" ] || { echo "no pi = [...] block under rows_by_agent"; false; }
    [[ "$pi_block" == *'"terminal_title_stripped"'* ]] || { echo "$pi_block"; false; }
}

@test "herdr config check validates rows_by_agent tokens, so the row above cannot rot" {
    command -v herdr >/dev/null 2>&1 || skip "herdr is not on PATH (mise installs it: dotf tools sync)"
    bad="$BATS_TEST_TMPDIR/config.toml"
    sed 's/"terminal_title_stripped"/"no_such_token"/' "$SRC" > "$bad"
    run env HERDR_CONFIG_PATH="$bad" herdr config check
    [ "$status" -ne 0 ] || false
    [[ "$output" == *"no_such_token"* ]] || false
}
