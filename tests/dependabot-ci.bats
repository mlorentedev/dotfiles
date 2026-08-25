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

@test "dependabot: every pull_request workflow needing a repo secret excludes dependabot" {
    # A workflow qualifies when BOTH hold: it triggers on `pull_request`, and it
    # reads a secret other than GITHUB_TOKEN (which is always present, merely
    # read-only for Dependabot). Such a job cannot do its work on a Dependabot
    # run, so it must not pretend to try.
    #
    # Read line by line rather than word-split: zsh does not split unquoted
    # parameters, so `set -- $list` would yield one field holding everything.
    offenders=""
    while IFS= read -r wf; do
        [ -n "$wf" ] || continue
        grep -qE '^[[:space:]]{2}pull_request:' "$wf" || continue
        grep -oE 'secrets\.[A-Z_]+' "$wf" | grep -qv 'secrets\.GITHUB_TOKEN' || continue
        grep -q "$ACTOR_GUARD" "$wf" || offenders="$offenders $(basename "$wf")"
    done <<< "$(find "$WF_DIR" -maxdepth 1 -name '*.yml' | sort)"

    if [ -n "$offenders" ]; then
        echo "These workflows run on pull_request, read a repository secret, and do not"
        echo "exclude dependabot[bot] — on a Dependabot PR that secret is the empty string:"
        echo "  $offenders"
        echo
        echo "Add \"github.actor != 'dependabot[bot]'\" to the job's \`if:\`, or move the"
        echo "credential into the Dependabot secrets store deliberately."
        return 1
    fi
}

@test "dependabot: the class guard actually inspects the workflow directory" {
    # Without this, the loop above passes vacuously the day the glob, the path
    # or the trigger pattern stops matching anything — a guard that inspects
    # nothing reports the same green as a guard that found nothing wrong. The
    # repository has paid for that distinction (#1203, #1206).
    matched=0
    while IFS= read -r wf; do
        [ -n "$wf" ] || continue
        grep -qE '^[[:space:]]{2}pull_request:' "$wf" || continue
        grep -oE 'secrets\.[A-Z_]+' "$wf" | grep -qv 'secrets\.GITHUB_TOKEN' || continue
        matched=$((matched + 1))
    done <<< "$(find "$WF_DIR" -maxdepth 1 -name '*.yml' | sort)"

    # add-to-project.yml (BITACORA_PAT) and pr-agent.yml (NAN_API_KEY).
    [ "$matched" -ge 2 ]
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
