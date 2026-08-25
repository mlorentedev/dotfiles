#!/usr/bin/env bats
# HARNESS-080 (#1221): a Dependabot-triggered workflow run cannot read this
# repository's secrets.
#
# GitHub serves those runs from a SEPARATE store — `dependabot/secrets`, empty
# here — so `secrets.ANYTHING` expands to the empty string with no error and no
# warning. Measured 2026-08-24 on #1219 and #1220: two red checks on every
# weekly dependency PR since 2026-08-07, and nobody looked, because a red check
# on a bot's PR is exactly what a reader learns to scroll past.
#
# The generic assertion below is the point of this file. Fixing the two current
# offenders one at a time would leave the NEXT workflow that reaches for a
# secret to rediscover this the same way — a class of defect the repository
# already decided is answered with a guard, not with a memo.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    WF_DIR="$REPO/.github/workflows"
    ACTOR_GUARD="github.actor != 'dependabot\[bot\]'"
}

# --- the class guard ---------------------------------------------------------

@test "dependabot: every pull_request job needing a repo secret excludes dependabot" {
    # Scoped PER JOB, not per file, and that distinction is the whole assertion.
    # A file-wide `grep` accepts a guard that sits on some OTHER job — or in a
    # comment — while the job actually holding the credential runs unguarded.
    # It would report green on a workflow whose `lint` job carries the guard and
    # whose `publish` job reads a deploy token without it, which is precisely
    # the arrangement this file exists to make impossible. Raised by CodeRabbit
    # on the first revision of this test, correctly: a guard that looks like it
    # covers the class is the failure mode of the very lesson this PR writes.
    #
    # Secret names are matched EXACTLY. `secrets.GITHUB_TOKEN2` is not
    # GITHUB_TOKEN, and a substring test silently exempted the job that read it.
    #
    # Parsed with PyYAML rather than grepped, following tests/pr-agent-config.bats.
    # Note `d[True]`: YAML 1.1 resolves a bare `on:` key to the boolean true, so
    # `d['on']` is a KeyError on every workflow file in this repository.
    run python3 -c "
import pathlib, re, sys, yaml

GUARD = \"github.actor != 'dependabot[bot]'\"
offenders = []
for wf in sorted(pathlib.Path('$WF_DIR').glob('*.yml')):
    doc = yaml.safe_load(open(wf))
    if not isinstance(doc, dict):
        continue
    triggers = doc.get(True, doc.get('on'))
    if not isinstance(triggers, dict) or 'pull_request' not in triggers:
        continue
    for name, job in (doc.get('jobs') or {}).items():
        body = yaml.safe_dump(job)
        secrets = set(re.findall(r'secrets\.([A-Za-z_][A-Za-z0-9_]*)', body))
        secrets.discard('GITHUB_TOKEN')
        if not secrets:
            continue
        if GUARD not in str(job.get('if', '')):
            offenders.append('%s job %s reads %s' % (wf.name, name, ', '.join(sorted(secrets))))

if offenders:
    print('These jobs run on pull_request and read a repository secret, but their')
    print('own \`if:\` does not exclude dependabot[bot]. On a Dependabot run that')
    print('secret is the empty string, so the job cannot succeed and never will:')
    for o in offenders:
        print('  ' + o)
    print('')
    print('Add \"github.actor != %s\" to THAT JOB (a guard on a' % repr('dependabot[bot]'))
    print('sibling job does not protect this one), or put the credential into the')
    print('Dependabot secrets store as a deliberate decision.')
    sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "dependabot: the class guard actually inspects some secret-using jobs" {
    # Without this, the check above passes vacuously the day the glob, the `on:`
    # key or the secret pattern stops matching anything — a guard that inspected
    # nothing reports the same green as a guard that found nothing wrong. The
    # repository has paid for that distinction (#1203, #1206).
    run python3 -c "
import pathlib, re, sys, yaml

inspected = []
for wf in sorted(pathlib.Path('$WF_DIR').glob('*.yml')):
    doc = yaml.safe_load(open(wf))
    if not isinstance(doc, dict):
        continue
    triggers = doc.get(True, doc.get('on'))
    if not isinstance(triggers, dict) or 'pull_request' not in triggers:
        continue
    for name, job in (doc.get('jobs') or {}).items():
        secrets = set(re.findall(r'secrets\.([A-Za-z_][A-Za-z0-9_]*)', yaml.safe_dump(job)))
        secrets.discard('GITHUB_TOKEN')
        if secrets:
            inspected.append('%s:%s' % (wf.name, name))

# add-to-project.yml (BITACORA_PAT) and pr-agent.yml (NAN_API_KEY).
print('inspected: ' + ', '.join(inspected))
sys.exit(0 if len(inspected) >= 2 else 1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# --- add-to-project ----------------------------------------------------------

@test "add-to-project: skips dependabot without dropping the fork guard" {
    wf="$WF_DIR/add-to-project.yml"
    grep -q "$ACTOR_GUARD" "$wf"
    # The fork test is a SEPARATE failure mode with the same symptom, and the
    # Dependabot fix must not be mistaken for a replacement: a Dependabot PR is
    # a branch in this repo, so `head.repo.fork` is false and never caught it.
    grep -q 'github.event.pull_request.head.repo.fork == false' "$wf"
}

@test "add-to-project: still runs for issues, which carry no fork or bot concern" {
    grep -q "github.event_name == 'issues'" "$WF_DIR/add-to-project.yml"
}

# --- pr-agent ----------------------------------------------------------------

@test "pr-agent: gates exactly one half of the trigger on the dependabot actor" {
    wf="$WF_DIR/pr-agent.yml"
    # Exactly one occurrence: the `pull_request` half. A human typing `/review`
    # on a Dependabot PR is asking on purpose, and that run is triggered by the
    # human — so its secrets are present and the reviewer works. Gating the
    # issue_comment half too would silently refuse them.
    [ "$(grep -c "$ACTOR_GUARD" "$wf")" -eq 1 ]
    grep -q "github.event_name == 'issue_comment'" "$wf"
}

@test "pr-agent: keeps the release-please exclusion it already had" {
    # Two independent conditions that each only ever REMOVE a review. Losing one
    # while adding the other would be a silent regression.
    grep -q "startsWith(github.event.pull_request.head.ref, 'release-please--')" \
        "$WF_DIR/pr-agent.yml"
}

# --- review-attestation ------------------------------------------------------

@test "review-attestation: does not tell readers it is unrequired when it is required" {
    # Verified 2026-08-24: required_status_checks.contexts includes
    # "review-attestation". The workflow's failure message said the opposite,
    # and that message is what a reader sees at the moment the check is red.
    wf="$WF_DIR/review-attestation.yml"
    refute_grep 'This check is NOT required in branch protection' "$wf"
    refute_grep 'NOT `required` in branch protection yet' "$wf"
}

@test "review-attestation: still offers both ways out of a red verdict" {
    # Narrowing what the message CLAIMS must not narrow what it OFFERS.
    wf="$WF_DIR/review-attestation.yml"
    grep -q 'Get the PR reviewed' "$wf"
    grep -q 'merged-unreviewed' "$wf"
}
