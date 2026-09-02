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

# The other half of this file. The two tests above ask whether a RULE EXISTS;
# this one asks whether the WORKING TREE OBEYS IT, and nothing did before.
#
# Measured 2026-09-01: `scripts/obs-cli.ps1` resolved `eol: crlf` and was LF on
# disk in the main checkout, while both worktrees had it CRLF -- and
# `git status` reported clean in all three, because the blob round-trips
# through `text=auto` either way. git converts at CHECKOUT, never
# retroactively, so a file authored with LF, committed normalized, and never
# re-checked-out keeps the wrong endings indefinitely. Two clean checkouts of
# one commit differed byte-for-byte for three and a half months.
#
# It surfaced only as a `dotf doctor` deploy-dir drift FAIL whose printed
# remedy ("run setup to refresh") could not fix it, because setup deploys from
# the tree that is already correct. A guard that proves a rule is declared, and
# never that it is honoured, is the reason nobody saw the cause.
_declared_eol_files() {
    git -C "$REPO" ls-files | git -C "$REPO" check-attr --stdin text eol
}

@test "every tracked file whose eol is declared has a working tree that obeys it" {
    cd "$REPO" || return 1

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
    done < <(_declared_eol_files)

    local lf=() crlf=() p
    for p in "${!eol_of[@]}"; do
        # `binary` (text unset) is exempt: it is never converted either way.
        [ "${text_of[$p]:-}" = "unset" ] && continue
        [ -f "$p" ] || continue
        case "${eol_of[$p]}" in
            lf)   lf+=("$p") ;;
            crlf) crlf+=("$p") ;;
        esac
    done

    # Same fixture-drift guard as the test above: an empty set here would pass
    # vacuously, and an empty result reading as a finding is the failure class
    # this whole file exists to prevent.
    [ ${#lf[@]} -gt 0 ] && [ ${#crlf[@]} -gt 0 ] || {
        printf 'Expected both eol=lf and eol=crlf files to be tracked; found\n' >&2
        printf 'lf=%d crlf=%d. Either .gitattributes changed or this test is\n' "${#lf[@]}" "${#crlf[@]}" >&2
        printf 'no longer measuring what it claims.\n' >&2
        return 1
    }

    local bad=()
    # An eol=lf file must carry no CR at all. -U keeps grep from stripping it.
    local hit
    while IFS= read -r hit; do
        [ -n "$hit" ] && bad+=("$hit (declared lf, contains CR)")
    done < <(printf '%s\0' "${lf[@]}" | xargs -0 grep -lU $'\r' -- 2>/dev/null)

    # An eol=crlf file must carry CR. Empty files are exempt: they have no line
    # ending to be wrong about, and flagging them would be noise.
    for p in "${crlf[@]}"; do
        [ -s "$p" ] || continue
        grep -qU $'\r' -- "$p" || bad+=("$p (declared crlf, contains none)")
    done

    if [ ${#bad[@]} -ne 0 ]; then
        printf 'Working-tree line endings disagree with .gitattributes.\n' >&2
        printf 'git may well report these CLEAN: whether it notices depends on\n' >&2
        printf 'its stat cache, so an untouched file that was wrong at checkout\n' >&2
        printf 'is never re-examined. That is how scripts/obs-cli.ps1 stayed\n' >&2
        printf 'byte-wrong from 2026-05-15 to 2026-09-01. Do not conclude from a\n' >&2
        printf 'clean `git status` that this is a false positive -- compare the\n' >&2
        printf 'bytes:\n' >&2
        printf '%s\n' "${bad[@]}" | sort | while IFS= read -r f; do printf '  %s\n' "$f" >&2; done
        printf '\nFix with: git add --renormalize <path>, or remove the file and\n' >&2
        printf '`git checkout -- <path>` to re-convert it on the way out.\n' >&2
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
