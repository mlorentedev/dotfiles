#!/usr/bin/env bats
# GUARD: no dual-shell script binds a zsh-tied variable name (#1530).
#
# In zsh, `path` IS `$PATH` as an array, and `status`, `cdpath`, `manpath` and
# `fignore` are tied the same way. Assigning one is not shadowing: `local
# path=x` or `read -r path` replaces the command search path inside that
# function, so every later external command is "command not found". A script
# that reads a failed command as "no match" then reports a clean result.
# `zsh -n` and shellcheck both pass it, because it is not a syntax error and
# not a bash defect. Measured 2026-09-05: a mutation harness reported all 8 of
# its mutations as ANCHOR-MISS when the cause was a wiped PATH.
#
# Scope: the files .claude/CLAUDE.md requires to run under bash AND zsh, which
# are every tracked *.sh and *.bash, plus *.bats. Excluded:
# - zsh-native files (.zshrc, *.zsh). There, `path=(...)` is the idiom.
# - specs/archive/. Its scripts are frozen records.
# - `status` in *.bats. bats owns it: `run` sets it, and bats runs only under
#   bash.
# - this file, whose fixture spells out every binding shape.

TIED='path|status|cdpath|manpath|fignore'

# tied_bindings FILE NAMES: print FILE's non-comment lines that bind one of
# NAMES (an ERE alternation). Four shapes: an assignment (`n=`, `n+=`,
# `n[i]=`), a `read` target, a `for` variable, and a bare declaration
# (`local n`). A grep error (exit 2: unreadable file, bad pattern) returns 2,
# so a tool failure is never read as "no bindings".
tied_bindings() {
    local file="$1" names="$2" sep='(^|[[:space:];(&|])' end='([[:space:];]|$)' hits rc=0
    hits="$(grep -nE \
        -e "${sep}(${names})(\\[[^]]*\\])?\\+?=" \
        -e "${sep}read([[:space:]]+[^[:space:]&;|]+)*[[:space:]]+(${names})${end}" \
        -e "${sep}for[[:space:]]+(${names})[[:space:]]+in${end}" \
        -e "${sep}(local|declare|typeset|readonly|export)([[:space:]]+[^[:space:]&;|]+)*[[:space:]]+(${names})${end}" \
        -- "$file")" || rc=$?
    [ "$rc" -le 1 ] || return 2
    [ -n "$hits" ] || return 0
    printf '%s\n' "$hits" | grep -vE '^[0-9]+:[[:space:]]*#' || true
}

@test "guard: no dual-shell script binds a zsh-tied variable name" {
    local offenders=() f names hits seen=0
    while IFS= read -r f; do
        # Deleted in the working tree but still in the index: nothing to bind.
        [ -e "$BATS_TEST_DIRNAME/../$f" ] || continue
        case "$f" in
            specs/archive/*) continue ;;
            tests/guard-zsh-tied-names.bats) continue ;;  # its fixture is every shape
            *.bats) names='path|cdpath|manpath|fignore' ;;
            *) names="$TIED" ;;
        esac
        seen=$((seen + 1))
        hits="$(tied_bindings "$BATS_TEST_DIRNAME/../$f" "$names")" || {
            offenders+=("$f: grep could not read it")
            continue
        }
        [ -z "$hits" ] || offenders+=("$f: ${hits//$'\n'/; }")
    done < <(git -C "$BATS_TEST_DIRNAME/.." ls-files -- '*.sh' '*.bash' '*.bats')

    # An empty file list is a broken environment (no git, not a checkout),
    # not a clean tree.
    if [ "$seen" -eq 0 ]; then
        printf 'no tracked shell file was scanned: is this a git checkout?\n' >&2
        return 1
    fi

    if [ "${#offenders[@]}" -gt 0 ]; then
        printf 'zsh-tied variable names bound in a dual-shell file:\n' >&2
        printf '  %s\n' "${offenders[@]}" >&2
        printf 'Under zsh this rewrites $PATH (or $?) inside the function. Rename it:\n' >&2
        printf 'file_path, rc, ... any name zsh does not tie.\n' >&2
        return 1
    fi
}

@test "guard: the detector actually detects, on a fixture with a known answer" {
    # A guard that silently matches nothing reports a clean tree forever. Six
    # bindings must count; six look-alikes must not, including a comment, a
    # flag, a URL query, an expansion and a longer name.
    local probe n
    probe="$BATS_TEST_TMPDIR/probe.sh"
    {
        printf 'local path="$1"\n'
        printf 'while IFS=$%s read -r added removed path; do :; done\n' "'\\t'"
        printf 'for status in a b; do :; done\n'
        printf '    local num path base\n'
        printf 'x=1; cdpath+=foo\n'
        printf 'manpath[1]=x\n'
        printf '# local path=x\n'
        printf 'gh api --path=x\n'
        printf 'curl "https://h/x?path=1"\n'
        printf 'echo "$path ${status}"\n'
        printf 'local file_path="$1" paths=x\n'
        printf 'read -r xpath\n'
    } > "$probe"

    n="$(tied_bindings "$probe" "$TIED" | wc -l | tr -d ' ')"
    if [ "$n" -ne 6 ]; then
        tied_bindings "$probe" "$TIED" >&2
        printf 'expected 6 bindings, matched %s\n' "$n" >&2
        return 1
    fi

    # A file grep cannot read is an error, never "no bindings".
    run tied_bindings "$BATS_TEST_TMPDIR/absent.sh" "$TIED"
    [ "$status" -eq 2 ]
}
