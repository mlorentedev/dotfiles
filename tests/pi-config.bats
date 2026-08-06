#!/usr/bin/env bats
# Guard (incident->guard, AI-025): the pi coding agent config carries no
# plaintext secret in git, and the curated model set stays consistent with
# its own README (NaN + non-big-3 paid OpenRouter).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export PI_MODELS="$DOTFILES_DIR/ai/pi/models.json"
    export PI_SETTINGS="$DOTFILES_DIR/ai/pi/settings.json"
    export PI_README="$DOTFILES_DIR/ai/pi/README.md"
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

# --- Referential integrity between the two files -------------------------
# Incident (2026-08-04): a model added as `deepseek-v4-flash-0731` in
# models.json was referenced as `nan/deepseek-v4-flash 0731` in settings.json
# -- a space where the id has a hyphen, so the model never resolved. Every
# assertion above passed: both files were valid JSON, carried no secret and
# named no banned provider. Nothing cross-checked one file against the other.

@test "ai/pi/settings.json nan/* models all resolve to an id in models.json" {
    command -v jq >/dev/null || skip "jq not available"
    ids="$(jq -r '.. | objects | select(has("id")) | .id' "$PI_MODELS" | LC_ALL=C sort -u)"
    missing=""
    # Read line by line: a reference may legitimately contain spaces, and word
    # splitting would silently report only the fragment after the space.
    while IFS= read -r ref; do
        [ -n "$ref" ] || continue
        printf '%s\n' "$ids" | grep -qxF "${ref#nan/}" || missing="$missing '$ref'"
    done <<< "$(jq -r '.enabledModels[] | select(startswith("nan/"))' "$PI_SETTINGS")"
    [ -z "$missing" ] || {
        echo "enabledModels referencing no model id in models.json:$missing"
        return 1
    }
}

@test "ai/pi/settings.json defaultModel resolves to an id in models.json" {
    command -v jq >/dev/null || skip "jq not available"
    want="$(jq -r '.defaultModel' "$PI_SETTINGS")"
    jq -r '.. | objects | select(has("id")) | .id' "$PI_MODELS" | grep -qxF "$want"
}

@test "ai/pi/models.json model ids are unique" {
    command -v jq >/dev/null || skip "jq not available"
    dupes="$(jq -r '.. | objects | select(has("id")) | .id' "$PI_MODELS" \
        | LC_ALL=C sort | uniq -d)"
    [ -z "$dupes" ] || { echo "duplicate model ids: $dupes"; return 1; }
}

@test "ai/pi/models.json display names are unique (pickable in the model list)" {
    # Same incident: the new model shipped with the display name of the model it
    # sits beside, so the picker showed two identical entries. The id was unique,
    # so the check above would not have caught it.
    command -v jq >/dev/null || skip "jq not available"
    dupes="$(jq -r '.. | objects | select(has("id") and has("name")) | .name' "$PI_MODELS" \
        | LC_ALL=C sort | uniq -d)"
    [ -z "$dupes" ] || { echo "duplicate model display names: $dupes"; return 1; }
}

# --- Config vs its own documentation -------------------------------------
# Same class one level up (2026-08-05): dropping the `:free` OpenRouter tier
# from enabledModels left ai/pi/README.md advertising three models the picker
# no longer offers -- and exposed two older drifts nobody had noticed, both
# from edits that changed the config without reading the doc beside it. A
# reader trusts the README; nothing made the README answer to the config.

@test "ai/pi/README.md model list matches settings.json enabledModels" {
    command -v jq >/dev/null || skip "jq not available"
    # The tier bullets under "## Model environment" are the documented set:
    # take every backticked token from the "- **Tier**: `a`, `b`" lines.
    doc="$(sed -n '/^## Model environment/,/^## /p' "$PI_README" \
        | grep '^- \*\*' | grep -o '`[^`]*`' | tr -d '`' | LC_ALL=C sort -u)"
    # enabledModels are provider-qualified (`nan/x`, `openrouter/vendor/y`)
    # while the README names the bare id, so compare on the last path segment.
    cfg="$(jq -r '.enabledModels[]' "$PI_SETTINGS" | sed 's|.*/||' | LC_ALL=C sort -u)"
    # Set equality, not containment: one-directional would have missed the
    # model added to the config in #749 and never listed in the README.
    [ "$doc" = "$cfg" ] || {
        echo "README model list and settings.json enabledModels disagree (< README, > config):"
        diff <(printf '%s\n' "$doc") <(printf '%s\n' "$cfg") | sed 's/^/  /'
        return 1
    }
}

@test "ai/pi/README.md documents the actual default model" {
    command -v jq >/dev/null || skip "jq not available"
    # The README advertised `nan/mimo-v2.5` while the config had shipped
    # `qwen3.6` for who knows how long: the default is the single value a new
    # user meets first, and it was the one the doc got wrong.
    want="$(jq -r '.defaultProvider + "/" + .defaultModel' "$PI_SETTINGS")"
    got="$(grep -m1 '^Default: ' "$PI_README" | grep -o '`[^`]*`' | head -1 | tr -d '`')"
    [ "$got" = "$want" ] || {
        echo "README Default: says '$got', settings.json says '$want'"
        return 1
    }
}

# CLI-018: the "missing age identity is optional for pi models.json" assertion
# lived in healthcheck.ps1; it now lives in go test (TestCheckOpenCode pi
# models.json branch) after the .ps1 was retired.
