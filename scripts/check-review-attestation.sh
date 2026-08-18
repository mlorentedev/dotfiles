#!/usr/bin/env bash

# check-review-attestation.sh: GUARD-002 (#906).
#
# Answers one question — did a review actually happen? — and refuses to let the
# answer be silence.
#
# The defect this exists for: on 2026-08-16, PRs #1007, #1009 and #1013 each
# carried a CodeRabbit comment reading "Review limit reached — we couldn't start
# this review", zero entries in reviews[], and `gh pr checks` reporting
# `CodeRabbit  pass`. Three PRs, none reviewed, all green. The one mechanical
# signal available asserted the opposite of the truth.
#
# States, decided by CONTENT rather than by check status:
#
#   attested  a review happened (bot or human, not the author)   -> exit 0
#   exempt    the diff carries nothing a review could act on     -> exit 0
#   disclosed no review, but the escape is declared in full      -> exit 0
#   declined  a reviewer said it could not review                -> exit 1
#   pending   no reviewer output yet                             -> exit 1
#
# `declined` and `pending` both refuse, and are reported distinctly on purpose.
# Collapsing them would repeat the second half of #988, where an error path
# discarded the HTTP status and showed `invalid character 'I'` instead of
# `HTTP 500` — the diagnostic threw away the diagnosis, which is why that bug
# survived weeks of being looked at.
#
# This check OBSERVES and never acts. It will not retrigger a rate-limited
# reviewer: a gate that spent the quota it measures would perturb its own
# subject, which is the exact shape of #988's `/status` probe poisoning the
# reads it was verifying. (Measured moot as well — an explicit re-review command
# does not reclaim a slot while the quota is spent.)
#
# No reviewer is named anywhere in this file, deliberately and under test. Every
# vendor-specific string lives in the registry, including the prose explaining
# each one; a comment naming a vendor here is how "the config is authoritative"
# quietly stops being true.
#
# Usage:
#   check-review-attestation.sh [--pr N] [--payload FILE] [--config FILE] [--quiet]
#
# Env:
#   REVIEW_ATTESTATION_CONFIG   override the reviewer registry path
#
# Exit:
#   0  attested or disclosed
#   1  declined or pending — not reviewed, and not declared as such
#   2  usage/setup error, or the PR state could not be read (FAILS CLOSED)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONFIG="${REVIEW_ATTESTATION_CONFIG:-$REPO_ROOT/harness/review-attestation.json}"
PR=""
PAYLOAD=""
QUIET=0

usage() {
    cat <<'EOF'
Usage: check-review-attestation.sh [options]

Reports whether a pull request has actually been reviewed.

Options:
  --pr N            PR number to inspect (fetched via gh)
  --payload FILE    read a pre-fetched `gh pr view --json ...` payload instead
                    (offline; this is how the bats fixtures drive it)
  --config FILE     reviewer registry (default: harness/review-attestation.json)
  --quiet           suppress the human-readable report; exit code only
  -h, --help        show this help

Exit: 0 attested|disclosed, 1 declined|pending, 2 setup error / unreadable.
EOF
}

# `shift 2` on a flag given without a value fails under `set -e`, and the shell
# exits 1 — which this script's own contract defines as "not reviewed" rather
# than "setup error". A typo would therefore have been reported as a verdict.
need_value() {
    [ -n "${2:-}" ] || { printf '[ERROR] %s requires a value\n' "$1" >&2; usage >&2; exit 2; }
}

while [ $# -gt 0 ]; do
    case "$1" in
        --pr)      need_value "$1" "${2:-}"; PR="$2"; shift 2 ;;
        --payload) need_value "$1" "${2:-}"; PAYLOAD="$2"; shift 2 ;;
        --config)  need_value "$1" "${2:-}"; CONFIG="$2"; shift 2 ;;
        --quiet)   QUIET=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) printf '[ERROR] Unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
    esac
done

say() { [ "$QUIET" -eq 1 ] || printf '%s\n' "$1"; }

# Every bail-out below is exit 2, never 0. `pattern-verification-fails-toward-
# unproven`: for a gate, "I could not determine" must never be delivered as
# "fine" — that IS the bug being fixed, and it would be absurd to reproduce it
# in the fix.
die_unreadable() {
    printf '[FAIL] review attestation: %s\n' "$1" >&2
    printf '       Cannot determine whether this PR was reviewed, so it is NOT treated as reviewed.\n' >&2
    exit 2
}

command -v jq >/dev/null 2>&1 || die_unreadable "jq is not installed"
[ -f "$CONFIG" ] || die_unreadable "reviewer registry not found: $CONFIG"
jq -e . "$CONFIG" >/dev/null 2>&1 || die_unreadable "reviewer registry is not valid JSON: $CONFIG"

# Shape, not just syntax. Valid JSON that is the wrong shape — a bare array, a
# reviewers entry with no login, an escape with no section — parsed fine and
# then silently produced empty strings downstream, which classify as "no
# reviewer declined" and "no escape configured". A malformed registry would
# have made the gate quietly more permissive, which is the direction it must
# never fail in.
jq -e '
    type == "object"
    and (.reviewers | type == "array")
    and (.reviewers | length > 0)
    and (.reviewers | all(
            (.login | type == "string" and length > 0)
        and (.declined_markers | type == "array")
        and (.declined_markers | all(type == "string" and length > 0))
        and (.review_markers   | type == "array")
        and (.review_markers   | all(type == "string" and length > 0))
    ))
    and (.escape | type == "object")
    and (.escape.label   | type == "string" and length > 0)
    and (.escape.section | type == "string" and length > 0)
' "$CONFIG" >/dev/null 2>&1 \
    || die_unreadable "reviewer registry is valid JSON but the wrong shape (needs reviewers[].login, reviewers[].declined_markers[], reviewers[].review_markers[], escape.label, escape.section): $CONFIG"

# --- acquire the payload -------------------------------------------------------
# One shape either way, so the offline fixture path and the live path exercise
# exactly the same classifier. A test that drove a different code path than
# production would be evidence about the test.
FIELDS="number,state,labels,body,comments,reviews,author,files"

if [ -n "$PAYLOAD" ]; then
    [ -f "$PAYLOAD" ] || die_unreadable "payload file not found: $PAYLOAD"
    [ -s "$PAYLOAD" ] || die_unreadable "payload file is empty: $PAYLOAD"
    PR_JSON="$(cat "$PAYLOAD")"
elif [ -n "$PR" ]; then
    command -v gh >/dev/null 2>&1 || die_unreadable "gh is not installed and no --payload was given"
    PR_JSON="$(gh pr view "$PR" --json "$FIELDS" 2>/dev/null)" \
        || die_unreadable "could not read PR #$PR (unauthenticated, or no such PR)"
else
    command -v gh >/dev/null 2>&1 || die_unreadable "gh is not installed and no --payload was given"
    PR_JSON="$(gh pr view --json "$FIELDS" 2>/dev/null)" \
        || die_unreadable "no PR for the current branch, and none was named with --pr"
fi

printf '%s' "$PR_JSON" | jq -e . >/dev/null 2>&1 \
    || die_unreadable "PR payload is not valid JSON"

# --- attested? -----------------------------------------------------------------
# Any entry in reviews[] counts, from a bot or a human alike: the question is
# whether a review happened, never who performed it.
#
# Except the author's own. Self-review is the one case where a non-empty
# reviews[] does not mean anyone independent looked — and independence is the
# whole point of the artifact. This mirrors harness/reviewer-pool.json's
# standing rule that the reviewer must not be the implementer.
PR_AUTHOR="$(printf '%s' "$PR_JSON" | jq -r '.author.login // ""')"
INDEPENDENT="$(printf '%s' "$PR_JSON" | jq -r --arg a "$PR_AUTHOR" \
    '[.reviews[]? | select((.author.login // "") != $a)] | length')"

if [ "$INDEPENDENT" -gt 0 ]; then
    REVIEWERS="$(printf '%s' "$PR_JSON" | jq -r --arg a "$PR_AUTHOR" \
        '[.reviews[]? | select((.author.login // "") != $a) | .author.login // "?"] | unique | join(", ")')"
    say "[OK] attested — reviewed by: $REVIEWERS"
    exit 0
fi

# --- attested by a comment-shaped review? --------------------------------------
# A reviewer that does not use the reviews API. PR-Agent (#786) runs under
# GITHUB_TOKEN as github-actions and publishes through the COMMENTS api, so
# reviews[] is empty on a PR it genuinely reviewed — measured on #1037 and #1042,
# both carrying a real Guide and both reading as unattested (#1045). The gate
# built to make green mean "reviewed" was calling the repo's own reviewer
# unreviewed, which is #906's failure with the sign flipped.
#
# Matched on a DECLARED (login, marker) pair, never on "a bot commented": the
# latter would let a labeler or release bot attest through the comments door,
# which is exactly #1033's defect arriving by another route.
#
# Runs BEFORE the declined check on purpose. Both can be true at once — PR-Agent
# reviews while CodeRabbit's quota is spent — and the gate asks whether a review
# happened, so one that did outranks one that did not.
ATTESTED_BY=""
ATTESTED_MARKER=""
while IFS=$'\t' read -r login marker; do
    [ -n "$login" ] || continue
    hit="$(printf '%s' "$PR_JSON" | jq -r --arg l "$login" --arg m "$marker" --arg a "$PR_AUTHOR" '
        def norm: (. // "") | ascii_downcase | sub("\\[bot\\]$"; "");
        [ .comments[]?
          | select((.author.login | norm) == ($l | norm))
          | select((.author.login | norm) != ($a | norm))
          | select((.body // "") | contains($m))
        ] | length')"
    if [ "$hit" -gt 0 ]; then
        ATTESTED_BY="$login"
        ATTESTED_MARKER="$marker"
        break
    fi
done < <(jq -r '.reviewers[]? | . as $r | ($r.review_markers[]? | [$r.login, .] | @tsv)' "$CONFIG")

if [ -n "$ATTESTED_BY" ]; then
    say "[OK] attested — reviewed by: $ATTESTED_BY (comment-shaped review: \"$ATTESTED_MARKER\")"
    exit 0
fi

# --- exempt? -------------------------------------------------------------------
# Some changes carry nothing a review could act on. A release commit restates
# commits that were each reviewed when they landed; reporting `declined` on it
# is not strictness, it is a false statement, and a gate that is wrong about a
# whole class of change teaches its readers to discount it — the failure this
# gate exists to prevent, arrived at from the other side.
#
# Matched by DIFF SIGNATURE and nothing else. Not by author: the tools that open
# such changes run under a human token, so an author rule never fires (verified
# on a live one, whose author reads as a person). Not by branch: any branch can
# be named to match. Every changed file must appear in the declared set, so
# touching one other file ends the exemption — nothing to borrow, nothing to
# game.
#
# The signature itself, and the name of whatever produces it, live in the
# registry. This file names no tool and no path, which is a rule it already had
# and which the first draft of this very comment broke — caught by the test that
# enforces it, not by me.
#
# Checked AFTER both attestation paths: a real review is better evidence than an
# exemption, and a release that did get reviewed should say so.
EXEMPT_NAME="$(printf '%s' "$PR_JSON" | jq -r --slurpfile cfg "$CONFIG" '
    ($cfg[0].exempt.signatures // []) as $sigs
    | [.files[]?.path] as $changed
    | if ($changed | length) == 0 then ""
      else
        [ $sigs[]
          | select(
              (($changed - .files) | length == 0)
              and ((.files - $changed) | length == 0)
            )
          | .name
        ] | first // ""
      end')"

if [ -n "$EXEMPT_NAME" ]; then
    say "[OK] exempt — every changed file is in the \"$EXEMPT_NAME\" signature; nothing here is reviewable"
    exit 0
fi

# --- declined? -----------------------------------------------------------------
# A reviewer posted, but what it posted says no review ran. Told apart by the
# vendor's own machine marker, never by prose: a review names files, lines or
# claims; a notice talks about the review itself.
DECLINED_BY=""
DECLINED_MARKER=""
while IFS=$'\t' read -r login marker; do
    [ -n "$login" ] || continue
    hit="$(printf '%s' "$PR_JSON" | jq -r --arg l "$login" --arg m "$marker" '
        def norm: (. // "") | ascii_downcase | sub("\\[bot\\]$"; "");
        [.comments[]? | select((.author.login | norm) == ($l | norm)) | select((.body // "") | contains($m))] | length')"
    if [ "$hit" -gt 0 ]; then
        DECLINED_BY="$login"
        DECLINED_MARKER="$marker"
        break
    fi
done < <(jq -r '.reviewers[]? | . as $r | ($r.declined_markers[]? | [$r.login, .] | @tsv)' "$CONFIG")

# --- the declared escape -------------------------------------------------------
# BOTH halves, never one. A label alone is a click; a rationale alone is a note
# nobody agreed to. Together they are a decision with a name on it.
ESCAPE_LABEL="$(jq -r '.escape.label // ""' "$CONFIG")"
ESCAPE_SECTION="$(jq -r '.escape.section // ""' "$CONFIG")"

HAS_LABEL="$(printf '%s' "$PR_JSON" | jq -r --arg l "$ESCAPE_LABEL" \
    '[.labels[]? | select((.name // "") == $l)] | length')"

# Non-empty means non-empty: a heading with nothing under it is the same
# disclosure as no heading at all, and is the likelier way to satisfy this by
# accident.
# `\r` is stripped and trailing whitespace trimmed BEFORE the heading is matched.
# A PR body edited in GitHub's web UI arrives CRLF-terminated, which left every
# line as "## Unreviewed merge rationale\r" and made the exact-match index fail —
# so a PR carrying both halves of the escape was refused, and told to add a
# section it already had. Measured: identical payloads, LF exits 0 and CRLF
# exits 1. The repo has met this class before (`Set-Content` writing CRLF into
# `.sh` files, `.gitattributes eol=lf`); it reappears at every boundary where
# text crosses from a Windows-flavoured producer.
RATIONALE="$(printf '%s' "$PR_JSON" | jq -r --arg s "$ESCAPE_SECTION" '
    (.body // "")
    | gsub("\r"; "")
    | split("\n")
    | map(sub("[ \t]+$"; ""))
    | (index($s)) as $i
    | if $i == null then ""
      else .[$i+1:]
        | (map(select(startswith("## "))) | first) as $next
        | (if $next == null then . else .[0:index($next)] end)
        | join("\n")
      end
' | sed 's/[[:space:]]*$//' | tr -d '[:space:]')"

if [ "$HAS_LABEL" -gt 0 ] && [ -n "$RATIONALE" ]; then
    if [ -n "$DECLINED_BY" ]; then
        say "[OK] disclosed — no review ran ($DECLINED_BY declined), and the merge is declared: \"$ESCAPE_LABEL\" + \"$ESCAPE_SECTION\""
    else
        say "[OK] disclosed — not reviewed, and the merge is declared: \"$ESCAPE_LABEL\" + \"$ESCAPE_SECTION\""
    fi
    exit 0
fi

# --- refuse, and say which refusal it is ---------------------------------------
if [ -n "$DECLINED_BY" ]; then
    printf '[FAIL] declined — %s posted a notice that no review ran (marker: "%s")\n' "$DECLINED_BY" "$DECLINED_MARKER" >&2
    printf '       A notice that no review ran leaves this PR UNREVIEWED. Its checks may still be green:\n' >&2
    printf '       the reviewer reports its own status, and a skipped review is not a failed one.\n' >&2
else
    printf '[FAIL] pending — no reviewer output on this PR yet\n' >&2
fi

printf '\n       Options:\n' >&2
printf '         (a) Get it reviewed, then re-run (this gate re-runs on new comments).\n' >&2
printf '         (b) Declare the unreviewed merge: add the "%s" label AND a non-empty\n' "$ESCAPE_LABEL" >&2
printf '             "%s" section to the PR body.\n' "$ESCAPE_SECTION" >&2
printf '\n       Proceeding on an unreviewed PR is allowed. Proceeding silently is not.\n' >&2
exit 1
