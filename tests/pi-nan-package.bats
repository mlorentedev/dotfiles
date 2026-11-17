#!/usr/bin/env bats
# AI-046 (#1764): @gtrabanco/pi-nan-provider owns pi's NaN model ids.
#
# Every assertion here is made against what the PACKAGE registers, installed
# into an isolated agent dir with no models.json. With ai/pi/models.json in the
# dir, pi lists the union of both, so a model the package dropped would still
# be listed from our own definition and each check would pass for the wrong
# reason (verification.md, "Package plus today's models.json").
#
# It needs a real pi and the network to install the package, so it runs in the
# `pi-nan-package` CI job, which sets PI_BIN and PI_NAN_PACKAGE_REQUIRED=1.
# Without PI_BIN the file skips; with the REQUIRED flag a skip becomes a
# failure, so the job cannot go green on a setup that never ran.
#
#   PI_BIN=~/.local/bin/pi bats tests/pi-nan-package.bats
#
# The key is a dummy: pi hides a provider without usable auth. It never reaches
# NaN: pi 1.2.0 builds the model runtime without network refresh for
# --list-models, so the package's catalog fetch is not called on this path and
# the listing is the build-time snapshot (measured for #1810 with a preload
# that records every fetch and socket connect: zero attempts). Deterministic,
# and it costs no quota.

setup_file() {
    REPO="$BATS_TEST_DIRNAME/.."
    export REPO
    if [ -z "${PI_BIN:-}" ]; then
        return 0
    fi

    AGENT="$BATS_FILE_TMPDIR/agent"
    mkdir -p "$AGENT"
    cp "$REPO/ai/pi/nan-provider.json" "$AGENT/nan-provider.json"
    export AGENT

    # The pin lives in the manifest; reading it here keeps one source of truth.
    SOURCE="$(jq -r '.packages[].source | select(test("/pi-nan-provider@"))' "$REPO/ai/pi/packages.json")"
    [ -n "$SOURCE" ] || { echo "no pi-nan-provider entry in ai/pi/packages.json" >&2; return 1; }

    PI_CODING_AGENT_DIR="$AGENT" "$PI_BIN" install "$SOURCE" >"$BATS_FILE_TMPDIR/install.log" 2>&1 \
        || { cat "$BATS_FILE_TMPDIR/install.log" >&2; return 1; }

    # Two steps, not a pipe: a pipe would report awk's status and hide pi's.
    NAN_API_KEY=dummy PI_CODING_AGENT_DIR="$AGENT" "$PI_BIN" --list-models nan >"$BATS_FILE_TMPDIR/list.txt"
    awk '$1 == "nan" { print $2 }' "$BATS_FILE_TMPDIR/list.txt" | LC_ALL=C sort -u >"$BATS_FILE_TMPDIR/listed"

    # Exact limits come from the package's snapshot module, not from the listing,
    # which rounds (1000000 and 1048576 both print as "1M"). Node refuses to
    # strip types under node_modules, so the module is copied out first; its
    # only import is a type, which stripping removes.
    PKG="${SOURCE#npm:}"
    PKG="${PKG%@*}"
    cp "$AGENT/npm/node_modules/$PKG/scripts/models.generated.ts" "$BATS_FILE_TMPDIR/snapshot.ts"
    node --input-type=module -e "
        const { pathToFileURL } = await import('node:url');
        const m = await import(pathToFileURL(process.argv[1]).href);
        console.log(JSON.stringify(m.NAN_GENERATED_MODELS));
    " "$BATS_FILE_TMPDIR/snapshot.ts" >"$BATS_FILE_TMPDIR/snapshot.json"
}

setup() {
    if [ -z "${PI_BIN:-}" ]; then
        [ "${PI_NAN_PACKAGE_REQUIRED:-}" != "1" ] || { echo "PI_NAN_PACKAGE_REQUIRED=1 but PI_BIN is unset"; return 1; }
        skip "PI_BIN unset: this file runs in the pi-nan-package CI job"
    fi
    LISTED="$BATS_FILE_TMPDIR/listed"
    SNAPSHOT="$BATS_FILE_TMPDIR/snapshot.json"
    PI_SETTINGS="$REPO/ai/pi/settings.json"
}

@test "pi-nan-package: the package alone registers NaN models, so no check below is vacuous" {
    [ "$(grep -c . "$LISTED")" -ge 1 ]
    jq -e 'length >= 1' "$SNAPSHOT" >/dev/null
}

# AI-048 (#1810): AI-046 AC5, re-runnable. The network is cut inside pi's own
# process by tests/fixtures/pi-network-cut.mjs, which throws on every fetch and
# socket connect, the way the one-off 2026-09-26 measurement did. A dead
# HTTPS_PROXY would prove nothing: Node's fetch ignores proxy variables unless
# the caller opts in, and the package does not. The marker check is the
# positive control: without it a dropped NODE_OPTIONS leaves pi online.
#
# Scope: this is the start path. The package's fetch-failure fallback on the
# session path (where pi refreshes with the network) is not exercised here.
@test "pi-nan-package: with the network cut, pi starts and lists the package snapshot (AI-046 AC5)" {
    marker="$BATS_TEST_TMPDIR/network-cut"
    PI_NETWORK_CUT_MARKER="$marker" NODE_OPTIONS="--import=$REPO/tests/fixtures/pi-network-cut.mjs" \
        NAN_API_KEY=dummy PI_CODING_AGENT_DIR="$AGENT" "$PI_BIN" --list-models nan \
        >"$BATS_TEST_TMPDIR/offline.txt" 2>"$BATS_TEST_TMPDIR/offline.err" \
        || { echo "pi failed to start with the network cut:"; cat "$BATS_TEST_TMPDIR/offline.err"; cat "$marker" 2>/dev/null; false; }
    grep -q '^loaded ' "$marker" || { echo "the network-cut preload never loaded (NODE_OPTIONS dropped?)"; false; }

    awk '$1 == "nan" { print $2 }' "$BATS_TEST_TMPDIR/offline.txt" | LC_ALL=C sort -u >"$BATS_TEST_TMPDIR/offline-listed"
    jq -r '.[].id' "$SNAPSHOT" | LC_ALL=C sort -u >"$BATS_TEST_TMPDIR/snapshot-ids"
    [ -s "$BATS_TEST_TMPDIR/snapshot-ids" ] || { echo "the package snapshot is empty"; false; }
    diff "$BATS_TEST_TMPDIR/snapshot-ids" "$BATS_TEST_TMPDIR/offline-listed" \
        || { echo "offline listing (>) differs from the package snapshot (<)"; false; }
}

@test "pi-nan-package: every nan/* id in enabledModels is registered by the package alone (AC2)" {
    missing=""
    while IFS= read -r ref; do
        [ -n "$ref" ] || continue
        grep -qxF "${ref#nan/}" "$LISTED" || missing="$missing $ref"
    done <<< "$(jq -r '.enabledModels[] | select(startswith("nan/"))' "$PI_SETTINGS" | tr -d '\r')"
    [ -z "$missing" ] || { echo "enabledModels the package does not register:$missing"; return 1; }
}

@test "pi-nan-package: defaultModel is registered by the package alone" {
    [ "$(jq -r '.defaultProvider' "$PI_SETTINGS")" = "nan" ] || skip "the default provider is not nan"
    grep -qxF "$(jq -r '.defaultModel' "$PI_SETTINGS")" "$LISTED"
}

@test "pi-nan-package: every nan member of the reviewer pool is reasoning-class in the package snapshot" {
    while IFS= read -r model; do
        [ -n "$model" ] || continue
        jq -e --arg m "$model" '.[] | select(.id == $m and .reasoning == true)' "$SNAPSHOT" >/dev/null \
            || { echo "pool member nan/$model is not a reasoning model in the package snapshot"; return 1; }
    done < <(jq -r '.pool[] | select(.runner == "pi" and .provider == "nan") | .model' "$REPO/harness/reviewer-pool.json" | tr -d '\r')
}

# #1866: opencode is held to pi's windows offline, against the committed copy in
# ai/pi/nan-package-windows.json (tests/nan-context-windows.bats). This holds the
# copy to the installed package, so a pin bump that changes a window fails here
# instead of leaving the offline test comparing against stale numbers.
@test "pi-nan-package: ai/pi/nan-package-windows.json equals the installed package's windows" {
    want="$(jq -S 'map({key: .id, value: .contextWindow}) | from_entries' "$SNAPSHOT")"
    got="$(jq -S '.contextWindow' "$REPO/ai/pi/nan-package-windows.json")"
    [ "$got" = "$want" ] || { echo "regenerate the committed snapshot's contextWindow as:"; echo "$want"; false; }
}

@test "pi-nan-package: effective package limits equal NaN's published windows and output caps" {
    run python3 - "$REPO/ai/pi/models.json" "$SNAPSHOT" <<'PY'
import json, sys
overrides = json.load(open(sys.argv[1]))["providers"]["nan"]["modelOverrides"]
models = {m["id"]: m for m in json.load(open(sys.argv[2]))}
for model, override in overrides.items():
    if model in models:
        models[model].update(override)
want = {
    "glm5.3-flash": (1000000, 131072),
    "deepseek-v4-flash": (1000000, 384000),
    "qwen3.8-flash": (1048576, 131072),
    "qwen3.6": (262144, 65536),
    "mimo-v2.6-flash": (1048576, 131072),
    "gemma4": (262144, 32768),
}
bad = [f"{model}: got context={models.get(model, {}).get('contextWindow')} output={models.get(model, {}).get('maxTokens')}"
       for model, limits in want.items()
       if (models.get(model, {}).get("contextWindow"), models.get(model, {}).get("maxTokens")) != limits]
print("\n".join(bad))
sys.exit(1 if bad else 0)
PY
    [ "$status" -eq 0 ] || { echo "effective pi limits disagree: $output"; false; }
}
