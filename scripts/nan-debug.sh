#!/usr/bin/env bash
# nan-debug.sh — query NaN deepseek-v4-flash and print reasoning + answer
# separately. Workaround: opencode TUI drops non-standard reasoning_content
# fields silently, so the chain-of-thought is invisible inside `oc`.
#
# Usage:  dbg "tu pregunta"
#         dbg -m qwen3.6 "tu pregunta"      # otro modelo (raro tendrá reasoning)
#         dbg -f long-prompt.txt            # leer prompt de fichero

set -euo pipefail

MODEL="deepseek-v4-flash"
PROMPT_FILE=""
ARGS=()

while [ $# -gt 0 ]; do
    case "$1" in
        -m|--model) MODEL="$2"; shift 2 ;;
        -f|--file)  PROMPT_FILE="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

if [ -n "$PROMPT_FILE" ]; then
    PROMPT="$(cat "$PROMPT_FILE")"
elif [ ${#ARGS[@]} -gt 0 ]; then
    PROMPT="${ARGS[*]}"
else
    echo "usage: dbg [-m MODEL] [-f FILE] <prompt...>" >&2
    exit 1
fi

# NAN_API_KEY: injected by `dotf secrets run`, or self-fetched via `dotf secrets
# show` on demand (ADR-028 — never the ambient shell env).
if [ -z "${NAN_API_KEY:-}" ] && command -v dotf >/dev/null 2>&1; then
    NAN_API_KEY="$(dotf secrets show NAN_API_KEY 2>/dev/null || true)"
    export NAN_API_KEY
fi
[ -z "${NAN_API_KEY:-}" ] && { echo "ERROR: NAN_API_KEY not set. Run: dotf secrets run -- $0" >&2; exit 1; }

BASE_URL="${NAN_BASE_URL:-https://api.nan.builders/v1}"

# ANSI colors: dim for reasoning, bold for answer
DIM=$'\033[2;36m'   # dim cyan
BOLD=$'\033[1;32m'  # bold green
RST=$'\033[0m'

body=$(jq -n --arg model "$MODEL" --arg prompt "$PROMPT" \
    '{model: $model, messages: [{role: "user", content: $prompt}], max_tokens: 2000, temperature: 0.6, top_p: 0.95}')

# Auth token via -K stdin config, not -H on argv (argv is world-readable
# in /proc/<pid>/cmdline). Unquoted heredoc so $NAN_API_KEY expands.
response=$(curl -sS -m 120 \
    -K - \
    -H "Content-Type: application/json" \
    -d "$body" \
    "$BASE_URL/chat/completions" <<CURL_CFG
header = "Authorization: Bearer $NAN_API_KEY"
CURL_CFG
) || { echo "curl failed" >&2; exit 1; }

reasoning=$(echo "$response" | jq -r '.choices[0].message.reasoning_content // .choices[0].message.provider_specific_fields.reasoning // ""')
answer=$(echo "$response" | jq -r '.choices[0].message.content // "<no content>"')
out_tok=$(echo "$response" | jq -r '.usage.completion_tokens // 0')
reason_tok=$(echo "$response" | jq -r '.usage.completion_tokens_details.reasoning_tokens // 0')

if [ -n "$reasoning" ] && [ "$reasoning" != "null" ]; then
    printf '%s┌─ reasoning (model: %s, %s tokens) ─%s\n' "$DIM" "$MODEL" "$reason_tok" "$RST"
    printf '%s│ %s%s\n' "$DIM" "$reasoning" "$RST" | sed "s/$/$RST/"
    printf '%s└─%s\n\n' "$DIM" "$RST"
else
    printf '%s(no reasoning in response — model %s doesn'\''t expose it)%s\n\n' "$DIM" "$MODEL" "$RST"
fi

printf '%s┌─ answer (%s tokens) ─%s\n' "$BOLD" "$out_tok" "$RST"
printf '%s\n' "$answer"
printf '%s└─%s\n' "$BOLD" "$RST"
