#!/usr/bin/env bats
# Tests for HARNESS-041: CI matrix path filtering (Issue #552)
#
# The workflow is parsed, not grepped: a grep for `needs.changes.outputs.code`
# anywhere in the file stayed green with a heavy step's guard deleted, and a
# grep for `changes` inside a job block stayed green with its `needs:` deleted
# (round-1 review, mutations 1 and 2).

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CI_YML="$DOTFILES_DIR/.github/workflows/ci.yml"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

# The jobs that stay active on every PR (so their required checks report) and
# skip only their heavy steps.
FILTERED_JOBS="lint lint-powershell test test-windows integration"

@test "HARNESS-041: the changes job runs a SHA-pinned dorny/paths-filter" {
    run python3 - "$CI_YML" <<'EOF'
import re, sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))["jobs"]["changes"]["steps"]
uses = [s.get("uses", "") for s in steps]
ok = [u for u in uses if re.fullmatch(r"dorny/paths-filter@[0-9a-f]{40}", u)]
print(uses)
sys.exit(0 if ok else 1)
EOF
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "HARNESS-041: every filtered job needs the changes job" {
    run python3 - "$CI_YML" $FILTERED_JOBS <<'EOF'
import sys, yaml
jobs = yaml.safe_load(open(sys.argv[1]))["jobs"]
bad = []
for name in sys.argv[2:]:
    needs = jobs[name].get("needs", [])
    needs = [needs] if isinstance(needs, str) else needs
    if "changes" not in needs:
        bad.append(name)
print("jobs without needs: [changes]:", bad)
sys.exit(1 if bad else 0)
EOF
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "HARNESS-041: every step after checkout in a filtered job is guarded by its own changes output" {
    run python3 - "$CI_YML" $FILTERED_JOBS <<'EOF'
import sys, yaml
jobs = yaml.safe_load(open(sys.argv[1]))["jobs"]
# Which filter decides each job. A guard on the wrong output passes a
# "has any guard" check and still skips the job on the PRs it exists for
# (round-2 review, mutation 6).
output = {"lint-powershell": "powershell"}
bad = []
for name in sys.argv[2:]:
    want = f"needs.changes.outputs.{output.get(name, 'code')} =="
    for i, step in enumerate(jobs[name]["steps"]):
        if step.get("uses", "").startswith("actions/checkout@"):
            continue
        if want not in str(step.get("if", "")):
            bad.append(f"{name}: {step.get('name') or step.get('uses') or i}")
print("unguarded steps:", bad)
sys.exit(1 if bad else 0)
EOF
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# Every tracked top-level entry is either in the `code` filter or declared
# docs-only here, with the reason. A new directory that is in neither would be
# classified as docs and skip every test without anyone deciding that.
@test "HARNESS-041: every top-level entry is in the code filter or declared docs-only" {
    command -v git >/dev/null 2>&1 || skip "git required to list tracked files"
    run python3 - "$CI_YML" "$DOTFILES_DIR" <<'EOF'
import subprocess, sys, yaml
jobs = yaml.safe_load(open(sys.argv[1]))["jobs"]
steps = jobs["changes"]["steps"]
filters = next(s["with"]["filters"] for s in steps if s.get("id") == "filter")
code = set(yaml.safe_load(filters)["code"])

docs_only = {
    "docs/":    "prose; check-doc-paths and check-lessons run in pre-commit, docs-drift on the push to main",
    "specs/":   "spec records; archive PRs are the common case, and the spec-id guard runs on the push to main",
    "README.md": "prose",
    "CHANGELOG.md": "written by release-please",
    "LICENSE": "legal text",
    "SECURITY.md": "prose",
    ".coderabbit.yaml": "reviewer config, read by no test",
    "release-please-config.json": "release tooling; a release PR's content was tested on main",
    ".release-please-manifest.json": "release tooling; a release PR's content was tested on main",
    "diff.patch": "stray file, removal tracked in #1869",
}

tracked = subprocess.run(["git", "-C", sys.argv[2], "ls-files"],
                         capture_output=True, text=True, check=True).stdout.split()
entries = {p.split("/")[0] + "/" if "/" in p else p for p in tracked}
if not entries:
    print("git ls-files listed nothing"); sys.exit(1)

def covered(e):
    return (e[:-1] + "/**" if e.endswith("/") else e) in code

bad = sorted(e for e in entries if not covered(e) and e not in docs_only)
print("top-level entries in neither list:", bad)
sys.exit(1 if bad else 0)
EOF
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}
