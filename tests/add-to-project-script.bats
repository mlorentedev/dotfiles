#!/usr/bin/env bats
# Runs add-to-project.yml's own `run:` script against a gh stub (#3160 items 10, 11).
#
# The script captures the project lookup with $(gh_graphql ...). A retry notice
# printed to stdout landed inside PROJECT_ID, so a transient error that the retry
# recovered still failed the mutation on a malformed id. And `--jq` prints the
# string `null` for a missing project, which passed the non-empty guard. These
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

    # gh stub: the project query answers $STUB_PROJECT, failing first with a
    # transient error when STUB_FAIL_ONCE is set; the mutation logs its args.
    cat > "$FIX/bin/gh" <<'STUB'
#!/usr/bin/env bash
case "$*" in
    *addProjectV2ItemById*)
        printf '%s\n' "$*" >> "$FIX/mutation.log"
        printf '{}' ;;
    *projectV2*)
        if [ -n "${STUB_FAIL_ONCE:-}" ] && [ ! -e "$FIX/failed" ]; then
            touch "$FIX/failed"
            echo "HTTP 502: Bad Gateway" >&2
            exit 1
        fi
        printf '%s\n' "$STUB_PROJECT" ;;
esac
STUB
    printf '#!/bin/sh\nexit 0\n' > "$FIX/bin/sleep"
    chmod +x "$FIX/bin/gh" "$FIX/bin/sleep"
    export FIX PATH="$FIX/bin:$PATH" CONTENT_NODE_ID=I_node GITHUB_EVENT_NAME=issues
}

@test "a transient error the retry recovers still adds the item with a clean project id" {
    STUB_PROJECT=PVT_abc STUB_FAIL_ONCE=1 run bash "$FIX/script.sh"
    [ "$status" -eq 0 ]
    grep -q 'project=PVT_abc ' "$FIX/mutation.log"
    # The notice still reaches the log, outside the captured value.
    [[ "$output" == *"::warning::transient failure (attempt 1/3)"* ]]
}

@test "a project that resolves to null fails without sending the mutation" {
    STUB_PROJECT=null run bash "$FIX/script.sh"
    [ "$status" -eq 1 ]
    [[ "$output" == *"project id resolved empty"* ]]
    [ ! -e "$FIX/mutation.log" ]
}
