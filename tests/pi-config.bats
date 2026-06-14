#!/usr/bin/env bats
# Guard (incident->guard, AI-025): the pi coding agent config carries no
# plaintext secret in git, and the curated model set stays consistent with
# opencode (NaN + free + non-big-3 paid OpenRouter).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PI_MODELS="$DOTFILES_DIR/ai/pi/models.json"
    export PI_SETTINGS="$DOTFILES_DIR/ai/pi/settings.json"
}

@test "ai/pi/models.json exists" {
    [[ -f "$PI_MODELS" ]]
}

@test "ai/pi/models.json has no literal API key" {
    ! grep -qE '"apiKey"[[:space:]]*:[[:space:]]*"sk-' "$PI_MODELS"
}

@test "ai/pi/models.json uses the {env:NAN_API_KEY} placeholder" {
    grep -qF '{env:NAN_API_KEY}' "$PI_MODELS"
}

@test "ai/pi/models.json is valid JSON" {
    command -v jq >/dev/null || skip "jq not available"
    jq empty "$PI_MODELS"
}

@test "ai/pi/settings.json is valid JSON" {
    command -v jq >/dev/null || skip "jq not available"
    jq empty "$PI_SETTINGS"
}

@test "ai/pi/settings.json omits the volatile lastChangelogVersion (seed-if-missing)" {
    ! grep -qF 'lastChangelogVersion' "$PI_SETTINGS"
}

@test "ai/pi/settings.json enabledModels exclude OpenAI/Google/Anthropic OpenRouter providers" {
    ! grep -qE 'openrouter/(openai|google|anthropic)/' "$PI_SETTINGS"
}

# The age identity is optional per box: a leftover {env:} token in the deployed
# models.json without a local age key is the expected state (runtime env resolver
# is the fallback), so it must SKIP, not FAIL. The .sh side is now go test
# (TestCheckOpenCode pi models.json branch); healthcheck.ps1 keeps the assertion.
@test "healthcheck.ps1 treats a missing age identity as optional for pi models.json" {
    grep -qF 'age identity absent' "$DOTFILES_DIR/scripts/healthcheck.ps1"
}
