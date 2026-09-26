#!/usr/bin/env bash

# check-knowledge-gate.sh: every PR names where its knowledge went (HARNESS-160).
#
# The PR body carries a "## Knowledge" section with three lines:
#
#   - Lesson: docs/lessons/lesson-301-some-slug.md
#   - ADR: none: no contract or policy changed
#   - Runbook: none: no new procedure
#
# Each line is one or more paths this PR adds, changes or renames under the
# kind's directory, or "none: <reason>". Lessons and ADRs used to be asked for
# only by text at the end of the work (Definition of Done §2, the handoff's
# harvest step), and that text lost to the next task: dotfiles went from 0.42
# lessons per commit to 0.18 in three weeks while the knowledge moved into
# ticket and PR bodies (vault note 2026-09-25-knowledge-capture-routing). This
# gate asks in the change that produced the knowledge, while it is fresh.
#
# PR context comes from the env the gate adapters export (spec-gate-pr.sh
# --gate): SDD_PR_BODY, SDD_LABELS, SDD_PR_AUTHOR. An UNSET body means the
# gate was run with no PR behind it, and it refuses to judge (exit 2) rather
# than pass; a set but empty body is a PR without the section (exit 1).
#
# Usage:
#   check-knowledge-gate.sh --base-ref REF --head-ref REF [--explain]
#
# Exit:
#   0  every line answered, or a dependency bot's PR
#   1  the section, a line, a reason or a counting path is missing
#   2  usage error, no PR context, or a range git cannot diff

set -euo pipefail

# The repo's scripts run under bash and zsh (.claude/CLAUDE.md). zsh matches
# bash here with 0-based arrays and BASH_REMATCH filled by =~; no case-folding
# option is used, since shopt is bash-only.
if [ -n "${ZSH_VERSION:-}" ]; then
    setopt KSH_ARRAYS BASH_REMATCH
fi

usage() {
    cat <<'EOF'
Usage: check-knowledge-gate.sh --base-ref REF --head-ref REF [--explain]

Checks that the PR body has a "## Knowledge" section whose Lesson, ADR and
Runbook lines each name a path this PR adds, changes or renames under
docs/lessons/, docs/adr/ or docs/runbooks/ (not _index.md), or say
"none: <reason>".

  --base-ref REF  base of the PR (e.g. origin/main)
  --head-ref REF  head of the PR (e.g. HEAD)
  --explain       on failure, print the expected shape

Env (exported by spec-gate-pr.sh --gate):
  SDD_PR_BODY    PR body text (required; unset exits 2)
  SDD_LABELS     comma-separated PR labels
  SDD_PR_AUTHOR  PR author login; gates the "dependencies" bot skip
EOF
}

BASE_REF=""
HEAD_REF=""
EXPLAIN=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base-ref) BASE_REF="${2:-}"; shift 2 ;;
        --head-ref) HEAD_REF="${2:-}"; shift 2 ;;
        --explain) EXPLAIN=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) printf '[ERROR] Unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$BASE_REF" || -z "$HEAD_REF" ]]; then
    printf '[ERROR] --base-ref and --head-ref are required\n' >&2
    exit 2
fi

if [[ -z "${SDD_PR_BODY+set}" ]]; then
    printf '[ERROR] no PR context: SDD_PR_BODY is unset. Run through spec-gate-pr.sh --gate.\n' >&2
    exit 2
fi

SDD_LABELS="${SDD_LABELS:-}"
SDD_PR_AUTHOR="${SDD_PR_AUTHOR:-}"

_has_label() {
    case ",${SDD_LABELS}," in
        *",${1},"*) return 0 ;;
        *) return 1 ;;
    esac
}

# The same list as check-spec-gate.sh's, and a test holds them equal. Exact
# match on the author, so a human who adds the label is still judged.
_is_dependency_bot() {
    [[ "$1" == "dependabot[bot]" || "$1" == "dependabot-preview[bot]" || "$1" == "renovate[bot]" ]]
}

if _has_label "dependencies" && _is_dependency_bot "$SDD_PR_AUTHOR"; then
    printf '[OK] knowledge-gate skipped: "dependencies" label on a bot-authored PR (%s)\n' "$SDD_PR_AUTHOR"
    exit 0
fi

# The body without fenced code blocks, so a section quoted as an example does
# not count. Unlike spec-gate's _strip_markdown_code, inline code stays: paths
# are usually written as `docs/lessons/...`, and stripping spans would erase
# the very answer being checked.
_strip_fences() {
    local line marker fence=""
    local fence_re='^[[:space:]]{0,3}(`{3,}|~{3,})'
    while IFS= read -r line; do
        if [[ "$line" =~ $fence_re ]]; then
            marker="${BASH_REMATCH[1]}"
            if [[ -z "$fence" ]]; then
                fence="$marker"
            elif [[ "${marker:0:1}" == "${fence:0:1}" && ${#marker} -ge ${#fence} ]]; then
                fence=""
            fi
            continue
        fi
        [[ -z "$fence" ]] && printf '%s\n' "$line"
    done <<< "$1"
}

# The body without HTML comments, which GitHub does not render: a template's
# guidance, or an example inside one, is not an answer the author gave.
_strip_comments() {
    awk '
        {
            line = $0; out = ""
            while (line != "") {
                if (inside) {
                    i = index(line, "-->")
                    if (i == 0) { line = "" } else { line = substr(line, i + 3); inside = 0 }
                } else {
                    i = index(line, "<!--")
                    if (i == 0) { out = out line; line = "" }
                    else { out = out substr(line, 1, i - 1); line = substr(line, i + 4); inside = 1 }
                }
            }
            print out
        }
    '
}

# The lines under "## Knowledge", up to the next heading of level 1 or 2.
_section() {
    awk '
        tolower($0) ~ /^##[[:space:]]+knowledge[[:space:]]*$/ { on=1; found=1; next }
        on && /^##?[[:space:]]/ { exit }
        on { print }
        END { if (!found) exit 3 }
    '
}

# CR first: a body saved from GitHub's web editor has CRLF line endings.
body=$(_strip_fences "${SDD_PR_BODY//$'\r'/}" | _strip_comments)

problems=()

if ! section=$(printf '%s\n' "$body" | _section); then
    problems+=('the PR body has no "## Knowledge" section')
fi

if [[ ${#problems[@]} -eq 0 ]]; then
    if ! changed=$(git diff --name-only --diff-filter=AMR -M "$BASE_REF...$HEAD_REF"); then
        printf '[ERROR] cannot diff %s...%s\n' "$BASE_REF" "$HEAD_REF" >&2
        printf '[ERROR] The gate fails closed: it will not judge paths it cannot see.\n' >&2
        exit 2
    fi
fi

# A regex matching WORD in any letter case, as [Ww][Oo]...: portable where
# bash's nocasematch and zsh's (#i) are not.
_ci() {
    printf '%s' "$1" | awk '{ for (i = 1; i <= length($0); i++) { c = substr($0, i, 1); printf "[%s%s]", toupper(c), tolower(c) } }'
}

# The directory a kind's paths must sit under.
_dir_for() {
    case "$1" in
        Lesson) printf 'docs/lessons/' ;;
        ADR) printf 'docs/adr/' ;;
        Runbook) printf 'docs/runbooks/' ;;
    esac
}

# Judges one line's value for a kind, appending what is wrong to problems.
_judge() {
    local kind="$1" dir="$2" value="$3" target token
    local none_re
    none_re="^$(_ci none)[[:space:]]*(:[[:space:]]*(.*))?\$"
    value="${value//\`/}"
    if [[ "$value" =~ $none_re ]]; then
        local reason="${BASH_REMATCH[2]}"
        if [[ -z "${reason// /}" ]]; then
            problems+=("$kind: \"none\" needs a reason, as \"none: <reason>\"")
        elif [[ "$reason" =~ ^\<[^\>]*\>$ ]]; then
            problems+=("$kind: \"$reason\" is the template's placeholder; write the reason")
        fi
        return 0
    fi
    local tokens
    tokens=$(printf '%s\n' "$value" | grep -oE '[A-Za-z0-9._/-]+\.md' || true)
    if [[ -z "$tokens" ]]; then
        problems+=("$kind: \"$value\" is neither a path nor \"none: <reason>\"")
        return 0
    fi
    while IFS= read -r token; do
        # Not "path": in zsh that name is tied to $PATH (.claude/CLAUDE.md).
        target="${token#./}"
        if [[ "$target" != "$dir"* ]]; then
            problems+=("$kind: $target is not under $dir")
        elif [[ "${target##*/}" == "_index.md" ]]; then
            problems+=("$kind: $target is an index, not a $kind")
        elif ! grep -qxF -- "$target" <<< "$changed"; then
            problems+=("$kind: $target is not changed by this PR (added, modified or renamed)")
        fi
    done <<< "$tokens"
}

summary=()

if [[ ${#problems[@]} -eq 0 ]]; then
    for kind in Lesson ADR Runbook; do
        line_re="^[[:space:]]*([-*+][[:space:]]+)?$(_ci "$kind")[[:space:]]*:[[:space:]]*(.*)\$"
        values=()
        while IFS= read -r line; do
            line="${line//\*\*/}"
            if [[ "$line" =~ $line_re ]]; then
                values+=("${BASH_REMATCH[2]%"${BASH_REMATCH[2]##*[![:space:]]}"}")
            fi
        done <<< "$section"
        case ${#values[@]} in
            0) problems+=("$kind: the line is missing") ;;
            1) _judge "$kind" "$(_dir_for "$kind")" "${values[0]}"
               summary+=("$(printf '  %-8s %s' "$kind" "${values[0]}")") ;;
            *) problems+=("$kind: the line appears ${#values[@]} times; give one answer") ;;
        esac
    done
fi

if [[ ${#problems[@]} -eq 0 ]]; then
    printf '[OK] knowledge-gate: every line is answered\n'
    printf '%s\n' "${summary[@]}"
    exit 0
fi

printf '[FAIL] knowledge-gate: %d problem(s) in the PR body\n' "${#problems[@]}"
printf '  - %s\n' "${problems[@]}"
if [[ $EXPLAIN -eq 1 ]]; then
    cat <<'EOF'

The PR body needs this section, one line per kind. Each line names the files
this PR adds or changes for that kind, or says why there are none:

  ## Knowledge

  - Lesson: docs/lessons/lesson-NNN-<slug>.md
  - ADR: none: <reason>
  - Runbook: none: <reason>

Edit the PR body; the gate re-runs on the edit.
EOF
fi
exit 1
