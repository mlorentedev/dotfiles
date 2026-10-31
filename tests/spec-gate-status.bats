#!/usr/bin/env bats
# The required `spec-gate` context is a commit status the workflow posts, not a
# job's check run (#2866). One push can fire two events in the same second; the
# concurrency group cancels the older pending run, and a cancelled check run
# under the required name kept the PR BLOCKED next to a green sibling. These
# tests hold the three things that fix depends on.

setup() {
    WORKFLOW="$BATS_TEST_DIRNAME/../.github/workflows/spec-gate.yml"
    command -v python3 >/dev/null 2>&1 || skip "python3 required to parse workflow YAML"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

# Prints one line per job: its display name, then the `if:` of every step that
# posts the spec-gate status.
inspect() {
    python3 - "$WORKFLOW" <<'PY'
import re
import sys

import yaml

doc = yaml.safe_load(open(sys.argv[1]))
print("statuses=" + str((doc.get("permissions") or {}).get("statuses")))
for key, job in doc["jobs"].items():
    print("job=" + str(job.get("name", key)))
    for step in job.get("steps", []):
        if re.search(r"""-f\s+context=["']?spec-gate["'\s]""", step.get("run", "")):
            print("publish-if=" + str(step.get("if")))
PY
}

@test "spec-gate status: no job reports a check run named spec-gate" {
    run inspect
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    if printf '%s\n' "$output" | grep -qx 'job=spec-gate'; then
        printf 'a job is named spec-gate; its cancelled runs block the merge (#2866)\n' >&2
        false
    fi
}

@test "spec-gate status: the verdict is posted on success or failure, never on cancel" {
    run inspect
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    # Exactly one publishing step. always() would post for a cancelled run; no
    # condition at all would never post a failure, leaving the context pending.
    [ "$(printf '%s\n' "$output" | grep -c '^publish-if=')" -eq 1 ] || { printf '%s\n' "$output" >&2; false; }
    printf '%s\n' "$output" | grep -qx 'publish-if=success() || failure()' || { printf '%s\n' "$output" >&2; false; }
}

@test "spec-gate status: the workflow may write commit statuses" {
    run inspect
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    printf '%s\n' "$output" | grep -qx 'statuses=write' || { printf '%s\n' "$output" >&2; false; }
}
