#!/usr/bin/env bats
# Tests for scripts/pr-agent-push-gate.sh (TOOL-023): a push is reviewed only
# once 3 non-merge commits are newer than the review PR-Agent would pick as the
# previous one. The inputs are the PR's comments and commits as the REST API
# returns them, so the fixtures below are real jq over real JSON, no stubs.

setup() {
    GATE="$BATS_TEST_DIRNAME/../scripts/pr-agent-push-gate.sh"
    COMMENTS="$BATS_TEST_TMPDIR/comments.json"
    COMMITS="$BATS_TEST_TMPDIR/commits.json"
    echo '[]' > "$COMMENTS"
    echo '[]' > "$COMMITS"
}

# comment LOGIN CREATED_AT BODY: appends one issue comment.
comment() {
    jq --arg u "$1" --arg t "$2" --arg b "$3" \
        '. + [{user: {login: $u}, created_at: $t, updated_at: $t, body: $b}]' \
        "$COMMENTS" > "$COMMENTS.tmp" && mv "$COMMENTS.tmp" "$COMMENTS"
}

# commit AUTHOR_DATE [PARENTS]: appends one PR commit (PARENTS defaults to 1).
commit() {
    jq --arg t "$1" --argjson p "${2:-1}" \
        '. + [{sha: ("c" + (length | tostring)), parents: [range($p) | {sha: "p"}],
               commit: {author: {date: $t}, message: "change"}}]' \
        "$COMMITS" > "$COMMITS.tmp" && mv "$COMMITS.tmp" "$COMMITS"
}

FULL=$'## PR Reviewer Guide \xf0\x9f\x94\x8d\n\n<!-- pr-agent:review:full -->\n\nfindings'
INCREMENTAL=$'## Incremental PR Reviewer Guide \xf0\x9f\x94\x8d\n\n<!-- pr-agent:review:incremental -->\n\nmore'

# Every case runs under zsh and bash, which must agree: the repo's scripts run
# under both (.claude/CLAUDE.md).
_gate() {
    run zsh "$GATE" --comments "$COMMENTS" --commits "$COMMITS" "$@"
    local zsh_status="$status" zsh_output="$output"
    run bash "$GATE" --comments "$COMMENTS" --commits "$COMMITS" "$@"
    if [ "$status" != "$zsh_status" ] || [ "$output" != "$zsh_output" ]; then
        printf 'bash (%s) and zsh (%s) disagree\n--- bash\n%s\n--- zsh\n%s\n' \
            "$status" "$zsh_status" "$output" "$zsh_output"
        return 1
    fi
}

@test "--help exits 0 and names the threshold" {
    run "$GATE" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"--threshold"* ]]
}

@test "a missing argument is a usage error" {
    run "$GATE" --comments "$COMMENTS"
    [ "$status" -eq 2 ]
}

@test "below the threshold: two new commits since the review do not run it" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z
    _gate
    [ "$status" -eq 0 ]
    [[ "$output" == *"run=false"* ]]
    [[ "$output" == *"2 new commit"* ]]
}

@test "at the threshold: three new commits run it" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z
    commit 2026-09-25T13:00:00Z
    _gate
    [ "$status" -eq 0 ]
    [[ "$output" == *"run=true"* ]]
}

@test "commits older than the review do not count" {
    commit 2026-09-25T08:00:00Z
    commit 2026-09-25T09:00:00Z
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    _gate
    [[ "$output" == *"run=false"* ]]
    [[ "$output" == *"1 new commit"* ]]
}

@test "merge commits do not count: updating the branch from main is not new work" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z 2
    commit 2026-09-25T13:00:00Z 2
    commit 2026-09-25T14:00:00Z 2
    _gate
    [[ "$output" == *"run=false"* ]]
}

@test "the latest review is the baseline: an incremental review resets the count" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z
    commit 2026-09-25T13:00:00Z
    comment "github-actions[bot]" 2026-09-25T14:00:00Z "$INCREMENTAL"
    commit 2026-09-25T15:00:00Z
    _gate
    [[ "$output" == *"run=false"* ]]
    [[ "$output" == *"1 new commit"* ]]
}

@test "a review is recognised by its heading alone, as PR-Agent's legacy prefix" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "## PR Reviewer Guide"$'\n\nno identity line'
    commit 2026-09-25T11:00:00Z
    _gate
    [[ "$output" == *"run=false"* ]]
}

@test "prose that mentions the Guide is not a review, nor is an identity past line 5" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z
    commit 2026-09-25T13:00:00Z
    comment "mlorentedev" 2026-09-25T14:00:00Z "The ## PR Reviewer Guide above is triaged."
    comment "mlorentedev" 2026-09-25T14:30:00Z $'a\nb\nc\nd\ne\n<!-- pr-agent:review:full -->'
    _gate
    [[ "$output" == *"run=true"* ]]
}

@test "an identity line with CRLF endings still marks a review" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z $'## Something else\r\n\r\n<!-- pr-agent:review:full -->\r\n'
    commit 2026-09-25T11:00:00Z
    _gate
    [[ "$output" == *"run=false"* ]]
}

@test "with no previous review it runs, and PR-Agent reviews in full" {
    comment "mlorentedev" 2026-09-25T10:00:00Z "a note"
    commit 2026-09-25T11:00:00Z
    _gate
    [[ "$output" == *"run=true"* ]]
    [[ "$output" == *"no previous review"* ]]
}

@test "a push by a bot is skipped, as PR-Agent skips it" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    commit 2026-09-25T12:00:00Z
    commit 2026-09-25T13:00:00Z
    _gate --sender-type Bot
    [[ "$output" == *"run=false"* ]]
    [[ "$output" == *"bot"* ]]
}

@test "unreadable input reviews: a skip must be justified, never assumed" {
    run "$GATE" --comments "$BATS_TEST_TMPDIR/absent.json" --commits "$COMMITS"
    [ "$status" -eq 0 ]
    [[ "$output" == *"run=true"* ]]
    echo 'not json' > "$COMMITS"
    run "$GATE" --comments "$COMMENTS" --commits "$COMMITS"
    [[ "$output" == *"run=true"* ]]
}

@test "--threshold changes the bound" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    commit 2026-09-25T11:00:00Z
    _gate --threshold 1
    [[ "$output" == *"run=true"* ]]
}

@test "the output is two key=value lines, ready for GITHUB_OUTPUT" {
    comment "github-actions[bot]" 2026-09-25T10:00:00Z "$FULL"
    _gate
    [ "${#lines[@]}" -eq 2 ]
    [[ "${lines[0]}" =~ ^run=(true|false)$ ]]
    [[ "${lines[1]}" == reason=* ]]
}
