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
# The key is a dummy: pi hides a provider without usable auth, and NaN answers a
# dummy key with 401, after which the package keeps its build-time snapshot.
# The listing is therefore the snapshot, deterministic, and costs no quota.

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
        const m = await import('$BATS_FILE_TMPDIR/snapshot.ts');
        console.log(JSON.stringify(m.NAN_GENERATED_MODELS));
    " >"$BATS_FILE_TMPDIR/snapshot.json"
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

@test "pi-nan-package: every nan/* id in enabledModels is registered by the package alone (AC2)" {
    missing=""
    while IFS= read -r ref; do
        [ -n "$ref" ] || continue
        grep -qxF "${ref#nan/}" "$LISTED" || missing="$missing $ref"
    done <<< "$(jq -r '.enabledModels[] | select(startswith("nan/"))' "$PI_SETTINGS")"
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

# #1772: opencode and pi describe the same endpoint, so a context window they
# disagree on is wrong in one of them. Since AI-046, pi's side is the package.
@test "pi-nan-package: opencode.jsonc and the package snapshot declare the same context window for every NaN model both carry" {
    run python3 - "$REPO/ai/opencode/opencode.jsonc" "$SNAPSHOT" <<'PY'
import json, re, sys
src = "".join(l for l in open(sys.argv[1]) if not l.lstrip().startswith("//"))
oc = json.loads(re.sub(r",(\s*[}\]])", r"\1", src))["provider"]["nan"]["models"]
pkg = {m["id"]: m for m in json.load(open(sys.argv[2]))}
both = sorted(set(oc) & set(pkg))
if not both:
    print("no NaN model is carried by both, so the comparison would be vacuous")
    sys.exit(1)
bad = [f"{k}: opencode {oc[k]['limit']['context']} package {pkg[k]['contextWindow']}"
       for k in both if oc[k]["limit"]["context"] != pkg[k]["contextWindow"]]
print("\n".join(bad))
sys.exit(1 if bad else 0)
PY
    [ "$status" -eq 0 ] || { echo "context windows disagree: $output"; false; }
}
