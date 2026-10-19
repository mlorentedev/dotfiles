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
    # A table this file does not already define: appending a second [ui] would be
    # a duplicate-table parse error, and the test would pass on the wrong
    # message (it did, once [ui] arrived for agent_panel_sort).
    { cat "$SRC"; printf '\n[terminal]\nno_such_key = 1\n'; } > "$bad"
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

@test "the agent panel answers who needs me first, not who was opened first" {
    # One key, no other setting depends on it; revert to grouping by deleting it.
    run grep -E '^agent_panel_sort[[:space:]]*=[[:space:]]*"priority"$' "$SRC"
    [ "$status" -eq 0 ] || { echo "ui.agent_panel_sort is not priority: $output"; false; }
}

@test "prefix+e opens a file explorer over the focused pane" {
    # Static: the binding is what a person presses, and it can rot without the
    # binary noticing (herdr ignores an unknown key silently).
    block=$(grep -A5 'key = "prefix+e"' "$SRC")
    [[ "$block" == *'type = "popup"'* ]] || { echo "prefix+e is not a popup: $block"; false; }
    [[ "$block" == *'command = "yazi"'* ]] || { echo "prefix+e does not run yazi: $block"; false; }
}

@test "the explorer's yazi comes from a pin, so a new machine provisions it (#2814)" {
    # A binding that runs a tool nobody provisions is a key that fails on a
    # fresh box. mise, not a hand install, is the provisioning path on every OS.
    run grep -E '^YAZI_VERSION=[0-9]+\.[0-9]+' "$REPO/versions.conf"
    [ "$status" -eq 0 ] || { echo "versions.conf pins no YAZI_VERSION, so the prefix+e popup has no provisioner"; false; }
}

# _pin_ge <a> <b>: true when dotted version a is at or above b.
_pin_ge() {
    awk -v a="$1" -v b="$2" 'BEGIN{
        split(a, x, "."); split(b, y, ".")
        for (i = 1; i <= 3; i++) if (x[i] + 0 != y[i] + 0) exit (x[i] + 0 > y[i] + 0 ? 0 : 1)
        exit 0
    }'
}

@test "a binding never names a dotf subcommand the pinned release does not have (#2814)" {
    # DOTF_VERSION is what a NEW machine installs; CI builds the tree, so a
    # binding naming a tree-only subcommand deploys cleanly here and fails on
    # every other machine (lesson 373). floor is the first release that carries
    # `dotf herdr explorer`: raise it in the follow-up PR that adds the binding,
    # never before. Both halves fail rather than skip, because a skip is a guard
    # that stopped guarding (#2815 review).
    floor="0.76.0"
    dotf_pin=$(sed -n 's/^DOTF_VERSION=//p' "$REPO/versions.conf")
    [ -n "$dotf_pin" ] || { echo "versions.conf has no DOTF_VERSION"; false; }
    run grep -E "^command[[:space:]]*=[[:space:]]*['\"]dotf herdr explorer" "$SRC"
    if [ "$status" -eq 0 ]; then
        _pin_ge "$dotf_pin" "$floor" \
            || { echo "the binding names dotf herdr explorer, but DOTF_VERSION=$dotf_pin predates the release carrying it ($floor)"; false; }
        return 0
    fi
    if _pin_ge "$dotf_pin" "$floor"; then
        echo "DOTF_VERSION=$dotf_pin is at or past $floor, so the release carries the subcommand: land the docked binding and delete this guard"
        false
    fi
}

@test "dotf herdr explorer exists and names its provisioner" {
    # The target of the follow-up binding. go run keeps it tied to the tree, so a
    # rename here cannot leave a binding pointing at nothing.
    command -v go >/dev/null 2>&1 || skip "Go toolchain not on PATH"
    run bash -c "cd '$REPO/cli' && go run ./cmd/dotf herdr explorer --help"
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    [[ "$output" == *"YAZI_VERSION"* ]] || { echo "$output"; false; }
}
