#!/usr/bin/env bats
# Guard (incident->guard, AI-025): the pi coding agent config carries no
# plaintext secret in git, and the curated model set stays consistent with
# its own README (NaN + non-big-3 paid OpenRouter).

load 'lib/refute'

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
    refute_grep '"apiKey"[[:space:]]*:[[:space:]]*"sk-' "$PI_MODELS"
}

@test "ai/pi/models.json uses the \${NAN_API_KEY} placeholder, resolved at runtime" {
    grep -qF '${NAN_API_KEY}' "$PI_MODELS"
}

# The regression guard for BUG-081b itself (ADR-034). pi does NOT implement its own
# {env:...} syntax, so a config carrying that placeholder ships a literal, unresolvable
# string as the API key -- and pi's preflight reported it as "configured", which is why
# this shipped broken and looked fine. The value is resolved before pi ever sees it
# (`dotf secrets run`), so the only correct placeholder is the shell form.
#
# This assertion is the inverse of the one it replaced: that test required the broken
# form and passed for as long as the bug existed.
@test "ai/pi/models.json carries no {env:...} placeholder pi cannot resolve [BUG-081b]" {
    refute_grep_fixed '{env:' "$PI_MODELS"
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
    refute_grep_fixed 'lastChangelogVersion' "$PI_SETTINGS"
}

@test "ai/pi/settings.json enabledModels exclude OpenAI/Google/Anthropic OpenRouter providers" {
    refute_grep 'openrouter/(openai|google|anthropic)/' "$PI_SETTINGS"
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
    # Both sides normalise to `provider/leaf`. Comparing bare leaf ids would
    # make `nan/foo` equal `openrouter/vendor/foo`, so a model documented under
    # the wrong tier would pass; carrying the provider keeps the tiers honest.
    # A bullet whose label is neither known tier yields `?/...`, which cannot
    # match anything -- a new tier has to teach this test about itself.
    doc="$(sed -n '/^## Model environment/,/^## /p' "$PI_README" | awk '
        /^- \*\*/ {
            p = "?"
            if ($0 ~ /^- \*\*NaN\*\*/) p = "nan"
            else if ($0 ~ /^- \*\*Paid OpenRouter\*\*/) p = "openrouter"
            line = $0
            while (match(line, /`[^`]*`/)) {
                print p "/" substr(line, RSTART + 1, RLENGTH - 2)
                line = substr(line, RSTART + RLENGTH)
            }
        }' | LC_ALL=C sort -u)"
    # `openrouter/deepseek/deepseek-v4-pro` -> `openrouter/deepseek-v4-pro`:
    # the README names the bare id under its tier, not the vendor path.
    cfg="$(jq -r '.enabledModels[] | split("/") | .[0] + "/" + .[-1]' "$PI_SETTINGS" \
        | LC_ALL=C sort -u)"
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

# --- The deploy contract the docs promise ---------------------------------
# Both setup scripts shipped the neighbouring "copy unless byte-identical"
# shape for settings.json. pi rewrites that file at runtime and the committed
# copy is forbidden to carry lastChangelogVersion (asserted above), so the
# comparison could never match once pi had run: the "already in sync" branch
# was dead code and every setup run reset the user's theme and default model,
# while README and tests had described it as seed-if-missing since AI-025.
# Source-level assertions, matching tests/harness-refresh-announce.bats -- the
# integration container seeds a fresh HOME, so it cannot observe the second run.

# extract_block <file> <start-regex> <end-regex>: the deploy block, CR stripped.
# setup-windows.ps1 is CRLF (.gitattributes), so an end anchor like /^}$/ never
# matches -- sed then prints to EOF and the negative assertions below would
# silently cover the whole rest of the file instead of this block. The caller
# checks the size for exactly that reason.
extract_block() {
    tr -d '\r' < "$1" | sed -n "/$2/,/$3/p"
}

# assert_absent <needle> <message>: grep for a literal that may start with `-`.
# `grep -qF '-Force'` parses the pattern as options (-F -o -r -c -e), leaves -e
# without its argument and exits 2, so the guard never fires -- a check that
# only knows how to pass. `-e` states that the next word is the pattern.
assert_absent() {
    printf '%s\n' "$block" | grep -qF -e "$1" && { echo "$2"; return 1; }
    return 0
}

@test "setup-linux.sh seeds pi settings.json only when absent" {
    block="$(extract_block "$DOTFILES_DIR/setup-linux.sh" '^PI_SETTINGS_SRC=' '^fi$')"
    [ -n "$block" ] || { echo "pi settings deploy block not found"; return 1; }
    [ "$(printf '%s\n' "$block" | wc -l)" -le 20 ] \
        || { echo "range never closed -- assertions would cover unrelated code"; return 1; }
    printf '%s\n' "$block" | grep -qF -e 'if [ -f "$PI_SETTINGS_DST" ]' \
        || { echo "deploy is not guarded on the destination being absent:"; printf '%s\n' "$block"; return 1; }
    assert_absent 'cmp -s "$PI_SETTINGS_SRC"' \
        "settings.json compares against the destination again -- that branch is dead code for a self-mutating file"
}

@test "setup-windows.ps1 seeds pi settings.json only when absent (Linux parity)" {
    block="$(extract_block "$DOTFILES_DIR/setup-windows.ps1" '^\$piSettingsSrc = ' '^}$')"
    [ -n "$block" ] || { echo "pi settings deploy block not found"; return 1; }
    [ "$(printf '%s\n' "$block" | wc -l)" -le 20 ] \
        || { echo "range never closed -- assertions would cover unrelated code"; return 1; }
    printf '%s\n' "$block" | grep -qF -e 'if (Test-Path -LiteralPath $piSettingsDst)' \
        || { echo "deploy is not guarded on the destination being absent:"; printf '%s\n' "$block"; return 1; }
    assert_absent 'Compare-Object' \
        "settings.json compares against the destination again -- that branch is dead code for a self-mutating file"
    # -Force would overwrite the very file the guard exists to protect.
    assert_absent '-Force' "Copy-Item still passes -Force"
}

# CLI-018: the "missing age identity is optional for pi models.json" assertion
# lived in healthcheck.ps1; it now lives in go test (TestCheckOpenCode pi
# models.json branch) after the .ps1 was retired.
