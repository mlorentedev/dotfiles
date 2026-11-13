#!/usr/bin/env bats
# CI-013 (#3149): the Windows and macOS jobs do not start on a documentation-only
# pull request, and every push runs them.
#
# A job skipped by its job-level `if` reports success, so a required check
# stays green; a matrix job loses only the legs its OS list leaves out. The
# workflows are parsed, not grepped, for the reason tests/ci-path-filtering.bats
# gives.

setup() {
    export WF="$BATS_TEST_DIRNAME/../.github/workflows"
    export FILTER="$BATS_TEST_DIRNAME/../.github/filters/code.yml"
    python3 -c "import yaml" 2>/dev/null || skip "PyYAML required to parse workflow YAML"
}

_py() { # python body with `load(name)` returning a parsed workflow; exit status is the verdict
    python3 - "$WF" "$FILTER" "$BATS_TEST_DIRNAME/../forge/branch-protection.json" "$1" <<'EOF'
import glob, json, os, re, sys, yaml
def load(name):
    wf = yaml.safe_load(open(os.path.join(sys.argv[1], name)))
    wf["on"] = wf.pop(True, wf.get("on"))  # PyYAML reads the bare key `on` as True
    return wf
FILTER = sys.argv[2]
# This repository's required checks, from the protection branch protection applies.
_prot = json.load(open(sys.argv[3]))["repos"]["mlorentedev/dotfiles"]["protection"]
REQUIRED = {c["context"] for c in _prot["required_status_checks"]["checks"]}
exec(sys.argv[4])
EOF
}

# The structural rule, so a Windows or macOS job added later is held to it too:
# on a pull request it is gated at job level, or its OS list is a `changes`
# output whose expression keeps Linux alone on a pull request. A required check is exempt: a job-level `if` would let it
# pass without running (CI-004, tests/lib/check-workflow-contexts.py). Since
# CI-017 no Windows or macOS job is required: ci-gate aggregates them, so
# test-windows is gated like the rest (HARNESS-041 took the step-level route).
@test "CI-013: every Windows or macOS job on a pull request is gated before a runner starts" {
    run _py '
bad = []
for wf_file in sorted(glob.glob(os.path.join(sys.argv[1], "*.yml"))):
    name = os.path.basename(wf_file)
    wf = load(name)
    on = wf["on"]
    events = [on] if isinstance(on, str) else list(on or [])
    if "pull_request" not in events:
        continue
    jobs = wf.get("jobs") or {}
    for job_id, job in jobs.items():
        os_list = str((job.get("strategy") or {}).get("matrix", {}).get("os", ""))
        # A dynamic OS list counts as gated only when it is a `changes` output
        # whose expression drops to Linux alone on a pull request; anything else
        # computed is a source this rule cannot read, so it is flagged.
        m = re.fullmatch(r"\$\{\{\s*fromJSON\(needs\.changes\.outputs\.(\w+)\)\s*\}\}", os_list)
        if m:
            expr = str(((jobs.get("changes") or {}).get("outputs") or {}).get(m.group(1), ""))
            q = chr(39)  # this body sits in a single-quoted shell word
            if not (f"github.event_name != {q}pull_request{q}" in expr
                    and q + "[\"ubuntu-latest\"]" + q in expr):
                bad.append(f"{name}:{job_id} (OS output {m.group(1)} has no Linux-only branch)")
            continue
        if "${{" in os_list:
            bad.append(f"{name}:{job_id} (OS list from an expression this rule cannot read)")
            continue
        literal = str(job.get("runs-on", "")) + os_list
        if not any(k in literal for k in ("windows", "macos")):
            continue
        if job.get("name", job_id) in REQUIRED:
            continue
        gated = "needs.changes.outputs." in str(job.get("if", ""))
        if not gated:
            bad.append(f"{name}:{job_id}")
print("ungated Windows/macOS jobs:", bad)
sys.exit(1 if bad else 0)'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# cli.yml and cli-smoke.yml: the OS list is a `changes` output. Linux is in it
# on every branch of the expression (#1888: the Go tests run on every diff), and
# a push takes both OSes without consulting the filter.
_os_list_case() { # $1 workflow, $2 job, $3 output
    _py "
wf = load('$1')
job = wf['jobs']['$2']
want = 'fromJSON(needs.changes.outputs.$3)'
matrix_os = str(job['strategy']['matrix']['os'])
expr = str(wf['jobs']['changes']['outputs']['$3'])
print('matrix.os:', matrix_os); print('output:', expr)
both = '[\"ubuntu-latest\",\"windows-latest\"]'
checks = [
    job.get('needs') in ('changes', ['changes']),
    want in matrix_os,
    \"github.event_name != 'pull_request'\" in expr,
    both in expr,
    \"'[\\\"ubuntu-latest\\\"]'\" in expr,
]
print(checks)
sys.exit(0 if all(checks) else 1)"
}

@test "CI-013: cli test takes its OS list from cli-changes, both OSes on a push" {
    run _os_list_case cli.yml test test_os
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "CI-013: cli-smoke takes its OS list from its changes job, both OSes on a push" {
    run _os_list_case cli-smoke.yml smoke smoke_os
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# The docs-only decision is read from one file, with the quantifier that makes
# `**` plus exclusions mean "a file outside the docs-only set".
@test "CI-013: the OS-list filters read the shared file with some-with-excludes" {
    run _py '
bad = []
for name in ("cli.yml", "cli-smoke.yml"):
    steps = load(name)["jobs"]["changes"]["steps"]
    hits = [s for s in steps if (s.get("with") or {}).get("filters") == ".github/filters/code.yml"]
    if len(hits) != 1 or hits[0]["with"].get("predicate-quantifier") != "some-with-excludes":
        bad.append(name)
print("workflows without the shared filter step:", bad)
sys.exit(1 if bad else 0)'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

# An exclusion that ci.yml calls code would skip the OS replicas on a diff the
# rest of CI tests: the two lists must not overlap.
@test "CI-013: no docs-only exclusion is in ci.yml's code filter" {
    run _py '
steps = load("ci.yml")["jobs"]["changes"]["steps"]
code = set(yaml.safe_load(next(s["with"]["filters"] for s in steps if s.get("id") == "filter"))["code"])
shared = yaml.safe_load(open(FILTER))["code"]
excl = [p[1:] for p in shared if p.startswith("!")]
overlap = [p for p in excl if p in code or p.removesuffix("/**") + "/**" in code]
print("first pattern:", shared[0], "overlap:", overlap)
sys.exit(0 if shared[0] == "**" and excl and not overlap else 1)'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "CI-013: the integration container builds with a gha layer cache and the pinned build args" {
    run _py '
steps = load("ci.yml")["jobs"]["integration"]["steps"]
build = [s for s in steps if str(s.get("uses", "")).startswith("docker/build-push-action@")]
print([s.get("uses") or s.get("name") for s in steps])
if len(build) != 1:
    sys.exit(1)
w = build[0]["with"]
args = str(w.get("build-args", ""))
checks = [
    any(str(s.get("uses", "")).startswith("docker/setup-buildx-action@") for s in steps),
    "type=gha" in str(w.get("cache-from", "")),
    "type=gha" in str(w.get("cache-to", "")) and "github.event_name == " in str(w.get("cache-to", "")),
    str(w.get("load")).lower() == "true",
    w.get("tags") == "dotfiles-integration-test",
    w.get("file") == "tests/Dockerfile.integration",
    all(f"{v}=${{{{ steps.versions.outputs.{v} }}}}" in args
        for v in ("BATS_VERSION", "AGE_VERSION", "GO_VERSION")),
]
print(checks)
sys.exit(0 if all(checks) else 1)'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}
