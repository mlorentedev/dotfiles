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

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    WF="$REPO/.github/workflows/review-attestation.yml"
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
