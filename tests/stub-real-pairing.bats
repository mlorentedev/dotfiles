#!/usr/bin/env bats
# A stub proves what we ASKED a tool to do; only the real tool proves it accepts
# it. BUG-055 shipped green with 15 passing cases over the exact broken branch,
# because `tests/precommit-fallback.bats` stubs `pre-commit` — by design, since a
# stub is what makes the *invocation* assertable. The cost is that "the command
# we built" and "a command the tool survives" are different claims, and only the
# first was ever tested. `git commit` then aborted in every repo on the machine.
#
# This guard does not forbid stubbing. It forbids stubbing a third-party binary
# as the ONLY evidence, by requiring each such suite to declare its position:
# either a `<name>-real.bats` sibling that drives the real dependency, or a line
# in the exemption table below saying why one cannot exist.
#
# The table is the known backlog, not an approval. It exists so the set can only
# shrink deliberately — a NEW stub suite fails this test until someone chooses.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    TESTS="$REPO/tests"
}

# Suites that stub a binary and have no real-dependency sibling, each with the
# reason. Add a row only when a real test genuinely cannot exist; prefer writing
# the sibling.
#
# Single-sourced: both exempt() and the "no stale entries" test below read this
# one list, rather than each keeping its own copy. They used to be two
# independently-maintained copies, and had already drifted — `bitacora-rollout`
# was exempted here but missing from the stale-entries loop, so an exemption
# going stale for that one suite specifically would have gone undetected. That
# is the exact failure mode this file exists to catch, reproduced inside itself.
#
#   bitacora-reconcile        stubs `gh` — a real run mutates the live GitHub project board
#   bitacora-rollout          stubs `gh` — same; a real run adds items to the live board. This
#                             suite is a worked example of the limitation BUG-055 names: #884 was
#                             a call the real `gh` rejects and the stub accepts, so no stub could
#                             have caught it. What the suite CAN pin is which call is made, and it
#                             does; the API's verdict on that call only arrives from a real run.
#   board-pickup              stubs `gh` — same; a real run self-assigns real issues
#   guard-memory-sink         stubs `git` — the real path is covered end-to-end by the dispatcher's own commit-time behaviour
#   guard-no-gui              stubs `obsidian` — and here a real run is not merely inconvenient, it is
#                             the defect. That suite's whole subject is that a test must never launch a
#                             GUI application; a real-dependency sibling would be a test that opens
#                             Obsidian against the developer's live vault, which is the thing measured
#                             on 2026-08-23 (18 stray processes) and the reason the guard exists. The
#                             stub there is not standing in for a real run — it proves the guard yields
#                             to a test's own stub.
#   hermes-setup              stubs remote installers — a real run provisions an agent host
#   install-dotf              stubs the release download — a real run fetches from GitHub releases
#   shell-profile             stubs `zsh`/`bash` timing probes — a real run measures this machine, not a fixture
#   skills-pipeline           stubs the deploy targets — a real run writes into the caller's own $HOME
#   vault-health              stubs `hive` — a real run needs the daemon and a live vault
#   vault-health-go-parity    stubs `obsidian` (from tests/golden/vault-health/lib.sh, the SAME
#                             stub vault-health-golden uses, shared by both suites) — same
#                             rationale: a real run needs the AppImage and a live vault. This
#                             suite's own subject is Go/shell PARITY against frozen goldens, not
#                             the stub's fidelity to the real CLI, which vault-health-golden's
#                             ORACLE hash check already guards.
#   vault-health-golden       stubs `obsidian` (from tests/golden/vault-health/lib.sh, not the
#                             .bats file itself — see #892) — same rationale as vault-health: a
#                             real run needs the AppImage and a live vault
#   vault-maintenance-weekly  stubs `cron`/`hive` — a real run installs a crontab entry
EXEMPT_SUITES="bitacora-reconcile bitacora-rollout board-pickup guard-memory-sink guard-no-gui
hermes-setup install-dotf shell-profile skills-pipeline vault-health vault-health-go-parity
vault-health-golden vault-maintenance-weekly"

exempt() {
    local base
    for base in $EXEMPT_SUITES; do
        [ "$base" = "$1" ] && return 0
    done
    return 1
}

# A suite "stubs a binary" when it makes something executable and puts its
# directory on PATH — the shape that shadows a real tool for the suite's own
# process. Deliberately structural: it matches intent, not a naming convention.
_stubs_a_binary_in_file() {
    grep -q 'chmod +x' "$1" && grep -qE 'PATH="?\$' "$1"
}

# Resolves `. "$VAR/name"` / `source "$VAR/name"` lines to a path under tests/,
# one level of same-file variable substitution deep — the shape every golden
# corpus suite uses: `HERE="$BATS_TEST_DIRNAME/golden/x"; . "$HERE/lib.sh"`.
# Deliberately one level: this resolves what THE SUITE sources, not what a
# sourced library goes on to source itself.
_sourced_test_libs() {
    local f="$1" line name val target rest
    local -A vars=()

    while read -r name val; do
        vars["$name"]="$val"
    done < <(
        # shellcheck disable=SC2016  # the $BATS_TEST_DIRNAME is a literal
        # pattern matched against the source file's text, not an expansion here.
        grep -oE '^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*="\$BATS_TEST_DIRNAME/[^"]*"' "$f" |
        sed -E 's#^[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)="\$BATS_TEST_DIRNAME/(.*)"$#\1 \2#'
    )

    while IFS= read -r line; do
        target=$(printf '%s' "$line" | grep -oE '"\$[A-Za-z_][A-Za-z0-9_]*/[^"]*"' | tr -d '"')
        [ -n "$target" ] || continue
        name="${target#\$}"; name="${name%%/*}"
        rest="${target#*/}"
        if [ "$name" = "BATS_TEST_DIRNAME" ]; then
            printf '%s/%s\n' "$TESTS" "$rest"
        elif [ -n "${vars[$name]:-}" ]; then
            printf '%s/%s/%s\n' "$TESTS" "${vars[$name]}" "$rest"
        fi
    done < <(grep -E '(^|[^A-Za-z_])(\.|source)[[:space:]]+"\$[A-Za-z_]' "$f")
}

stubs_a_binary() {
    _stubs_a_binary_in_file "$1" && return 0
    local lib
    while IFS= read -r lib; do
        [ -f "$lib" ] || continue
        case "$lib" in "$TESTS"/*) ;; *) continue ;; esac
        _stubs_a_binary_in_file "$lib" && return 0
    done < <(_sourced_test_libs "$1")
    return 1
}

@test "every suite that stubs a binary either pairs with a real test or is a declared exemption" {
    unpaired=()
    for f in "$TESTS"/*.bats; do
        base="$(basename "$f" .bats)"
        case "$base" in *-real) continue ;; esac
        stubs_a_binary "$f" || continue
        [ -f "$TESTS/$base-real.bats" ] && continue
        exempt "$base" && continue
        unpaired+=("$base")
    done

    if [ ${#unpaired[@]} -ne 0 ]; then
        printf 'Suites stubbing a binary with no real-dependency sibling and no exemption:\n' >&2
        printf '  %s\n' "${unpaired[@]}" >&2
        printf '\nWrite tests/<name>-real.bats driving the real tool, or add the suite to\n' >&2
        printf 'exempt() with the reason a real test cannot exist. See BUG-055.\n' >&2
        return 1
    fi
}

@test "the documented exemption table matches the list the code reads" {
    # The comment table above EXEMPT_SUITES is a THIRD copy of this fact. The
    # header narrates a drift between the first two (exempt() and the stale-entries
    # loop) and fixed it by single-sourcing them to EXEMPT_SUITES — but the
    # human-readable table stayed a parallel copy with nothing comparing them.
    #
    # It is the only copy a reader consults, so a drift here does not disable the
    # guard, it misinforms the person deciding whether to add a row. Measured in
    # sync when this was written; that is exactly when to pin it.
    local documented authoritative
    documented=$(sed -n '/^#   bitacora-reconcile/,/^EXEMPT_SUITES=/p' "$TESTS/stub-real-pairing.bats" \
        | grep -oE '^#   [a-z][a-z0-9-]+' | awk '{print $2}' | sort -u)
    # Split explicitly, never by unquoted expansion: zsh does not word-split an
    # unquoted parameter, so `for x in $VAR` yields ONE field there and N in bash.
    # That is a row in this repo's prohibited-pattern table, and it fails silently
    # — an empty or single-element result reads as agreement.
    authoritative=$(printf '%s' "$EXEMPT_SUITES" | tr ' \n' '\n\n' | grep -v '^$' | sort -u)
    if [ "$documented" != "$authoritative" ]; then
        printf 'The documented table and EXEMPT_SUITES disagree:\n' >&2
        diff <(printf '%s\n' "$documented") <(printf '%s\n' "$authoritative") >&2
        return 1
    fi
}

@test "the exemption list has no stale entries" {
    # An exemption that no longer applies is worse than none: it silently grants
    # cover to a suite that has since grown a real sibling, or to one that no
    # longer exists at all.
    stale=()
    for base in $EXEMPT_SUITES; do
        [ -f "$TESTS/$base.bats" ] || { stale+=("$base (suite gone)"); continue; }
        [ -f "$TESTS/$base-real.bats" ] && stale+=("$base (now has a real sibling)")
    done

    if [ ${#stale[@]} -ne 0 ]; then
        printf 'Stale exemptions — remove these rows from exempt():\n' >&2
        printf '  %s\n' "${stale[@]}" >&2
        return 1
    fi
}

@test "the pairing rule is enforceable: precommit-fallback is paired and detected as stubbing" {
    # Guards the detector itself. If stubs_a_binary stopped matching, both tests
    # above would pass vacuously by finding nothing to check -- the failure mode
    # that makes a green suite meaningless.
    run stubs_a_binary "$TESTS/precommit-fallback.bats"
    [ "$status" -eq 0 ]
    [ -f "$TESTS/precommit-fallback-real.bats" ]
}

@test "the source-following is enforceable: vault-health-golden stubs obsidian only via its sourced lib.sh" {
    # Guards the #892 fix itself. The .bats file greps clean on its own -- the
    # stubbing lives entirely in tests/golden/vault-health/lib.sh -- so this
    # pins that stubs_a_binary only catches it by following the `. "$HERE/..."`
    # line. Without this pin, the source-following could silently regress (e.g.
    # a refactor changes the variable name) and both tests above would pass
    # vacuously again, exactly the failure #892 reported.
    run _stubs_a_binary_in_file "$TESTS/vault-health-golden.bats"
    [ "$status" -ne 0 ]

    run stubs_a_binary "$TESTS/vault-health-golden.bats"
    [ "$status" -eq 0 ]
}
