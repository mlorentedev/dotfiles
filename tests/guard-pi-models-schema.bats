#!/usr/bin/env bats
#
# ai/pi/models.json is deployed verbatim to ~/.pi/agent/models.json, and pi
# validates the WHOLE file against its own schema. One bad field does not
# degrade a single model — pi rejects the file and reports "No models
# available", which takes out `pi`, the `qq`/`qf` wrappers, and every NaN arm of
# the adversarial-review pool at once.
#
# Measured 2026-09-05: #1471 shipped `"audio"` in mimo-v2.5's input array. pi
# 0.84.4 rejects it:
#
#   Invalid models.json schema:
#     - providers.nan.models.4.input.2: must be equal to constant
#   No models available.
#
# It sat in the repo unnoticed because the DEPLOYED copy was still an older
# valid one; the breakage only appeared when a deploy overwrote it, four
# commits later, and presented as "the reviewer pool is down".

setup() {
    REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
    MODELS="$REPO_ROOT/ai/pi/models.json"
}

@test "guard: ai/pi/models.json exists and is valid JSON" {
    [ -f "$MODELS" ]
    run python3 -c "import json,sys; json.load(open('$MODELS'))"
    [ "$status" -eq 0 ]
}

@test "guard: every model's input modality is one pi accepts" {
    # pi 0.84.4 accepts "text" and "image". Anything else makes it discard the
    # entire file. Kept as an explicit allow-list: a new modality should fail
    # here and be verified against the installed pi before being added, which is
    # exactly the step #1471 skipped.
    run python3 - "$MODELS" <<'PY'
import json, sys
allowed = {"text", "image"}
bad = []
d = json.load(open(sys.argv[1]))
for pname, p in d.get("providers", {}).items():
    for i, m in enumerate(p.get("models", [])):
        for v in m.get("input", []):
            if v not in allowed:
                bad.append(f"providers.{pname}.models.{i} ({m.get('id')}): input {v!r}")
if bad:
    print("\n".join(bad))
    sys.exit(1)
PY
    [ "$status" -eq 0 ]
}

@test "guard: if pi is installed, it actually accepts the file we ship" {
    # The allow-list above is a proxy; this is the real thing. Skipped where pi
    # is absent (CI) rather than faked, and it names why.
    if ! command -v pi >/dev/null 2>&1; then
        skip "pi is not installed here; the modality allow-list above is the standing check"
    fi
    # PI_CODING_AGENT_DIR redirects pi's config directory. Verified empirically
    # that it is honoured: pointing it at a deliberately broken copy reproduces
    # "Invalid models.json schema". A first version used PI_CONFIG_DIR, which pi
    # ignores -- so it read the real config and passed while the file under test
    # was broken. It was caught by mutating the fixture, not by reading it.
    cp "$MODELS" "$BATS_TEST_TMPDIR/models.json"
    run env PI_CODING_AGENT_DIR="$BATS_TEST_TMPDIR" pi --list-models
    # pi exits 0 even on a rejected file, so assert on the message, not status.
    [[ "$output" != *"Invalid models.json schema"* ]]
}
