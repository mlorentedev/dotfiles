#!/usr/bin/env bats
# Tests for scripts/pr-agent-route.sh (AI-045 AC9, #1923): draw the CI review
# pool's first reviewer, by the weights in harness/reviewer-pool.json, from the
# members that answered their probe; route a risky PR to the `route: risk`
# member; and name the provider of the second attempt.
#
# `curl` is a stub on PATH. It answers each Anthropic probe with STUB_CODE, or
# STUB_SONNET_CODE for the Sonnet probe (`timeout` makes it behave like curl's
# exit 28), and logs its argv and stdin, so a test can prove where the key
# travels. tests/pr-agent-route-real.bats drives the real curl.
#
# The pool is a fixture with weights 3, 2 and 5, so a point in [0, 10) names
# exactly one member: 0-2 mimo, 3-4 deepseek, 5-9 haiku.

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
code="${STUB_CODE:-200}"
case "$*" in *claude-sonnet*) code="${STUB_SONNET_CODE:-$code}" ;; esac
if [ "$code" = "timeout" ]; then printf '000'; exit 28; fi
printf '%s' "$code"
STUB
    chmod +x "$BATS_TEST_TMPDIR/bin/curl"
    export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
    export POOL_FILE="$BATS_TEST_TMPDIR/pool.json"
    cat > "$POOL_FILE" <<'JSON'
{"pool": [
 {"id": "nan/mimo-v2.6-flash", "pr_agent": {"model": "openai/mimo-v2.6-flash", "weight": 3}},
 {"id": "nan/deepseek-v4-flash", "pr_agent": {"model": "openai/deepseek-v4-flash", "weight": 2}},
 {"id": "nan/qwen3.8-flash"},
 {"id": "anthropic-review/claude-haiku-5-5",
  "pr_agent": {"model": "anthropic/claude-haiku-5-5", "weight": 5, "reasoning_effort": "high"}},
 {"id": "anthropic-review/claude-sonnet-5-5",
  "pr_agent": {"model": "anthropic/claude-sonnet-5-5", "route": "risk", "reasoning_effort": "medium"}}
], "pr_agent_risk": {"min_changed_lines": 900, "label": "deep-review"}}
JSON
    export NAN_OUTCOME=success NAN_MODEL=openai/mimo-v2.6-flash
    export NAN_FALLBACKS='["openai/deepseek-v4-flash"]'
    export PR_AGENT_ANTHROPIC_API_KEY="sk-ant-test-not-a-real-key"
    export GITHUB_REPOSITORY=o/r
    unset PR_AGENT_PROVIDER PR_AGENT_DRAW GITHUB_STEP_SUMMARY STUB_CODE STUB_SONNET_CODE
    unset PR_CHANGED_LINES PR_LABELS NAN_PREFLIGHT_EXIT
}

route() { run "$ROUTE" --output "$OUT"; }
out() { sed -n "s/^$1=//p" "$OUT"; }

# every_point TOTAL: how many points of [0, TOTAL) draw each reviewer, as
# `model=count ` pairs in byte order.
every_point() {
    local p
    for p in $(seq 0 $(($1 - 1))); do
        : > "$OUT"
        PR_AGENT_DRAW=$p "$ROUTE" --output "$OUT" >/dev/null
        out reviewer
    done | LC_ALL=C sort | uniq -c | awk '{print $2 "=" $1}' | tr '\n' ' '
}

# --- the draw ------------------------------------------------------------------

@test "route: each member is drawn at its own points, with the other provider second" {
    PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out reviewer)" = "openai/mimo-v2.6-flash" ]
    [ "$(out route)" = "draw" ]
    [ "$(out first)" = "nan" ]
    [ "$(out second)" = "anthropic" ]
    [ "$(out nan_model)" = "openai/mimo-v2.6-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/deepseek-v4-flash"]' ]
    # The second attempt is the drawn Anthropic member, at its effort.
    [ "$(out anthropic_model)" = "anthropic/claude-haiku-5-5" ]
    [ "$(out anthropic_effort)" = "high" ]

    : > "$OUT"; PR_AGENT_DRAW=3 route
    [ "$(out reviewer)" = "openai/deepseek-v4-flash" ]
    # The drawn NaN model goes first; the other stays behind it.
    [ "$(out nan_model)" = "openai/deepseek-v4-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/mimo-v2.6-flash"]' ]

    : > "$OUT"; PR_AGENT_DRAW=5 route
    [ "$(out reviewer)" = "anthropic/claude-haiku-5-5" ]
    [ "$(out first)" = "anthropic" ]
    [ "$(out second)" = "nan" ]
    [ "$(out anthropic_model)" = "anthropic/claude-haiku-5-5" ]
    # NaN is the second attempt, in declared order.
    [ "$(out nan_model)" = "openai/mimo-v2.6-flash" ]
    [ "$(out nan_fallbacks)" = '["openai/deepseek-v4-flash"]' ]
}

@test "route: across every point, each member is drawn exactly its weight" {
    run every_point 10
    [ "$output" = "anthropic/claude-haiku-5-5=5 openai/deepseek-v4-flash=2 openai/mimo-v2.6-flash=3 " ]
}

@test "route: a member that did not answer gives up its points, and the rest keep theirs" {
    export NAN_FALLBACKS='[]'
    run every_point 8
    [ "$output" = "anthropic/claude-haiku-5-5=5 openai/mimo-v2.6-flash=3 " ]
    : > "$OUT"; PR_AGENT_DRAW=8 route
    [ "$status" -eq 2 ]
}

@test "route: a NaN model with no weight is never drawn first, but stays in the chain" {
    export NAN_FALLBACKS='["openai/deepseek-v4-flash", "openai/glm5.3-flash"]'
    run every_point 10
    [ "$output" = "anthropic/claude-haiku-5-5=5 openai/deepseek-v4-flash=2 openai/mimo-v2.6-flash=3 " ]
    : > "$OUT"; PR_AGENT_DRAW=0 route
    [ "$(out nan_fallbacks)" = '["openai/deepseek-v4-flash","openai/glm5.3-flash"]' ]
}

@test "route: a point outside the total weight is a usage error, never a silent pick" {
    PR_AGENT_DRAW=10 route
    [ "$status" -eq 2 ]
    PR_AGENT_DRAW=x route
    [ "$status" -eq 2 ]
}

@test "route: the random draw reaches every member, the heaviest most" {
    local i counts="$BATS_TEST_TMPDIR/counts"
    for i in $(seq 1 300); do
        : > "$OUT"
        "$ROUTE" --output "$OUT" >/dev/null
        out reviewer
    done | sort | uniq -c > "$counts"
    [ "$(wc -l < "$counts" | tr -d ' ')" = "3" ] || { cat "$counts"; false; }
    # Expected 90, 60 and 150; 30 is more than four standard deviations below
    # the smallest, and haiku leads deepseek by about seven.
    while read -r n _; do [ "$n" -ge 30 ] || { cat "$counts"; false; }; done < "$counts"
    [ "$(awk '/haiku/ {print $1}' "$counts")" -gt "$(awk '/deepseek/ {print $1}' "$counts")" ]
}

# The shipped pool, not the fixture: every point of its total weight draws the
# member that owns it, so the shares in harness/reviewer-pool.json are the
# shares CI runs.
@test "route: the shipped pool draws each member exactly its declared weight" {
    local pool="$BATS_TEST_DIRNAME/../harness/reviewer-pool.json" total want
    export POOL_FILE="$pool"
    NAN_MODEL=$(jq -r '[.pool[] | .pr_agent.model // empty | select(startswith("openai/"))] | first' "$pool")
    NAN_FALLBACKS=$(jq -c '[.pool[] | .pr_agent.model // empty | select(startswith("openai/"))] | .[1:]' "$pool")
    export NAN_MODEL NAN_FALLBACKS
    total=$(jq '[.pool[] | .pr_agent.weight // 0] | add' "$pool")
    want=$(jq -r '[.pool[] | select(.pr_agent.weight) | "\(.pr_agent.model)=\(.pr_agent.weight)"] | sort | join(" ")' "$pool")
    run every_point "$total"
    [ "$output" = "$want " ] || { printf 'drawn:   %s\nweights: %s\n' "$output" "$want" >&2; false; }
}

# The shipped risk line is the budget lever (ADR-046): 1,500 changed lines put
# about 3% of PRs on Sonnet. Moving it moves the month's spend, so the number is
# pinned here and the route is driven against the shipped pool on both sides.
@test "route: the shipped pool routes to Sonnet at 1,500 changed lines and not below" {
    local pool="$BATS_TEST_DIRNAME/../harness/reviewer-pool.json"
    export POOL_FILE="$pool"
    NAN_MODEL=$(jq -r '[.pool[] | .pr_agent.model // empty | select(startswith("openai/"))] | first' "$pool")
    NAN_FALLBACKS=$(jq -c '[.pool[] | .pr_agent.model // empty | select(startswith("openai/"))] | .[1:]' "$pool")
    export NAN_MODEL NAN_FALLBACKS
    [ "$(jq '.pr_agent_risk.min_changed_lines' "$pool")" = "1500" ]

    PR_CHANGED_LINES=1500 PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out route)" = "risk" ]
    [ "$(out reviewer)" = "anthropic/claude-sonnet-5-5" ]

    : > "$OUT"; PR_CHANGED_LINES=1499 PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out route)" = "draw" ]
}

# --- the risk route --------------------------------------------------------------

@test "route: a PR at the line threshold reviews first on the risk member, NaN second" {
    PR_CHANGED_LINES=900 PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out route)" = "risk" ]
    [ "$(out reviewer)" = "anthropic/claude-sonnet-5-5" ]
    [ "$(out first)" = "anthropic" ]
    [ "$(out second)" = "nan" ]
    [ "$(out anthropic_model)" = "anthropic/claude-sonnet-5-5" ]
    [ "$(out anthropic_effort)" = "medium" ]
    grep -qF '"model":"claude-sonnet-5-5"' "$STUB_ARGV_LOG"

    : > "$OUT"; PR_CHANGED_LINES=899 PR_AGENT_DRAW=0 route
    [ "$(out route)" = "draw" ]
    [ "$(out reviewer)" = "openai/mimo-v2.6-flash" ]
}

@test "route: the label routes a small PR to the risk member, and only the exact label" {
    PR_CHANGED_LINES=3 PR_LABELS='["docs", "deep-review"]' PR_AGENT_DRAW=0 route
    [ "$(out route)" = "risk" ]
    [ "$(out reviewer)" = "anthropic/claude-sonnet-5-5" ]
    : > "$OUT"; PR_CHANGED_LINES=3 PR_LABELS='["deep-review-later"]' PR_AGENT_DRAW=0 route
    [ "$(out route)" = "draw" ]
}

@test "route: the risk member is probed only for a risky PR" {
    PR_CHANGED_LINES=10 PR_AGENT_DRAW=0 route
    refute_grep 'claude-sonnet' "$STUB_ARGV_LOG"
}

@test "route: a risk member that does not answer sends the PR to the draw, with a note" {
    STUB_SONNET_CODE=529 PR_CHANGED_LINES=5000 PR_AGENT_DRAW=5 route
    [ "$status" -eq 0 ]
    [ "$(out route)" = "draw" ]
    [ "$(out reviewer)" = "anthropic/claude-haiku-5-5" ]
    [ "$(out anthropic_model)" = "anthropic/claude-haiku-5-5" ]
    [[ "$(out note)" == *"crossed the risk line (5000 changed lines"*"drawn by weight instead"* ]] || false
}

@test "route: an unread size is named, and the label still routes" {
    PR_CHANGED_LINES="" PR_AGENT_DRAW=0 route
    [ "$(out route)" = "draw" ]
    [[ "$(out note)" == *"size could not be read"* ]] || false
    : > "$OUT"; PR_CHANGED_LINES="" PR_LABELS='["deep-review"]' PR_AGENT_DRAW=0 route
    [ "$(out route)" = "risk" ]
}

@test "route: PR_AGENT_PROVIDER=nan and nan-only never take the risk route" {
    PR_AGENT_PROVIDER=nan PR_CHANGED_LINES=5000 PR_AGENT_DRAW=0 route
    [ "$(out first)" = "nan" ]
    [ "$(out route)" = "draw" ]
    : > "$OUT"; : > "$STUB_ARGV_LOG"
    PR_AGENT_PROVIDER=nan-only PR_CHANGED_LINES=5000 PR_AGENT_DRAW=0 route
    [ "$(out first)" = "nan" ]
    [ ! -s "$STUB_ARGV_LOG" ]
}

# --- the allowlist -----------------------------------------------------------------

@test "route: a pool naming an Anthropic model outside the allowlist fails before any probe" {
    jq '.pool[3].pr_agent.model = "anthropic/claude-opus-5-5"' "$POOL_FILE" > "$POOL_FILE.new"
    mv "$POOL_FILE.new" "$POOL_FILE"
    PR_AGENT_DRAW=0 route
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::"*"anthropic/claude-opus-5-5"*"allowlist"* ]] || false
    [ ! -s "$STUB_ARGV_LOG" ]
    [ ! -s "$OUT" ]
}

@test "route: the allowlist is exactly Haiku and Sonnet" {
    run grep -c '^ALLOWED_ANTHROPIC="anthropic/claude-haiku-5-5 anthropic/claude-sonnet-5-5"$' "$ROUTE"
    [ "$output" = "1" ]
}

@test "route: an unreadable pool is a usage error" {
    POOL_FILE="$BATS_TEST_TMPDIR/missing.json" route
    [ "$status" -eq 2 ]
}

# --- members out of the draw ---------------------------------------------------

@test "route: an Anthropic member that does not answer is never drawn" {
    local code
    for code in 401 404 429 529 timeout; do
        : > "$OUT"
        STUB_CODE=$code PR_AGENT_DRAW=3 route
        [ "$status" -eq 0 ]
        [ "$(out first)" = "nan" ]
        [ "$(out second)" = "none" ]
        [ -z "$(out anthropic_model)" ]
        [[ "$output" == *"::warning::"*"claude-haiku-5-5"* ]] || false
        [ -n "$(out note)" ]
        # Only NaN's five points remain.
        : > "$OUT"
        STUB_CODE=$code PR_AGENT_DRAW=5 route
        [ "$status" -eq 2 ]
    done
}

@test "route: the warning names the fix by status class" {
    STUB_CODE=401 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"key is refused"*"dotf secrets rotate PR_AGENT_ANTHROPIC_API_KEY"* ]] || false
    : > "$OUT"; STUB_CODE=404 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"model id is wrong"*"reviewer-pool.json"* ]] || false
    : > "$OUT"; STUB_CODE=429 PR_AGENT_DRAW=0 route
    [[ "$(out note)" == *"rate limit, overload or spend limit"* ]] || false
}

@test "route: no key means no probe and no Anthropic member, with the remedy" {
    PR_AGENT_ANTHROPIC_API_KEY="" PR_CHANGED_LINES=5000 PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out second)" = "none" ]
    [ "$(out route)" = "draw" ]
    [ ! -s "$STUB_ARGV_LOG" ]
    [[ "$(out note)" == *"dotf secrets sync ci --repo o/r"* ]] || false
}

@test "route: NaN down as a whole leaves Anthropic alone, with no second attempt" {
    NAN_OUTCOME=failure NAN_MODEL="" NAN_FALLBACKS="" PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "anthropic" ]
    [ "$(out second)" = "none" ]
    [ -z "$(out nan_model)" ]
}

@test "route: a preflight that failed on its setup stops the draw instead of paying for every review" {
    NAN_OUTCOME=failure NAN_MODEL="" NAN_FALLBACKS="" NAN_PREFLIGHT_EXIT=2 route
    [ "$status" -eq 1 ]
    [[ "$output" == *"::error::The NaN preflight failed on its setup (exit 2)"* ]] || false
    [ ! -s "$STUB_ARGV_LOG" ]
    [ ! -s "$OUT" ]
}

@test "route: a preflight in which no model answered (exit 1) still hands over to Anthropic" {
    NAN_OUTCOME=failure NAN_MODEL="" NAN_FALLBACKS="" NAN_PREFLIGHT_EXIT=1 PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "anthropic" ]
}

@test "route: no member at all answers first=none, and the job summary says so" {
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary"
    NAN_OUTCOME=failure NAN_MODEL="" STUB_CODE=529 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "none" ]
    [ "$(out second)" = "none" ]
    grep -q 'No member answered' "$GITHUB_STEP_SUMMARY"
}

# --- the override --------------------------------------------------------------

@test "route: PR_AGENT_PROVIDER=nan draws among NaN only, with Anthropic second" {
    PR_AGENT_PROVIDER=nan PR_AGENT_DRAW=3 route
    [ "$(out reviewer)" = "openai/deepseek-v4-flash" ]
    [ "$(out second)" = "anthropic" ]
    : > "$OUT"; PR_AGENT_PROVIDER=nan PR_AGENT_DRAW=5 route
    [ "$status" -eq 2 ]
}

@test "route: PR_AGENT_PROVIDER=anthropic puts Anthropic first, NaN second" {
    PR_AGENT_PROVIDER=anthropic PR_AGENT_DRAW=0 route
    [ "$(out first)" = "anthropic" ]
    [ "$(out second)" = "nan" ]
}

@test "route: PR_AGENT_PROVIDER=nan-only never probes or schedules Anthropic" {
    PR_AGENT_PROVIDER=nan-only PR_AGENT_DRAW=0 route
    [ "$status" -eq 0 ]
    [ "$(out first)" = "nan" ]
    [ "$(out second)" = "none" ]
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
    PR_CHANGED_LINES=5000 PR_AGENT_DRAW=0 route
    refute_grep 'sk-ant-test-not-a-real-key' "$STUB_ARGV_LOG"
    grep -qF 'header = "x-api-key: sk-ant-test-not-a-real-key"' "$STUB_STDIN_LOG"
    [[ "$output" != *"sk-ant-test"* ]] || false
    refute_grep 'sk-ant-test' "$OUT"
}

@test "route: the probe asks for the pool's model, briefly, on the Messages API" {
    PR_AGENT_DRAW=0 route
    grep -qF '"model":"claude-haiku-5-5"' "$STUB_ARGV_LOG"
    grep -qF '"max_tokens":64' "$STUB_ARGV_LOG"
    grep -qF 'anthropic-version: 2023-06-01' "$STUB_ARGV_LOG"
    grep -qF 'https://api.anthropic.com/v1/messages' "$STUB_ARGV_LOG"
}

@test "route: the job summary names the member drawn, the risk route and the second attempt" {
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary"
    PR_AGENT_DRAW=5 route
    grep -qF 'Drawn to review first: `anthropic/claude-haiku-5-5`' "$GITHUB_STEP_SUMMARY"
    grep -qF 'nothing is published: nan.' "$GITHUB_STEP_SUMMARY"
    : > "$GITHUB_STEP_SUMMARY"
    PR_LABELS='["deep-review"]' route
    grep -qF 'Routed by risk (the deep-review label) to `anthropic/claude-sonnet-5-5`' "$GITHUB_STEP_SUMMARY"
}

@test "route: runs under bash -e as a workflow step does" {
    run bash -e "$ROUTE" --output "$OUT"
    [ "$status" -eq 0 ]
    [ -n "$(out first)" ]
}
