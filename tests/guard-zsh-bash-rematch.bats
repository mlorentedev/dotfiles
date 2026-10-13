#!/usr/bin/env bats
# GUARD: a dual-shell script that reads BASH_REMATCH turns it on under zsh.
#
# zsh fills BASH_REMATCH from `[[ =~ ]]` only under `setopt BASH_REMATCH`, and
# indexes it from 1 unless KSH_ARRAYS is set. Without the shim the read is
# empty, or under `set -u` an abort, and `zsh -n` and shellcheck both pass it.
# check-knowledge-gate.sh carried the shim; check-spec-gate.sh did not, and its
# code-span stripping aborted under zsh unnoticed because every suite ran it
# under its bash shebang.
#
# Scope: the files .claude/CLAUDE.md requires to run under bash AND zsh, every
# tracked *.sh and *.bash. specs/archive/ is excluded: frozen records.

# A line that names BASH_REMATCH outside a whole-line comment.
READS='^[[:space:]]*[^#[:space:]].*BASH_REMATCH'

# missing_shim FILE: succeed when FILE reads BASH_REMATCH and nothing in it
# sets the zsh option. A grep error returns 2, so it is never read as "fine".
missing_shim() {
    local file="$1" rc=0
    grep -qE "$READS" "$file" || rc=$?
    [ "$rc" -le 1 ] || return 2
    [ "$rc" -eq 0 ] || return 1
    rc=0
    grep -qE '^[[:space:]]*setopt[[:space:]].*BASH_REMATCH' "$file" || rc=$?
    [ "$rc" -le 1 ] || return 2
    [ "$rc" -eq 1 ]
}

@test "guard: every dual-shell script that reads BASH_REMATCH sets it under zsh" {
    local offenders=() f readers=0 rc
    while IFS= read -r f; do
        [ -e "$BATS_TEST_DIRNAME/../$f" ] || continue
        case "$f" in specs/archive/*) continue ;; esac
        grep -qE "$READS" "$BATS_TEST_DIRNAME/../$f" && readers=$((readers + 1))
        rc=0
        missing_shim "$BATS_TEST_DIRNAME/../$f" || rc=$?
        [ "$rc" -ne 2 ] || { echo "cannot read $f" >&2; return 1; }
        [ "$rc" -eq 1 ] || offenders+=("$f")
    done < <(git -C "$BATS_TEST_DIRNAME/.." ls-files -- '*.sh' '*.bash')

    # A population floor: two scripts read BASH_REMATCH today. Zero would mean
    # the file list came back empty, not that the repo is clean.
    [ "$readers" -ge 2 ] || { echo "only $readers readers found" >&2; return 1; }
    if [ "${#offenders[@]}" -gt 0 ]; then
        printf 'reads BASH_REMATCH without `setopt KSH_ARRAYS BASH_REMATCH` under zsh: %s\n' \
            "${offenders[@]}" >&2
        return 1
    fi
}

@test "guard: the predicate flags a reader without the shim and passes one with it" {
    printf '%s\n' '[[ x =~ (x) ]] && echo "${BASH_REMATCH[1]}"' > "$BATS_TEST_TMPDIR/bare.sh"
    printf '%s\n' 'if [ -n "${ZSH_VERSION:-}" ]; then' '    setopt KSH_ARRAYS BASH_REMATCH' 'fi' \
        '[[ x =~ (x) ]] && echo "${BASH_REMATCH[1]}"' > "$BATS_TEST_TMPDIR/shimmed.sh"
    printf '%s\n' '# BASH_REMATCH is only named here' 'echo plain' > "$BATS_TEST_TMPDIR/plain.sh"

    missing_shim "$BATS_TEST_TMPDIR/bare.sh"
    run missing_shim "$BATS_TEST_TMPDIR/shimmed.sh"
    [ "$status" -eq 1 ]
    run missing_shim "$BATS_TEST_TMPDIR/plain.sh"
    [ "$status" -eq 1 ]
    run missing_shim "$BATS_TEST_TMPDIR/absent.sh"
    [ "$status" -eq 2 ]
}
