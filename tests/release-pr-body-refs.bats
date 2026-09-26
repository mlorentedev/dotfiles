#!/usr/bin/env bats
# CI-005 (#1668): the release PR body must close nothing.
#
# release-please builds the release PR body from conventional-changelog notes,
# whose template renders EVERY reference a commit carries as ", closes [#N]",
# whatever keyword the commit used. `Refs #N` included. When the release PR
# merges into main, GitHub closes each issue its body says it "closes". Measured
# on 0.57.0 (#1613): #1451, #1626 and #1596 closed as completed two seconds
# after the merge, each referenced only with `Refs`.
#
# An issue that should close was already closed by its own PR's merge, so the
# release PR body has no legitimate closing keyword to keep. The workflow step
# rewrites every one of them to `refs`.
#
# These tests execute the step's OWN `run:` script, extracted from the workflow,
# against a stub `gh` that serves fixtures through the real jq. Asserting on the
# workflow text alone would pass for a substitution that matches nothing.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    WF="$REPO/.github/workflows/release-please.yml"
    STEP_NAME="Neutralise closing keywords in the release PR body"

    WORK="$BATS_TEST_TMPDIR"
    mkdir -p "$WORK/bin" "$WORK/fixtures"
    export GH_LOG="$WORK/gh.log" FIXTURES="$WORK/fixtures" PATCHED="$WORK/patched"
    : >"$GH_LOG"

    # A stub that answers the two reads the step makes, and records the write.
    # --jq is applied with the real jq, as gh would apply it.
    cat >"$WORK/bin/gh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$GH_LOG"
jq_expr="" method=GET endpoint="" body_file=""
while [ $# -gt 0 ]; do
    case "$1" in
        api) ;;
        --jq) jq_expr="$2"; shift ;;
        -X) method="$2"; shift ;;
        -F|-f) case "$2" in body=@*) body_file="${2#body=@}" ;; esac; shift ;;
        *) endpoint="$1" ;;
    esac
    shift
done
if [ "$method" = PATCH ]; then
    pr="${endpoint##*/}"
    cp "$body_file" "$PATCHED.$pr"
    exit 0
fi
case "$endpoint" in
    */pulls\?*) fixture="$FIXTURES/pulls.json" ;;
    */pulls/*) fixture="$FIXTURES/pull-${endpoint##*/}.json" ;;
    *) echo "stub gh: unexpected endpoint $endpoint" >&2; exit 2 ;;
esac
jq -r "${jq_expr:-.}" "$fixture"
STUB
    chmod +x "$WORK/bin/gh"

    # Two open PRs: the release PR (#10) and an ordinary one (#11), which the
    # step must never touch whatever its body says.
    cat >"$FIXTURES/pulls.json" <<'JSON'
[
  {"number": 10, "labels": [{"name": "autorelease: pending"}]},
  {"number": 11, "labels": [{"name": "bug"}]}
]
JSON
    set_body 11 'Closes #99'
}

# Writes the body of pull request $1 as the fixture the stub serves.
set_body() {
    jq -n --arg b "$2" '{body: $b}' >"$FIXTURES/pull-$1.json"
}

# Prints the step's run: script. Empty when the step is absent.
step_script() {
    python3 -c "
import sys, yaml
d = yaml.safe_load(open('$WF'))
for step in d['jobs']['release-please']['steps']:
    if step.get('name') == '$STEP_NAME':
        print(step['run'])
"
}

run_step() {
    local script
    script="$(step_script)"
    [ -n "$script" ] || { echo "step '$STEP_NAME' not found in $WF" >&2; return 1; }
    PATH="$WORK/bin:$PATH" GITHUB_REPOSITORY=owner/repo RUNNER_TEMP="$WORK" \
        bash -eo pipefail -c "$script"
}

@test "release-please: the neutralising step runs after release-please-action" {
    run python3 -c "
import sys, yaml
d = yaml.safe_load(open('$WF'))
steps = d['jobs']['release-please']['steps']
names = [s.get('name') or s.get('uses', '') for s in steps]
rp = [i for i, n in enumerate(names) if n.startswith('googleapis/release-please-action')]
ours = [i for i, n in enumerate(names) if n == '$STEP_NAME']
if not rp or not ours:
    print(f'steps: {names}'); sys.exit(1)
if ours[0] < rp[0]:
    print('the step runs before release-please writes the body'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "release-please: the template's ', closes [#N](url)' becomes ', refs [#N](url)'" {
    set_body 10 '* **forge:** declare branch protection ([da16c65](https://github.com/o/r/commit/da16c65)), closes [#1451](https://github.com/o/r/issues/1451)'
    run run_step
    [ "$status" -eq 0 ]
    grep -qF '), refs [#1451](https://github.com/o/r/issues/1451)' "$PATCHED.10"
    refute_grep_fixed 'closes' "$PATCHED.10"
}

@test "release-please: several references after one keyword, and a cross-repo one" {
    set_body 10 '* a, closes [#1](u) [#2](u)
* b, closes [owner/other#3](u)'
    run run_step
    [ "$status" -eq 0 ]
    grep -qF '* a, refs [#1](u) [#2](u)' "$PATCHED.10"
    grep -qF '* b, refs [owner/other#3](u)' "$PATCHED.10"
}

# A commit subject is rendered verbatim, so its own prose can carry a keyword
# GitHub honours. Every closing keyword counts, in any case, with or without a
# colon, bare or as a URL.
@test "release-please: every GitHub closing keyword is neutralised, not only the template's" {
    set_body 10 '* fix: Fixes #4
* fix: resolved: #5
* fix: the release closed #6
* fix: FIX https://github.com/o/r/issues/7'
    run run_step
    [ "$status" -eq 0 ]
    tr '[:upper:]' '[:lower:]' <"$PATCHED.10" >"$WORK/lower"
    refute_grep '(close[sd]?|fix(e[sd])?|resolve[sd]?):? +(#|https)' "$WORK/lower"
    grep -qF 'refs #4' "$PATCHED.10"
    grep -qF 'refs #5' "$PATCHED.10"
    grep -qF 'refs #6' "$PATCHED.10"
    grep -qF 'refs https://github.com/o/r/issues/7' "$PATCHED.10"
}

@test "release-please: a word that merely contains a keyword is left alone" {
    set_body 10 '* feat: prefixes #8 and unresolved #9, closes [#10](u)'
    run run_step
    [ "$status" -eq 0 ]
    grep -qF 'prefixes #8 and unresolved #9, refs [#10](u)' "$PATCHED.10"
}

@test "release-please: a body with nothing to rewrite is not written back" {
    set_body 10 '* feat: something ([abc](u))'
    run run_step
    [ "$status" -eq 0 ]
    [ ! -e "$PATCHED.10" ]
    refute_grep_fixed PATCH "$GH_LOG"
}

@test "release-please: an already neutralised body is not written again" {
    set_body 10 '* a, refs [#1](u)'
    run run_step
    [ "$status" -eq 0 ]
    refute_grep_fixed PATCH "$GH_LOG"
}

@test "release-please: a pull request without the release label is never touched" {
    set_body 10 '* a, closes [#1](u)'
    run run_step
    [ "$status" -eq 0 ]
    [ ! -e "$PATCHED.11" ]
    refute_grep_fixed 'pulls/11' "$GH_LOG"
}
