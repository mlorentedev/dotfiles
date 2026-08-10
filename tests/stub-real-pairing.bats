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
#   bitacora-reconcile        stubs `gh` — a real run mutates the live GitHub project board
#   bitacora-rollout          stubs `gh` — same; a real run adds items to the live board. This
#                             suite is a worked example of the limitation BUG-055 names: #884 was
#                             a call the real `gh` rejects and the stub accepts, so no stub could
#                             have caught it. What the suite CAN pin is which call is made, and it
#                             does; the API's verdict on that call only arrives from a real run.
#   board-pickup              stubs `gh` — same; a real run self-assigns real issues
#   guard-memory-sink         stubs `git` — the real path is covered end-to-end by the dispatcher's own commit-time behaviour
#   hermes-setup              stubs remote installers — a real run provisions an agent host
#   install-dotf              stubs the release download — a real run fetches from GitHub releases
#   shell-profile             stubs `zsh`/`bash` timing probes — a real run measures this machine, not a fixture
#   skills-pipeline           stubs the deploy targets — a real run writes into the caller's own $HOME
#   vault-health              stubs `hive` — a real run needs the daemon and a live vault
#   vault-maintenance-weekly  stubs `cron`/`hive` — a real run installs a crontab entry
exempt() {
    case "$1" in
        bitacora-reconcile|bitacora-rollout|board-pickup|guard-memory-sink|hermes-setup|install-dotf|\
        shell-profile|skills-pipeline|vault-health|vault-maintenance-weekly) return 0 ;;
        *) return 1 ;;
    esac
}

# A suite "stubs a binary" when it makes something executable and puts its
# directory on PATH — the shape that shadows a real tool for the suite's own
# process. Deliberately structural: it matches intent, not a naming convention.
stubs_a_binary() {
    grep -q 'chmod +x' "$1" && grep -qE 'PATH="?\$' "$1"
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

@test "the exemption list has no stale entries" {
    # An exemption that no longer applies is worse than none: it silently grants
    # cover to a suite that has since grown a real sibling, or to one that no
    # longer exists at all.
    stale=()
    for base in bitacora-reconcile board-pickup guard-memory-sink hermes-setup install-dotf \
                shell-profile skills-pipeline vault-health vault-maintenance-weekly; do
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
