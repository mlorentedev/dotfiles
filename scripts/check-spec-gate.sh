#!/bin/bash

# check-spec-gate.sh: SDD Tier 4 enforcement gate.
#
# Computes production diff LOC between two refs and validates that PRs above
# the threshold include at least one file under specs/<feature-id>/ (active
# spec folder, NOT specs/archive/). Used by .github/workflows/spec-gate.yml
# and opt-in by pre-push hooks installed via scripts/install-precommit.sh.
#
# Usage:
#   check-spec-gate.sh --base-ref REF --head-ref REF [--threshold N] [--explain]
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

usage() {
    cat <<'EOF'
Usage: check-spec-gate.sh --base-ref REF --head-ref REF [--threshold N] [--explain]

  --base-ref REF    Base ref to diff against (e.g. origin/main)
  --head-ref REF    Head ref of the change (e.g. HEAD)
  --threshold N     LOC threshold above which a spec folder is required (default 50)
  --explain         Print the LOC breakdown per file
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
    local extracted
    extracted=$(printf '%s\n' "$SDD_PR_BODY" | awk '
        /^## SDD skip rationale[[:space:]]*$/ { in_block=1; next }
        in_block && /^## / { exit }
        in_block { print }
    ' | tr -d '[:space:]')
    [[ -n "$extracted" ]]
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

if _has_label "dependencies"; then
    if _is_dependency_bot "$SDD_PR_AUTHOR"; then
        printf '[OK] spec-gate skipped: "dependencies" label on a bot-authored PR (%s)\n' "$SDD_PR_AUTHOR"
        exit 0
    fi
    printf '[INFO] "dependencies" label present but author "%s" is not a recognised dependency bot — not honouring the skip; evaluating the gate normally.\n' "${SDD_PR_AUTHOR:-<unset>}" >&2
fi

if _has_label "skip-sdd"; then
    if _skip_rationale_nonempty; then
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

       Reference: AGENTS.md "Discipline Gate (NON-NEGOTIABLE)" section.
EOF
exit 1
