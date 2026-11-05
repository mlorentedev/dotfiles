#!/usr/bin/env bats
# Runs bitacora-status.yml's own github-script against a stubbed `github`
# (dotfiles#3160 items 3, 4 and 8, and the consumer review of its rollout). The script is read from the workflow, so the
# tests exercise what ships. setTimeout is stubbed, so a retry costs no time; each
# wait it asked for is recorded.

setup() {
    command -v node >/dev/null || skip "node not installed"
    ROOT="$BATS_TEST_DIRNAME/.."
    FIX="$BATS_TEST_TMPDIR"
    python3 -c '
import sys, yaml
wf = yaml.safe_load(open(sys.argv[1], encoding="utf-8"))
print(wf["jobs"]["in-progress"]["steps"][0]["with"]["script"])' \
        "$ROOT/.github/workflows/bitacora-status.yml" > "$FIX/script.js"

    # SCENARIO (JSON): state = live issue state, or states = one per read of it;
    # values = the item's single-select values, or reads = one list per read;
    # fail = errors thrown by the first calls, in order. A fail entry with
    # graphql=true is shaped as @octokit/graphql shapes it: headers on err.headers,
    # the parsed body on err.response.
    cat > "$FIX/run.cjs" <<'JS'
const fs = require('fs');
const sc = JSON.parse(process.env.SCENARIO);
const calls = [], waits = [], log = [];
const fail = [...(sc.fail ?? [])];
const states = [...(sc.states ?? [])], reads = [...(sc.reads ?? [])];
const maybeFail = () => {
  const f = fail.shift();
  if (!f) return;
  const err = new Error(f.message ?? 'boom');
  const headers = { ...(f.headers ?? {}) };
  if (f.resetIn) headers['x-ratelimit-reset'] = String(Math.floor(Date.now() / 1000) + f.resetIn);
  if (f.graphql) { err.headers = headers; err.response = { data: null, errors: f.errors }; }
  else if (f.status) { err.status = f.status; err.response = { status: f.status, headers }; }
  if (f.errors) err.errors = f.errors;
  throw err;
};
globalThis.setTimeout = (fn, ms) => { waits.push(ms / 1000); fn(); };
const github = {
  rest: { issues: { get: async () => { calls.push('issues.get'); maybeFail(); return { data: { state: states.shift() ?? sc.state ?? 'open' } }; } } },
  graphql: async (q, vars) => {
    const op = q.includes('addProjectV2ItemById') ? 'add' : q.includes('updateProjectV2ItemFieldValue') ? 'update' : 'read';
    calls.push(op === 'update' ? `update:${vars.optionId}` : op);
    maybeFail();
    if (op === 'add') return { addProjectV2ItemById: { item: { id: 'PVTI_1' } } };
    if (op === 'read') return { node: { fieldValues: { nodes: reads.shift() ?? sc.values ?? [], pageInfo: { hasNextPage: !!sc.more } } } };
    return {};
  },
};
const core = { info: (m) => log.push(m), warning: (m) => log.push(`warning: ${m}`) };
const context = { repo: { owner: 'o', repo: 'r' }, payload: { issue: { number: 7, node_id: 'I_7' } } };
const body = fs.readFileSync(process.argv[2], 'utf8');
const AsyncFunction = Object.getPrototypeOf(async () => {}).constructor;
new AsyncFunction('github', 'context', 'core', body)(github, context, core)
  .then(() => 0, (e) => { log.push(`error: ${e.message}`); return 1; })
  .then((rc) => { console.log(JSON.stringify({ calls, waits, log })); process.exit(rc); });
JS
}

go() { SCENARIO="$1" run node "$FIX/run.cjs" "$FIX/script.js"; }

STATUS_FIELD='{"id":"PVTSSF_lAHOAM7xJs4BZ6GYzhU1dtI"}'
PRIORITY_FIELD='{"id":"PVTSSF_priority"}'

@test "an issue closed while the run was queued is left off the board" {
    go '{"state":"closed"}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"calls":["issues.get"]'* ]] || false
}

@test "a Backlog issue moves to In Progress" {
    go "{\"values\":[{\"optionId\":\"f65b1174\",\"name\":\"Backlog\",\"field\":$STATUS_FIELD}]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *'"update:6c133cc8"'* ]] || false
}

@test "a parked issue is left as is, whatever the field and option are named" {
    go "{\"values\":[{\"optionId\":\"7705a647\",\"name\":\"Bloqueado\",\"field\":$STATUS_FIELD}]}"
    [ "$status" -eq 0 ]
    [[ "$output" != *'update:'* ]] || false
    [[ "$output" == *'is \"Bloqueado\"; left as is'* ]] || false
}

@test "another field's option is not read as the status" {
    go "{\"values\":[{\"optionId\":\"7705a647\",\"name\":\"P1\",\"field\":$PRIORITY_FIELD}]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *'"update:6c133cc8"'* ]] || false
}

@test "a transient failure is retried and the move still lands" {
    go "{\"fail\":[null,{\"status\":502}],\"values\":[]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *'"update:6c133cc8"'* ]] || false
    [[ "$output" == *'"waits":[5]'* ]] || false
}

@test "a secondary rate limit waits for retry-after, and at least a minute" {
    go '{"fail":[{"status":403,"message":"You have exceeded a secondary rate limit","headers":{"retry-after":"90"}},{"status":429}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"waits":[90,60]'* ]] || false
}

@test "a GraphQL RATE_LIMITED error is retried, though it arrives with HTTP 200" {
    go '{"fail":[null,{"errors":[{"type":"RATE_LIMITED"}]}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"waits":[60]'* ]] || false
}

@test "a missing permission fails at once, with no retry" {
    go '{"fail":[{"status":403,"message":"Resource not accessible by personal access token"}]}'
    [ "$status" -eq 1 ]
    [[ "$output" == *'"calls":["issues.get"]'* ]] || false
    [[ "$output" == *'"waits":[]'* ]] || false
}

@test "a GraphQL error that is not a rate limit fails at once" {
    go '{"fail":[null,{"errors":[{"type":"NOT_FOUND"}]}]}'
    [ "$status" -eq 1 ]
    [[ "$output" == *'"waits":[]'* ]] || false
}

@test "a primary rate limit waits until its reset" {
    go '{"fail":[{"status":403,"message":"API rate limit exceeded","headers":{"x-ratelimit-remaining":"0"},"resetIn":120}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" =~ \"waits\":\[1(19|20|21)\] ]] || false
}

@test "a rate limit that resets after the job's budget fails at once" {
    go '{"fail":[{"status":403,"message":"API rate limit exceeded","headers":{"x-ratelimit-remaining":"0"},"resetIn":3600}]}'
    [ "$status" -eq 1 ]
    [[ "$output" == *'"waits":[]'* ]] || false
}

@test "a Status beyond the values read fails rather than reading as unset" {
    go "{\"more\":true,\"values\":[{\"optionId\":\"7705a647\",\"name\":\"P1\",\"field\":$PRIORITY_FIELD}]}"
    [ "$status" -eq 1 ]
    [[ "$output" != *'update:'* ]] || false
    [[ "$output" == *'Status not in the first 100 field values'* ]] || false
    # The script's own error is not an API failure: no retry.
    [[ "$output" == *'"waits":[]'* ]] || false
}

BACKLOG="{\"optionId\":\"f65b1174\",\"name\":\"Backlog\",\"field\":$STATUS_FIELD}"
BLOCKED="{\"optionId\":\"7705a647\",\"name\":\"Blocked\",\"field\":$STATUS_FIELD}"

@test "a secondary limit waits its retry-after, though the primary reset is far off" {
    go '{"fail":[{"status":403,"message":"You have exceeded a secondary rate limit","headers":{"retry-after":"60","x-ratelimit-remaining":"4000"},"resetIn":3000}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"waits":[60]'* ]] || false
    [[ "$output" == *'"update:6c133cc8"'* ]] || false
}

@test "a GraphQL rate limit reads the reset from the error's own headers" {
    go '{"fail":[null,{"graphql":true,"errors":[{"type":"RATE_LIMITED"}],"headers":{"x-ratelimit-remaining":"0"},"resetIn":120}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" =~ \"waits\":\[1(19|20|21)\] ]] || false
}

@test "a GraphQL rate limit that resets after the job's budget fails at once" {
    go '{"fail":[null,{"graphql":true,"errors":[{"type":"RATE_LIMITED"}],"headers":{"x-ratelimit-remaining":"0"},"resetIn":3600}]}'
    [ "$status" -eq 1 ]
    [[ "$output" == *'"waits":[]'* ]] || false
}

@test "a GraphQL error typed RATE_LIMIT is a rate limit too" {
    go '{"fail":[null,{"errors":[{"type":"RATE_LIMIT"}]}],"values":[]}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"waits":[60]'* ]] || false
}

@test "an error from the script itself is not retried" {
    # A TypeError from an unexpected response shape has no status.
    go '{"fail":[null,{"message":"Cannot read properties of undefined"}]}'
    [ "$status" -eq 1 ]
    [[ "$output" == *'"waits":[]'* ]] || false
}

@test "a Status parked during a retry wait is not overwritten" {
    go "{\"fail\":[null,null,null,{\"status\":502}],\"reads\":[[$BACKLOG],[$BLOCKED]]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *'"calls":["issues.get","add","read","update:6c133cc8","issues.get","add","read"]'* ]] || false
    [[ "$output" == *'is \"Blocked\"; left as is'* ]] || false
}

@test "an issue closed during a retry wait is left off the board" {
    go '{"fail":[null,{"status":502}],"states":["open","closed"]}'
    [ "$status" -eq 0 ]
    [[ "$output" == *'"calls":["issues.get","add","issues.get"]'* ]] || false
    [[ "$output" == *'left off the board'* ]] || false
}
