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
    [[ -f "$PI_MODELS" ]] || false
}

@test "ai/pi/models.json has no literal API key" {
    refute_grep '"apiKey"[[:space:]]*:[[:space:]]*"sk-' "$PI_MODELS"
}

@test "every standalone provider in ai/pi/models.json takes its key from a \${VAR} placeholder, resolved at runtime" {
    command -v jq >/dev/null || skip "jq not available"
    bad="$(jq -r '.providers | to_entries[] | select(.value.models) | select((.value.apiKey // "") | test("^[$][{][A-Z0-9_]+[}]$") | not) | .key' "$PI_MODELS")"
    [ -z "$bad" ] || { echo "providers whose apiKey is not a \${VAR} placeholder: $bad"; return 1; }
}

# AI-046: pi-nan-provider owns the NaN provider and reads NAN_API_KEY from pi's
# environment, so models.json no longer names the key. What delivers it is the
# wrapper, which injects it per process (ADR-028); drop NAN_API_KEY from any one
# of them and pi on that shell loses NaN with no config change to review.
@test "the pi wrapper hands NAN_API_KEY to pi through dotf secrets run, on every shell" {
    for f in .zshrc .bashrc; do
        grep -qE '^[[:space:]]*pi\(\) \{ dotf secrets run --only [A-Z_,]*NAN_API_KEY[A-Z_,]* -- pi "\$@"; \}' "$DOTFILES_DIR/$f" \
            || { echo "$f: no pi wrapper injecting NAN_API_KEY"; return 1; }
    done
    grep -qE '^[[:space:]]*function pi \{' "$DOTFILES_DIR/powershell/profile.ps1" \
        || { echo "profile.ps1: no pi wrapper"; return 1; }
    grep -qE '^[[:space:]]*\$env_b64 = \(dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- powershell -NoProfile -Command "\[Convert\]::ToBase64String' "$DOTFILES_DIR/powershell/profile.ps1" \
        || { echo "profile.ps1: pi wrapper does not extract NAN_API_KEY through secrets run"; return 1; }
}

# AI-046 AC3. A NaN model defined here replaces the package's model with the
# same id (pi docs/models.md), so the live catalog and the package's limits
# would silently stop applying to it. Per-model changes belong in
# providers.nan.modelOverrides, which composes above the package instead.
@test "ai/pi/models.json defines no NaN model: pi-nan-provider owns them (AI-046 AC3)" {
    command -v jq >/dev/null || skip "jq not available"
    run jq -r '.providers.nan.models // [] | .[].id' "$PI_MODELS"
    [ "$status" -eq 0 ]
    [ -z "$output" ] || { echo "NaN models defined in ai/pi/models.json: $output"; return 1; }
}

@test "pi model overrides carry NaN's published windows and output caps (AI-045 AC5)" {
    command -v jq >/dev/null || skip "jq not available"
    run jq -e '
      .providers.nan.modelOverrides as $m
      | ($m["glm5.3-flash"]      == {"contextWindow":1000000,"maxTokens":131072})
      and ($m["deepseek-v4-flash"] == {"contextWindow":1000000,"maxTokens":384000})
      and ($m["qwen3.8-flash"]     == {"contextWindow":1048576,"maxTokens":131072})
      and ($m["qwen3.6"]           == {"contextWindow":262144,"maxTokens":65536})
      and ($m["mimo-v2.6-flash"]   == {"contextWindow":1048576,"maxTokens":131072})
      and ($m["gemma4"]            == {"contextWindow":262144,"maxTokens":32768})
    ' "$PI_MODELS"
    [ "$status" -eq 0 ] || { echo "pi NaN modelOverrides do not match the published limits"; false; }
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

# pi decides the thinking shape per model, not per API: it sends adaptive thinking
# only when the model declares compat.forceAdaptiveThinking, and otherwise the
# budget form ({"type": "enabled"}). Claude 5.x rejects the budget form with a
# 400, so a custom Claude model without the flag fails every request it makes;
# the second-signer and fallback reviews (ADR-046) died this way on their first
# live run. An older Claude model that needs budget thinking must say so here.
@test "every reasoning model on the Anthropic Messages API declares adaptive thinking" {
    command -v jq >/dev/null || skip "jq not available"
    # One filter feeds both checks, so the non-empty guard counts exactly the set
    # the flag is asserted on: a renamed `reasoning` or `api` empties it and fails.
    local set='[.providers[] | . as $p | (.models // [])[]
        | select((.api // $p.api) == "anthropic-messages" and .reasoning == true)]'
    [ "$(jq "$set | length" "$PI_MODELS")" -gt 0 ] || { echo "no Anthropic reasoning model found: the filter matches nothing"; return 1; }
    bad="$(jq -r "$set | .[] | select(.compat.forceAdaptiveThinking != true) | .id" "$PI_MODELS")"
    [ -z "$bad" ] || { echo "Anthropic reasoning models without compat.forceAdaptiveThinking: $bad"; return 1; }
}

@test "ai/pi/models.json is valid JSON" {
    command -v jq >/dev/null || skip "jq not available"
    jq empty "$PI_MODELS"
}

@test "ai/pi/settings.json is valid JSON" {
    command -v jq >/dev/null || skip "jq not available"
    jq empty "$PI_SETTINGS"
}

# AI-046 AC4: pi-nan-provider's media bridge runs `npx -y nan-mcp-server@<v>` per
# tool call, a second npm package the pinned manifest never sees. The package
# resolves the toggle env var > ~/.pi/agent/nan-provider.json > default ON, so
# the managed file is what keeps it off on every launch path. `merge`, not
# `replace`: webSearch in the same file is the box's own /nan-mcp toggle.
@test "pi-nan-provider's media MCP bridge is deployed OFF, merged into the package's state file" {
    command -v jq >/dev/null || skip "jq not available"
    local entry
    entry="$(jq -c '.configs[] | select(.dst == "{HOME}/.pi/agent/nan-provider.json")' "$DOTFILES_DIR/ai/deploy.json")"
    [ -n "$entry" ]
    [ "$(printf '%s' "$entry" | jq -r '.strategy')" = "merge" ]
    [ "$(printf '%s' "$entry" | jq -r '.requires')" = "pi" ]
    [ "$(jq -r '.mediaMcp' "$DOTFILES_DIR/$(printf '%s' "$entry" | jq -r '.src')")" = "false" ]
}

# #843 / #2100: models.json is rendered with NAN_API_KEY into a 0600 file,
# and on a box without pi that file is read by nothing -- the integration
# image deployed it half-rendered ("NAN_API_KEY could not be resolved").
# pi-compaction is deliberately NOT gated: it merges into the settings.json
# setup seeds on every box (#1938), and verify-setup asserts the merge there.
@test "pi's models.json is deployed only where pi is installed" {
    command -v jq >/dev/null || skip "jq not available"
    local entry
    entry="$(jq -c '.configs[] | select(.dst == "{HOME}/.pi/agent/models.json")' "$DOTFILES_DIR/ai/deploy.json")"
    [ -n "$entry" ]
    [ "$(printf '%s' "$entry" | jq -r '.requires')" = "pi" ]
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
#
# Since AI-046 the NaN ids come from pi-nan-provider, not models.json, so the
# enabledModels and defaultModel checks run against the installed package in
# tests/pi-nan-package.bats. What stays here is models.json's own consistency.

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
        }' | tr -d '\r' | LC_ALL=C sort -u)"
    # `openrouter/deepseek/deepseek-v4-pro` -> `openrouter/deepseek-v4-pro`:
    # the README names the bare id under its tier, not the vendor path.
    cfg="$(jq -r '.enabledModels[] | split("/") | .[0] + "/" + .[-1]' "$PI_SETTINGS" \
        | tr -d '\r' | LC_ALL=C sort -u)"
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

# --- #1938: declarative compaction, merged into the seeded settings.json ---
# pi auto-compacts when contextTokens > contextWindow - reserveTokens, so a 40%
# trigger is reserveTokens = round(0.6 x contextWindow). The window lives in
# models.json; deriving the expectation from it means a window change that
# forgets compaction.json fails here instead of silently moving the trigger.

@test "ai/pi/compaction.json holds only the compaction key (it is merged into a pi-owned file)" {
    command -v jq >/dev/null || skip "jq not available"
    keys="$(jq -c 'keys' "$DOTFILES_DIR/ai/pi/compaction.json")"
    [ "$keys" = '["compaction"]' ] || { echo "top-level keys: $keys"; return 1; }
    # enabled is global in pi; owning it here would override a box's own choice.
    run jq -e '.compaction | has("enabled")' "$DOTFILES_DIR/ai/pi/compaction.json"
    [ "$status" -ne 0 ] || { echo "compaction.json sets enabled -- native auto-compaction stays the box's"; return 1; }
}

@test "every enabled NaN model compacts at 40% of its models.json window (#1938)" {
    command -v jq >/dev/null || skip "jq not available"
    diff="$(jq -rn \
        --slurpfile s "$PI_SETTINGS" \
        --slurpfile m "$PI_MODELS" \
        --slurpfile c "$DOTFILES_DIR/ai/pi/compaction.json" '
        ($c[0].compaction.modelOverrides) as $o
        | ($m[0].providers.nan.modelOverrides) as $w
        | ($s[0].enabledModels | map(select(startswith("nan/")))) as $enabled
        | ([$enabled[] | . as $k | ($k | ltrimstr("nan/")) as $id
            | if ($w[$id].contextWindow | type) != "number" then "\($k): no contextWindow in models.json"
              elif $o[$k].reserveTokens != ($w[$id].contextWindow * 6 / 10 | round)
              then "\($k): reserveTokens \($o[$k].reserveTokens), want \($w[$id].contextWindow * 6 / 10 | round)"
              else empty end]
          + [$o | keys[] | select(. as $k | $enabled | index($k) | not) | "\(.): override for a model not in enabledModels"])
        | .[]')"
    [ -z "$diff" ] || { echo "$diff"; return 1; }
    # Pin two values by hand, so a broken derivation cannot agree with itself.
    [ "$(jq '.compaction.modelOverrides["nan/qwen3.8-flash"].reserveTokens' "$DOTFILES_DIR/ai/pi/compaction.json")" = 629146 ]
    [ "$(jq '.compaction.modelOverrides["nan/qwen3.6"].reserveTokens' "$DOTFILES_DIR/ai/pi/compaction.json")" = 157286 ]
}

@test "ai/deploy.json merges compaction.json into pi's settings.json" {
    command -v jq >/dev/null || skip "jq not available"
    entry="$(jq -c '.configs[] | select(.name == "pi-compaction") | [.src, .dst, .strategy]' "$DOTFILES_DIR/ai/deploy.json")"
    [ "$entry" = '["ai/pi/compaction.json","{HOME}/.pi/agent/settings.json","merge"]' ] \
        || { echo "pi-compaction entry: ${entry:-missing}"; return 1; }
}

# A merge onto a missing destination creates it, so if `dotf deploy` ran first
# on a fresh box it would write a settings.json holding only compaction, and
# the seed above would then see a file and never write defaultModel. The order
# of two blocks in a script is incidental until something asserts it.
@test "both setups seed pi settings.json BEFORE dotf deploy merges compaction into it" {
    seed="$(grep -n '^PI_SETTINGS_SRC=' "$DOTFILES_DIR/setup-linux.sh" | cut -d: -f1)"
    deploy="$(grep -n '"$_dotf" deploy' "$DOTFILES_DIR/setup-linux.sh" | cut -d: -f1)"
    [ -n "$seed" ] && [ -n "$deploy" ] || { echo "linux anchors missing: seed=$seed deploy=$deploy"; return 1; }
    [ "$seed" -lt "$deploy" ] || { echo "setup-linux.sh: seed at $seed, dotf deploy at $deploy"; return 1; }
    seed="$(grep -n '^\$piSettingsSrc = ' "$DOTFILES_DIR/setup-windows.ps1" | cut -d: -f1)"
    deploy="$(grep -n '& dotf deploy' "$DOTFILES_DIR/setup-windows.ps1" | cut -d: -f1)"
    [ -n "$seed" ] && [ -n "$deploy" ] || { echo "windows anchors missing: seed=$seed deploy=$deploy"; return 1; }
    [ "$seed" -lt "$deploy" ] || { echo "setup-windows.ps1: seed at $seed, dotf deploy at $deploy"; return 1; }
}

# --- AI-032 (#1247): enabledModels field-level sync on an EXISTING settings.json ---
# The two tests above guard the whole-file seed-if-missing contract; these guard
# the layer on top of it that lets a catalog addition (e.g. #1254's
# qwen3.8-flash/glm5.3-flash) reach a machine that already has settings.json,
# without resetting theme/defaultModel/lastChangelogVersion -- the exact bug #756
# fixed by making the whole file seed-once. This one IS fully executable outside
# a container (pure jq over temp files, no $HOME involved), unlike the seed
# block's second-run behavior that lesson-150 documents as uncoverable here.

# bats test_tags=os-sensitive
@test "setup-linux.sh pi enabledModels sync updates the model list and preserves theme/defaultModel/lastChangelogVersion" {
    command -v jq >/dev/null || skip "jq not available"
    PI_SETTINGS_SRC="$BATS_TEST_TMPDIR/src.json"
    PI_SETTINGS_DST="$BATS_TEST_TMPDIR/dst.json"
    cat > "$PI_SETTINGS_SRC" <<'JSON'
{"enabledModels": ["nan/qwen3.6", "nan/glm5.3-flash"]}
JSON
    cat > "$PI_SETTINGS_DST" <<'JSON'
{"theme": "light", "defaultModel": "nan/mimo-v2.5", "lastChangelogVersion": "0.50.0", "enabledModels": ["nan/qwen3.6"]}
JSON

    # Stubs for the real script's logger calls -- this test only cares about
    # the file mutation, not the log lines. eval runs in THIS shell, so no
    # export/subshell is needed for the block to see them.
    log_info() { :; }
    log_success() { :; }
    log_warning() { :; }

    block="$(sed -n '/^# Field-level sync (AI-032/,/^fi$/p' "$DOTFILES_DIR/setup-linux.sh")"
    [ -n "$block" ] || { echo "pi enabledModels sync block not found"; return 1; }
    eval "$block"

    run jq -e '.enabledModels == ["nan/qwen3.6","nan/glm5.3-flash"]
        and .theme == "light"
        and .defaultModel == "nan/mimo-v2.5"
        and .lastChangelogVersion == "0.50.0"' "$PI_SETTINGS_DST"
    [ "$status" -eq 0 ] || { echo "post-sync dst.json:"; cat "$PI_SETTINGS_DST"; return 1; }

    # Second run: the already-in-sync branch must not error and must not
    # change the result (idempotent -- same jq block, same fixtures now equal).
    eval "$block"
    run jq -e '.enabledModels == ["nan/qwen3.6","nan/glm5.3-flash"] and .theme == "light"' "$PI_SETTINGS_DST"
    [ "$status" -eq 0 ] || { echo "second run changed dst.json:"; cat "$PI_SETTINGS_DST"; return 1; }
}

@test "setup-windows.ps1 pi enabledModels sync block guards both files, preserves other fields, avoids the pipe/ConvertTo-Json array-collapse bug (Linux parity)" {
    block="$(extract_block "$DOTFILES_DIR/setup-windows.ps1" '^# Field-level sync (AI-032' '^}$')"
    [ -n "$block" ] || { echo "pi enabledModels sync block not found"; return 1; }
    printf '%s\n' "$block" | grep -qF -e 'Test-Path -LiteralPath $piSettingsDst -PathType Leaf' \
        || { echo "sync is not guarded on the destination existing:"; printf '%s\n' "$block"; return 1; }
    printf '%s\n' "$block" | grep -qF -e 'Test-Path -LiteralPath $piSettingsSrc -PathType Leaf' \
        || { echo "sync is not guarded on the source existing:"; printf '%s\n' "$block"; return 1; }
    # @($null).Count is 1, not 0 -- wrapping the source's enabledModels in @()
    # before checking for null/missing would make the "no models" guard never
    # fire and go on to write a literal null into a live settings.json. The
    # null check must happen BEFORE the @() wrap (reviewer-found, adversarial
    # review of this spec, initially FAIL on exactly this).
    printf '%s\n' "$block" | grep -qF -e '$null -eq $piSrcModelsRaw' \
        || { echo "src guard does not null-check before wrapping in @() -- @(\$null).Count is 1, not 0:"; printf '%s\n' "$block"; return 1; }
    # -InputObject, never a bare pipe into ConvertTo-Json: piping unwraps a
    # single-element array before the cmdlet sees a collection.
    printf '%s\n' "$block" | grep -qF -e 'ConvertTo-Json -InputObject $piSrcModels' \
        || { echo "src comparison pipes into ConvertTo-Json instead of using -InputObject:"; printf '%s\n' "$block"; return 1; }
    printf '%s\n' "$block" | grep -qF -e 'ConvertTo-Json -InputObject $piDstModels' \
        || { echo "dst comparison pipes into ConvertTo-Json instead of using -InputObject:"; printf '%s\n' "$block"; return 1; }
    # Only .enabledModels may be assigned -- theme/defaultModel/lastChangelogVersion
    # are pi's own runtime state and this block must never write them.
    assert_absent '.theme =' "the sync block assigns .theme -- that is pi's own runtime state"
    assert_absent '.defaultModel =' "the sync block assigns .defaultModel -- that is pi's own runtime state"
    assert_absent '.lastChangelogVersion =' "the sync block assigns .lastChangelogVersion -- that is pi's own runtime state"
}

# CLI-018: the "missing age identity is optional for pi models.json" assertion
# lived in healthcheck.ps1; it now lives in go test (TestCheckOpenCode pi
# models.json branch) after the .ps1 was retired.
