#!/usr/bin/env bats
# #1866: opencode and pi describe the same NaN endpoint, so a context window
# they disagree on is wrong in one of them.
#
# pi's windows come from the pinned @gtrabanco/pi-nan-provider package (AI-046),
# which only exists once installed, so tests/pi-nan-package.bats can only run in
# CI. ai/pi/nan-package-windows.json is a committed copy of the package's
# windows; this file holds opencode to it offline, and pi-nan-package.bats holds
# the copy to the installed package, so neither side can drift unseen.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    WINDOWS="$REPO/ai/pi/nan-package-windows.json"
}

@test "nan-context-windows: the snapshot names the package version packages.json pins" {
    pin="$(jq -r '.packages[].source | select(test("/pi-nan-provider@"))' "$REPO/ai/pi/packages.json")"
    [ -n "$pin" ]
    [ "$(jq -r '.source' "$WINDOWS")" = "$pin" ] \
        || { echo "snapshot is for $(jq -r '.source' "$WINDOWS"), packages.json pins $pin"; false; }
}

@test "nan-context-windows: opencode.jsonc declares pi's effective window for every NaN model both carry" {
    run python3 - "$REPO/ai/opencode/opencode.jsonc" "$REPO/ai/pi/models.json" "$WINDOWS" <<'PY'
import json, re, sys
src = "".join(l for l in open(sys.argv[1]) if not l.lstrip().startswith("//"))
oc = json.loads(re.sub(r",(\s*[}\]])", r"\1", src))["provider"]["nan"]["models"]
overrides = json.load(open(sys.argv[2]))["providers"]["nan"]["modelOverrides"]
pi = dict(json.load(open(sys.argv[3]))["contextWindow"])
# models.json overrides the package per model, as pi applies it.
for model, override in overrides.items():
    if model in pi and "contextWindow" in override:
        pi[model] = override["contextWindow"]
both = sorted(set(oc) & set(pi))
if not both:
    print("no NaN model is carried by both, so the comparison would be vacuous")
    sys.exit(1)
bad = [f"{k}: opencode {oc[k]['limit']['context']} pi {pi[k]}"
       for k in both if oc[k]["limit"]["context"] != pi[k]]
print("\n".join(bad))
sys.exit(1 if bad else 0)
PY
    [ "$status" -eq 0 ] || { echo "context windows disagree: $output"; false; }
}
