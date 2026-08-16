#!/usr/bin/env bats
# GUARD-002 (#906): the review-attestation gate.
#
# Every case is offline, driven by a committed payload, so the classifier under
# test is byte-for-byte the one that runs in CI — a suite that exercised a
# different code path than production would be evidence about the suite.
#
# The load-bearing fixture is `pr-1009.raw.json`: the real 2026-08-16 payload
# from a PR that carried a CodeRabbit rate-limit notice, showed `CodeRabbit
# pass` in `gh pr checks`, was never reviewed, and was merged. If this suite
# ever stops classifying that file as `declined`, the gate has stopped doing
# the one thing it was built for.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    SCRIPT="$REPO/scripts/check-review-attestation.sh"
    F="$BATS_TEST_DIRNAME/fixtures/review-attestation"
    TMP="$(mktemp -d)"
}

teardown() { [ -z "${TMP:-}" ] || rm -rf "$TMP"; }

# --- syntax, in both shells (repo rule) ---

@test "check-review-attestation.sh valid bash syntax" {
    bash -n "$SCRIPT"
}

@test "check-review-attestation.sh valid zsh syntax" {
    if command -v zsh >/dev/null 2>&1; then zsh -n "$SCRIPT"; else skip "zsh not available"; fi
}

# --- AC1/AC3: classifies the three states, by content ---

@test "AC1: classifies the REAL #1009 rate-limit payload as declined" {
    run "$SCRIPT" --payload "$F/pr-1009.raw.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"declined"* ]]
    [[ "$output" == *"coderabbitai"* ]]
}

@test "AC1: classifies the other two real captures as declined too" {
    for pr in 1007 1013; do
        run "$SCRIPT" --payload "$F/pr-${pr}.raw.json"
        [ "$status" -eq 1 ]
        [[ "$output" == *"declined"* ]]
    done
}

@test "AC3: a PR with no reviewer output is pending, not declined" {
    run "$SCRIPT" --payload "$F/no-output.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"pending"* ]]
    # The fourth constraint: refusing is not enough, the refusal must say WHICH
    # refusal it is. Collapsing these two would repeat #988's second defect,
    # where the error path discarded the HTTP status and showed the wrong cause.
    [[ "$output" != *"declined"* ]]
}

# --- AC2: a real review attests, whoever performed it ---

@test "AC2: a bot review attests" {
    run "$SCRIPT" --payload "$F/bot-reviewed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
}

@test "AC2: a human review attests" {
    run "$SCRIPT" --payload "$F/human-reviewed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
}

@test "AC2: the author reviewing their own PR does NOT attest" {
    # Independence is the point of the artifact — the same rule
    # harness/reviewer-pool.json enforces for spec reviews, where the reviewer
    # must not be the implementer.
    run "$SCRIPT" --payload "$F/self-review.json"
    [ "$status" -eq 1 ]
    [[ "$output" != *"attested"* ]]
}

# --- AC4: the escape needs BOTH halves ---

@test "AC4: label plus a non-empty rationale is disclosed" {
    run "$SCRIPT" --payload "$F/disclosed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"disclosed"* ]]
}

@test "AC4: the label alone is not a disclosure" {
    run "$SCRIPT" --payload "$F/label-only.json"
    [ "$status" -eq 1 ]
}

@test "AC4: the rationale alone is not a disclosure" {
    run "$SCRIPT" --payload "$F/section-only.json"
    [ "$status" -eq 1 ]
}

@test "AC4: an empty rationale section is not a disclosure" {
    # The likeliest way to satisfy this by accident: paste the heading, merge
    # before writing under it. A heading with nothing beneath it discloses
    # exactly as much as no heading.
    run "$SCRIPT" --payload "$F/empty-rationale.json"
    [ "$status" -eq 1 ]
}

# --- AC5: a reviewer is config, not code ---

@test "AC5: a second reviewer declared in config is recognized, with no code change" {
    # This is what makes #786 (PR-Agent) an entry rather than a rewrite.
    cat > "$TMP/config.json" <<'EOF'
{ "reviewers": [ { "login": "pr-agent",
                   "declined_markers": ["pr-agent: provider unavailable"] } ],
  "escape": { "label": "merged-unreviewed", "section": "## Unreviewed merge rationale" } }
EOF
    run "$SCRIPT" --config "$TMP/config.json" --payload "$F/second-reviewer.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"declined"* ]]
    [[ "$output" == *"pr-agent"* ]]
}

@test "AC5: the script names no reviewer of its own" {
    # If a login is hardcoded here, the config is decorative and the next
    # reviewer migration is a code change again.
    ! grep -q 'coderabbitai' "$SCRIPT"
    ! grep -q 'pr-agent' "$SCRIPT"
}

@test "AC5: the shipped registry declares at least one reviewer and the escape" {
    jq -e '.reviewers | length >= 1' "$REPO/harness/review-attestation.json" >/dev/null
    jq -e '.escape.label != null and .escape.section != null' "$REPO/harness/review-attestation.json" >/dev/null
}

# --- AC6: unreadable input fails closed ---

@test "AC6: malformed JSON exits 2, never 0" {
    run "$SCRIPT" --payload "$F/malformed.json"
    [ "$status" -eq 2 ]
}

@test "AC6: an empty payload exits 2, never 0" {
    : > "$TMP/empty.json"
    run "$SCRIPT" --payload "$TMP/empty.json"
    [ "$status" -eq 2 ]
}

@test "AC6: a missing payload file exits 2, never 0" {
    run "$SCRIPT" --payload "$TMP/does-not-exist.json"
    [ "$status" -eq 2 ]
}

@test "AC6: a missing or malformed reviewer registry exits 2, never 0" {
    run "$SCRIPT" --config "$TMP/no-such-config.json" --payload "$F/no-output.json"
    [ "$status" -eq 2 ]
    printf 'not json {[' > "$TMP/bad-config.json"
    run "$SCRIPT" --config "$TMP/bad-config.json" --payload "$F/no-output.json"
    [ "$status" -eq 2 ]
}

@test "AC6: every failure path says it could not determine, rather than implying fine" {
    run "$SCRIPT" --payload "$F/malformed.json"
    [[ "$output" == *"NOT treated as reviewed"* ]]
}

# --- usage ---

@test "unknown argument exits 2" {
    run "$SCRIPT" --bogus
    [ "$status" -eq 2 ]
}

@test "--help exits 0 and prints usage" {
    run "$SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
}

@test "--quiet suppresses the report but keeps the exit code" {
    run "$SCRIPT" --quiet --payload "$F/human-reviewed.json"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# --- AC7: the gate re-runs when a review lands, without a push ---

@test "AC7: the workflow triggers on pull_request AND issue_comment" {
    # Without issue_comment the gate would only ever answer at the one moment
    # the answer is always "pending" — reviewers land after the checks settle.
    # That timing gap is half of the bug, not a detail.
    python3 -c "
import sys, yaml
d = yaml.safe_load(open('$REPO/.github/workflows/review-attestation.yml'))
on = d[True] if True in d else d['on']
sys.exit(0 if 'pull_request' in on and 'issue_comment' in on else 1)
"
}

@test "AC7: the workflow does not swallow the gate's failure" {
    # A `continue-on-error: true` here would rebuild the exact defect: a check
    # that runs, decides 'not reviewed', and reports green anyway.
    #
    # Anchored to a YAML KEY, not the bare string. The first version of this
    # assertion matched the workflow's own comment explaining why the key is
    # absent — a guard tripping on documentation *about* the thing rather than
    # the thing, which is the same false positive ScanUnresolvedTags strips code
    # spans to avoid, and the same one that made #998's archive gate unpassable.
    ! grep -qE '^[[:space:]]*continue-on-error[[:space:]]*:' "$REPO/.github/workflows/review-attestation.yml"
}

@test "AC7: the issue_comment path publishes a status onto the PR head" {
    # Found by running this workflow on its own first PR (#1019). A run
    # triggered by issue_comment is associated with the DEFAULT BRANCH, not the
    # PR's head commit, so its check-run never appears on the PR. Without an
    # explicit commit status the trigger executes and changes nothing visible —
    # a re-run that only appears to re-run, which is this spec's own defect
    # class wearing a different hat.
    local wf="$REPO/.github/workflows/review-attestation.yml"
    grep -qE '^[[:space:]]*statuses:[[:space:]]*write' "$wf"
    grep -q 'statuses/' "$wf"
}

@test "AC7: the run still fails when the PR is not attested" {
    # The status publish captures the exit code, so the final step must restore
    # it. Otherwise capturing it to publish the status would silently convert
    # every verdict into a green run — the bug, rebuilt inside the fix for it.
    grep -qE 'exit "\$CODE"' "$REPO/.github/workflows/review-attestation.yml"
}

# --- AC4 (cont.): line-ending and whitespace robustness ---

@test "AC4: a CRLF body still satisfies the escape" {
    # GitHub's web editor produces CRLF. Before the fix, every line arrived as
    # "## Unreviewed merge rationale\r", the exact-match index missed, and a PR
    # carrying BOTH halves of the escape was refused — and told to add the
    # section it already had. Identical payloads: LF exited 0, CRLF exited 1.
    #
    # This is the case the hand-written fixtures could not catch, because they
    # were all written here with LF heredocs. Same class as the repo's
    # `Set-Content`/`.gitattributes eol=lf` lesson: it reappears wherever text
    # crosses from a Windows-flavoured producer.
    run "$SCRIPT" --payload "$F/disclosed-crlf.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"disclosed"* ]]
}

@test "AC4: trailing whitespace on the heading still satisfies the escape" {
    run "$SCRIPT" --payload "$F/disclosed-trailing-ws.json"
    [ "$status" -eq 0 ]
}

@test "AC4: line-ending tolerance does not weaken the negatives" {
    # The risk of normalising input is over-accepting. Re-assert the three
    # refusals after the fix, so tolerance never quietly becomes permissiveness.
    for f in label-only section-only empty-rationale; do
        run "$SCRIPT" --payload "$F/$f.json"
        [ "$status" -eq 1 ]
    done
}
