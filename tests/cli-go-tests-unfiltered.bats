#!/usr/bin/env bats
# cli.yml runs the Go tests on every diff, never behind a path filter (#1888).
#
# Go tests guard files outside cli/: the golden corpora under tests/golden/,
# harness/*.json, the shell rc files. A filter on cli/** skipped each of those
# guards on the very diff that changed its file, on the PR and on main, while
# cli-gate reported green (lesson 322: an allow-list path filter treats every
# unlisted path as safe to skip). The workflow is parsed, not grepped, for the
# reason tests/ci-path-filtering.bats gives.

setup() {
    CLI_YML="$BATS_TEST_DIRNAME/../.github/workflows/cli.yml"
}

_yaml() { # python body reading the parsed workflow as `wf`; exit status is the verdict
    python3 - "$CLI_YML" "$1" <<'EOF'
import sys, yaml
wf = yaml.safe_load(open(sys.argv[1]))
# PyYAML reads the bare key `on` as the boolean True.
wf["on"] = wf.pop(True, wf.get("on"))
exec(sys.argv[2])
EOF
}

@test "cli.yml: the Go test job has no condition, so no filter can skip it" {
    run _yaml '
job = wf["jobs"]["test"]
print("if:", job.get("if"), "needs:", job.get("needs"))
sys.exit(1 if "if" in job or "needs" in job else 0)'
    [ "$status" -eq 0 ] || { echo "$output"; false; }
}

@test "cli.yml: neither trigger is path-filtered" {
    run _yaml '
bad = [ev for ev, cfg in wf["on"].items()
       if isinstance(cfg, dict) and ({"paths", "paths-ignore"} & set(cfg))]
print(bad)
sys.exit(1 if bad else 0)'
    [ "$status" -eq 0 ] || { echo "path-filtered triggers: $output"; false; }
}

@test "cli.yml: cli-gate still waits on the test job" {
    run _yaml '
needs = wf["jobs"]["cli-gate"]["needs"]
sys.exit(0 if "test" in needs else 1)'
    [ "$status" -eq 0 ]
}
