#!/usr/bin/env bash
# nan-quality-bench.sh — quality A/B/C benchmark across NaN chat models.
# Sends the same set of representative prompts to qwen3.6, gemma4, deepseek-v4-flash
# and dumps responses side-by-side. Use to choose your default by quality not latency.
#
# Usage: ./scripts/nan-quality-bench.sh [output_dir]
#        default output_dir = /tmp/nan-bench-<timestamp>

set -euo pipefail

# Load secrets
if [ -z "${NAN_API_KEY:-}" ]; then
    [ -f scripts/load-secrets.sh ] && . scripts/load-secrets.sh && secrets_refresh >/dev/null 2>&1
fi
[ -z "${NAN_API_KEY:-}" ] && { echo "ERROR: NAN_API_KEY not set" >&2; exit 1; }

OUT="${1:-/tmp/nan-bench-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$OUT"
BASE_URL="${NAN_BASE_URL:-https://api.nan.builders/v1}"
MODELS=(qwen3.6 gemma4 deepseek-v4-flash)

# Representative prompts: real workloads, not toy.
declare -A PROMPTS=(
    [01-code-review]="Revisa este código Go y di si tiene problemas de concurrencia. Sé conciso:
\`\`\`go
type Counter struct { count int }
func (c *Counter) Inc() { c.count++ }
func main() {
    c := &Counter{}
    for i := 0; i < 1000; i++ { go c.Inc() }
    time.Sleep(time.Second)
    fmt.Println(c.count)
}
\`\`\`"
    [02-architecture]="Diseña en 3 párrafos la arquitectura de un sistema de pagos que: (1) acepta tarjeta y transferencia, (2) procesa 1000 tps, (3) cumple PCI-DSS. Habla de servicios, base de datos, y un trade-off importante."
    [03-refactor]="Refactoriza esta función Python para mejor legibilidad SIN cambiar el comportamiento. Devuelve solo el código:
\`\`\`python
def f(l):
    r=[]
    for i in range(len(l)):
        if l[i]%2==0 and l[i]>10:
            r.append(l[i]*2)
    return r
\`\`\`"
    [04-debug]="Tengo un test que falla intermitentemente solo en CI, nunca local. Es un test de integración con Postgres. ¿Cuáles son las 3 causas más probables y cómo las debugeo? Sé conciso."
)

echo "Output dir: $OUT"
echo "Models: ${MODELS[*]}"
echo "Prompts: ${#PROMPTS[@]}"
echo ""

# Save prompts
for name in "${!PROMPTS[@]}"; do
    printf '%s\n' "${PROMPTS[$name]}" > "$OUT/prompt-$name.txt"
done

# Run each prompt × model with throttle (avoid 100rpm/5concurrent rate limit)
for prompt_name in $(printf '%s\n' "${!PROMPTS[@]}" | sort); do
    prompt="${PROMPTS[$prompt_name]}"
    echo "========================================================"
    echo "[$prompt_name]"
    echo "========================================================"
    echo "$prompt" | head -3
    echo "..."
    echo ""

    for model in "${MODELS[@]}"; do
        sleep 2  # throttle
        body=$(jq -n --arg model "$model" --arg prompt "$prompt" \
            '{model: $model, messages: [{role: "user", content: $prompt}], max_tokens: 800, temperature: 0.6, top_p: 0.95}')

        out_file="$OUT/${prompt_name}--${model}.md"
        echo "  → $model ..."
        response=$(curl -sS -m 90 \
            -w '\n__METRICS__:%{time_total}|%{time_starttransfer}' \
            -H "Authorization: Bearer $NAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d "$body" \
            "$BASE_URL/chat/completions" 2>/dev/null) || { echo "    ERROR curl" >&2; continue; }

        json="${response%$'\n__METRICS__:'*}"
        metrics="${response##*__METRICS__:}"
        wall="${metrics%|*}"

        content=$(echo "$json" | jq -r '.choices[0].message.content // "<no content>"')
        out_tok=$(echo "$json" | jq -r '.usage.completion_tokens // 0')
        reason_tok=$(echo "$json" | jq -r '.usage.completion_tokens_details.reasoning_tokens // 0')

        {
            printf '# %s — %s\n\n' "$prompt_name" "$model"
            printf '**Wall**: %.2fs | **Output tokens**: %s | **Reasoning tokens**: %s\n\n---\n\n' "$wall" "$out_tok" "$reason_tok"
            printf '%s\n' "$content"
        } > "$out_file"

        printf '    wall=%.2fs out=%s reason=%s\n' "$wall" "$out_tok" "$reason_tok"
    done
    echo ""
done

echo "========================================================"
echo "DONE. Results in: $OUT"
echo "========================================================"
echo "Compare side-by-side:"
echo "  for f in $OUT/01-code-review--*.md; do echo \"=== \$f ===\"; cat \"\$f\"; done | less"
echo ""
echo "Or open in your editor:"
echo "  ls $OUT/*.md"
