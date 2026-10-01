#!/usr/bin/env bash

# model-canary.sh: run `dotf harness canary` and keep ONE issue in step with it
# (AI-045 AC8, #1860).
#
# NaN retired `mimo-v2.5` on 2026-09-30 and the first thing to notice was a PR
# whose review never came. The canary probes every bound model daily instead,
# and this keeps a single labelled issue current:
#
#   canary exit 0  every model answered (or is only over quota): close the issue
#                  if one is open, with a comment saying so
#   canary exit 1  a model was refused or stayed unavailable: open the issue, or
#                  rewrite the open one's body in place (one issue, never one
#                  per day), and exit 1 so the scheduled run is red
#   anything else  the canary could not look: exit with its status and leave the
#                  issue untouched, because it says nothing about the models
#
# The classification lives here rather than in the workflow's `run:` block,
# because Actions runs that block under `bash -e`, which would make the exit-1
# branch unreachable (BUG-063). bats drives it with a stub `gh` and a stub
# canary.
#
# Usage: model-canary.sh <dotf-binary> [repo-root]
# Env:   GH_TOKEN and GH_REPO for gh; NAN_API_KEY for the canary;
#        GITHUB_STEP_SUMMARY and GITHUB_SERVER_URL/GITHUB_RUN_ID when set.

set -u

if [ $# -lt 1 ]; then
    printf 'usage: %s <dotf-binary> [repo-root]\n' "${0##*/}" >&2
    exit 2
fi
dotf_bin=$1
repo_root=${2:-.}

for tool in gh mktemp; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        printf '%s is required and was not found\n' "$tool" >&2
        exit 2
    fi
done
if [ -z "${GH_REPO:-}" ]; then
    printf 'GH_REPO is not set\n' >&2
    exit 2
fi

label="model-canary"
title="NaN model canary: a bound model is not answering"

report=$(mktemp)
body=$(mktemp)
trap 'rm -f "$report" "$body"' EXIT

# `|| rc=$?`, never `cmd; rc=$?`: under the `bash -e` Actions injects, the
# canary's exit 1 would end this script on the spot (BUG-063).
rc=0
"$dotf_bin" harness canary --repo-root "$repo_root" --report "$report" || rc=$?

if [ -n "${GITHUB_STEP_SUMMARY:-}" ] && [ -s "$report" ]; then
    cat "$report" >> "$GITHUB_STEP_SUMMARY"
fi

if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
    printf '::error::the canary could not run (exit %s); the canary issue is left as it is\n' "$rc"
    exit "$rc"
fi

if ! existing=$(gh issue list --state open --label "$label" --json number --jq '.[0].number // empty'); then
    printf '::error::could not list open %s issues\n' "$label"
    exit 2
fi

run_url=""
if [ -n "${GITHUB_SERVER_URL:-}" ] && [ -n "${GITHUB_RUN_ID:-}" ]; then
    run_url="${GITHUB_SERVER_URL}/${GH_REPO}/actions/runs/${GITHUB_RUN_ID}"
fi

if [ "$rc" -eq 0 ]; then
    if [ -n "$existing" ]; then
        gh issue comment "$existing" --body "Every bound NaN model answered on $(date -u +%F)${run_url:+ ($run_url)}. Closing." >/dev/null &&
            gh issue close "$existing" >/dev/null || exit 2
        printf '::notice::every bound model answered; closed #%s\n' "$existing"
    fi
    exit 0
fi

{
    cat "$report"
    printf '\nLast checked %s%s. This issue is rewritten by every failing run and closed by the first run in which every model answers.\n' \
        "$(date -u +%F)" "${run_url:+ by $run_url}"
    # shellcheck disable=SC2016 # the backticks are Markdown code spans, not expansions
    printf '\nFiled by `.github/workflows/model-canary.yml`. Locally: `dotf secrets run --only NAN_API_KEY -- dotf harness canary`.\n'
} > "$body"

if [ -n "$existing" ]; then
    # REST, not `gh issue edit`: the CLI's edit path queries the retired
    # projectCards field and fails.
    gh api -X PATCH "repos/${GH_REPO}/issues/${existing}" -F "body=@${body}" >/dev/null || exit 2
    printf '::warning::a bound NaN model is not answering; updated #%s\n' "$existing"
else
    gh label create "$label" --description "A NaN model this repository binds is not answering" \
        --color D93F0B --force >/dev/null || exit 2
    gh issue create --title "$title" --label "$label" --body-file "$body" >/dev/null || exit 2
    printf '::warning::a bound NaN model is not answering; opened a %s issue\n' "$label"
fi
exit 1
