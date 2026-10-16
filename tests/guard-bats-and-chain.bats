#!/usr/bin/env bats
# GUARD: no `[ a ] && [ b ]` assertion chain in the bats suite (#2199).
#
# bats runs a @test body under `set -e`, and `set -e` never fires on a failing
# command that is not the last one of an `&&` list. So `[ 1 = 2 ] && [ 1 = 1 ]`
# on any line but a test's last skips the rest of the chain and the test goes
# on, green. Measured with bats 1.x on msi, 2026-10-08; 22 such lines were in
# the suite when this guard landed. It holds on every bash, unlike the bare
# `[[ ]]` of guard-bats-dbracket.bats, which is vacuous only before bash 4.1.
#
# The rule is the whole form, not the dead subset, for the reason that guard
# gives: a chain on a test's last line works, and stops working the moment
# someone appends an assertion below it. Write one assertion per line.
#
# Only a line made of nothing but tests joined by `&&` is this shape. A chain
# continued with `\` onto an `||`, one that ends in `|| false` or `|| { …; }`,
# and `[ x ] && cmd` (a condition, not an assertion) are not matched.

# Lines that are a chain of `[ ]` / `[[ ]]` tests joined by `&&`, optionally
# followed by a comment. A read that errors is a failure, never a count of 0.
and_chain_count() {
    awk '
        /^[[:space:]]*\[\[?[[:space:]].*\]\]?[[:space:]]*&&[[:space:]]*\[\[?[[:space:]].*\]\]?[[:space:]]*(#.*)?$/ { n++ }
        END { print n + 0 }
    ' "$1" || { echo "awk could not read $1" >&2; return 2; }
}

# The suite and the helpers it sources: a helper runs under the calling test's
# set -e, so a chain there is vacuous in the same way.
suite_files() {
    local f
    for f in "$BATS_TEST_DIRNAME"/*.bats "$BATS_TEST_DIRNAME"/*.bash "$BATS_TEST_DIRNAME"/lib/*.bash; do
        [ -e "$f" ] || continue
        printf '%s\n' "$f"
    done
}

@test "guard: no bats file carries an [ ] && [ ] assertion chain" {
    local offenders=() f n files
    files="$(suite_files)"
    [ -n "$files" ] || { echo "no suite files found under $BATS_TEST_DIRNAME" >&2; return 1; }
    while IFS= read -r f; do
        [ -r "$f" ] || { echo "cannot read $f" >&2; return 1; }
        n="$(and_chain_count "$f")" || return 1
        [ "$n" -eq 0 ] || offenders+=("${f#"$BATS_TEST_DIRNAME"/} ($n)")
    done <<EOF_FILES
$files
EOF_FILES

    if [ "${#offenders[@]}" -gt 0 ]; then
        printf '`[ a ] && [ b ]` assertion chains found:\n' >&2
        printf '  %s\n' "${offenders[@]}" >&2
        printf 'Anywhere but the last line of a @test, a false `a` skips `b` and\n' >&2
        printf 'the test passes. Write one assertion per line.\n' >&2
        return 1
    fi
}

@test "guard: the detector actually detects, on a fixture with a known answer" {
    # Four chains, and seven shapes that must not count: the || forms, a
    # continued line, && onto a command, a comment and a single test.
    local probe n
    probe="$(mktemp)"
    {
        printf '@test "x" {\n'
        printf '    [ a = b ] && [ c = d ]\n'
        printf '\t[ -n "$x" ] && [ -d "$x" ] && [ -x "$x" ]  # why\n'
        printf '    [ "$status" -eq 0 ] && [[ "$output" == *ok* ]]\n'
        printf '[ a = b ]&&[ c = d ]\n'
        printf '    [ a = b ] && [ c = d ] || false\n'
        printf '    [ a = b ] && [ c = d ] || { echo "$output"; false; }\n'
        printf '    [ a = b ] && [ c = d ] \\\n'
        printf '        || skip "why"\n'
        printf '    [ -n "$x" ] && export Y="$x"\n'
        printf '    # [ a = b ] && [ c = d ]\n'
        printf '    [ a = b ]\n'
        printf '}\n'
    } > "$probe"
    n="$(and_chain_count "$probe")"
    rm -f "$probe"
    [ "$n" -eq 4 ]
}

@test "guard: a read error is reported, never counted as clean" {
    run and_chain_count "$BATS_TEST_DIRNAME/does-not-exist.bats"
    [ "$status" -eq 2 ]
    [[ "$output" == *"could not read"* ]] || false
}

@test "guard: a mid-body chain really is a false green on the bash running bats" {
    # The premise, measured rather than asserted.
    local dir
    dir="$(mktemp -d)"
    printf '@test "chain" {\n    [ 1 = 2 ] && [ 1 = 1 ]\n    true\n}\n' > "$dir/probe.bats"
    run bats --tap "$dir/probe.bats"
    rm -rf "$dir"
    [[ "$output" == *"ok 1 chain"* ]] || false
    [[ "$output" != *"not ok 1 chain"* ]] || false
}
