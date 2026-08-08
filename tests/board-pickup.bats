#!/usr/bin/env bats
# Tests for git-hooks/lib/board-pickup.sh — the pickup forcing-function helper
# (knowledge#140). The board move itself is done by the bitacora-status Action on
# issues:[assigned] (HARNESS-010); this helper only self-assigns. We lock down the
# pure local contract: branch flag/name parsing, open-state guard, current-repo→
# knowledge resolution order, and the fire-and-forget guarantee (a board hiccup
# never errors a checkout).
#
# Hermetic: a fake `gh` on PATH logs each `issue edit` (repo + number) and answers
# `issue view --json state` from a controllable fixture, so no network is touched.

setup() {
    HELPER="$BATS_TEST_DIRNAME/../git-hooks/lib/board-pickup.sh"
    TMP="$(mktemp -d "/tmp/bats_pickup_XXXXXX")"
    GHLOG="$TMP/edit.log"
    export GHLOG

    # Fixture knobs the fake gh reads from the environment:
    #   STUB_STATE    — state `issue view` reports (default OPEN)
    #   STUB_BAD_REPO — a repo for which `issue view` fails (simulates "no such issue")
    #   STUB_FAIL_ALL — non-empty → every gh call exits 1 (fire-and-forget probe)
    export STUB_STATE="OPEN" STUB_BAD_REPO="" STUB_FAIL_ALL=""

    mkdir -p "$TMP/bin"
    cat > "$TMP/bin/gh" <<'EOF'
#!/usr/bin/env bash
[ -n "${STUB_FAIL_ALL:-}" ] && exit 1
sub="${1:-}"; action="${2:-}"; num="${3:-}"
repo=""
while [ $# -gt 0 ]; do [ "$1" = "--repo" ] && { shift; repo="$1"; }; shift; done
if [ "$sub" = "issue" ] && [ "$action" = "view" ]; then
    [ "$repo" = "${STUB_BAD_REPO:-}" ] && exit 1
    echo "${STUB_STATE:-OPEN}"; exit 0
fi
if [ "$sub" = "issue" ] && [ "$action" = "edit" ]; then
    echo "$repo $num" >> "$GHLOG"; exit 0
fi
exit 0
EOF
    chmod +x "$TMP/bin/gh"
    export PATH="$TMP/bin:$PATH"

    REPO="$TMP/repo"
    mkdir -p "$REPO"
    git -C "$REPO" init -q

    # GUARD-001 wires core.hooksPath globally, so an unqualified `git checkout`
    # in this fixture fires the machine's real post-checkout dispatcher — which
    # backgrounds *this very helper* and appends its own line to $GHLOG. Every
    # count assertion below would then measure the machine's hooks as well as the
    # helper under test. Neutralising it in the fixture's own config (rather than
    # per call, as tests/precommit-fallback.bats does) covers all 13 git
    # invocations here and any added later.
    mkdir -p "$TMP/no-hooks"
    git -C "$REPO" config core.hooksPath "$TMP/no-hooks"
    git -C "$REPO" config user.email t@t
    git -C "$REPO" config user.name t
    git -C "$REPO" remote add origin "git@github.com:mlorentedev/dotfiles.git"
    git -C "$REPO" commit -q --allow-empty -m init
}

teardown() { rm -rf "$TMP"; }

run_helper() { ( cd "$REPO" && bash "$HELPER" "$@" ); }

@test "open issue in the current repo: self-assigns there, no fallback" {
    git -C "$REPO" checkout -q -b feat/77-some-slug
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
    grep -q "mlorentedev/dotfiles 77" "$GHLOG"
    ! grep -q "mlorentedev/knowledge" "$GHLOG"
}

@test "current-repo issue missing: falls back to the knowledge bitacora home" {
    export STUB_BAD_REPO="mlorentedev/dotfiles"
    git -C "$REPO" checkout -q -b fix/88-thing
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
    grep -q "mlorentedev/knowledge 88" "$GHLOG"
}

@test "closed issue is never assigned (no resurrecting Done work)" {
    export STUB_STATE="CLOSED"
    git -C "$REPO" checkout -q -b feat/77-x
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
    [ ! -s "$GHLOG" ]
}

@test "file checkout (flag=0) is an immediate no-op -- gh never called" {
    git -C "$REPO" checkout -q -b feat/77-x
    run run_helper "" "" "0"
    [ "$status" -eq 0 ]
    [ ! -s "$GHLOG" ]
}

@test "branch without a leading issue number: no-op, gh never called" {
    git -C "$REPO" checkout -q -b chore/no-number-here
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
    [ ! -s "$GHLOG" ]
}

@test "fire-and-forget: a failing gh never errors the checkout" {
    export STUB_FAIL_ALL="1"
    git -C "$REPO" checkout -q -b feat/77-x
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
}

@test "knowledge repo itself: assigns once, no redundant fallback" {
    git -C "$REPO" remote set-url origin "git@github.com:mlorentedev/knowledge.git"
    git -C "$REPO" checkout -q -b fix/52-thing
    run run_helper "" "" "1"
    [ "$status" -eq 0 ]
    [ "$(grep -c 'mlorentedev/knowledge 52' "$GHLOG")" -eq 1 ]
}

@test "fixture isolation: a checkout must not fire the machine's post-checkout dispatcher" {
    # Guard for the isolation itself, not for the helper. Without the
    # core.hooksPath neutralisation in setup(), the global GUARD-001 dispatcher
    # runs the real post-checkout, which launches board-pickup.sh in the
    # background -- so every count assertion in this file silently measures two
    # runs. That is exactly how "assigns once, no redundant fallback" came to
    # fail on a clean main while the helper itself was correct.
    git -C "$REPO" checkout -q -b feat/77-x
    # The dispatcher backgrounds the helper (`... &`), so an immediate check can
    # pass on timing alone. Wait long enough that a leaked run would have landed.
    sleep 1
    [ ! -s "$GHLOG" ]
}
