#!/usr/bin/env bats
# GUARD-003 (#1052): the gate's WORKFLOW shape, as distinct from its classifier.
#
# Separate from check-review-attestation.bats on purpose. That file asserts what
# the script decides; this one asserts that the workflow ever asks. The two fail
# for different reasons and a change to one rarely touches the other.
#
# Why this file exists at all: GitHub suppresses workflow events for comments
# created with GITHUB_TOKEN, to prevent recursion. PR-Agent publishes with
# GITHUB_TOKEN. Measured 2026-08-18 — two PR-Agent comments on #1047 triggered
# ZERO gate runs, while two human comments elsewhere each triggered one within
# three seconds. So `issue_comment` cannot be the trigger that notices our own
# reviewer, and on a PR where PR-Agent is the only reviewer (CodeRabbit's quota
# spent, this repo's normal state) the verdict freezes seconds after the PR
# opens and nothing re-asks.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    WF="$REPO/.github/workflows/review-attestation.yml"
    PRA="$REPO/.github/workflows/pr-agent.yml"
}

# Parses the workflow and answers one question, printing nothing on success.
#
# `on:` is read as the BOOLEAN True, not the string "on" — YAML 1.1 spells true
# as y|yes|on|true and PyYAML still honours it. A test that looked up d["on"]
# would raise KeyError and could be "fixed" by deleting the assertion.
wf_query() {
    python3 -c "
import sys, yaml
d = yaml.safe_load(open('$WF'))
triggers = d.get(True, d.get('on', {}))
$1
" 
}

@test "review-attestation: the workflow exists" {
    [ -f "$WF" ]
}

@test "review-attestation: re-evaluates when the pr-agent reviewer finishes [AC1]" {
    run wf_query "
wr = triggers.get('workflow_run')
if wr is None:
    print('no workflow_run trigger: our own reviewer finishing wakes nothing'); sys.exit(1)
names = wr.get('workflows') or []
if 'pr-agent' not in names:
    print(f'workflow_run does not name pr-agent (names={names})'); sys.exit(1)
if 'completed' not in (wr.get('types') or []):
    print('workflow_run does not listen for completed'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "review-attestation: the job admits workflow_run events [AC1]" {
    run wf_query "
cond = str(d['jobs']['attestation'].get('if', ''))
if 'workflow_run' not in cond:
    print('job if: does not admit workflow_run — the trigger would fire and skip'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# A workflow_run payload carries neither pull_request.number nor issue.number,
# so the expression the other two triggers rely on evaluates to empty. Resolving
# the PR from the run's head SHA is the substance of the trigger, not a detail:
# a gate that re-evaluates the wrong pull request is worse than one that does
# not re-evaluate at all.
@test "review-attestation: the workflow_run path resolves its pull request [AC4]" {
    grep -q 'workflow_run.head_sha' "$WF"
    grep -qE 'commits/.*\/pulls' "$WF"
}

# A workflow_run payload carries no PR number, so a group key built only from
# `pull_request.number || issue.number` collapses EVERY workflow_run into one
# group keyed on the empty string. `completed` is not in the cancel-exempt list,
# so a reviewer finishing on one PR would cancel the re-evaluation of another —
# the #1040 defect, one workflow over, arrived by a different route.
@test "review-attestation: the concurrency key cannot conflate two pull requests [AC4]" {
    run wf_query "
group = str(d['concurrency']['group'])
if 'workflow_run.head_sha' not in group:
    print('group key has no workflow_run discriminator: all such runs share one group'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The verdict must reach the PR head as a commit status on every path. A
# workflow_run run is associated with the default branch, exactly like an
# issue_comment one, so without this it would compute a verdict nothing displays.
@test "review-attestation: every path still publishes the verdict as a commit status [AC2]" {
    grep -q 'statuses: write' "$WF"
    grep -qE 'statuses/\$\{SHA\}' "$WF"
}

# #1041: a check-run belongs to the run that created it and can never be updated.
# The pull_request run fires seconds after the PR opens, when no reviewer has
# posted and the only possible verdict is `pending` — so it froze red on every
# correct PR, permanently, while the commit status beside it went green. A gate
# whose normal state is red teaches its readers to scroll past it.
@test "review-attestation: the pull_request run never freezes the check-run red [#1041]" {
    # WIDENED from `pending` to every kind. A check-run cannot be revised, and on
    # the pull_request event NO verdict is final -- the reviewers have not run
    # yet, by construction. Restricting the exemption to `pending` narrowed the
    # window instead of closing it: measured 2026-08-22, `declined` was the first
    # verdict on every PR opened in a full session, and each froze red for good.
    grep -q '\[ "$EVENT" = "pull_request" \]; then' "$WF"
}

# ...and the exemption is scoped to the ONE event that cannot mean anything yet.
# Every other path still fails: by then a reviewer has had its turn.
@test "review-attestation: every non-pull_request path still fails [#1041]" {
    run bash -c '
      body=$(sed -n "/Fail the run when the PR is not attested/,\$p" "'"$WF"'")
      # the exemption must be guarded on the EVENT; losing that guard would make
      # the gate never fail anywhere, which is widening it into silence
      echo "$body" | grep -q "EVENT\" = \"pull_request\"" || exit 1
      # and the unconditional refusal must survive after it
      echo "$body" | grep -q "exit \"\$CODE\"" || exit 1
      exit 0
    '
    [ "$status" -eq 0 ]
}

@test "review-attestation: the verdict still reaches a REVISABLE signal [#1041]" {
    # The exemption is only defensible because the commit status carries the
    # verdict and every later run can revise it. If the publish step ever went
    # away, softening the check-run would become plain silence.
    grep -q 'statuses' "$WF"
    grep -q 'Publish the verdict onto the PR head' "$WF"
}

# The workflow extracts the verdict KIND from the classifier's own structured
# prefix. That is a coupling between two files, so it is tested against the real
# strings rather than assumed: a sed that silently matches nothing would make
# every verdict read as an empty kind, and the exemption above would stop firing.
@test "review-attestation: the kind extractor matches what the classifier emits [#1041]" {
    local script="$REPO/scripts/check-review-attestation.sh"
    for kind in pending declined; do
        grep -q "\[FAIL\] $kind" "$script" || {
            printf 'classifier no longer emits "[FAIL] %s"\n' "$kind" >&2; return 1; }
    done
    # the extractor itself, run over the real emitted shapes
    got=$(printf '[FAIL] pending — no reviewer output on this PR yet\n' \
          | sed -n 's/^\[[A-Z]*\] \([a-z]*\).*/\1/p' | head -1)
    [ "$got" = "pending" ] || { printf 'extractor produced "%s"\n' "$got" >&2; return 1; }
    got=$(printf '[FAIL] declined — coderabbitai posted a notice\n' \
          | sed -n 's/^\[[A-Z]*\] \([a-z]*\).*/\1/p' | head -1)
    [ "$got" = "declined" ] || { printf 'extractor produced "%s"\n' "$got" >&2; return 1; }
}

# AC6: promoting this gate to a required check has an order, and getting it
# wrong blocks the repository. The order lives beside the thing it constrains
# rather than in a ticket nobody reads at the moment of enabling it — and it
# must name the STATUS, because requiring the check-run pins every PR to the
# verdict computed seconds after it opened.
@test "review-attestation: the required-check adoption order is documented here [AC6]" {
    grep -q 'Require the commit status `review-attestation`, never the `attestation`' "$WF"
    for step in 1047 1041 1052; do
        grep -q "#$step" "$WF" || { printf 'adoption order does not name #%s\n' "$step" >&2; return 1; }
    done
}

# --- #1128: a review requested with /review --------------------------------
#
# A pr-agent run triggered by `issue_comment` is associated with the DEFAULT
# BRANCH, so the workflow_run it completes carries main's SHA and an empty
# pull_requests[]. Resolving by SHA finds no open PR and skips: correct, since a
# run with nothing to judge must not publish, and useless, since the PR identity
# was lost at the trigger. Measured live 2026-10-20: slash run 39365425846 on an
# open PR, attestation 39365734960 "no open pull request for fd0be466".
#
# The fix carries the number from the run being judged, never infers it: the
# issue_comment path of pr-agent uploads it as an artifact, and the gate reads
# that artifact only for runs whose event is issue_comment. Those always execute
# the default branch's pr-agent.yml, so the number comes from reviewed code.

# Prints the `run:` body of the step with the given id in the given job.
step_run() {
    python3 -c "
import sys, yaml
d = yaml.safe_load(open(sys.argv[1]))
for s in d['jobs'][sys.argv[2]]['steps']:
    if s.get('id') == sys.argv[3]:
        print(s['run']); break
else:
    sys.exit('no step with id ' + sys.argv[3])
" "$1" "$2" "$3"
}

# Runs the gate's "Resolve the pull request" step under a stub gh. The stub
# answers each endpoint from a STUB_* variable with the value the step's --jq
# would have produced, so these cases test the step's control flow, not jq.
resolve() {
    local dir="$BATS_TEST_TMPDIR" script
    mkdir -p "$dir/bin"
    cat > "$dir/bin/gh" <<'STUB'
#!/usr/bin/env bash
case "$2" in
  */commits/*/pulls)          printf '%s\n' "${STUB_SHA_PR:-}" ;;
  */actions/runs/*/artifacts) printf '%s\n' "${STUB_ARTIFACT_ID:-}" ;;
  */actions/artifacts/*/zip)  cat "$STUB_ZIP" ;;
  */pulls/*)                  printf '%s\n' "${STUB_PR_STATE:-open}" ;;
  *) echo "stub gh: unexpected call: $*" >&2; exit 64 ;;
esac
STUB
    chmod +x "$dir/bin/gh"
    script=$(step_run "$WF" attestation pr) || return 1
    : > "$dir/out"
    (cd "$dir" && PATH="$dir/bin:$PATH" GITHUB_OUTPUT="$dir/out" \
        GITHUB_REPOSITORY=o/r bash -eo pipefail -c "$script")
}

# Builds the artifact's zip with the given pr-number content.
make_zip() {
    STUB_ZIP="$BATS_TEST_TMPDIR/fixture-artifact.zip"
    python3 -c "
import sys, zipfile
with zipfile.ZipFile(sys.argv[1], 'w') as z:
    z.writestr('pr-number', sys.argv[2])
" "$STUB_ZIP" "$1"
    export STUB_ZIP
}

@test "review-attestation: a /review run re-evaluates the PR it was asked on [#1128]" {
    make_zip $'2828\n'
    DIRECT='' RUN_EVENT=issue_comment RUN_ID=1 RUN_SHA=mainsha STUB_ARTIFACT_ID=7 \
        run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'number=2828' "$BATS_TEST_TMPDIR/out" \
        || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
}

@test "review-attestation: a /review run without the artifact skips, never guesses [#1128]" {
    DIRECT='' RUN_EVENT=issue_comment RUN_ID=1 RUN_SHA=mainsha STUB_ARTIFACT_ID='' \
        STUB_SHA_PR=99 run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'skip=true' "$BATS_TEST_TMPDIR/out" || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
    refute_grep '^number=' "$BATS_TEST_TMPDIR/out"
}

@test "review-attestation: a non-numeric artifact is refused [#1128]" {
    make_zip '12; echo pwned'
    DIRECT='' RUN_EVENT=issue_comment RUN_ID=1 RUN_SHA=mainsha STUB_ARTIFACT_ID=7 \
        run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'skip=true' "$BATS_TEST_TMPDIR/out" || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
}

@test "review-attestation: an artifact naming a closed PR is not attested [#1128]" {
    make_zip '2828'
    DIRECT='' RUN_EVENT=issue_comment RUN_ID=1 RUN_SHA=mainsha STUB_ARTIFACT_ID=7 \
        STUB_PR_STATE=closed run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'skip=true' "$BATS_TEST_TMPDIR/out" || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
}

# The automatic path must not start trusting an artifact: a pull_request run
# executes the PR's own copy of pr-agent.yml, so whatever it uploads was written
# by the change under review.
@test "review-attestation: a pull_request reviewer run still resolves by head SHA [AC4]" {
    make_zip '1'
    DIRECT='' RUN_EVENT=pull_request RUN_ID=1 RUN_SHA=prsha STUB_ARTIFACT_ID=7 \
        STUB_SHA_PR=2844 run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'number=2844' "$BATS_TEST_TMPDIR/out" || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
}

@test "review-attestation: a direct trigger keeps its own number [AC4]" {
    DIRECT=1234 RUN_EVENT='' RUN_ID='' RUN_SHA='' run resolve
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    grep -qx 'number=1234' "$BATS_TEST_TMPDIR/out" || { cat "$BATS_TEST_TMPDIR/out" >&2; false; }
}

# Every issue_comment-triggered reviewer run, skipped ones included, completes
# with main's SHA. Keyed on head_sha alone they share one group, `completed`
# cancels in progress, and the triage comment that follows a /review cancels
# its re-evaluation (39539977021, cancelled, 2026-10-21).
@test "review-attestation: re-evaluations of two /review runs do not share a group [#1128]" {
    # Ordered, not merely present: an id term placed after head_sha in the ||
    # chain is never reached, and one not gated on the event splits every
    # pull_request reviewer run too.
    run wf_query "
group = ' '.join(str(d['concurrency']['group']).split())
term = \"github.event.workflow_run.event == 'issue_comment' && github.event.workflow_run.id\"
if term not in group:
    print('group has no per-run discriminator gated on issue_comment'); sys.exit(1)
if group.index(term) > group.index('github.event.workflow_run.head_sha'):
    print('the run id comes after head_sha, so the || chain never reaches it'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "review-attestation: the gate may read another run's artifacts [#1128]" {
    run wf_query "
if d['permissions'].get('actions') != 'read':
    print('permissions lack actions: read; the artifact list is unreadable'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The producer half. The upload runs only on issue_comment (the path whose
# workflow file is the default branch's), carries the issue number through env,
# runs before any step that can end the job, and the gate's resolve step selects
# that artifact by name and reads the member the upload stores.
@test "pr-agent: the /review path publishes the PR number for the gate [#1128]" {
    run python3 -c "
import re, sys, yaml
pra = yaml.safe_load(open(sys.argv[1]))
wf = open(sys.argv[2]).read()
steps = pra['jobs']['review']['steps']
ids = [s.get('id') for s in steps]
up = [s for s in steps if 'upload-artifact' in str(s.get('uses', ''))]
if len(up) != 1:
    sys.exit('expected one upload-artifact step, found %d' % len(up))
up = up[0]
if \"github.event_name == 'issue_comment'\" not in str(up.get('if', '')):
    sys.exit('the upload is not restricted to issue_comment')
name = up['with']['name']
member = str(up['with']['path']).rsplit('/', 1)[-1]
gate = yaml.safe_load(wf)
resolve = [x for x in gate['jobs']['attestation']['steps'] if x.get('id') == 'pr'][0]['run']
if 'select(.name == \"' + name + '\")' not in resolve:
    sys.exit('the resolve step does not select the artifact named ' + name)
if not re.search(r'unzip -p \S+ ' + re.escape(member) + r'(\s|$)', resolve):
    sys.exit('the resolve step does not read the member ' + member + ' the upload stores')
if steps.index(up) > ids.index('credential'):
    sys.exit('the upload runs after a step that can end the job')
pub = [s for s in steps if s.get('id') == 'pr_number']
if not pub or 'github.event.issue.number' not in str(pub[0].get('env', {})):
    sys.exit('the PR number does not come from github.event.issue.number via env')
if '\"\$RUNNER_TEMP/' + member + '\"' not in str(pub[0].get('run', '')):
    sys.exit('the number is not written to the file the upload publishes: ' + member)
if up['with'].get('if-no-files-found') != 'error':
    sys.exit('a missing file would upload nothing and the gate would skip silently')
" "$PRA" "$WF"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# A comment that is not a slash command still starts a pr-agent run, which skips.
# Grouped by run id, each of those would run the gate to a skip of its own.
@test "review-attestation: a skipped reviewer run is not re-evaluated [#1128]" {
    run wf_query "
cond = ' '.join(str(d['jobs']['attestation'].get('if', '')).split())
if \"github.event.workflow_run.conclusion != 'skipped'\" not in cond:
    print('job if: admits skipped reviewer runs'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}
