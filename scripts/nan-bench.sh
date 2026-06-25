#!/usr/bin/env bash
# nan-bench.sh — latency + throughput benchmark across NaN chat models.
# Surfaces reasoning-chain overhead per model. Use to pick the right default
# for opencode and to validate provider responsiveness.
#
# Usage: ./scripts/nan-bench.sh [prompt]
#        (default prompt: "responde solo: pong")

set -euo pipefail

# NAN_API_KEY: injected by `dotf secrets run -- <this script>`, or self-fetched
# on demand via `dotf secrets show` (ADR-028 — never the ambient shell env).
if [ -z "${NAN_API_KEY:-}" ] && command -v dotf >/dev/null 2>&1; then
    NAN_API_KEY="$(dotf secrets show nan-api-key 2>/dev/null || true)"
    export NAN_API_KEY
fi

if [ -z "${NAN_API_KEY:-}" ]; then
    echo "ERROR: NAN_API_KEY not set. Run: dotf secrets run -- $0" >&2
    exit 1
fi

BASE_URL="${NAN_BASE_URL:-https://api.nan.builders/v1}"
PROMPT="${1:-responde solo: pong}"
MODELS=(qwen3.6 gemma4 deepseek-v4-flash)

printf '%-22s %8s %8s %10s %10s %8s %s\n' "model" "wall(s)" "ttfb(s)" "in_tokens" "out_tokens" "reason" "content"
printf '%-22s %8s %8s %10s %10s %8s %s\n' "----------------------" "-------" "-------" "----------" "----------" "------" "-------"

for model in "${MODELS[@]}"; do
    # Throttle to avoid NaN's 100 rpm / 5 concurrent limit when running back-to-back
    sleep 1
    body=$(jq -n --arg model "$model" --arg prompt "$PROMPT" \
        '{model: $model, messages: [{role: "user", content: $prompt}], max_tokens: 200}')

    # -w: wall time. ttfb = time_starttransfer (TLS+TCP+server-first-byte).
    response=$(curl -sS -m 60 \
        -w '\n__METRICS__:%{time_total}|%{time_starttransfer}' \
        -H "Authorization: Bearer $NAN_API_KEY" \
        -H "Content-Type: application/json" \
        -d "$body" \
        "$BASE_URL/chat/completions" 2>/dev/null) || {
        printf '%-22s %s\n' "$model" "  ERROR (curl exit)"
        continue
    }

    json="${response%$'\n__METRICS__:'*}"
    metrics="${response##*__METRICS__:}"
    wall="${metrics%|*}"
    ttfb="${metrics##*|}"

    content=$(echo "$json" | jq -r '.choices[0].message.content // "<null>"' 2>/dev/null | tr '\n' ' ' | cut -c1-30)
    in_tokens=$(echo "$json" | jq -r '.usage.prompt_tokens // 0' 2>/dev/null)
    out_tokens=$(echo "$json" | jq -r '.usage.completion_tokens // 0' 2>/dev/null)
    reason_tokens=$(echo "$json" | jq -r '.usage.completion_tokens_details.reasoning_tokens // 0' 2>/dev/null)

    printf '%-22s %8.2f %8.2f %10s %10s %8s %s\n' \
        "$model" "$wall" "$ttfb" "$in_tokens" "$out_tokens" "$reason_tokens" "$content"
done

echo ""
echo "Notes:"
echo "  wall      = total request time end-to-end (TLS+TCP+inference+streaming-close)"
echo "  ttfb      = time-to-first-byte (TLS+TCP+server-first-token)"
echo "  reason    = reasoning_tokens — invisible chain-of-thought billed but not returned"
echo "  content   = first 30 chars of returned answer (truncated)"
echo ""
echo "If wall ≫ ttfb: model spent time generating tokens (normal)."
echo "If reason > 0:  model is doing reasoning chain — adds latency. Disable for fast paths."
echo "If wall on deepseek-v4-flash ≫ qwen3.6: confirms reasoning-chain overhead on default."
