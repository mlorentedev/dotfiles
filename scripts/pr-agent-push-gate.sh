#!/usr/bin/env bash

# pr-agent-push-gate.sh: decide whether a push to a pull request is reviewed (TOOL-023).
#
# pr-agent used to run a full /review on every push (#1053). In one week that was
# 39 reviews across 18 PRs, and up to 7 on one PR. Each review was one request to
# a NaN pool of 5 shared slots, and each re-review reopened the triage queue. A
# push is now reviewed incrementally (/review -i), and only once THRESHOLD
# non-merge commits are newer than the previous review. Below the bound, or
# whenever the baseline cannot be trusted, it asks for a FULL review instead
# (mode=full): PR-Agent's own incremental range is not safe to hand a review it
# did not earn.
#
# "The previous review" is the one PR-Agent itself picks (v0.45.0,
# github_provider.get_previous_review and utils.comment_matches_identity): the
# LAST issue comment carrying a review identity line within its first 5 lines,
# or starting with a Guide heading — by BODY alone, with no author check. This
# gate only trusts a review posted by github-actions[bot] (CWE-345: on a public
# repo anyone can comment the same marker text), and if a comment from anyone
# else, newer than that review, still carries one, the push is reviewed in
# full rather than incrementally: PR-Agent would pick that forged comment as
# ITS OWN baseline too, and /review never picks a previous review at all.
#
# A commit counts as new past that review's HEAD SHA — the persistent state
# block's last_run.head_sha (`<!-- pr-agent-review-state:v1 {...} -->`, present
# on a full review) — by POSITION in the commit list, which survives a rebase
# that preserves author dates (this repo rebases with --onto routinely). If
# that sha is no longer in the PR (a rebase or force-push rewrites it too), the
# range cannot be trusted: review in full. If no state block carries a
# head_sha at all — true of every INCREMENTAL review, which never writes one
# (_review_finding_state_enabled returns False once self.incremental.is_incremental) —
# this falls back to the original author-date count, so a PR whose latest
# review was incremental is exposed to the same rebase blind spot until its
# next FULL one.
#
# It fails OPEN: input it cannot read means run=true, mode=full. Skipping a
# review has to be justified by the data; running one costs only inference.
#
# Usage:
#   pr-agent-push-gate.sh --comments FILE --commits FILE [--threshold N] [--sender-type TYPE]
#
# Output (stdout, for $GITHUB_OUTPUT): run=true|false, then reason=<text>, and
# for run=true a third line, mode=full|incremental — full whenever the
# baseline is not one this gate can vouch for, incremental once it is.
# Exit: 0 with a decision; 2 on a usage error.

set -uo pipefail

usage() {
    cat <<'EOF'
Usage: pr-agent-push-gate.sh --comments FILE --commits FILE [--threshold N] [--sender-type TYPE]

  --comments FILE     the PR's issue comments, as GET /issues/N/comments returns them
  --commits FILE      the PR's commits, as GET /pulls/N/commits returns them
  --threshold N       new non-merge commits needed to review a push (default 3)
  --sender-type TYPE  the push event's sender.type; "Bot" is skipped, as PR-Agent skips it

Prints run=true|false, reason=<text>, and (when run=true) mode=full|incremental.
Unreadable input prints run=true, mode=full.
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
    if [ -n "${3:-}" ]; then
        printf 'run=%s\nreason=%s\nmode=%s\n' "$1" "$2" "$3"
    else
        printf 'run=%s\nreason=%s\n' "$1" "$2"
    fi
    exit 0
}

if [ "$SENDER_TYPE" = "Bot" ]; then
    decide false "a bot pushed, and PR-Agent skips bot pushes (push_trigger_ignore_bot_commits)"
fi

# Shared predicate: true when a comment body carries a review identity (in its
# first 5 lines, compared after trimming) or either Guide heading (a prefix of
# the whole body), as PR-Agent v0.45.0 matches them (comment_matches_identity).
# shellcheck disable=SC2016  # single-quoted on purpose: this is jq source, $b is its variable
JQ_IS_REVIEW='
    def identity: . == "<!-- pr-agent:review:full -->" or . == "<!-- pr-agent:review:incremental -->";
    def is_review: (.body // "") as $b
      | ($b | split("\n")[:5] | map(gsub("^\\s+|\\s+$"; "")) | any(identity))
        or ($b | startswith("## PR Reviewer Guide"))
        or ($b | startswith("## Incremental PR Reviewer Guide"));
'

# Only github-actions[bot] may set the baseline (CWE-345): unfiltered, a
# comment from anyone else carrying the marker text stood in for a review that
# never happened.
baseline_json=$(jq -c "$JQ_IS_REVIEW"'
    [ .[] | select((.user.login // "") == "github-actions[bot]") | select(is_review) ]
    | last // empty' "$COMMENTS" 2>/dev/null) \
    || decide true "the PR comments cannot be read, so the push is reviewed" full

if [ -z "$baseline_json" ]; then
    decide true "no previous review, so PR-Agent reviews the whole PR" full
fi
baseline=$(printf '%s' "$baseline_json" | jq -r '.created_at // ""')

# PR-Agent's own get_previous_review has no author check either, so a forged
# comment newer than the real review would become ITS baseline too, not just
# this gate's: reviewing in full sidesteps that entirely.
forged=$(jq --arg since "$baseline" "$JQ_IS_REVIEW"'
    [ .[] | select((.user.login // "") != "github-actions[bot]") | select(is_review)
      | select(.created_at > $since) ] | length' "$COMMENTS" 2>/dev/null) \
    || decide true "the PR comments cannot be read, so the push is reviewed" full
if [ "$forged" -gt 0 ]; then
    decide true "a review marker from a non-bot commenter is newer than the review of $baseline by github-actions[bot]; PR-Agent's own incremental baseline is not author-checked either, so this is reviewed in full" full
fi

# The state block is the last thing PR-Agent appends to a full review's body
# (append_review_state), so everything after the marker, with the trailing
# "-->" trimmed, is its JSON verbatim.
state_tail=$(printf '%s' "$baseline_json" | jq -r '
    (.body // "") as $b
    | if ($b | test("pr-agent-review-state:v1"))
      then ($b | split("pr-agent-review-state:v1") | last
                | sub("^\\s*"; "") | sub("\\s*-->\\s*$"; ""))
      else empty end' 2>/dev/null) || state_tail=""

# state_tail is already the JSON text itself (not a string containing it), so
# jq parses it directly as input; malformed JSON is a parse error here, caught
# the same way as any other unreadable input, and falls back below.
head_sha=""
if [ -n "$state_tail" ]; then
    head_sha=$(printf '%s' "$state_tail" | jq -r '.last_run.head_sha // empty' 2>/dev/null) \
        || head_sha=""
fi

if [ -n "$head_sha" ]; then
    sha_index=$(jq -r --arg sha "$head_sha" '[ .[].sha ] | index($sha) // -1' "$COMMITS" 2>/dev/null) \
        || decide true "the PR commits cannot be read, so the push is reviewed" full
    if [ "$sha_index" = "-1" ]; then
        decide true "the reviewed commit ($head_sha) is no longer in this PR, most likely a rebase or force-push" full
    fi
    new=$(jq --argjson idx "$sha_index" \
        '[ .[($idx + 1):][] | select((.parents | length) < 2) ] | length' \
        "$COMMITS" 2>/dev/null) \
        || decide true "the PR commits cannot be read, so the push is reviewed" full
    if [ "$new" -ge "$THRESHOLD" ]; then
        decide true "$new new commit(s) since the reviewed commit $head_sha, at or past the threshold of $THRESHOLD" incremental
    fi
    decide false "$new new commit(s) since the reviewed commit $head_sha, below the threshold of $THRESHOLD; comment /review to ask for one now"
fi

# No state block, or one without a head_sha (every incremental review, and any
# full review whose block does not carry one): fall back to counting by date.
new=$(jq --arg since "$baseline" \
    '[ .[] | select((.parents | length) < 2) | select(.commit.author.date > $since) ] | length' \
    "$COMMITS" 2>/dev/null) \
    || decide true "the PR commits cannot be read, so the push is reviewed" full

if [ "$new" -ge "$THRESHOLD" ]; then
    decide true "$new new commit(s) since the review of $baseline, at or past the threshold of $THRESHOLD" incremental
fi
decide false "$new new commit(s) since the review of $baseline, below the threshold of $THRESHOLD; comment /review to ask for one now"
