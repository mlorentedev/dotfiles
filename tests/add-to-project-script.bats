#!/usr/bin/env bats
# Runs add-to-project.yml's own `run:` script against a gh stub (#3160 items 1, 2, 10, 11, 12).
#
# The script captures the project lookup with $(gh_graphql ...). A retry notice
# printed to stdout landed inside PROJECT_ID, so a transient error that the retry
# recovered still failed the mutation on a malformed id. And `--jq` prints the
# string `null` for a missing project, which passed the non-empty guard. A refused
# connection went red with no retry, and a secondary rate limit got 5 s. These
# tests read the script from the workflow, so they exercise what ships.

setup() {
    ROOT="$BATS_TEST_DIRNAME/.."
    FIX="$BATS_TEST_TMPDIR"
    mkdir -p "$FIX/bin"
    python3 -c '
import sys, yaml
wf = yaml.safe_load(open(sys.argv[1], encoding="utf-8"))
step = next(s for s in wf["jobs"]["add-to-project"]["steps"] if s.get("name") == "Add to bitácora board")
print(step["run"])' "$ROOT/.github/workflows/add-to-project.yml" > "$FIX/script.sh"

    # gh stub: the project query answers $STUB_PROJECT. STUB_FAIL_ONCE makes its
    # first call fail with STUB_FAIL_MSG (default a 502); the mutation logs its
    # args. sleep logs how long it was asked to wait.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
case "$*" in
    *addProjectV2ItemById*)
        printf '%s\n' "$*" >> "$FIX/mutation.log"
        printf '{}' ;;
    *projectV2*)
        if [ -n "${STUB_FAIL_ONCE:-}" ] && [ ! -e "$FIX/failed" ]; then
            touch "$FIX/failed"
            echo "${STUB_FAIL_MSG:-HTTP 502: Bad Gateway}" >&2
            exit 1
        fi
        printf '%s\n' "$STUB_PROJECT" ;;
esac
STUB
    printf '#!/bin/sh\necho "$1" >> "$FIX/sleep.log"\n' > "$FIX/bin/sleep"
    chmod +x "$FIX/bin/gh" "$FIX/bin/sleep"
    export FIX PATH="$FIX/bin:$PATH" CONTENT_NODE_ID=I_node GITHUB_EVENT_NAME=issues
}

@test "a transient error the retry recovers still adds the item with a clean project id" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    grep -q 'project=PVT_abc ' "$FIX/mutation.log"
    # The notice still reaches the log, outside the captured value.
    [[ "$output" == *"::warning::transient failure (attempt 1/3)"* ]] || false
}

@test "a project that resolves to null fails without sending the mutation" {
    # Also the fallback for a gh that exited 0 on a body carrying `errors`:
    # `--jq .data.user.projectV2.id` prints `null` for it, and this guard rejects it.
    STUB_PROJECT=null run bash "$FIX/script.sh"
    [ "$status" -eq 1 ]
    [[ "$output" == *"project id resolved empty"* ]] || false
    [ ! -e "$FIX/mutation.log" ]
}

@test "a refused connection is retried as transient (item 1)" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 \
        STUB_FAIL_MSG='Post "https://api.github.com/graphql": dial tcp 140.82.112.6:443: connect: connection refused' \
        run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    grep -q 'project=PVT_abc ' "$FIX/mutation.log"
}

@test "a secondary rate limit waits a minute before the retry (item 2)" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 \
        STUB_FAIL_MSG='HTTP 403: You have exceeded a secondary rate limit.' run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    [ "$(cat "$FIX/sleep.log")" = 60 ]
}

@test "a 5xx backs off 5 seconds, not a minute" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    [ "$(cat "$FIX/sleep.log")" = 5 ]
}

@test "a GraphQL RATE_LIMITED error, which gh exits 1 on, takes the rate-limit path (item 12)" {
    # GitHub sends RATE_LIMITED with HTTP 200. gh api graphql still exits 1 on a
    # body carrying `errors` (checked live on 2026-11-05) and prints this line,
    # so the stub exits 1 with it. A gh that exited 0 instead is test 2's case.
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 \
        STUB_FAIL_MSG='GraphQL: API rate limit already exceeded for user ID 13562150.' run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    [[ "$output" == *"primary GraphQL pool exhausted"* ]] || false
    # One annotation line: a split message would leave its tail outside the warning.
    [[ "$output" == *"::warning::primary GraphQL pool exhausted"*"if it is still open, mlorentedev/dotfiles bitacora-reconcile.yml adds it within a day"* ]] || false
    [ "$(printf '%s\n' "$output" | grep -c 'bitacora-reconcile')" -eq 1 ]
    printf '%s\n' "$output" | grep -q '^::warning::.*bitacora-reconcile.yml adds it within a day$'
    [ ! -e "$FIX/mutation.log" ]
}

@test "a GraphQL permission error, which gh exits 1 on, fails the job (item 12)" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 \
        STUB_FAIL_MSG='GraphQL: Resource not accessible by personal access token (user.projectV2)' run bash "$FIX/script.sh"
    [ "$status" -eq 1 ]
    [[ "$output" == *"(not a rate limit)"* ]] || false
    [ ! -e "$FIX/sleep.log" ]
}
