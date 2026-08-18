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
    # `review_markers` is required (#1045), empty here because this reviewer is
    # declared only to test the declined path. Required rather than optional
    # because an absent field would silently mean "never attests", which is the
    # exact shape of the bug #1045 fixed — a reviewer reviewing and the gate not
    # seeing it. Adding a reviewer is still config-only, which is the claim.
    cat > "$TMP/config.json" <<'EOF'
{ "reviewers": [ { "login": "pr-agent",
                   "declined_markers": ["pr-agent: provider unavailable"],
                   "review_markers": [] } ],
  "escape": { "label": "merged-unreviewed", "section": "## Unreviewed merge rationale" } }
EOF
    run "$SCRIPT" --config "$TMP/config.json" --payload "$F/second-reviewer.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"declined"* ]]
    [[ "$output" == *"pr-agent"* ]]
}

# refute_grep exists because `! cmd` is EXEMPT from `set -e` in bash. A bare
# `! grep -q X file` anywhere except the final line of a @test is silently
# ignored, so the test passes on the strength of its last assertion alone.
#
# That is not hypothetical here: this file's AC5 case asserted the script names
# no reviewer, the script DID name one in a comment, and the suite reported
# green for as long as the violated assertion was not the last line. Found via a
# reviewer comment about a different defect in the same predicate.
refute_grep() {
    local pattern="$1" file="$2"
    if grep -qE "$pattern" "$file"; then
        printf 'expected NOT to find /%s/ in %s, but it is there:\n' "$pattern" "$file" >&2
        grep -nE "$pattern" "$file" >&2
        return 1
    fi
}

@test "AC5: the script names no reviewer of its own" {
    # If a login appears here at all — even in prose — the registry has stopped
    # being the single place a reviewer is declared, and the next migration is a
    # code change again.
    refute_grep 'coderabbitai' "$SCRIPT"
    refute_grep 'pr-agent' "$SCRIPT"
    refute_grep 'rate limited by' "$SCRIPT"
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
    refute_grep '^[[:space:]]*continue-on-error[[:space:]]*:' "$REPO/.github/workflows/review-attestation.yml"
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

# --- review findings on #1019 ---

@test "AC6: a flag given without a value exits 2, not 1" {
    # `shift 2` on a valueless flag fails under set -e and the shell exits 1 —
    # which this script's contract defines as "not reviewed" rather than "setup
    # error". A typo would have been reported as a verdict.
    for flag in --pr --payload --config; do
        run "$SCRIPT" "$flag"
        [ "$status" -eq 2 ]
    done
}

@test "AC6: a registry that is valid JSON but the wrong shape exits 2" {
    # Valid JSON of the wrong shape parsed fine and produced empty strings
    # downstream, which classify as "nobody declined" and "no escape
    # configured" — a malformed registry made the gate quietly MORE permissive.
    printf '[]' > "$TMP/array.json"
    run "$SCRIPT" --config "$TMP/array.json" --payload "$F/no-output.json"
    [ "$status" -eq 2 ]

    printf '{"reviewers":[{"declined_markers":["x"]}],"escape":{"label":"l","section":"s"}}' > "$TMP/nologin.json"
    run "$SCRIPT" --config "$TMP/nologin.json" --payload "$F/no-output.json"
    [ "$status" -eq 2 ]

    printf '{"reviewers":[{"login":"x","declined_markers":["m"]}],"escape":{"label":"l"}}' > "$TMP/nosection.json"
    run "$SCRIPT" --config "$TMP/nosection.json" --payload "$F/no-output.json"
    [ "$status" -eq 2 ]
}

@test "AC6: an absent gh with no payload exits 2, never 0" {
    # The live path with no way to read the PR must fail closed like every other
    # unreadable case.
    #
    # A hermetic PATH rather than an empty one: emptying PATH also removes bash
    # and jq, so the script dies 127 before reaching the branch under test and
    # the case passes for the wrong reason. gh, jq and bash all live in /usr/bin
    # here, so the only honest way to remove one is to build a PATH containing
    # exactly the others. Everything else this path touches is a shell builtin.
    mkdir -p "$TMP/bin"
    local b
    for b in bash jq dirname cat; do
        ln -s "$(command -v "$b")" "$TMP/bin/$b"
    done
    [ ! -e "$TMP/bin/gh" ]

    run env PATH="$TMP/bin" "$SCRIPT" --pr 1
    [ "$status" -eq 2 ]
    [[ "$output" == *"gh is not installed"* ]]
}

@test "AC3: the declined message names both the reviewer and the marker" {
    # Preserve-why, asserted rather than assumed: an operator must be able to
    # tell WHICH reviewer declined and on what evidence.
    run "$SCRIPT" --payload "$F/coderabbit-rate-limited.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"coderabbitai"* ]]
    [[ "$output" == *"rate limited by coderabbit.ai"* ]]
}

@test "meta: no bare negated assertion survives in this suite" {
    # The pitfall that hid a real violation here: `! cmd` is exempt from set -e,
    # so a failing negation anywhere but the last line of a @test is ignored.
    # refute_grep is the replacement; this keeps the pattern from creeping back.
    refute_grep '^[[:space:]]*![[:space:]]*grep' "$BATS_TEST_FILENAME"
}

@test "AC7: the workflow tolerates the gate not yet existing on the base branch" {
    # Chicken-and-egg found in production on #1019: pinning checkout to the base
    # branch (the fix for a PR being able to edit the gate that judges it) means
    # the PR which first INTRODUCES the gate checks out a base that lacks it.
    # The job then died with "No such file or directory" and reported failure
    # for a gate that simply was not deployed yet.
    #
    # Not a hole: a PR cannot remove a file from its own base, so this branch is
    # only reachable before the first merge.
    local wf="$REPO/.github/workflows/review-attestation.yml"
    grep -q 'not active on' "$wf"
    grep -qE 'if \[ ! -x scripts/check-review-attestation.sh \]' "$wf"
}

@test "AC7: the failing step states the verdict, not just an exit code" {
    # A bare `exit "$CODE"` made the RED step say nothing — the diagnostic was
    # in an earlier, green step, so anyone opening the failure saw a code and no
    # reason. This gate's own fourth constraint (fail closed, but say why),
    # violated at the job level inside the check that enforces it.
    local wf="$REPO/.github/workflows/review-attestation.yml"
    grep -q '::error title=Review attestation::' "$wf"
    grep -q 'merged-unreviewed' "$wf"
    grep -q 'does not veto' "$wf"
}

# --- #1045: a reviewer that posts comments instead of reviews ---
#
# PR-Agent (TOOL-013) runs under GITHUB_TOKEN as github-actions and publishes
# through the comments API, so reviews[] is empty on a PR it genuinely reviewed.
# The gate built to make green mean "reviewed" was calling this repo's own
# reviewer unreviewed — #906's failure with the sign flipped. Measured on #1037
# and #1042, both carrying a real Guide and both classified `declined`.

@test "AC: a declared reviewer's comment-shaped review attests" {
    run "$SCRIPT" --payload "$F/pr-agent-reviewed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
    [[ "$output" == *"github-actions"* ]]
}

@test "AC: the [bot] login suffix does not change the verdict" {
    # The two GitHub APIs disagree: `gh pr view --json comments` (GraphQL) returns
    # `github-actions`, `/issues/N/comments` (REST) returns `github-actions[bot]`.
    # Matching the raw string made the verdict depend on which API produced the
    # payload — half of #1033. Both spellings must attest identically.
    run "$SCRIPT" --payload "$F/pr-agent-reviewed-bot-suffix.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
}

@test "AC: suggestions are not a review" {
    # `## PR Code Suggestions` is the improve command's output and is deliberately
    # absent from review_markers. Attesting on it would make the gate mean "the
    # reviewer ran" rather than "the reviewer reviewed".
    run "$SCRIPT" --payload "$F/pr-agent-suggestions-only.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"pending"* ]]
}

@test "AC: an undeclared bot cannot attest by posting the same marker" {
    # The failure mode this guards is #1033 arriving through the comments door.
    # Matching "a bot commented" would let a labeler or release bot attest; the
    # pair (login, marker) must both be declared in the registry.
    run "$SCRIPT" --payload "$F/undeclared-bot-guide.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"pending"* ]]
}

@test "AC: a reviewer cannot attest its own PR by commenting on it" {
    # Self-review is the one case where reviewer output does not mean anyone
    # independent looked. The reviews[] path already excludes the author; the
    # comment path must exclude it identically or it becomes the way around.
    run "$SCRIPT" --payload "$F/pr-agent-self-authored.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"pending"* ]]
}

@test "AC: a real review outranks another reviewer's declined notice" {
    # Both are true at once whenever PR-Agent reviews while CodeRabbit's quota is
    # spent — which is the normal state of this repo, and was the exact state of
    # #1042. The gate asks whether a review happened, so the one that did wins.
    run "$SCRIPT" --payload "$F/pr-agent-reviewed-coderabbit-declined.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
    [[ "$output" == *"github-actions"* ]]
}

@test "AC: the registry must declare review_markers for every reviewer" {
    # Shape, not syntax. A registry entry missing review_markers parsed fine and
    # produced an empty marker list downstream, which classifies as "this reviewer
    # never attests" — the gate failing toward MORE refusals, but silently.
    printf '%s' '{"reviewers":[{"login":"x","declined_markers":[]}],
                  "escape":{"label":"l","section":"## s"}}' > "$TMP/bad.json"
    run "$SCRIPT" --config "$TMP/bad.json" --payload "$F/pr-agent-reviewed.json"
    [ "$status" -eq 2 ]
    [[ "$output" == *"wrong shape"* ]]
}

# --- exempt: changes carrying nothing a review could act on --------------------
#
# The pairing is the point. Excluding release PRs from the reviewer WITHOUT this
# exemption manufactures a PR with no reviewer output that nothing can ever
# clear — #1061's unreachable state, created on purpose. These cases pin both
# halves of the behaviour, including the two ways it must NOT fire.

@test "exempt: a release-shaped diff with no review passes" {
    run "$SCRIPT" --payload "$F/release-unreviewed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"exempt"* ]]
    [[ "$output" == *"release-please"* ]]
}

@test "exempt: one extra file ends the exemption" {
    # The signature cannot be borrowed. A change that touches the three release
    # paths AND anything else is an ordinary change wearing a release's clothes,
    # and this is what makes the exemption unavailable to game.
    run "$SCRIPT" --payload "$F/release-plus-one-file.json"
    [ "$status" -eq 1 ]
    [[ "$output" != *"exempt"* ]]
}

@test "exempt: a strict subset of the signature is NOT exempt" {
    # Exact set equality: changing only CHANGELOG.md requires review.
    run "$SCRIPT" --payload "$F/release-strict-subset.json"
    [ "$status" -eq 1 ]
    [[ "$output" != *"exempt"* ]]
}

@test "exempt: an empty file list does not exempt everything" {
    # Set subtraction says every element of an empty list is present in any set,
    # so a payload with no files matches EVERY signature. That would turn an
    # unreadable or truncated payload into a blanket pass — the failure this gate
    # exists to prevent, reached through a set-theory identity rather than a bug.
    run "$SCRIPT" --payload "$F/no-files-at-all.json"
    [ "$status" -eq 1 ]
    [[ "$output" != *"exempt"* ]]
}

@test "exempt: a real review outranks the exemption in the report" {
    # Both exit 0, so this is about what the record says. A release that WAS
    # reviewed should say attested, because that is the stronger fact and the
    # one a later reader wants.
    run "$SCRIPT" --payload "$F/release-reviewed.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"attested"* ]]
}

@test "exempt: the signature is declared in the registry, not in the script" {
    # This file names no vendor and no path, under test, and the exemption must
    # not be where that rule finally breaks.
    refute_grep 'release-please|CHANGELOG\.md|\.release-please-manifest' "$SCRIPT"
}
