#!/usr/bin/env bats
# GUARD: no bare `[[ ... ]]` assertion in the bats suite (#2164).
#
# Before bash 4.1, a failing `[[ ]]` does not trip `set -e`, and bats runs a
# @test body under `set -e`. So on bash 3.2, `[[ $output == *x* ]]` anywhere but
# the LAST line of a @test cannot fail that test. macOS `/bin/bash` is 3.2, and
# the test-macos job runs the os-sensitive tier under it on purpose
# (`run-bats.sh --expect-bash 3`), so that leg was where the false greens lived.
# Measured 2026-10-07: 668 bare lines suite-wide, 54 of them mid-test in the
# tier. Linux and Git Bash run bash 5, where every one was sound.
#
# The rule is the whole form, not the dead subset, for the reason
# guard-bats-negation.bats gives for `!`: a bare `[[ ]]` on the last line works
# and stops working the moment someone appends an assertion below it. Write
# `[[ ... ]] || false` (the form the bats documentation gives for bash 3.2), or
# chain your own `|| { echo ...; false; }`.
#
# Only a line whose first token is `[[` is this shape. `if [[ ... ]]`, `while`,
# and a line that continues with `\` onto an `||` are read where the status is
# used, and are not matched.
#
# `[[ ... ]] && cmd` is worse: no bash fails a test on it, because a failing
# command before `&&` never trips `set -e`. It counts too, unless what follows is
# loop or test control (`continue`, `break`, `return`, `skip`), where the `[[ ]]`
# is a condition rather than an assertion.

# Lines that start with `[[` and end at its `]]`, optionally followed by a
# comment, or that chain it with `&&` onto anything but control flow. A read
# that errors is a failure, never a count of 0.
bare_dbracket_count() {
    awk '
        /^[[:space:]]*\[\[[[:space:]].*\]\][[:space:]]*(#.*)?$/ { n++; next }
        /^[[:space:]]*\[\[[[:space:]].*\]\][[:space:]]*&&/ &&
            !/\]\][[:space:]]*&&[[:space:]]*(continue|break|return|skip)([[:space:];]|$)/ { n++ }
        END { print n + 0 }
    ' "$1" || { echo "awk could not read $1" >&2; return 2; }
}

# The suite and the helpers it sources: a helper runs under the calling test's
# set -e, so a bare [[ ]] there is vacuous in the same way.
suite_files() {
    local f
    for f in "$BATS_TEST_DIRNAME"/*.bats "$BATS_TEST_DIRNAME"/*.bash "$BATS_TEST_DIRNAME"/lib/*.bash; do
        [ -e "$f" ] && printf '%s\n' "$f"
    done
}

@test "guard: no bats file carries a bare [[ ]] assertion" {
    local offenders=() f n files
    files="$(suite_files)"
    [ -n "$files" ] || { echo "no suite files found under $BATS_TEST_DIRNAME" >&2; return 1; }
    while IFS= read -r f; do
        [ -r "$f" ] || { echo "cannot read $f" >&2; return 1; }
        n="$(bare_dbracket_count "$f")" || return 1
        [ "$n" -eq 0 ] || offenders+=("${f#"$BATS_TEST_DIRNAME"/} ($n)")
    done <<EOF_FILES
$files
EOF_FILES

    if [ "${#offenders[@]}" -gt 0 ]; then
        printf 'bare `[[ ... ]]` assertions found:\n' >&2
        printf '  %s\n' "${offenders[@]}" >&2
        printf 'On bash 3.2 (macOS /bin/bash) one that is not the last line of its\n' >&2
        printf '@test cannot fail it. Append `|| false`.\n' >&2
        return 1
    fi
}

@test "guard: the detector actually detects, on a fixture with a known answer" {
    # Four bare assertions (one an && chain onto another assertion), and nine
    # shapes that must not count: the || forms, && onto control flow, the
    # conditional contexts, a comment and the single-bracket test.
    local probe n
    probe="$(mktemp)"
    {
        printf '@test "x" {\n'
        printf '    [[ a == b ]]\n'
        printf '\t[[ -n "$x" ]]  # why\n'
        printf '[[ "$output" == *"]]"* ]]\n'
        printf '    [[ a == b ]] || false\n'
        printf '    [[ a == b ]] || { echo "$output"; false; }\n'
        printf '    [[ a == b ]] && continue\n'
        printf '    [[ a == b ]] && return 0\n'
        printf '    [[ a == b ]] && skip "why"\n'
        printf '    [[ "$output" == *x* ]] && grep -q y f\n'
        printf '    if [[ a == b ]]; then :; fi\n'
        printf '    [[ a == b ]] \\\n'
        printf '        || false\n'
        printf '    # [[ a == b ]]\n'
        printf '    [ a = b ]\n'
        printf '}\n'
    } > "$probe"
    n="$(bare_dbracket_count "$probe")"
    rm -f "$probe"
    [ "$n" -eq 4 ]
}

@test "guard: a read error is reported, never counted as clean" {
    run bare_dbracket_count "$BATS_TEST_DIRNAME/does-not-exist.bats"
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not read"* ]] || false
}

# bats test_tags=os-sensitive
@test "guard: the || false form fails a test mid-body on the bash running bats" {
    # The premise, measured rather than asserted: on the test-macos leg this runs
    # under /bin/bash 3.2, where the bare form passes and this form must not.
    local dir
    dir="$(mktemp -d)"
    {
        printf '@test "bare" {\n    [[ a == b ]]\n    true\n}\n'
        printf '@test "guarded" {\n    [[ a == b ]] || false\n    true\n}\n'
    } > "$dir/probe.bats"
    run bats --tap "$dir/probe.bats"
    rm -rf "$dir"
    [[ "$output" == *"not ok 2 guarded"* ]] || false
    # From bash 4.1 the bare form fails too; before it, it is the false green.
    if [ $((BASH_VERSINFO[0] * 100 + BASH_VERSINFO[1])) -lt 401 ]; then
        [[ "$output" == *"ok 1 bare"* && "$output" != *"not ok 1 bare"* ]] || false
    fi
}
