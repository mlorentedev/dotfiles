#!/usr/bin/env bash
# Hold `dotf gate spec` to scripts/check-spec-gate.sh while both exist (CI-018).
# Load after lib/both-shells and lib/dotf-bin, and call dotf_bin_resolve in setup:
#   load 'lib/both-shells'
#   load 'lib/dotf-bin'
#   load 'lib/spec-gate-parity'

SPEC_GATE_SCRIPT="$BATS_TEST_DIRNAME/../scripts/check-spec-gate.sh"

# spec_gate_both [script args...]: run the script under zsh and bash and,
# when the arguments name both refs, `dotf gate spec` on an event built from
# the same SDD_* environment. Requires the same status and, short of exit 2
# (the two word a wiring error differently), the same output. Leaves the
# script's result in $status and $output for the caller's assertions.
#
# The dotf leg runs with no token and a PATH holding git alone, so a gh call
# cannot succeed and would show as a disagreement (AC3). The adjacency report
# and its step summary are compared too: dotf writes the table to its own
# --summary file, which must equal what each shell appended. --help skips the
# dotf leg; a flag the helper does not map fails unless the script rejects it.
# shellcheck disable=SC2154  # bats' run assigns status and output
spec_gate_both() {
    local base="" head="" skip=""
    local -a flags=() a=("$@")
    while [ $# -gt 0 ]; do
        case "$1" in
            --base-ref) base="$2"; shift 2 ;;
            --head-ref) head="$2"; shift 2 ;;
            --threshold) flags+=(--threshold "$2"); shift 2 ;;
            --explain) flags+=(--explain); shift ;;
            --adjacency-issues) flags+=(--adjacency "$2"); shift 2 ;;
            --help) skip=1; shift ;;
            *) skip=unknown; shift ;;
        esac
    done
    # The dotf leg runs first, so the script's run is the last `run` and its
    # status and output are what the caller asserts on.
    local dotf_status="" dotf_output=""
    # Each side writes its step summary to a file of this call's own: on a CI
    # runner GITHUB_STEP_SUMMARY is the job's real summary, shared by every test.
    local sum="${GITHUB_STEP_SUMMARY:-}" dotf_sum="$BATS_TEST_TMPDIR/dotf-summary.md" sh_sum="$BATS_TEST_TMPDIR/script-summary.md"
    : >"$dotf_sum"
    : >"$sh_sum"
    flags+=(--summary "$dotf_sum")
    if [ -z "$skip" ] && [ -n "$base" ] && [ -n "$head" ]; then
        _spec_gate_event "$BATS_TEST_TMPDIR/spec-event.json"
        local bin="$BATS_TEST_TMPDIR/git-only"
        mkdir -p "$bin"
        [ -e "$bin/git" ] || ln -s "$(command -v git)" "$bin/git"
        run env -u GH_TOKEN -u GITHUB_TOKEN PATH="$bin" "$DOTF_BIN" gate spec \
            --event "$BATS_TEST_TMPDIR/spec-event.json" --base "$base" --head "$head" \
            ${flags[@]+"${flags[@]}"}
        dotf_status="$status" dotf_output="$output"
    fi
    GITHUB_STEP_SUMMARY="$sh_sum" run_both_shells "$SPEC_GATE_SCRIPT" ${a[@]+"${a[@]}"} || return 1
    # A caller that named a summary file reads the script's table from it.
    [ -z "$sum" ] || cat "$sh_sum" >>"$sum"
    # A flag this helper does not know is only safe to skip when the script
    # rejects it too; one it accepts means the dotf leg is missing a case.
    if [ "$skip" = unknown ] && [ "$status" != 2 ]; then
        printf 'spec_gate_both: the script accepted a flag this helper does not map: %s\n' "${a[*]}"
        return 1
    fi
    if [ -z "$skip" ] && [ "$status" != 2 ] && { [ -n "$base" ] || [ -n "$head" ]; } &&
        { [ -z "$base" ] || [ -z "$head" ]; }; then
        printf 'spec_gate_both: the script ran with one ref; give both or neither\n'
        return 1
    fi
    [ -n "$dotf_status" ] || return 0
    if ! { cat "$dotf_sum" "$dotf_sum" | cmp -s - "$sh_sum"; }; then
        printf 'script and dotf gate spec wrote different step summaries\n--- script (both shells)\n%s\n--- dotf\n%s\n' \
            "$(cat "$sh_sum")" "$(cat "$dotf_sum")"
        return 1
    fi
    if [ "$dotf_status" != "$status" ] ||
        { [ "$status" != 2 ] && [ "$(_spec_gate_common "$dotf_output")" != "$(_spec_gate_common "$output")" ]; }; then
        printf 'script (%s) and dotf gate spec (%s) disagree\n--- script\n%s\n--- dotf\n%s\n' \
            "$status" "$dotf_status" "$output" "$dotf_output"
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
