#!/usr/bin/env bats
# harness/reviewer-pool.json is the allow-list of models that may sign a
# review.md, and since HARNESS-093 (#1370) `dotf spec review` draws one member
# at random by default. The gate is only as strong as its weakest member, so a
# pi member must be a reasoning-class model -- the line this pool draws (a
# latency-only daily driver that PASSes cheaply is worse than no gate). Measured
# on the ids, never on prose. Since AI-046 the NaN models belong to
# pi-nan-provider, so that check runs against the installed package in
# tests/pi-nan-package.bats.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export POOL="$DOTFILES_DIR/harness/reviewer-pool.json"
    export MODELS="$DOTFILES_DIR/ai/pi/models.json"
}

@test "the pool is valid JSON with unique, non-blank ids" {
    jq -e '.pool | length >= 1' "$POOL" >/dev/null
    [ "$(jq -r '.pool[].id' "$POOL" | grep -c .)" = "$(jq -r '.pool[].id' "$POOL" | sort -u | grep -c .)" ]
    [ "$(jq -r '.pool[] | select(.id == "" or .id == null) | "blank"' "$POOL" | grep -c blank)" = "0" ]
}

@test "the pool draws from at least four members, so a random pick spreads the API buckets (HARNESS-093)" {
    [ "$(jq -r '.pool | length' "$POOL")" -ge 4 ]
    for id in nan/deepseek-v4-flash nan/mimo-v2.5 nan/glm5.3-flash nan/qwen3.8-flash; do
        jq -e --arg id "$id" '.pool[] | select(.id == $id)' "$POOL" >/dev/null
    done
}

@test "dotf spec review draws a member at random by default and --reviewer names one (HARNESS-093)" {
    grep -qF 'rand.IntN' "$DOTFILES_DIR/cli/internal/cmd/spec.go"
    grep -qF '"pool member to run (default: one drawn at random from the pool)"' "$DOTFILES_DIR/cli/internal/cmd/spec.go"
    # the skill's usage line tells the reader the same thing the launcher does
    grep -qF 'a pool member drawn at random' "$DOTFILES_DIR/harness/skills/adversarial-review/SKILL.md"
}
