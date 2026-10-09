#!/usr/bin/env bash

# pr-agent-route.sh: draw which pool member reviews this PR first (AI-045 AC9,
# #1923).
#
# The CI review pool is the NaN chain the preflight declares
# (scripts/pr-agent-model-preflight.sh) plus one Anthropic model. Every member
# that answered its probe is drawn with equal weight, the way `dotf spec review`
# draws from harness/reviewer-pool.json. A member that did not answer (dead,
# over quota, saturated, no credential) is not in the draw, so a busy NaN model
# hands its share to the others without anything measuring load.
#
# One PR-Agent attempt talks to ONE provider: PR-Agent's own fallback chain
# shares one transport, and NaN's (`LITELLM__CUSTOM_LLM_PROVIDER: openai`, its
# base URL) would misroute an Anthropic model. So the draw picks the provider of
# the first attempt and, within NaN, the model; the other provider, when one of
# its members answered, is the second attempt. The workflow runs that second
# attempt only on a measured absence of a published review.
#
# PR_AGENT_PROVIDER (a repository variable) overrides the draw:
#   draw (or unset)  equal weight over every member that answered
#   nan              NaN first, Anthropic second
#   anthropic        Anthropic first, NaN second
#   nan-only         NaN only; the Anthropic model is not even probed, so this
#                    is the switch that stops all spending on the Anthropic key
# A forced provider with no member that answered falls back to the draw, with a
# warning: forcing is for testing a route, and a review is still worth having.
# An unknown value fails: it may be a misspelt `nan-only`, and spending against
# a switch someone meant to turn off is the one outcome this must not produce.
#
# The key goes to curl on stdin (`-K -`), never on its argv.
#
# Env:
#   NAN_OUTCOME            the preflight step's outcome (`success` when a NaN model answered)
#   NAN_MODEL              the preflight's first NaN model that answered
#   NAN_FALLBACKS          JSON array of the other NaN models that answered, in declared order
#   ANTHROPIC_MODEL        the pool's Anthropic member, `anthropic/<id>`
#   PR_AGENT_ANTHROPIC_API_KEY  its key; empty means the member is unavailable (never printed)
#   PR_AGENT_PROVIDER      the override above
#   PR_AGENT_DRAW          tests only: the index to draw instead of a random one
#   ANTHROPIC_API_BASE     default https://api.anthropic.com
#   ROUTE_PROBE_TIMEOUT    seconds for the Anthropic probe, default 30
#   GITHUB_REPOSITORY      names the repository in the remedy
#   GITHUB_STEP_SUMMARY    when set, the draw is appended to it
#
# Output, as `key=value` lines to --output (default stdout):
#   first=nan|anthropic|none     the provider of the first attempt
#   second=nan|anthropic|none    the provider of the attempt after it
#   reviewer=<model>             the member drawn to review first
#   nan_model=<model>            the NaN attempt's model, when NaN is first or second
#   nan_fallbacks=<JSON array>   the NaN attempt's fallback chain
#   note=<text>                  why a member was out of the draw, for the final guard
#
# Exit: 0 routed (first may be `none`), 1 unknown PR_AGENT_PROVIDER, 2 usage.

set -uo pipefail

output="/dev/stdout"
while [ $# -gt 0 ]; do
    case "$1" in
        --output) output="${2:-}"; shift 2 ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
    esac
done
for tool in curl jq; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        printf '%s is required and was not found\n' "$tool" >&2
        exit 2
    fi
done
anthropic_model="${ANTHROPIC_MODEL:-}"
case "$anthropic_model" in
    anthropic/?*) ;;
    *) printf 'ANTHROPIC_MODEL must be anthropic/<id>, got: %s\n' "$anthropic_model" >&2; exit 2 ;;
esac

provider="${PR_AGENT_PROVIDER:-draw}"
case "$provider" in
    draw|nan|anthropic|nan-only) ;;
    *)
        printf '::error::PR_AGENT_PROVIDER is %s, which is none of draw, nan, anthropic or nan-only.' "$provider"
        printf ' Nothing is reviewed until it is fixed: a misspelt nan-only must not spend on the Anthropic key.\n'
        exit 1
        ;;
esac

# The NaN members that answered, in declared order.
nan=()
if [ "${NAN_OUTCOME:-}" = "success" ] && [ -n "${NAN_MODEL:-}" ]; then
    nan+=("$NAN_MODEL")
    while IFS= read -r m; do
        [ -n "$m" ] && nan+=("$m")
    done < <(jq -r '.[]?' <<<"${NAN_FALLBACKS:-[]}" 2>/dev/null)
fi

# probe_anthropic: prints the HTTP status, or 000 when nothing answered in time.
# 64 output tokens: any 2xx is an answer, and a truncated one costs the least.
probe_anthropic() {
    local body code
    body=$(jq -cn --arg m "${anthropic_model#anthropic/}" \
        '{model: $m, max_tokens: 64, messages: [{role: "user", content: "Reply with OK."}]}')
    if ! code=$(printf 'header = "x-api-key: %s"\n' "$PR_AGENT_ANTHROPIC_API_KEY" \
        | curl -sS -K - -o /dev/null -w '%{http_code}' -m "${ROUTE_PROBE_TIMEOUT:-30}" \
            -H 'anthropic-version: 2023-06-01' -H 'Content-Type: application/json' \
            -d "$body" "${ANTHROPIC_API_BASE:-https://api.anthropic.com}/v1/messages" 2>/dev/null); then
        code=000
    fi
    printf '%s' "${code:-000}"
}

note=""
anthropic=()
if [ "$provider" != "nan-only" ]; then
    if [ -z "${PR_AGENT_ANTHROPIC_API_KEY:-}" ]; then
        note="PR_AGENT_ANTHROPIC_API_KEY is not set, so ${anthropic_model} was out of the review pool."
        note="${note} Remedy: add ci:${GITHUB_REPOSITORY:-<repo>} to its consumers in dotfiles/secrets/registry.yaml,"
        note="${note} then run: dotf secrets sync ci --repo ${GITHUB_REPOSITORY:-<repo>}"
        printf '::warning::%s\n' "$note"
    else
        code=$(probe_anthropic)
        case "$code" in
            2??) anthropic+=("$anthropic_model") ;;
            401|403) note="${anthropic_model} answered HTTP ${code}: the key is refused. Rotate it: dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY" ;;
            404) note="${anthropic_model} answered HTTP 404: the model id is wrong. Fix it in .github/workflows/pr-agent.yml" ;;
            000) note="${anthropic_model} gave no answer within ${ROUTE_PROBE_TIMEOUT:-30}s and was out of the review pool for this run." ;;
            *) note="${anthropic_model} answered HTTP ${code} (rate limit, overload or spend limit) and was out of the review pool for this run." ;;
        esac
        if [ "${#anthropic[@]}" -eq 0 ]; then printf '::warning::%s\n' "$note"; fi
    fi
fi

# The candidates the draw picks from.
candidates=()
case "$provider" in
    draw) candidates=(${nan[@]+"${nan[@]}"} ${anthropic[@]+"${anthropic[@]}"}) ;;
    nan|nan-only) candidates=(${nan[@]+"${nan[@]}"}) ;;
    anthropic) candidates=(${anthropic[@]+"${anthropic[@]}"}) ;;
esac
if [ "${#candidates[@]}" -eq 0 ] && [ "$provider" != "draw" ] && [ "$provider" != "nan-only" ]; then
    printf '::warning::PR_AGENT_PROVIDER=%s, but no %s member answered; drawing from the rest of the pool.\n' \
        "$provider" "$provider"
    candidates=(${nan[@]+"${nan[@]}"} ${anthropic[@]+"${anthropic[@]}"})
fi

first=none second=none reviewer="" nan_model="" nan_fallbacks="[]"
if [ "${#candidates[@]}" -gt 0 ]; then
    n=${#candidates[@]}
    if [ -n "${PR_AGENT_DRAW:-}" ]; then
        case "$PR_AGENT_DRAW" in
            ''|*[!0-9]*) printf 'PR_AGENT_DRAW must be an index, got: %s\n' "$PR_AGENT_DRAW" >&2; exit 2 ;;
        esac
        if [ "$PR_AGENT_DRAW" -ge "$n" ]; then
            printf 'PR_AGENT_DRAW %s is outside the %s candidates\n' "$PR_AGENT_DRAW" "$n" >&2
            exit 2
        fi
        i=$PR_AGENT_DRAW
    else
        i=$((RANDOM % n))
    fi
    reviewer="${candidates[$i]}"
    case "$reviewer" in
        anthropic/*)
            first=anthropic
            if [ "${#nan[@]}" -gt 0 ]; then second=nan; fi
            ;;
        *)
            first=nan
            # Empty under nan-only, which never probes it.
            if [ "${#anthropic[@]}" -gt 0 ]; then second=anthropic; fi
            ;;
    esac
    # The NaN attempt starts on the drawn model when it was drawn, and on the
    # first NaN member otherwise; the rest stay behind it in declared order.
    if [ "${#nan[@]}" -gt 0 ]; then
        nan_model="${nan[0]}"
        if [ "$first" = "nan" ]; then nan_model="$reviewer"; fi
        nan_fallbacks=$(printf '%s\n' "${nan[@]}" \
            | jq -Rsc --arg m "$nan_model" 'split("\n") | map(select(. != "" and . != $m))')
    fi
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    # The backticks are Markdown, not command substitution.
    # shellcheck disable=SC2016
    {
        printf '### PR-Agent review pool\n\n'
        members=$(printf '`%s` ' ${nan[@]+"${nan[@]}"} ${anthropic[@]+"${anthropic[@]}"} | sed 's/ $//')
        printf 'Override: `%s`. Members that answered: %s.\n\n' "$provider" "${members:-none}"
        if [ "$first" = "none" ]; then
            printf 'No member answered, so no review attempt runs.\n'
        else
            printf 'Drawn to review first: `%s`. Second attempt, only if nothing is published: %s.\n' \
                "$reviewer" "$second"
        fi
        if [ -n "$note" ]; then printf '\n%s\n' "$note"; fi
    } >> "$GITHUB_STEP_SUMMARY"
fi
if [ "$first" != "none" ]; then
    printf '::notice::PR-Agent reviews first on %s (override: %s); second attempt: %s.\n' \
        "$reviewer" "$provider" "$second"
fi

{
    printf 'first=%s\nsecond=%s\nreviewer=%s\n' "$first" "$second" "$reviewer"
    printf 'nan_model=%s\nnan_fallbacks=%s\n' "$nan_model" "$nan_fallbacks"
    printf 'note=%s\n' "$note"
} >> "$output"
