#!/usr/bin/env bash

# check-doc-paths.sh: fail when an instruction file names a repo path that does
# not exist.
#
# Agent instruction files (.claude/CLAUDE.md, AGENTS.md, …) are read as fact.
# When a script is retired into `dotf` or a config is replaced by a new model,
# the file that told agents to run it keeps saying so and nothing complains:
# the repo checks content sync, markdown corruption and harness drift, but
# nothing checks that a path named in prose still resolves.
#
# Not hypothetical (#916): `.claude/CLAUDE.md` was committed by #907 while
# naming seven files that no longer existed, and two sessions in two days
# followed one of them — `./scripts/healthcheck.sh`, long retired into
# `dotf doctor`.
#
# ## What counts as a repo path
#
# Deliberately conservative, because a guard with false positives gets
# bypassed and then protects nothing. A backticked token is checked only when:
#
#   - it contains no whitespace, shell metacharacters or <placeholder> markers;
#   - it is repo-relative (no leading /, ~, $ or -) and not a URL;
#   - it contains no `..` component, so a token cannot resolve outside the repo;
#   - AND it contains a slash whose first segment is a real top-level entry of
#     this repo (computed from disk, so the rule needs no maintenance).
#
# Bare filenames are NOT checked. Resolving them by basename was tried and
# reverted: it flagged vault patterns, `machine.json` and `review.md` on
# AGENTS.md — files that legitimately live elsewhere. The cost is a real blind
# spot (a backticked `missing.md` passes), accepted deliberately.
#
# Everything else — model ids like `opencode-go/qwen3.6-plus`, command
# fragments, vault paths, deployed ~/ locations — is ignored by construction
# rather than by an exclusion list that would rot the same way the docs did.
#
# ## Convention this implies
#
# A backticked repo path is a LIVE claim. To name a path that no longer exists
# — "the old loader lived at scripts/load-secrets.sh" — write it in plain text.
# The same holds in the other direction: a path that does not exist YET
# — "Ollama docs will live at ai/ollama/ when wired" — is also not a live
# claim, so it also goes in plain text, not backticks (round 5 found this gap
# point-fixed on one file with no general rule written down).
# That keeps the invariant simple enough to state in one line, and needs no
# per-file exception list.
#
# It also means this guard is for INSTRUCTION files (things agents act on), not
# for historical ones. docs/lessons.md legitimately names retired scripts in its
# incident write-ups; running this against it would report a dozen "failures"
# that are all correct as history. Keep the target list to files that tell
# someone what to do.
#
# Usage:
#   check-doc-paths.sh <file>...
#
# Exit:
#   0  every referenced path resolves
#   1  one or more referenced paths are missing
#   2  usage error / unreadable file

set -euo pipefail

if [ "$#" -eq 0 ]; then
    cat >&2 <<'EOF'
Usage: check-doc-paths.sh <file>...
  Verifies that every repo-relative path named in backticks inside <file>
  exists on disk. Paths resolve against the repo root (the parent of the
  directory holding this script).
EOF
    exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

KNOWN_EXT='\.(sh|ps1|zsh|bats|conf|md|json|jsonc|ya?ml|toml)$'

# Top-level entries of this repo, one per line. Computed rather than listed:
# a new top-level directory is covered the day it is created.
TOP_LEVEL="$(ls -A "$REPO_ROOT")"

# True when the token's first path segment is a real top-level repo entry.
is_repo_rooted() {
    _first="${1%%/*}"
    # .git is a top-level entry but its internals are not repo content — docs
    # mention `.git/info/exclude` as a concept, not as a tracked file.
    [ "$_first" = ".git" ] && return 1
    printf '%s\n' "$TOP_LEVEL" | grep -qxF "$_first"
}

VAULT_ROOT="${VAULT_PATH:-}"
if [ -z "$VAULT_ROOT" ] && command -v dotf >/dev/null 2>&1; then
    VAULT_ROOT="$(dotf env path VAULT_PATH 2>/dev/null || true)"
fi

missing_total=0

for target in "$@"; do
    if [ ! -f "$target" ]; then
        printf 'check-doc-paths: not a file: %s\n' "$target" >&2
        exit 2
    fi

    missing_in_file=0
    # shellcheck disable=SC2016  # the backtick pattern is literal by design
    tokens="$(grep -o '`[^`[:space:]]*`' "$target" | tr -d '\140' | sort -u || true)"

    while IFS= read -r token; do
        [ -n "$token" ] || continue

        # Vault paths ($VAULT_PATH/...) are checked when VAULT_ROOT resolves.
        case "$token" in
            '$VAULT_PATH/'*|'${VAULT_PATH}/'*)
                case "$token" in
                    *'<'*|*'>'*|*'&'*|*'|'*|*';'*|*'='*|*'@'*|*'('*|*')'*|*'*'*) continue ;;
                esac
                if [ -n "$VAULT_ROOT" ] && [ -d "$VAULT_ROOT" ]; then
                    subpath="${token#\$VAULT_PATH/}"
                    subpath="${subpath#\$\{VAULT_PATH\}/}"
                    if [ ! -e "$VAULT_ROOT/$subpath" ]; then
                        printf '%s: referenced vault path does not exist: %s\n' "$target" "$token" >&2
                        missing_in_file=$((missing_in_file + 1))
                    fi
                fi
                continue
                ;;
            /*|'~'*|'$'*|-*|*://*) continue ;;
        esac

        # Shell metacharacters, assignments, or <placeholder> markers mean this
        # is a command fragment or a template, not a path on disk.
        case "$token" in
            *'<'*|*'>'*|*'&'*|*'|'*|*';'*|*'='*|*'@'*|*'('*|*')'*) continue ;;
        esac

        # An ALL-CAPS segment is usually a stand-in the reader substitutes
        # (`sensitive/KEYNAME.secret.age`). But plenty of real files are
        # ALL-CAPS — SKILL.md, AGENTS.md, MEMORY.md, README.md — so the
        # placeholder reading only applies when the token does NOT end in a
        # known extension. Without that second condition this rule silently
        # skipped `ai/skills/*/SKILL.md`, a genuinely dead reference.
        if printf '%s\n' "$token" | grep -qE '(^|/)[A-Z][A-Z0-9_]+(\.|/|$)' &&
            ! printf '%s' "$token" | grep -qE "$KNOWN_EXT"; then
            continue
        fi

        token="${token#./}"

        # Only rooted paths are checked. A bare filename is deliberately
        # ignored: prose names plenty of files that live elsewhere by design —
        # vault patterns (`pattern-language-standards.md`), per-machine config
        # (`machine.json`), spec artifacts (`review.md`), even files named only
        # to forbid them (`TODO.md`). Resolving those by basename produced a
        # dozen false alarms on AGENTS.md, and a guard that cries wolf is a
        # guard someone deletes.
        printf '%s' "$token" | grep -q '/' || continue
        is_repo_rooted "$token" || continue

        # A `..` component lets a token resolve outside REPO_ROOT while still
        # passing is_repo_rooted — `scripts/../../other-repo/README.md` has the
        # first segment `scripts` and was reported OK. That is a false negative,
        # the failure mode this guard exists to avoid, so reject it loudly.
        #
        # Order matters and was wrong once: this ran BEFORE the rooted checks
        # above, so it fired on `not-a-real-dir/../x.md` — a token the guard
        # promises to ignore by construction. Fixing a false negative created a
        # false positive. It must stay after the gate that decides "is this ours
        # to judge at all".
        case "/$token/" in
            */../*)
                printf '%s: path escapes the repo root: %s\n' "$target" "$token" >&2
                missing_in_file=$((missing_in_file + 1))
                continue
                ;;
        esac

        case "$token" in
            *'*'*)
                # A glob must match at least one path. Expanded under bash -c
                # with nullglob so an unmatched pattern yields nothing rather
                # than the literal pattern — and, under zsh's default NOMATCH,
                # rather than aborting the whole command.
                matches="$(bash -c 'shopt -s nullglob; cd "$1" || exit 0; set -- $2; printf "%s\n" "$@"' _ "$REPO_ROOT" "$token" | grep -c . || true)"
                if [ "$matches" -eq 0 ]; then
                    printf '%s: glob matches nothing: %s\n' "$target" "$token" >&2
                    missing_in_file=$((missing_in_file + 1))
                fi
                ;;
            *)
                if [ ! -e "$REPO_ROOT/$token" ]; then
                    printf '%s: referenced path does not exist: %s\n' "$target" "$token" >&2
                    missing_in_file=$((missing_in_file + 1))
                fi
                ;;
        esac
    done <<TOKENS
$tokens
TOKENS

    if [ "$missing_in_file" -eq 0 ]; then
        printf 'check-doc-paths: OK %s\n' "$target"
    else
        printf 'check-doc-paths: %s missing path(s) in %s\n' "$missing_in_file" "$target" >&2
        missing_total=$((missing_total + missing_in_file))
    fi
done

if [ "$missing_total" -ne 0 ]; then
    printf '\ncheck-doc-paths: %s referenced path(s) do not exist.\n' "$missing_total" >&2
    printf 'Either the path moved (update the doc) or the doc is stale (drop the claim).\n' >&2
    exit 1
fi

exit 0
