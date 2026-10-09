#!/usr/bin/env bats
# Tests for scripts/pr-agent-model-preflight.sh (AI-045): before PR-Agent runs,
# find the first model of its DECLARED chain that NaN answers, so a retired or
# hanging model degrades to the next one loudly instead of to a green job with
# no review. Measured 2026-09-30: `mimo-v2.5` hung for hours, then answered 401.
#
# `curl` is a stub on PATH. It answers per model from STUB_CODES
# ("model=code ..."; `timeout` makes it behave like curl's exit 28), and logs
# its argv and stdin so a test can prove where the key travels.

bats_require_minimum_version 1.5.0
load 'lib/refute'

setup() {
    PREFLIGHT="$BATS_TEST_DIRNAME/../scripts/pr-agent-model-preflight.sh"
    OUT="$BATS_TEST_TMPDIR/output"
    : > "$OUT"
    export STUB_ARGV_LOG="$BATS_TEST_TMPDIR/argv.log"
    export STUB_STDIN_LOG="$BATS_TEST_TMPDIR/stdin.log"
    mkdir -p "$BATS_TEST_TMPDIR/bin"
    cat > "$BATS_TEST_TMPDIR/bin/curl" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$STUB_ARGV_LOG"
cat >> "$STUB_STDIN_LOG"
body=""
while [ $# -gt 0 ]; do
    case "$1" in -d|--data) body="$2"; shift ;; esac
    shift
done
model=$(printf '%s' "$body" | jq -r '.model')
code=$(printf '%s\n' $STUB_CODES | sed -n "s/^${model}=//p")
if [ "$code" = "timeout" ]; then printf '000'; exit 28; fi
printf '%s' "${code:-404}"
STUB
    chmod +x "$BATS_TEST_TMPDIR/bin/curl"
    export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
    export NAN_API_KEY="sk-test-not-a-real-key"
    unset GITHUB_STEP_SUMMARY
}

run_preflight() {
    run "$PREFLIGHT" --model openai/mimo-v2.6-flash \
        --fallbacks '["openai/deepseek-v4-flash"]' --output "$OUT"
}

@test "preflight: the primary answers, so it reviews and the fallback stays behind it" {
    export STUB_CODES="mimo-v2.6-flash=200 deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/mimo-v2.6-flash' "$OUT"
    grep -qxF 'fallbacks=["openai/deepseek-v4-flash"]' "$OUT"
    refute_grep '^retry_model=' "$OUT"
    [[ "$output" != *"::warning::"* ]] || false
}

@test "preflight: a retired primary (401) hands the review to the fallback, and says so" {
    export STUB_CODES="mimo-v2.6-flash=401 deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/deepseek-v4-flash' "$OUT"
    grep -qxF 'fallbacks=[]' "$OUT"
    refute_grep '^retry_model=' "$OUT"
    [[ "$output" == *"::warning::"*"openai/mimo-v2.6-flash"*"HTTP 401"*"openai/deepseek-v4-flash"* ]] || false
}

@test "preflight: a primary that hangs is skipped like one that refuses" {
    export STUB_CODES="mimo-v2.6-flash=timeout deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    grep -qxF 'model=openai/deepseek-v4-flash' "$OUT"
    [[ "$output" == *"::warning::"*"openai/mimo-v2.6-flash"*"no answer within"* ]] || false
}

@test "preflight: the warning names the fix by status class, not one remedy for all" {
    # 401/403/404: the key cannot call it, so the chain is wrong. 402/429: quota
    # or NaN's per-model concurrency (measured in #1107), where editing the chain
    # is the wrong advice. 5xx and a hang: NaN did not answer this time.
    export STUB_CODES="mimo-v2.6-flash=429 deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    [[ "$output" == *"openai/mimo-v2.6-flash answered HTTP 429"*"quota or concurrency"* ]] || false
    [[ "$output" != *"Fix the chain"* ]] || false

    export STUB_CODES="mimo-v2.6-flash=401 deepseek-v4-flash=200"
    run_preflight
    [[ "$output" == *"HTTP 401"*"Fix the chain"* ]] || false

    export STUB_CODES="mimo-v2.6-flash=503 deepseek-v4-flash=200"
    run_preflight
    [[ "$output" == *"HTTP 503"*"NaN did not serve it"* ]] || false
    [[ "$output" != *"Fix the chain"* ]] || false
}

@test "preflight: when no declared model answers, the step fails and NaN leaves the pool" {
    export STUB_CODES="mimo-v2.6-flash=401 deepseek-v4-flash=429"
    run_preflight
    [ "$status" -eq 1 ]
    [[ "$output" == *"::warning::No model in PR-Agent's declared NaN chain answered"* ]] || false
    run ! grep -q '^model=' "$OUT"
}

@test "preflight: it never picks a model outside the declared chain" {
    # Every other id answers; only the two declared ones refuse.
    export STUB_CODES="mimo-v2.6-flash=401 deepseek-v4-flash=401 glm5.3-flash=200 qwen3.6=200"
    run_preflight
    [ "$status" -eq 1 ]
    run ! grep -q 'glm5.3-flash\|qwen3.6' "$OUT"
}

@test "preflight: it probes the bare NaN id, and hands PR-Agent the transport-prefixed one" {
    export STUB_CODES="mimo-v2.6-flash=200 deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    grep -q '"model":"mimo-v2.6-flash"' "$STUB_ARGV_LOG"
    run ! grep -q '"model":"openai/' "$STUB_ARGV_LOG"
}

@test "preflight: the key reaches curl on stdin, never on its argv" {
    export STUB_CODES="mimo-v2.6-flash=200 deepseek-v4-flash=200"
    run_preflight
    [ "$status" -eq 0 ]
    run ! grep -q "$NAN_API_KEY" "$STUB_ARGV_LOG"
    grep -q "Bearer $NAN_API_KEY" "$STUB_STDIN_LOG"
    [[ "$output" != *"$NAN_API_KEY"* ]] || false
}

@test "preflight: the job summary names the model that reviews and every one that did not answer" {
    export STUB_CODES="mimo-v2.6-flash=401 deepseek-v4-flash=200"
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary.md"
    run_preflight
    [ "$status" -eq 0 ]
    grep -q 'openai/deepseek-v4-flash' "$GITHUB_STEP_SUMMARY"
    grep -q 'openai/mimo-v2.6-flash.*401' "$GITHUB_STEP_SUMMARY"
}

@test "preflight: a missing key or model is a usage error, not a model failure" {
    run env -u NAN_API_KEY "$PREFLIGHT" --model openai/x --fallbacks '[]' --output "$OUT"
    [ "$status" -eq 2 ]
    run "$PREFLIGHT" --fallbacks '[]' --output "$OUT"
    [ "$status" -eq 2 ]
    run "$PREFLIGHT" --model openai/x --fallbacks 'not json' --output "$OUT"
    [ "$status" -eq 2 ]
}
