#!/usr/bin/env bats
# Every workflow job must declare timeout-minutes.
#
# The incident: on 2026-08-19 the `test` job hung in `apt-get update` against an
# Ubuntu mirror that had stopped answering. Last output 06:30:08, cancelled BY HAND
# at 06:38:22 — eight minutes of silence, and not the first time. `test-windows`
# already carried `timeout-minutes: 30`; the job that actually hung carried none, so
# nothing could end it but a person watching.
#
# This asserts the only mechanism that kills a hung job, on every job, so the next
# hang — which will have a different cause, because this one is now bounded — ends
# by itself. It is deliberately a presence check: here the declaration IS the
# mechanism. GitHub enforces it; there is no gap between saying it and it happening.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    command -v python3 >/dev/null 2>&1 || skip "python3 required to parse workflow YAML"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

@test "every workflow job declares timeout-minutes" {
    run python3 - "$REPO" <<'PY'
import glob, os, sys, yaml

missing = []
for path in sorted(glob.glob(os.path.join(sys.argv[1], ".github", "workflows", "*.yml"))):
    with open(path, encoding="utf-8") as fh:
        doc = yaml.safe_load(fh) or {}
    for name, job in (doc.get("jobs") or {}).items():
        # A `uses:` job calls a reusable workflow, whose own jobs carry the timeout.
        if not isinstance(job, dict) or "uses" in job:
            continue
        if "timeout-minutes" not in job:
            missing.append(f"{os.path.basename(path)}:{name}")

if missing:
    print("jobs with no timeout-minutes — a hang in one of these ends only when a human cancels it:")
    for m in missing:
        print("  " + m)
    sys.exit(1)
PY
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "the apt step that hung is bounded by retries and a timeout" {
    # Narrower than the rule above and kept separately: the timeout ends a hang,
    # this stops it starting. A dead mirror should cost ~45s and a readable error.
    grep -q 'Acquire::Retries' "$REPO/.github/workflows/ci.yml"
    grep -q 'Acquire::http::Timeout' "$REPO/.github/workflows/ci.yml"
}
