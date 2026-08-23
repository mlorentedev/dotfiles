#!/usr/bin/env bash
# Shared negative assertions for the bats suite.  Load with: load 'lib/refute'
#
# WHY THIS EXISTS
# ---------------
# bash exempts any command preceded by `!` from `set -e`, and bats runs a @test
# body under `set -e`.  So `! grep -q X file` in the MIDDLE of a @test cannot
# fail that test: the body's exit status is taken from its last command, and
# every earlier `!` line is discarded.  A bare `false` in the same position IS
# caught — the exemption is specific to the `!` form, which is why it survives
# casual inspection.
#
# That is not hypothetical.  tests/check-review-attestation.bats asserted that
# scripts/check-review-attestation.sh names no reviewer; the script DID name one
# (in a comment), and the suite stayed green for as long as the violated
# assertion was not the last line of its @test.
#
# Two hardenings over a plain `! grep`, both of which turn a vacuous pass into a
# loud failure:
#
#   * a grep ERROR is not an absence.  An unreadable file, or a pattern that is
#     invalid under the chosen dialect (`(` is a group in ERE and a literal in
#     BRE, so a BRE pattern moved to -E can start erroring), exits >=2.  `! grep`
#     reads that as "not found" and passes.  Here it fails and says so.
#   * the pattern is passed after `--`, so a pattern that begins with `-`
#     (`--vault knowledge`) is a pattern and not a bundle of grep options.
#
# The dialect is explicit in the function name: refute_grep is an extended
# regex, refute_grep_fixed is a literal string.  Do not "simplify" them into one
# entry point with a flags argument — the call sites are the place where the
# author has to state which one they mean.

# refute_grep <ere-pattern> <file>
# Fails if <file> matches the extended regular expression <ere-pattern>.
refute_grep() {
    _refute_grep_impl E "$@"
}

# refute_grep_fixed <literal> <file>
# Fails if <file> contains the literal string <literal>.
refute_grep_fixed() {
    _refute_grep_impl F "$@"
}

_refute_grep_impl() {
    local dialect="$1" pattern="$2" file="$3"

    if [ "$#" -ne 3 ]; then
        printf 'refute_grep: expected <pattern> <file>, got %d argument(s)\n' \
            "$(($# - 1))" >&2
        return 1
    fi

    local status=0
    grep "-q${dialect}" -- "$pattern" "$file" >/dev/null 2>&1 || status=$?

    case "$status" in
        0)
            printf 'expected NOT to find /%s/ in %s, but it is there:\n' \
                "$pattern" "$file" >&2
            grep "-n${dialect}" -- "$pattern" "$file" >&2
            return 1
            ;;
        1)
            return 0
            ;;
        *)
            printf 'refute_grep: grep exited %s on %s for /%s/ — an error is not an absence\n' \
                "$status" "$file" "$pattern" >&2
            return 1
            ;;
    esac
}
