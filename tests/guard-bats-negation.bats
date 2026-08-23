#!/usr/bin/env bats
# GUARD: no bare `! cmd` assertion in the bats suite (#1034).
#
# bash exempts a command preceded by `!` from `set -e`, and bats runs a @test
# body under `set -e`.  So `! grep -q X file` anywhere but the LAST line of a
# @test cannot fail that test — the body's status comes from its last command,
# and every earlier `!` line is discarded.  Measured 2026-08-23: 139 such lines
# in 30 files, 53 of them in that position, and one hiding a real violation for
# months (tests/check-review-attestation.bats, AC5).
#
# A `!` on the last line does work — and is one appended assertion away from not
# working, silently.  So the rule is the whole form, not the dead subset: use
# tests/lib/refute.bash (refute_grep / refute_grep_fixed), or `run cmd` plus an
# explicit status check.  All 139 are converted; this keeps the count at zero.
#
# The quarantine ratchet that carried the not-yet-converted files through the
# three-PR conversion is gone with the last entry, as designed.  If a batch of
# them ever has to come back, the shape was: a `file count` list, asserted exact
# in both directions, so a conversion had to lower its entry in the same commit
# and a regression could not hide behind an entry that over-counted.
#
# This is a bats meta-test rather than a scripts/check-*.sh gate on purpose.
# check-bats-names.sh has to be a script because bats cannot see its own
# duplicate @test names (the file is a parse error, so nothing runs).  Here
# there is no such blindness: a bats file can read its siblings, and the `test`
# job already runs the suite.

# Lines whose first non-blank token is `!` — the form bash exempts from set -e.
# `[ ! -f x ]`, `if ! grep …` and `foo || ! bar` are not this shape and are not
# matched: they are read by the shell in a context where the status is used.
bare_negation_count() {
    local file="$1" count
    count="$(grep -cE '^[[:space:]]*![[:space:]]' "$file" 2>/dev/null || true)"
    printf '%s\n' "${count:-0}"
}

@test "guard: no bats file carries a bare negated assertion" {
    local offenders=() f rel n
    for f in "$BATS_TEST_DIRNAME"/*.bats; do
        rel="tests/$(basename "$f")"
        n="$(bare_negation_count "$f")"
        [ "$n" -eq 0 ] || offenders+=("$rel ($n)")
    done

    if [ "${#offenders[@]}" -gt 0 ]; then
        printf 'bare `! cmd` assertions found:\n' >&2
        printf '  %s\n' "${offenders[@]}" >&2
        printf 'A `!` line that is not the last line of its @test cannot fail it,\n' >&2
        printf 'and one that is, stops being the last line the moment you append.\n' >&2
        printf "Use tests/lib/refute.bash (load 'lib/refute'), or run + a status check.\n" >&2
        return 1
    fi
}

@test "guard: the detector actually detects, on a fixture with a known answer" {
    # A guard that silently matches nothing reports a clean suite forever, and
    # one that overcounts cries wolf until someone deletes it. This pins the
    # detector against a file whose answer is counted by hand: three bare
    # negations, and six shapes that must NOT count — including a commented-out
    # negation and a `!` that is merely an argument, the plausible false
    # positives.
    local probe
    probe="$(mktemp)"
    {
        printf '@test "x" {\n'
        printf '    ! grep -q a b\n'
        printf '\t! [ -f x ]\n'
        printf '! true\n'
        printf '    if ! grep -q a b; then :; fi\n'
        printf '    [ ! -f x ]\n'
        printf '    refute_grep a b\n'
        printf '    printf "%%s" "!"\n'
        printf '    # ! grep -q a b\n'
        printf '    foo || ! bar\n'
        printf '}\n'
    } > "$probe"

    local n
    n="$(bare_negation_count "$probe")"
    rm -f "$probe"
    [ "$n" -eq 3 ]
}

@test "guard: every bats file that calls a refute helper loads the library" {
    # `load 'lib/refute'` missing is not a silent pass — the call errors — but it
    # errors at the first @test that reaches it, which may be far from the edit.
    # Cheaper to say so here, with the file named.
    local missing=() f
    for f in "$BATS_TEST_DIRNAME"/*.bats; do
        grep -qE '^[[:space:]]*refute_grep(_fixed)? ' "$f" || continue
        grep -qF "load 'lib/refute'" "$f" || missing+=("tests/$(basename "$f")")
    done

    if [ "${#missing[@]}" -gt 0 ]; then
        printf "these files call a refute helper without load 'lib/refute':\n" >&2
        printf '  %s\n' "${missing[@]}" >&2
        return 1
    fi
}
