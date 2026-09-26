#!/usr/bin/env bash

# pr-agent-push-gate.sh: decide whether a push to a pull request is reviewed (TOOL-023).
#
# pr-agent used to run a full /review on every push (#1053). In one week that was
# 39 reviews across 18 PRs, and up to 7 on one PR. Each review was one request to
# a NaN pool of 5 shared slots, and each re-review reopened the triage queue. A
# push is now reviewed incrementally (/review -i), and only once THRESHOLD
# non-merge commits are newer than the previous review.
#
# "The previous review" is the one PR-Agent itself picks (v0.45.0,
# github_provider.get_previous_review and utils.comment_matches_identity): the
# LAST issue comment carrying a review identity line within its first 5 lines,
# or starting with a Guide heading. Its created_at is the baseline, and a
# commit is new when its author date is later (get_commit_range). Mirroring that
# rule keeps the published-review guard honest: the gate never starts PR-Agent
# on a push that PR-Agent would then decline to review.
#
# It fails OPEN: input it cannot read means run=true. Skipping a review has to
# be justified by the data; running one costs only inference.
#
# Usage:
#   pr-agent-push-gate.sh --comments FILE --commits FILE [--threshold N] [--sender-type TYPE]
#
# Output (stdout, for $GITHUB_OUTPUT): run=true|false, then reason=<text>.
# Exit: 0 with a decision; 2 on a usage error.

set -uo pipefail

usage() {
    cat <<'EOF'
Usage: pr-agent-push-gate.sh --comments FILE --commits FILE [--threshold N] [--sender-type TYPE]

  --comments FILE     the PR's issue comments, as GET /issues/N/comments returns them
  --commits FILE      the PR's commits, as GET /pulls/N/commits returns them
  --threshold N       new non-merge commits needed to review a push (default 3)
  --sender-type TYPE  the push event's sender.type; "Bot" is skipped, as PR-Agent skips it

Prints run=true|false and reason=<text>. Unreadable input prints run=true.
EOF
}

COMMENTS=""
COMMITS=""
THRESHOLD=3
SENDER_TYPE=""

while [ $# -gt 0 ]; do
    case "$1" in
        --comments) COMMENTS="${2:-}"; shift 2 ;;
        --commits) COMMITS="${2:-}"; shift 2 ;;
        --threshold) THRESHOLD="${2:-}"; shift 2 ;;
        --sender-type) SENDER_TYPE="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

if [ -z "$COMMENTS" ] || [ -z "$COMMITS" ]; then
    usage >&2
    exit 2
fi
case "$THRESHOLD" in
    ''|*[!0-9]*) printf -- '--threshold must be a whole number\n' >&2; exit 2 ;;
esac

decide() {
    printf 'run=%s\nreason=%s\n' "$1" "$2"
    exit 0
}

if [ "$SENDER_TYPE" = "Bot" ]; then
    decide false "a bot pushed, and PR-Agent skips bot pushes (push_trigger_ignore_bot_commits)"
fi

# The identities are exact lines, compared after trimming; the headings are
# prefixes of the whole body. Both as PR-Agent v0.45.0 matches them.
baseline=$(jq -r '
    def identity: . == "<!-- pr-agent:review:full -->" or . == "<!-- pr-agent:review:incremental -->";
    [ .[] | (.body // "") as $b
      | select(($b | split("\n")[:5] | map(gsub("^\\s+|\\s+$"; "")) | any(identity))
               or ($b | startswith("## PR Reviewer Guide"))
               or ($b | startswith("## Incremental PR Reviewer Guide"))) ]
    | last | .created_at // ""' "$COMMENTS" 2>/dev/null) \
    || decide true "the PR comments cannot be read, so the push is reviewed"

if [ -z "$baseline" ]; then
    decide true "no previous review, so PR-Agent reviews the whole PR"
fi

new=$(jq --arg since "$baseline" \
    '[ .[] | select((.parents | length) < 2) | select(.commit.author.date > $since) ] | length' \
    "$COMMITS" 2>/dev/null) \
    || decide true "the PR commits cannot be read, so the push is reviewed"

if [ "$new" -ge "$THRESHOLD" ]; then
    decide true "$new new commit(s) since the review of $baseline, at or past the threshold of $THRESHOLD"
fi
decide false "$new new commit(s) since the review of $baseline, below the threshold of $THRESHOLD; comment /review to ask for one now"
