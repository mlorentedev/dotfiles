#!/usr/bin/env bats
# Tests for scripts/pr-agent-route.sh (AI-045 AC9, #1923): draw the CI review
# pool's first reviewer, with equal weight, from the members that answered
# their probe, and name the provider of the second attempt.
#
# `curl` is a stub on PATH. It answers the Anthropic probe with STUB_CODE
# (`timeout` makes it behave like curl's exit 28) and logs its argv and stdin,
# so a test can prove where the key travels. tests/pr-agent-route-real.bats
# drives the real curl.

bats_require_minimum_version 1.5.0
load 'lib/refute'

setup() {
    ROUTE="$BATS_TEST_DIRNAME/../scripts/pr-agent-route.sh"
    OUT="$BATS_TEST_TMPDIR/output"
    : > "$OUT"
    export STUB_ARGV_LOG="$BATS_TEST_TMPDIR/argv.log"
    export STUB_STDIN_LOG="$BATS_TEST_TMPDIR/stdin.log"
    : > "$STUB_ARGV_LOG"
    mkdir -p "$BATS_TEST_TMPDIR/bin"
    cat > "$BATS_TEST_TMPDIR/bin/curl" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$STUB_ARGV_LOG"
cat >> "$STUB_STDIN_LOG"
if [ "${STUB_CODE:-200}" = "timeout" ]; then printf '000'; exit 28; fi
printf '%s' "${STUB_CODE:-200}"
STUB
    chmod +x "$BATS_TEST_TMPDIR/bin/curl"
    export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
    export NAN_OUTCOME=success NAN_MODEL=openai/mimo-v2.6-flash
    export NAN_FALLBACKS='["openai/deepseek-v4-flash"]'
    export ANTHROPIC_MODEL=anthropic/claude-haiku-5-5
    export PR_AGENT_ANTHROPIC_API_KEY="sk-ant-test-not-a-real-key"
    export GITHUB_REPOSITORY=o/r
    unset PR_AGENT_PROVIDER PR_AGENT_DRAW GITHUB_STEP_SUMMARY STUB_CODE
}

route() { run "$ROUTE" --output "$OUT"; }
out() { sed -n "s/^$1=//p" "$OUT"; }

# --- the draw ------------------------------------------------------------------

@test "route: each member of a full pool can be drawn first, by index" {
    PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out reviewer)" = "openai/mimo-v2.6-flash" ]
    [ "$(out first)" = "nan" ] && [ "$(out second)" = "anthropic" ]
    [ "$(out nan_model)" = "openai/mimo-v2.6-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/deepseek-v4-flash"]' ]

    : > "$OUT"; PR_AGENT_DRAW=1 route
    [ "$(out reviewer)" = "openai/deepseek-v4-flash" ]
    [ "$(out first)" = "nan" ] && [ "$(out second)" = "anthropic" ]
    # The drawn NaN model goes first; the other stays behind it.
    [ "$(out nan_model)" = "openai/deepseek-v4-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/mimo-v2.6-flash"]' ]

    : > "$OUT"; PR_AGENT_DRAW=2 route
    [ "$(out reviewer)" = "anthropic/claude-haiku-5-5" ]
    [ "$(out first)" = "anthropic" ] && [ "$(out second)" = "nan" ]
    # NaN is the second attempt, in declared order.
    [ "$(out nan_model)" = "openai/mimo-v2.6-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/deepseek-v4-flash"]' ]
}

@test "route: an index outside the pool is a usage error, never a silent pick" {
    PR_AGENT_DRAW=3 route
    [ "$status" -eq 2 ]
    PR_AGENT_DRAW=x route
    [ "$status" -eq 2 ]
}

@test "route: the random draw reaches every member with roughly equal weight" {
    local i counts
    for i in $(seq 1 300); do
        : > "$OUT"
        "$ROUTE" --output "$OUT" >/dev/null
        out reviewer
    done | sort | uniq -c > "$BATS_TEST_TMPDIR/counts"
    counts=$(cat "$BATS_TEST_TMPDIR/counts")
    [ "$(wc -l < "$BATS_TEST_TMPDIR/counts")" -eq 3 ] || { echo "$counts"; false; }
    # Expected 100 each; 60 is more than six standard deviations below it.
    while read -r n _; do [ "$n" -ge 60 ] || { echo "$counts"; false; }; done < "$BATS_TEST_TMPDIR/counts"
}

# --- members out of the draw ---------------------------------------------------

@test "route: an Anthropic member that does not answer is never drawn" {
    local code
    for code in 401 404 429 529 timeout; do
        : > "$OUT"
        STUB_CODE=$code PR_AGENT_DRAW=1 route
        [ "$status" -eq 0 ]
        [ "$(out first)" = "nan" ]
        [ "$(out second)" = "none" ]
        [[ "$output" == *"::warning::"*"claude-haiku-5-5"* ]] || false
        [ -n "$(out note)" ]
        # Only two candidates remain, so index 2 is outside the pool.
        : > "$OUT"
        STUB_CODE=$code PR_AGENT_DRAW=2 route
        [ "$status" -eq 2 ]
    done
}

@test "route: the warning names the fix by status class" {
    STUB_CODE=401 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"key is refused"*"dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY"* ]] || false
    : > "$OUT"; STUB_CODE=404 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"model id is wrong"* ]] || false
    : > "$OUT"; STUB_CODE=429 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"rate limit, overload or spend limit"* ]] || false
}

@test "route: no key means no probe and no Anthropic member, with the remedy" {
    PR_AGENT_ANTHROPIC_API_KEY="" PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out second)" = "none" ]
    [ ! -s "$STUB_ARGV_LOG" ]
    [[ "$(out note)" == *"dotf secrets sync ci --repo o/r"* ]] || false
}

@test "route: a NaN member that failed the preflight is not in the draw" {
    NAN_FALLBACKS='[]' PR_AGENT_DRAW=1 route
    [ "$(out reviewer)" = "anthropic/claude-haiku-5-5" ]
    [ "$(out nan_fallbacks)" = '[]' ]
}

@test "route: NaN down as a whole leaves Anthropic alone, with no second attempt" {
    NAN_OUTCOME=failure NAN_MODEL="" NAN_FALLBACKS="" PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "anthropic" ]
    [ "$(out second)" = "none" ]
    [ -z "$(out nan_model)" ]
}

@test "route: no member at all answers first=none, and the job summary says so" {
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary"
    NAN_OUTCOME=failure NAN_MODEL="" STUB_CODE=529 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "none" ] && [ "$(out second)" = "none" ]
    grep -q 'No member answered' "$GITHUB_STEP_SUMMARY"
}

# --- the override --------------------------------------------------------------

@test "route: PR_AGENT_PROVIDER=nan draws among NaN only, with Anthropic second" {
    PR_AGENT_PROVIDER=nan PR_AGENT_DRAW=1 route
    [ "$(out reviewer)" = "openai/deepseek-v4-flash" ]
    [ "$(out second)" = "anthropic" ]
    : > "$OUT"; PR_AGENT_PROVIDER=nan PR_AGENT_DRAW=2 route
    [ "$status" -eq 2 ]
}

@test "route: PR_AGENT_PROVIDER=anthropic puts Anthropic first, NaN second" {
    PR_AGENT_PROVIDER=anthropic PR_AGENT_DRAW=0 route
    [ "$(out first)" = "anthropic" ] && [ "$(out second)" = "nan" ]
}

@test "route: PR_AGENT_PROVIDER=nan-only never probes or schedules Anthropic" {
    PR_AGENT_PROVIDER=nan-only PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "nan" ] && [ "$(out second)" = "none" ]
    [ ! -s "$STUB_ARGV_LOG" ]
    # Even with NaN down, the switch holds: no review rather than spending.
    : > "$OUT"; NAN_OUTCOME=failure NAN_MODEL="" PR_AGENT_PROVIDER=nan-only route
    [ "$(out first)" = "none" ]
    [ ! -s "$STUB_ARGV_LOG" ]
}

@test "route: a forced provider with no member that answered draws from the rest, loudly" {
    STUB_CODE=529 PR_AGENT_PROVIDER=anthropic PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "nan" ]
    [[ "$output" == *"::warning::PR_AGENT_PROVIDER=anthropic"* ]] || false
}

@test "route: an unknown override fails before any probe" {
    PR_AGENT_PROVIDER=nan_only route
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::PR_AGENT_PROVIDER is nan_only"* ]] || false
    [ ! -s "$STUB_ARGV_LOG" ]
    [ ! -s "$OUT" ]
}

# --- the key, the request, the outputs -------------------------------------------

@test "route: the key travels on curl's stdin, never on its argv" {
    PR_AGENT_DRAW=0 route
    refute_grep 'sk-ant-test-not-a-real-key' "$STUB_ARGV_LOG"
    grep -qF 'header = "x-api-key: sk-ant-test-not-a-real-key"' "$STUB_STDIN_LOG"
    [[ "$output" != *"sk-ant-test"* ]] || false
    refute_grep 'sk-ant-test' "$OUT"
}

@test "route: the probe asks for the pinned model, briefly, on the Messages API" {
    PR_AGENT_DRAW=0 route
    grep -qF '"model":"claude-haiku-5-5"' "$STUB_ARGV_LOG"
    grep -qF '"max_tokens":64' "$STUB_ARGV_LOG"
    grep -qF 'anthropic-version: 2023-06-01' "$STUB_ARGV_LOG"
    grep -qF 'https://api.anthropic.com/v1/messages' "$STUB_ARGV_LOG"
}

@test "route: a model that is not anthropic/<id> is a usage error" {
    ANTHROPIC_MODEL=claude-haiku-5-5 route
    [ "$status" -eq 2 ]
}

@test "route: the job summary names the member drawn and the second attempt" {
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary"
    PR_AGENT_DRAW=2 route
    grep -qF 'Drawn to review first: `anthropic/claude-haiku-5-5`' "$GITHUB_STEP_SUMMARY"
    grep -qF 'nothing is published: nan.' "$GITHUB_STEP_SUMMARY"
}

@test "route: runs under bash -e as a workflow step does" {
    run bash -e "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ -n "$(out first)" ]
}
