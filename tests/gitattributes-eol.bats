#!/usr/bin/env bats
# BUG-039: git-hooks/** got an explicit `text eol=lf` rule after a CRLF
# checkout disabled the GUARD-001 dispatcher on Windows (BUG-068) -- the shebang
# became "#!/usr/bin/env bash\r" and every hook died "No such file or
# directory". That fix covered one group of extensionless files. This is the
# CLASS-LEVEL guard: every tracked extensionless text file, repo-wide, must
# resolve an explicit eol, so the next one added outside an already-covered
# group (cli/internal/initrepo/templates/**, tests/golden/**, ssh/config,
# LICENSE) fails loudly here instead of depending on which OS happens to check
# it out.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

# Tracked files whose basename has no '.' -- the shape .gitattributes' *.sh/
# *.bash/*.bats extension-keyed rules cannot match by construction, and the
# reason every file in this class needs an explicit rule of its own.
_extensionless_tracked_files() {
    git -C "$REPO" ls-files | awk -F/ '{n=$NF; if (n !~ /\./) print $0}'
}

@test "every tracked extensionless file has an explicit eol (or is declared binary)" {
    cd "$REPO" || return 1
    local files
    files=$(_extensionless_tracked_files)
    # A fixture-drift guard on the guard itself: if this ever finds NOTHING,
    # the class the test protects has vanished from the repo and the test
    # would pass vacuously -- the exact failure mode this whole PR is about.
    [ -n "$files" ] || {
        printf 'No tracked extensionless files found. Either the repo layout\n' >&2
        printf 'changed (update this test) or something upstream is broken.\n' >&2
        return 1
    }

    # One process for the whole set (not one `check-attr` per file): --stdin
    # emits "<path>: <attr>: <value>" per attribute per path, two lines per
    # file for the two attributes requested here.
    local -A eol_of text_of
    local line path rest attr val
    while IFS= read -r line; do
        path="${line%%: *}"
        rest="${line#*: }"
        attr="${rest%%: *}"
        val="${rest#*: }"
        case "$attr" in
            eol) eol_of["$path"]="$val" ;;
            text) text_of["$path"]="$val" ;;
        esac
    done < <(printf '%s\n' "$files" | git check-attr --stdin text eol)

    # "unspecified" eol is only acceptable when text itself is explicitly
    # unset (`binary`) -- a deliberate escape hatch for a future extensionless
    # fixture that genuinely is binary, not a way to silence this guard for a
    # text file someone forgot to cover.
    local unresolved=() p
    for p in "${!eol_of[@]}"; do
        if [ "${eol_of[$p]}" = "unspecified" ] && [ "${text_of[$p]:-}" != "unset" ]; then
            unresolved+=("$p")
        fi
    done

    if [ ${#unresolved[@]} -ne 0 ]; then
        printf 'Tracked extensionless files with no explicit eol -- a Windows\n' >&2
        printf 'checkout CRLFs their shebang or fixture comparison (BUG-039/BUG-068):\n' >&2
        printf '%s\n' "${unresolved[@]}" | sort | while IFS= read -r f; do printf '  %s\n' "$f" >&2; done
        printf '\nAdd an explicit `text eol=lf` (or `binary`) rule in .gitattributes.\n' >&2
        return 1
    fi
}

@test "the five GUARD-001 hook dispatchers resolve eol=lf specifically" {
    # AC1: distinct from the class guard above -- this pins the exact value,
    # not just "something explicit", for the dispatchers whose CRLF breakage
    # (BUG-068) is what motivated the class guard in the first place.
    local f
    for f in pre-commit pre-push commit-msg post-checkout prepare-commit-msg; do
        run git -C "$REPO" check-attr eol -- "git-hooks/$f"
        [[ "$output" == *": eol: lf" ]]
    done
}
