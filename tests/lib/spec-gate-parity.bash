#!/usr/bin/env bash
# Hold `dotf gate spec` to scripts/check-spec-gate.sh while both exist (CI-018).
# Load after lib/both-shells and lib/dotf-bin, and call dotf_bin_resolve in setup:
#   load 'lib/both-shells'
#   load 'lib/dotf-bin'
#   load 'lib/spec-gate-parity'

SPEC_GATE_SCRIPT="$BATS_TEST_DIRNAME/../scripts/check-spec-gate.sh"

# spec_gate_both [script args...]: run the script under zsh and bash, then,
# when the arguments name both refs, `dotf gate spec` on an event built from
# the same SDD_* environment. Requires the same status and, short of exit 2
# (the two word a wiring error differently), the same output. Leaves the
# script's result in $status and $output for the caller's assertions.
#
# The dotf leg runs with no token and a PATH holding git alone, so a gh call
# cannot succeed and would show as a disagreement (AC3). The adjacency report
# is PR 3's: a run that asks for it skips the dotf leg.
# shellcheck disable=SC2154  # bats' run assigns status and output
spec_gate_both() {
    run_both_shells "$SPEC_GATE_SCRIPT" "$@" || return 1
    local sh_status="$status" sh_output="$output"
    local base="" head="" extra=""
    while [ $# -gt 0 ]; do
        case "$1" in
            --base-ref) base="$2"; shift 2 ;;
            --head-ref) head="$2"; shift 2 ;;
            --threshold) extra="$extra --threshold $2"; shift 2 ;;
            --explain) extra="$extra --explain"; shift ;;
            --adjacency-issues) return 0 ;;
            *) return 0 ;;
        esac
    done
    [ -n "$base" ] && [ -n "$head" ] || return 0

    _spec_gate_event "$BATS_TEST_TMPDIR/spec-event.json"
    local bin="$BATS_TEST_TMPDIR/git-only"
    mkdir -p "$bin"
    [ -e "$bin/git" ] || ln -s "$(command -v git)" "$bin/git"
    # $extra is flags this function built, never caller text: split on purpose.
    # shellcheck disable=SC2086
    run env -u GH_TOKEN -u GITHUB_TOKEN PATH="$bin" "$DOTF_BIN" gate spec \
        --event "$BATS_TEST_TMPDIR/spec-event.json" --base "$base" --head "$head" $extra
    local dotf_status="$status" dotf_output="$output"
    status="$sh_status" output="$sh_output"
    if [ "$dotf_status" != "$sh_status" ] ||
        { [ "$sh_status" != 2 ] && [ "$(_spec_gate_common "$dotf_output")" != "$(_spec_gate_common "$sh_output")" ]; }; then
        printf 'script (%s) and dotf gate spec (%s) disagree\n--- script\n%s\n--- dotf\n%s\n' \
            "$sh_status" "$dotf_status" "$sh_output" "$dotf_output"
        return 1
    fi
}

# The event the shared pr-checks workflow hands the gate. The script reads an
# unset SDD_PR_BODY as empty, so the event always carries a pull request.
_spec_gate_event() {
    jq -n --arg body "${SDD_PR_BODY:-}" --arg labels "${SDD_LABELS:-}" --arg author "${SDD_PR_AUTHOR:-}" \
        '{pull_request: {number: 1, body: $body,
          labels: ($labels | split(",") | map(select(. != "")) | map({name: .})),
          user: {login: $author}}}' >"$1"
}

# The output without the one paragraph that differs by design: how to re-run
# the gate by hand, which names the script on one side and not the other.
_spec_gate_common() {
    printf '%s\n' "$1" | awk '
        /If this already archived a spec/ { skip = 1 }
        skip && /^[[:space:]]*$/ { skip = 0; next }
        !skip { print }'
}
