#!/usr/bin/env bats
# GUARD: no bare `! cmd` assertion in the bats suite (#1034).
#
# bash exempts a command preceded by `!` from `set -e`, and bats runs a @test
# body under `set -e`.  So `! grep -q X file` anywhere but the LAST line of a
# @test cannot fail that test — the body's status comes from its last command,
# and every earlier `!` line is discarded.  Measured: 55 of the suite's 139 such
# lines were in that position, and one of them was hiding a real violation for
# months (tests/check-review-attestation.bats, AC5).
#
# A `!` on the last line does work — and is one appended assertion away from not
# working, silently.  So the rule this guard enforces is the whole form, not the
# dead subset: use tests/lib/refute.bash (refute_grep / refute_grep_fixed), or
# `run cmd` plus an explicit status check.
#
# This is a bats meta-test rather than a scripts/check-*.sh gate on purpose.
# check-bats-names.sh has to be a script because bats cannot see its own
# duplicate @test names (the file is a parse error, so nothing runs).  Here
# there is no such blindness: a bats file can read its siblings, and the `test`
# job already runs the suite.

# --- the quarantine list -----------------------------------------------------
#
# Files still carrying the bare form, with the exact count each carries today.
# It is a RATCHET, not a home: the guard fails if a listed file's count changes
# in EITHER direction, so a conversion must delete or lower its entry in the
# same commit, and a regression cannot hide behind an entry that over-counts.
# #1034 empties it; when the last entry goes, so does the list.
quarantine_list() {
    cat <<'LIST'
tests/agents-md.bats 2
tests/aliases.bats 10
tests/antigravity.bats 4
tests/bitacora-reconcile.bats 1
tests/bitacora-rollout.bats 1
tests/board-pickup.bats 1
tests/check-doc-paths.bats 1
tests/compile-harness-real.bats 2
tests/docs-drift.bats 2
tests/hermes-setup.bats 1
tests/hive-upgrade-timer.bats 4
tests/install-dotf-ps1.bats 1
tests/knowledge-crystallize-ps1.bats 2
tests/nan-scripts-secrets.bats 2
tests/opencode.bats 23
tests/pi-config.bats 4
tests/pwsh-analyzer-coverage.bats 1
tests/shell-wrapper-dedup.bats 2
tests/skills-pipeline.bats 4
tests/utils.bats 6
tests/vault-health-golden.bats 1
tests/verify-setup.bats 2
tests/versions-conf.bats 3
tests/versions-no-hardcode.bats 2
tests/windows-defaults.bats 2
LIST
}

# Lines whose first non-blank token is `!` — the form bash exempts from set -e.
# `[ ! -f x ]`, `if ! grep …` and `foo || ! bar` are not this shape and are not
# matched: they are read by the shell in a context where the status is used.
bare_negation_count() {
    local file="$1" count
    count="$(grep -cE '^[[:space:]]*![[:space:]]' "$file" 2>/dev/null || true)"
    printf '%s\n' "${count:-0}"
}

quarantined_count() {
    local rel="$1"
    quarantine_list | awk -v f="$rel" '$1 == f { print $2; found = 1 }
                                       END { if (!found) print "-" }'
}

@test "guard: no bats file outside the quarantine list carries a bare negated assertion" {
    local offenders=() f rel n
    for f in "$BATS_TEST_DIRNAME"/*.bats; do
        rel="tests/$(basename "$f")"
        n="$(bare_negation_count "$f")"
        [ "$n" -eq 0 ] && continue
        [ "$(quarantined_count "$rel")" = '-' ] && offenders+=("$rel ($n)")
    done

    if [ "${#offenders[@]}" -gt 0 ]; then
        printf 'bare `! cmd` assertions found in files that had none:\n' >&2
        printf '  %s\n' "${offenders[@]}" >&2
        printf 'A `!` line that is not the last line of its @test cannot fail it.\n' >&2
        printf "Use tests/lib/refute.bash (load 'lib/refute'), or run + a status check.\n" >&2
        return 1
    fi
}

@test "guard: every quarantined file still carries exactly its recorded count" {
    local drift=() rel want got
    while read -r rel want; do
        [ -n "$rel" ] || continue
        if [ ! -f "$BATS_TEST_DIRNAME/../$rel" ]; then
            drift+=("$rel: listed but does not exist")
            continue
        fi
        got="$(bare_negation_count "$BATS_TEST_DIRNAME/../$rel")"
        [ "$got" = "$want" ] || drift+=("$rel: recorded $want, found $got")
    done < <(quarantine_list)

    if [ "${#drift[@]}" -gt 0 ]; then
        printf 'the quarantine list has drifted from the suite:\n' >&2
        printf '  %s\n' "${drift[@]}" >&2
        printf 'Converting a file means deleting or lowering its entry in the same commit.\n' >&2
        return 1
    fi
}

@test "guard: the detector actually detects, on a fixture with a known answer" {
    # A guard that silently matches nothing reports a clean suite forever, and
    # one that overcounts turns the ratchet into noise. This pins the detector
    # against a file whose answer is counted by hand: three bare negations, and
    # six shapes that must NOT count — including a commented-out negation and a
    # `!` that is merely an argument, which are the plausible false positives.
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

@test "guard: the quarantine list is sorted and free of duplicate entries" {
    # Two entries for one file would let a conversion satisfy the stale one and
    # leave the other unnoticed; sorting keeps the merge conflicts honest when
    # parallel branches each convert a file.
    local names
    names="$(quarantine_list | awk '{ print $1 }')"
    [ "$names" = "$(printf '%s\n' "$names" | LC_ALL=C sort)" ]
    [ "$(printf '%s\n' "$names" | wc -l)" -eq "$(printf '%s\n' "$names" | sort -u | wc -l)" ]
}
