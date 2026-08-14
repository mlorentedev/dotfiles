#!/usr/bin/env bash

# check-spec-gate.sh: SDD Tier 4 enforcement gate.
#
# Computes production diff LOC between two refs and validates that PRs above
# the threshold include at least one file under specs/<feature-id>/ (active
# spec folder, NOT specs/archive/). Used by .github/workflows/spec-gate.yml
# and opt-in by pre-push hooks installed via scripts/install-precommit.sh.
#
# Usage:
#   check-spec-gate.sh --base-ref REF --head-ref REF [--threshold N] [--explain]
#                      [--adjacency-issues FILE]
#
# Env (consumed when set, normally populated by the CI workflow from PR):
#   SDD_LABELS     Comma-separated PR labels
#   SDD_PR_BODY    PR body text
#   SDD_PR_AUTHOR  PR author login (gates the "dependencies" bot skip)
#
# Exit:
#   0  OK (under threshold OR spec folder present OR valid skip)
#   1  Discipline Gate violation
#   2  Usage/setup error

set -euo pipefail

THRESHOLD=50
# Minimum added+removed LOC WITHIN active specs/<id>/ folders for the change to
# count as a genuine spec touch. Defeats the "edit one line of an unrelated stale
# spec to legitimise a large PR" bypass (#686/C25): a real new/updated spec is
# far larger than this floor, a one-line alibi is not. (The complete spec<->PR
# linkage pairs with #670, which archives shipped specs and shrinks the alibi
# surface.)
SPEC_FLOOR=10
BASE_REF=""
HEAD_REF=""
EXPLAIN=0
# Path to the open-issue feed for the advisory adjacency report (HARNESS-063).
# Empty on every local run: fetching every open issue needs a token, so this
# feed is populated by the workflow only -- this script itself never fetches
# anything. (Distinct from BUG-061/#854's PR-body/labels/author resolution:
# that is a single PR read, needs no token, and is handled by the separate
# scripts/spec-gate-prepush.sh wrapper on the local pre-push tier -- see its
# header for why it lives outside this script rather than inside it.)
ADJACENCY_ISSUES=""

usage() {
    cat <<'EOF'
Usage: check-spec-gate.sh --base-ref REF --head-ref REF [--threshold N] [--explain]
                          [--adjacency-issues FILE]

  --base-ref REF    Base ref to diff against (e.g. origin/main)
  --head-ref REF    Head ref of the change (e.g. HEAD)
  --threshold N     LOC threshold above which a spec folder is required (default 50)
  --explain         Print the LOC breakdown per file
  --adjacency-issues FILE
                    TSV of open issues ("<number><TAB><title and body, newlines
                    flattened>"). Emits an ADVISORY report of other open issues
                    naming a file this change touches. Never affects the verdict;
                    a missing or unreadable file is silently ignored.
  -h, --help        Show this help

Env:
  SDD_LABELS     Comma-separated PR labels (CI sets this; locally optional)
  SDD_PR_BODY    PR body text (CI sets this; locally optional)
  SDD_PR_AUTHOR  PR author login; gates the "dependencies" skip to bot authors

Exit codes:
  0  OK
  1  Discipline Gate violation
  2  Usage/setup error
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base-ref) BASE_REF="$2"; shift 2 ;;
        --head-ref) HEAD_REF="$2"; shift 2 ;;
        --threshold) THRESHOLD="$2"; shift 2 ;;
        --explain) EXPLAIN=1; shift ;;
        --adjacency-issues) ADJACENCY_ISSUES="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) printf '[ERROR] Unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$BASE_REF" || -z "$HEAD_REF" ]]; then
    printf '[ERROR] --base-ref and --head-ref are required\n' >&2
    usage >&2
    exit 2
fi

if ! git rev-parse --git-dir >/dev/null 2>&1; then
    printf '[ERROR] Not in a git repo\n' >&2
    exit 2
fi

SDD_LABELS="${SDD_LABELS:-}"
SDD_PR_BODY="${SDD_PR_BODY:-}"
SDD_PR_AUTHOR="${SDD_PR_AUTHOR:-}"

_has_label() {
    case ",${SDD_LABELS}," in
        *",${1},"*) return 0 ;;
        *) return 1 ;;
    esac
}

# The "dependencies" auto-skip is only legitimate for bot-authored dependency
# PRs. Gating it on the author closes the bypass where any human collaborator
# adds the "dependencies" label to neutralise the gate — spec-gate.yml re-runs
# on `labeled` (#686/C25). Exact string match, not a case glob: "[bot]" is all
# glob metacharacters.
_is_dependency_bot() {
    [[ "$1" == "dependabot[bot]" || "$1" == "dependabot-preview[bot]" || "$1" == "renovate[bot]" ]]
}

_skip_rationale_nonempty() {
    local heading="$1"
    local extracted
    extracted=$(printf '%s\n' "$SDD_PR_BODY" | awk -v h="## $heading" '
        $0 ~ ("^" h "[[:space:]]*$") { in_block=1; next }
        in_block && /^## / { exit }
        in_block { print }
    ' | tr -d '[:space:]')
    [[ -n "$extracted" ]]
}

# ---------------------------------------------------------------------------
# Archive-on-merge (SDD-038): a PR that CLOSES an issue must archive the active
# spec tracking it. Keyed on GitHub's closing keywords rather than on "a spec
# was touched" — `Refs #N` legitimately leaves a spec active when the issue's
# work continues in a follow-up.
# ---------------------------------------------------------------------------

# owner/name of this repo, used to reject cross-repo closing references.
# CI exports GITHUB_REPOSITORY; otherwise derive it from origin. Empty means
# ownership cannot be proven, in which case QUALIFIED references are ignored
# rather than guessed at (a bare `#N` is this repo by definition).
_repo_slug() {
    if [[ -n "${GITHUB_REPOSITORY:-}" ]]; then
        printf '%s' "$GITHUB_REPOSITORY"
        return 0
    fi
    local url
    url=$(git remote get-url origin 2>/dev/null) || return 0
    url="${url%.git}"
    case "$url" in
        *github.com[:/]*) printf '%s' "${url##*github.com}" | sed -E 's#^[:/]##' ;;
    esac
}

# A PR body with fenced code blocks and inline code spans removed.
#
# GitHub does not linkify — and therefore does not close from — a reference
# inside code, so neither may this gate. That is the whole criterion: the gate's
# contract is to fire exactly when GitHub would close the issue.
#
# Note what is deliberately NOT stripped: ordinary prose. Narrowing the match to
# line-leading declarations was considered and rejected — GitHub matches a
# closing keyword anywhere in the body, so a gate that only saw line-leading ones
# would stay silent while GitHub closed the issue, leaving the spec un-archived.
# For a check whose entire job is to demand the archive, a miss is far worse than
# a cry of wolf, and code blocks are the only region where "this is not a
# directive" is provable rather than guessed.
_strip_markdown_code() {
    local line marker fence=""
    # Single-quoted on purpose: these are ERE patterns, and the backticks must
    # reach the regex engine rather than the shell. Held in variables because an
    # unquoted backtick inside [[ =~ ]] would be command substitution.
    local fence_re='^[[:space:]]{0,3}(`{3,}|~{3,})'
    # shellcheck disable=SC2016
    local span_re='`[^`]*`'

    while IFS= read -r line; do
        if [[ "$line" =~ $fence_re ]]; then
            marker="${BASH_REMATCH[1]}"
            if [[ -z "$fence" ]]; then
                fence="$marker"
            elif [[ "${marker:0:1}" == "${fence:0:1}" && ${#marker} -ge ${#fence} ]]; then
                fence=""   # a closing fence must match the opener's char and length
            fi
            continue
        fi
        [[ -n "$fence" ]] && continue
        # Inline spans: the shape a worked example takes when it sits in a
        # sentence rather than in its own block.
        while [[ "$line" =~ $span_re ]]; do
            line="${line/"${BASH_REMATCH[0]}"/}"
        done
        printf '%s\n' "$line"
    done <<< "$1"
}

# Issue numbers this PR body closes, one per line. Only GitHub's own closing
# verbs count; `Refs`, `Part of` and a bare number in prose must not fire.
_closing_issue_numbers() {
    local slug="$1"
    local body="$2"

    # #773: a documentation PR about this very gate is the likeliest thing to
    # carry a realistic example of the string the gate looks for. #767's body
    # tripped its own check twice — once on a fenced `SDD_PR_BODY='Closes #670'`
    # transcript, once on a sentence describing a different, future PR.
    body=$(_strip_markdown_code "$body")

    # Fold the full-URL form into the qualified short form so one matcher covers
    # both. `|` as the sed delimiter because `#` is part of the replacement.
    body=$(printf '%s\n' "$body" | sed -E \
        's|https?://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/issues/([0-9]+)|\1#\2|g')

    # `|| true`: no closing reference is the COMMON case, but grep exits 1 on no
    # match and this script runs under `set -euo pipefail`. Left unguarded, an
    # ordinary "Refs #N" PR aborts the gate — failing closed for the wrong
    # reason, which blocks legitimate work.
    local matches
    matches=$(printf '%s\n' "$body" \
        | grep -oiE '\b(close[sd]?|fix(e[sd])?|resolve[sd]?):?[[:space:]]+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?#[0-9]+' \
        || true)
    [[ -z "$matches" ]] && return 0

    printf '%s\n' "$matches" \
        | awk '{ print $NF }' \
        | while IFS= read -r ref; do
            [[ -z "$ref" ]] && continue
            local qualifier="${ref%#*}"
            local number="${ref##*#}"
            if [[ -n "$qualifier" ]]; then
                # Someone else's issue number must never archive our spec.
                if [[ -z "$slug" || "$qualifier" != "$slug" ]]; then
                    continue
                fi
            fi
            printf '%s\n' "$number"
        done
}

# The issue number declared by a proposal.md on stdin, or nothing. Three shapes
# exist in-tree — "owner/repo#N", "repo#N" and a bare unquoted N — and the
# quoted forms are followed by a trailing `#` comment, so the value is extracted
# before any number is read out of it.
_frontmatter_issue_number() {
    awk '
        NR == 1 && $0 !~ /^---[[:space:]]*$/ { exit }
        NR > 1 && $0 ~ /^---[[:space:]]*$/ { exit }
        /^issue:[[:space:]]*/ {
            sub(/^issue:[[:space:]]*/, "")
            if (substr($0, 1, 1) == "\"") { sub(/^"/, ""); sub(/".*$/, "") }
            else { sub(/[[:space:]].*$/, "") }
            sub(/^.*#/, "")
            if ($0 ~ /^[0-9]+$/) print $0
            exit
        }
    '
}

# "<issue-number><TAB><spec-id>" for every ACTIVE spec at the given ref. Read at
# the base ref: at head an archived spec is (correctly) gone, so base is the only
# ref where the issue -> spec linkage is still observable.
_active_spec_issue_map() {
    local ref="$1"
    local paths
    paths=$(git ls-tree -r --name-only "$ref" -- specs/ 2>/dev/null) || return 1

    printf '%s\n' "$paths" | while IFS= read -r path; do
        case "$path" in
            specs/archive/*) continue ;;
            specs/*/proposal.md) ;;
            *) continue ;;
        esac
        local id="${path#specs/}"
        id="${id%/proposal.md}"
        case "$id" in */*) continue ;; esac   # only specs/<id>/proposal.md
        local num
        num=$(git show "$ref:$path" 2>/dev/null | _frontmatter_issue_number) || continue
        # An `if`, not `[[ ... ]] && printf`: the latter is the loop body's last
        # command, so a spec WITHOUT an issue: field would make the whole while
        # loop exit 1 and the caller fail closed on perfectly valid data.
        if [[ -n "$num" ]]; then
            printf '%s\t%s\n' "$num" "$id"
        fi
    done
}

# "<issue-number><TAB><spec-id>" for every ARCHIVED spec at the given ref — the
# mirror of _active_spec_issue_map, and the other half of the linkage. Once a
# spec is archived it leaves specs/<id>/ entirely, so the active map cannot see
# what a PR archived; a spec created AND archived in one change is invisible to
# the active map at base and at head alike.
_archived_spec_issue_map() {
    local ref="$1"
    local paths
    paths=$(git ls-tree -r --name-only "$ref" -- specs/archive/ 2>/dev/null) || return 1

    printf '%s\n' "$paths" | while IFS= read -r path; do
        case "$path" in
            specs/archive/*/proposal.md) ;;
            *) continue ;;
        esac
        local id="${path#specs/archive/}"
        id="${id%/proposal.md}"
        case "$id" in */*) continue ;; esac   # only specs/archive/<id>/proposal.md
        local num
        num=$(git show "$ref:$path" 2>/dev/null | _frontmatter_issue_number) || continue
        # Same `if` reasoning as _active_spec_issue_map: a trailing failed test
        # would make the whole while loop exit 1 on valid data.
        if [[ -n "$num" ]]; then
            printf '%s\t%s\n' "$num" "$id"
        fi
    done
}

# Spec IDs this PR archived in fulfilment of archive-on-merge: archived at head
# AND linked, through proposal.md's `issue:` frontmatter, to an issue this PR
# closes. Anything else archived in the same diff is not here, so #397's
# protection against a gratuitous archive-move dodging the gate is untouched.
_mandated_archive_ids() {
    [[ -z "$SDD_PR_BODY" ]] && return 0
    local slug numbers map num spec_num spec_id
    slug=$(_repo_slug)
    numbers=$(_closing_issue_numbers "$slug" "$SDD_PR_BODY" | sort -u)
    [[ -z "$numbers" ]] && return 0
    map=$(_archived_spec_issue_map "$HEAD_REF") || return 0
    [[ -z "$map" ]] && return 0

    while IFS= read -r num; do
        [[ -z "$num" ]] && continue
        while IFS=$'\t' read -r spec_num spec_id; do
            if [[ -n "$spec_id" && "$spec_num" == "$num" ]]; then
                printf '%s\n' "$spec_id"
            fi
        done <<< "$map"
    done <<< "$numbers"
}

_check_archive_on_merge() {
    # No PR body (local pre-push runs) means nothing to enforce.
    [[ -z "$SDD_PR_BODY" ]] && return 0

    local slug numbers
    slug=$(_repo_slug)
    numbers=$(_closing_issue_numbers "$slug" "$SDD_PR_BODY" | sort -u)
    [[ -z "$numbers" ]] && return 0

    local map base_map head_map
    if ! base_map=$(_active_spec_issue_map "$BASE_REF"); then
        printf '[ERROR] Could not read specs/ at base ref "%s".\n' "$BASE_REF" >&2
        printf '        The gate fails closed: it will not pass a PR whose spec state cannot be read.\n' >&2
        exit 2
    fi
    # Union with head: base alone would miss a PR that CREATES a spec and closes
    # its issue in the same change — precisely the "never archived" pattern this
    # gate exists to stop. A spec archived at head is no longer under specs/<id>/,
    # so it cannot appear here and cannot produce a false positive.
    if ! head_map=$(_active_spec_issue_map "$HEAD_REF"); then
        printf '[ERROR] Could not read specs/ at head ref "%s".\n' "$HEAD_REF" >&2
        exit 2
    fi
    map=$(printf '%s\n%s\n' "$base_map" "$head_map" | grep -v '^$' | sort -u || true)
    [[ -z "$map" ]] && return 0

    # Newline-delimited accumulator, not a bash array (AGENTS.md: avoid
    # bash-only syntax) — this script's shebang is bash-only anyway, but the
    # `<<<` here-string idiom already used throughout keeps this consistent
    # with the rest of the file rather than introducing a second convention.
    local violations=""
    local num spec_num spec_id archived
    while IFS= read -r num; do
        [[ -z "$num" ]] && continue
        while IFS=$'\t' read -r spec_num spec_id; do
            [[ -n "$spec_id" && "$spec_num" == "$num" ]] || continue
            # Ask the head TREE, not the diff: a move only shows up as a rename
            # when git's rename detection fires, and the same change can surface
            # as delete+add. Presence cannot be fooled by how git renders it.
            archived=$(git ls-tree -r --name-only "$HEAD_REF" -- "specs/archive/$spec_id/" 2>/dev/null) || archived=""
            [[ -n "$archived" ]] || violations="${violations}${spec_id} (#${num})"$'\n'
        done <<< "$map"
    done <<< "$numbers"

    [[ -z "$violations" ]] && return 0

    if _has_label "skip-archive"; then
        if _skip_rationale_nonempty "Archive skip rationale"; then
            printf '[OK] archive-on-merge skipped: "skip-archive" label + non-empty "## Archive skip rationale" in PR body\n'
            return 0
        fi
        cat >&2 <<'EOF'
[FAIL] "skip-archive" label present but the "## Archive skip rationale" section is empty or missing in the PR body.
EOF
        exit 1
    fi

    {
        printf '[FAIL] SDD archive-on-merge violation:\n'
        printf '       This PR closes an issue whose spec is still active:\n'
        while IFS= read -r v; do [[ -n "$v" ]] && printf '         %s\n' "$v"; done <<< "$violations"
        printf '\n'
        printf '       The SDD lifecycle ends by archiving the spec in the SAME PR\n'
        printf '       (Discipline Gate step 7). Run:\n\n'
        while IFS= read -r v; do [[ -n "$v" ]] && printf '         dotf spec archive %s\n' "${v%% *}"; done <<< "$violations"
        printf '\n'
        printf '       If the work genuinely continues elsewhere, reference the issue\n'
        printf '       without a closing keyword (e.g. "Refs #N"), or add the\n'
        printf '       "skip-archive" label AND a non-empty "## Archive skip rationale"\n'
        printf '       section to the PR body.\n'
    } >&2
    exit 1
}

# ---------------------------------------------------------------------------
# Advisory adjacency report (HARNESS-063): OTHER open issues naming a file this
# change touches. Purely informational — it never gates.
#
# The case it exists for: #851 merged a correct fix for #850 while #849 sat open
# describing the same file's other input shape. Every gate here looked only at
# the issue the PR closes, so nothing connected them and the defect shipped
# green (#857).
# ---------------------------------------------------------------------------

# Basenames too generic for a name match to carry information — matching these
# would report a large slice of the backlog on every PR that touches one.
ADJACENCY_GENERIC_BASENAMES="README.md AGENTS.md CLAUDE.md CHANGELOG.md Makefile Dockerfile go.mod go.sum"

# The changed files worth matching on: the ones this gate already treats as
# production. Measured against the live backlog, matching the whole diff reported
# 37 issues of which 3 carried signal — the rest matched template basenames
# (`proposal.md`, `tasks.md`, `docs/lessons.md`) that nearly every PR touches. A
# wall of noise is read once and ignored forever, which is the same outcome as
# having no check.
#
# Reusing _excluded() is the second helper-reuse decision here, so it gets the
# same scrutiny the first one failed: its error policy is "not production code",
# and inheriting it means an issue about a test file or a doc will not surface.
# That is acceptable where the first reuse was not — an unhandled INPUT SHAPE,
# the defect class this check exists for, lives in production code by definition,
# and precision is what makes the report legible enough to act on.
_adjacency_candidate_files() {
    local path
    while IFS= read -r path; do
        if [[ -z "$path" ]]; then continue; fi
        if _excluded "$path"; then continue; fi
        if _is_active_spec_path "$path"; then continue; fi
        printf '%s\n' "$path"
    done <<< "$1"
    return 0
}

# "<number><TAB><matched-path><TAB><haystack>" per adjacent issue.
#
# Matches over UNSTRIPPED title+body, unlike _closing_issue_numbers, and the
# difference is deliberate rather than an oversight. That matcher strips code
# spans because there a false POSITIVE is the expensive error: it would demand an
# archive GitHub never triggers, blocking legitimate work. Here the expensive
# error is the false NEGATIVE — #849 cites its path only inside an inline code
# span, so stripping would hide the one issue this check exists to surface.
# Reusing _strip_markdown_code() would inherit its error policy along with its
# code; tests/spec-gate-adjacency.bats pins that so a future refactor goes red.
_adjacent_open_issues() {
    local feed="$1" files="$2" closed="$3"
    local num haystack path base
    while IFS=$'\t' read -r num haystack; do
        if [[ -z "$num" || -z "${haystack:-}" ]]; then continue; fi
        if printf '%s\n' "$closed" | grep -qxF "$num"; then continue; fi
        while IFS= read -r path; do
            if [[ -z "$path" ]]; then continue; fi
            if [[ "$haystack" == *"$path"* ]]; then
                printf '%s\t%s\t%s\n' "$num" "$path" "$haystack"
                break
            fi
            base="${path##*/}"
            case " $ADJACENCY_GENERIC_BASENAMES " in *" $base "*) continue ;; esac
            if [[ "$haystack" == *"$base"* ]]; then
                printf '%s\t%s\t%s\n' "$num" "$base" "$haystack"
                break
            fi
        done <<< "$files"
    done < "$feed"
    return 0
}

# Emits the report to stdout, plus a CI annotation and step-summary table when
# running under Actions. Every failure path returns 0: an advisory check that can
# break the gate is worse than no advisory check at all.
_report_adjacent_issues() {
    if [[ -z "$ADJACENCY_ISSUES" || ! -r "$ADJACENCY_ISSUES" ]]; then return 0; fi

    local files closed rows num path haystack
    files=$(git diff --name-only "${BASE_REF}...${HEAD_REF}" 2>/dev/null) || return 0
    files=$(_adjacency_candidate_files "$files")
    if [[ -z "$files" ]]; then return 0; fi
    closed=$(_closing_issue_numbers "$(_repo_slug)" "$SDD_PR_BODY" | sort -u || true)
    rows=$(_adjacent_open_issues "$ADJACENCY_ISSUES" "$files" "$closed" || true)
    if [[ -z "$rows" ]]; then return 0; fi

    printf '[INFO] Adjacent open issues (advisory — this does not gate the PR):\n'
    while IFS=$'\t' read -r num path haystack; do
        if [[ -z "$num" ]]; then continue; fi
        printf '         #%-6s names %s\n' "$num" "$path"
        printf '                 %.88s\n' "$haystack"
    done <<< "$rows"
    printf '       One of these may describe the same defect on an input shape this\n'
    printf '       change does not cover. Fold it in, or record why it is separate,\n'
    printf '       before merging (HARNESS-063).\n'

    if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
        printf '::warning title=Adjacent open issues::%s\n' \
            "$(printf '%s' "$rows" | awk -F'\t' '{printf "#%s (%s) ", $1, $2}')"
    fi
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        {
            printf '\n### Adjacent open issues (advisory)\n\n'
            printf '| Issue | Names | Title |\n|---|---|---|\n'
            while IFS=$'\t' read -r num path haystack; do
                if [[ -z "$num" ]]; then continue; fi
                # shellcheck disable=SC2016  # backticks are markdown code spans
                # for the step-summary table, not command substitution.
                printf '| #%s | `%s` | %.88s |\n' "$num" "$path" "$haystack"
            done <<< "$rows"
        } >> "$GITHUB_STEP_SUMMARY"
    fi
    return 0
}

_excluded() {
    local path="$1"
    local base="${path##*/}"
    case "$path" in
        docs/*.md) return 0 ;;   # doc-only (ADRs, lessons, runbooks): prose, not production
        tests/*|specs/archive/*) return 0 ;;
        # No broad *generated* glob (#686/C25): it excluded any path merely
        # CONTAINING "generated" (e.g. a hand-written internal/generated_names.go)
        # — a silent bypass. Zero generated files exist in-tree today; when a real
        # one is introduced, add an explicit entry here (e.g. *.pb.go), never a
        # substring match.
    esac
    case "$base" in
        *_test.go) return 0 ;;   # Go test files are tests, not production (#517)
        *.lock|*.lockb) return 0 ;;
        package-lock.json|pnpm-lock.yaml|go.sum) return 0 ;;
        .gitignore|CHANGELOG.md) return 0 ;;
    esac
    return 1
}

_is_active_spec_path() {
    case "$1" in
        specs/archive/*) return 1 ;;
        specs/*/*) return 0 ;;
        *) return 1 ;;
    esac
}

# `specs/archive/<id>/...` where <id> is a spec this PR archived to satisfy
# archive-on-merge. Such a path counts as THE spec touch outright rather than
# feeding SPEC_LOC, because an archive move barely registers as a diff: a real
# one (#787) is 4 LOC — the `status:` rewrite plus three pure renames — against
# SPEC_FLOOR=10. Counting its lines would leave the gate exactly as
# unsatisfiable as before (#800).
#
# This does not reopen the "one-line alibi" #686/C25 closed. That bypass is a
# trivial edit to an UNRELATED active spec; this path is reachable only through
# the `issue:` frontmatter of a spec whose issue this very PR closes, so there is
# no unrelated spec to hide behind.
_is_mandated_archive_path() {
    [[ -n "$MANDATED_ARCHIVE_IDS" ]] || return 1
    case "$1" in specs/archive/*/*) ;; *) return 1 ;; esac
    local id="${1#specs/archive/}"
    id="${id%%/*}"
    printf '%s\n' "$MANDATED_ARCHIVE_IDS" | grep -qxF "$id"
}

# git diff --numstat compresses a rename into one path with the common
# prefix/suffix factored out: "specs/{ => archive}/<id>/proposal.md" (brace
# form) or "old/path => new/path" (full form). Both must collapse to the
# DESTINATION path before the matchers run, or an archived move reads as an
# active-spec touch and dodges the gate (#397).
_normalize_rename_path() {
    local p="$1"
    case "$p" in
        *'{'*' => '*'}'*)
            # Brace form: prefix{old => new}suffix -> prefix + new + suffix
            p=$(printf '%s' "$p" | sed 's/{[^}]* => \([^}]*\)}/\1/')
            ;;
    esac
    case "$p" in
        *' => '*) p="${p##* => }" ;;   # Full form: old => new -> new
    esac
    case "$p" in
        *//*) p=$(printf '%s' "$p" | sed 's#//*#/#g') ;;  # collapse moves OUT of a dir
    esac
    printf '%s' "$p"
}

# The advisory report runs before every gate and every early exit, so a PR that
# skips SDD or fails archive-on-merge still surfaces its adjacent issues — those
# are exactly the PRs where a second open issue is most likely to be missed.
_report_adjacent_issues

# Archive-on-merge runs FIRST and unconditionally (SDD-038). A three-line PR can
# still end an issue's life, so it must not sit behind the LOC early exit below;
# and `skip-sdd` asserts "this change needs no spec", which says nothing about
# whether an existing spec's work is finished.
_check_archive_on_merge

if _has_label "dependencies"; then
    if _is_dependency_bot "$SDD_PR_AUTHOR"; then
        printf '[OK] spec-gate skipped: "dependencies" label on a bot-authored PR (%s)\n' "$SDD_PR_AUTHOR"
        exit 0
    fi
    printf '[INFO] "dependencies" label present but author "%s" is not a recognised dependency bot — not honouring the skip; evaluating the gate normally.\n' "${SDD_PR_AUTHOR:-<unset>}" >&2
fi

if _has_label "skip-sdd"; then
    if _skip_rationale_nonempty "SDD skip rationale"; then
        printf '[OK] spec-gate skipped: "skip-sdd" label + non-empty "## SDD skip rationale" in PR body\n'
        exit 0
    fi
    cat >&2 <<'EOF'
[FAIL] "skip-sdd" label present but the "## SDD skip rationale" section is empty or missing in the PR body.
       Add a "## SDD skip rationale" section to the PR body with a real reason
       (e.g. "mechanical rename, no logic change"), or open a spec folder instead.
EOF
    exit 1
fi

TOTAL_LOC=0
SPEC_LOC=0
SPEC_TOUCHED=0
INCLUDED=()
EXCLUDED=()
# Computed once, before the diff walk: the specs this PR archived because
# archive-on-merge required it (#800). Empty for every PR that closes nothing.
MANDATED_ARCHIVE_IDS=$(_mandated_archive_ids || true)

# Fail CLOSED on a diff error. If BASE_REF/HEAD_REF do not resolve (shallow or
# fresh clone without origin/main, a typo'd ref, a detached worktree), git diff
# errors — and an enforcement gate must NOT let that silently pass. The previous
# `2>/dev/null || true` swallowed the error, the loop read nothing, TOTAL_LOC
# stayed 0, and the gate exited 0 on ANY PR (#686/C3). `if !` keeps `set -e`
# from aborting before we can emit a clear, actionable message.
if ! DIFF_OUTPUT=$(git diff --numstat "${BASE_REF}...${HEAD_REF}" 2>/dev/null); then
    printf '[ERROR] git diff "%s...%s" failed — base/head ref could not be resolved.\n' "$BASE_REF" "$HEAD_REF" >&2
    printf '        The Discipline Gate fails closed (exit 2): it will not pass a PR whose diff cannot be computed.\n' >&2
    printf '        In CI, ensure the base ref is fetched (fetch-depth: 0 or an explicit git fetch origin <base>).\n' >&2
    exit 2
fi

while IFS=$'\t' read -r added removed path; do
    [[ -z "${path:-}" ]] && continue
    # Resolve git's rename-compression to the destination path (#397).
    case "$path" in *' => '*) path=$(_normalize_rename_path "$path") ;; esac
    [[ "$added" == "-" ]] && added=0
    [[ "$removed" == "-" ]] && removed=0

    if _is_active_spec_path "$path"; then
        SPEC_LOC=$((SPEC_LOC + added + removed))
    elif _is_mandated_archive_path "$path"; then
        SPEC_TOUCHED=1
    fi

    if _excluded "$path"; then
        EXCLUDED+=("$path:$((added + removed))")
        continue
    fi

    file_loc=$((added + removed))
    TOTAL_LOC=$((TOTAL_LOC + file_loc))
    INCLUDED+=("$path:$file_loc")
done <<< "$DIFF_OUTPUT"

# A spec touch only counts if it is substantive (#686/C25). A trivial edit to an
# unrelated active spec (the "one-line alibi") must not legitimise a large PR.
if (( SPEC_LOC >= SPEC_FLOOR )); then
    SPEC_TOUCHED=1
fi

if [[ "$EXPLAIN" -eq 1 ]]; then
    printf '[INFO] Spec-gate breakdown (%s...%s)\n' "$BASE_REF" "$HEAD_REF"
    printf '  Threshold: %d LOC\n' "$THRESHOLD"
    printf '  Production LOC (added+removed): %d\n' "$TOTAL_LOC"
    printf '  Active-spec LOC (added+removed): %d (floor %d to count as a spec touch)\n' "$SPEC_LOC" "$SPEC_FLOOR"
    if [[ -n "$MANDATED_ARCHIVE_IDS" ]]; then
        printf '  Specs archived for archive-on-merge (count as a spec touch):\n'
        while IFS= read -r _id; do
            [[ -n "$_id" ]] && printf '    %s\n' "$_id"
        done <<< "$MANDATED_ARCHIVE_IDS"
    fi
    if [[ "$SPEC_TOUCHED" -eq 1 ]]; then
        printf '  Spec folder touched: yes\n'
    else
        printf '  Spec folder touched: no\n'
    fi
    if (( ${#INCLUDED[@]} > 0 )); then
        printf '  Files counted:\n'
        for f in "${INCLUDED[@]}"; do printf '    %s\n' "$f"; done
    fi
    if (( ${#EXCLUDED[@]} > 0 )); then
        printf '  Files excluded:\n'
        for f in "${EXCLUDED[@]}"; do printf '    %s\n' "$f"; done
    fi
fi

if (( TOTAL_LOC < THRESHOLD )); then
    printf '[OK] Production diff %d LOC < threshold %d (below threshold; spec not required)\n' "$TOTAL_LOC" "$THRESHOLD"
    exit 0
fi

if [[ "$SPEC_TOUCHED" -eq 1 ]]; then
    printf '[OK] Production diff %d LOC >= threshold %d AND spec folder touched in diff\n' "$TOTAL_LOC" "$THRESHOLD"
    exit 0
fi

cat >&2 <<EOF
[FAIL] SDD Discipline Gate violation:
       Production diff: $TOTAL_LOC LOC (>= threshold $THRESHOLD)
       No specs/<feature-id>/ folder touched in this PR.

       Options:
         (a) Create a spec folder: dotf spec init <feature-id>
         (b) Add the "skip-sdd" label to the PR AND a non-empty
             "## SDD skip rationale" section in the PR body.

       If this already archived a spec to satisfy archive-on-merge but still
       failed here: open the PR first (scripts/spec-gate-prepush.sh, wired to
       this hook, resolves its labels/body/author live via \`gh\` once one
       exists). Before a PR exists there is nothing to resolve; SDD_PR_BODY can
       be set by hand for a one-off check: SDD_PR_BODY='Closes #N' $0 ...

       Reference: AGENTS.md "Discipline Gate (NON-NEGOTIABLE)" section.
EOF
exit 1
