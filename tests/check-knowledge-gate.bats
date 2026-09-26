#!/usr/bin/env bats
# Tests for scripts/check-knowledge-gate.sh (HARNESS-160): every PR names, in a
# "## Knowledge" section, the lesson, ADR and runbook it wrote, or why none.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    REPO_FIXTURE="$BATS_TEST_TMPDIR/repo"
    mkdir -p "$REPO_FIXTURE"
    cd "$REPO_FIXTURE" || exit 1
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    mkdir -p docs/lessons docs/adr docs/runbooks
    echo "old" > docs/lessons/lesson-001-old.md
    echo "index" > docs/lessons/_index.md
    echo "seed" > README.md
    git add -A
    git commit -q -m "seed"
    git checkout -q -b feature
    unset SDD_PR_BODY SDD_LABELS SDD_PR_AUTHOR
}

_commit() {
    git add -A
    git commit -q -m "${1:-change}"
}

# Every case runs under zsh and bash, which must agree: the repo's scripts run
# under both (.claude/CLAUDE.md), and bash-only constructs fail silently in zsh.
_both() {
    run zsh "$SCRIPTS_DIR/check-knowledge-gate.sh" "$@"
    local zsh_status="$status" zsh_output="$output"
    run bash "$SCRIPTS_DIR/check-knowledge-gate.sh" "$@"
    if [ "$status" != "$zsh_status" ] || [ "$output" != "$zsh_output" ]; then
        printf 'bash (%s) and zsh (%s) disagree\n--- bash\n%s\n--- zsh\n%s\n' \
            "$status" "$zsh_status" "$output" "$zsh_output"
        return 1
    fi
}

_gate() {
    _both --base-ref main --head-ref feature "$@"
}

# A body whose Lesson line is $1; ADR and Runbook are reasoned nones.
_body_with_lesson() {
    SDD_PR_BODY="## Summary

Something changed.

## Knowledge

- Lesson: $1
- ADR: none: no contract or policy changed
- Runbook: none: no new procedure
"
    export SDD_PR_BODY
}

_add_lesson() {
    echo "new" > docs/lessons/lesson-002-new.md
    _commit "add lesson"
}

@test "--help shows usage and exits 0" {
    _both --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"## Knowledge"* ]]
}

@test "exits 2 when --base-ref is missing" {
    export SDD_PR_BODY="x"
    _both --head-ref feature
    [ "$status" -eq 2 ]
}

@test "exits 2 on an unknown argument" {
    _both --bogus
    [ "$status" -eq 2 ]
    [[ "$output" == *"Unknown"* ]]
}

@test "exits 2 when the PR body is unset: no PR context is a wiring error, not a pass" {
    _gate
    [ "$status" -eq 2 ]
    [[ "$output" == *"no PR context"* ]]
}

@test "exits 2 when the range cannot be diffed" {
    _body_with_lesson "none: nothing learned"
    _both --base-ref no-such-ref --head-ref feature
    [ "$status" -eq 2 ]
}

@test "passes a lesson this PR adds and two reasoned nones" {
    _add_lesson
    _body_with_lesson "docs/lessons/lesson-002-new.md"
    _gate
    [ "$status" -eq 0 ]
    [[ "$output" == *"[OK]"* ]]
    [[ "$output" == *"docs/lessons/lesson-002-new.md"* ]]
}

@test "passes three paths, one per kind" {
    _add_lesson
    echo "adr" > docs/adr/adr-001-x.md
    echo "rb" > docs/runbooks/release.md
    _commit "adr and runbook"
    SDD_PR_BODY="## Knowledge
- Lesson: docs/lessons/lesson-002-new.md
- ADR: docs/adr/adr-001-x.md
- Runbook: docs/runbooks/release.md"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 0 ]
}

@test "accepts bold labels, inline-code paths and CRLF line endings" {
    _add_lesson
    SDD_PR_BODY=$'## Knowledge\r\n\r\n- **Lesson:** `docs/lessons/lesson-002-new.md`\r\n- **ADR:** none: no decision\r\n- **Runbook**: none: no procedure\r\n'
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 0 ]
}

@test "labels and none are read in any letter case" {
    SDD_PR_BODY="## knowledge
- lesson: None: nothing learned
- adr: NONE: no decision
- RUNBOOK: none: no procedure"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 0 ]
}

@test "a changed existing lesson counts: extending one is capture too" {
    echo "more" >> docs/lessons/lesson-001-old.md
    _commit "extend"
    _body_with_lesson "docs/lessons/lesson-001-old.md"
    _gate
    [ "$status" -eq 0 ]
}

@test "a renamed lesson counts under its new path" {
    git mv docs/lessons/lesson-001-old.md docs/lessons/lesson-001-renamed.md
    _commit "rename"
    _body_with_lesson "docs/lessons/lesson-001-renamed.md"
    _gate
    [ "$status" -eq 0 ]
}

@test "fails when the section is missing" {
    export SDD_PR_BODY="## Summary

No knowledge section here."
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"## Knowledge"* ]]
}

@test "fails when the body is set but empty" {
    export SDD_PR_BODY=""
    _gate
    [ "$status" -eq 1 ]
}

@test "fails when a line is missing, naming it" {
    SDD_PR_BODY="## Knowledge
- Lesson: none: nothing learned
- ADR: none: no decision"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Runbook"* ]]
}

@test "fails on a bare none: the reason is the point" {
    _body_with_lesson "none"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Lesson"* ]]
    [[ "$output" == *"reason"* ]]
}

@test "fails on none: with an empty reason" {
    _body_with_lesson "none:   "
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"reason"* ]]
}

@test "fails on the template's placeholder reason copied verbatim" {
    _body_with_lesson "none: <reason>"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"placeholder"* ]]
}

@test "fails when a line is neither a path nor a reasoned none" {
    _body_with_lesson "yes, see the PR"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Lesson"* ]]
}

@test "fails on a lesson path this PR does not change" {
    _body_with_lesson "docs/lessons/lesson-001-old.md"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"lesson-001-old.md"* ]]
    [[ "$output" == *"not changed by this PR"* ]]
}

@test "fails on a lesson this PR deletes: a deletion is not capture" {
    git rm -q docs/lessons/lesson-001-old.md
    _commit "delete"
    _body_with_lesson "docs/lessons/lesson-001-old.md"
    _gate
    [ "$status" -eq 1 ]
}

@test "fails on a changed path outside the kind's directory" {
    echo "more" >> README.md
    _commit "readme"
    _body_with_lesson "README.md"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"docs/lessons/"* ]]
}

@test "fails on _index.md: an index row is not a lesson" {
    echo "row" >> docs/lessons/_index.md
    _commit "index"
    _body_with_lesson "docs/lessons/_index.md"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"_index.md"* ]]
}

@test "fails when one of several paths does not count, naming that one" {
    _add_lesson
    _body_with_lesson "docs/lessons/lesson-002-new.md, docs/lessons/lesson-009-ghost.md"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"lesson-009-ghost.md"* ]]
}

@test "fails on a duplicated line: two answers are no answer" {
    SDD_PR_BODY="## Knowledge
- Lesson: none: nothing learned
- Lesson: none: said twice
- ADR: none: no decision
- Runbook: none: no procedure"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Lesson"* ]]
}

@test "a section quoted in a fenced code block does not count" {
    SDD_PR_BODY='## Summary

The shape the gate wants:

```markdown
## Knowledge
- Lesson: none: an example
- ADR: none: an example
- Runbook: none: an example
```'
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"## Knowledge"* ]]
}

@test "a fence left open says so, instead of only reporting a missing section" {
    SDD_PR_BODY='## Summary

```bash
make test

## Knowledge
- Lesson: none: nothing learned
- ADR: none: no decision
- Runbook: none: no procedure'
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"line 3"* ]]
    [[ "$output" == *"never closed"* ]]
}

@test "lines inside an HTML comment do not count: GitHub does not render them" {
    SDD_PR_BODY="## Knowledge

<!--
Example:
- Lesson: none: an example in the template
-->
- Lesson: <!-- a lesson path --> none: nothing learned here
- ADR: none: no decision
- Runbook: none: no procedure"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 0 ]
    SDD_PR_BODY="## Knowledge
<!-- - Lesson: none: only in a comment -->
- ADR: none: no decision
- Runbook: none: no procedure"
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Lesson: the line is missing"* ]]
}

@test "the section ends at the next heading" {
    SDD_PR_BODY="## Knowledge
- Lesson: none: nothing learned
- ADR: none: no decision

## Test plan
- Runbook: none: this line belongs to another section"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Runbook"* ]]
}

@test "--explain prints the expected shape on failure" {
    export SDD_PR_BODY="no section"
    _gate --explain
    [ "$status" -eq 1 ]
    [[ "$output" == *"none: <reason>"* ]]
}

@test "dependencies: a dependency bot's PR with the label is skipped" {
    export SDD_PR_BODY="Bumps a dependency." SDD_LABELS="dependencies" SDD_PR_AUTHOR="dependabot[bot]"
    _gate
    [ "$status" -eq 0 ]
    [[ "$output" == *"[OK]"* ]]
    [[ "$output" == *"dependabot[bot]"* ]]
}

@test "dependencies: a human's PR with the label is judged" {
    export SDD_PR_BODY="Bumps a dependency." SDD_LABELS="dependencies" SDD_PR_AUTHOR="someone"
    _gate
    [ "$status" -eq 1 ]
}

@test "dependencies: a bot without the label is judged" {
    export SDD_PR_BODY="Bumps a dependency." SDD_LABELS="" SDD_PR_AUTHOR="dependabot[bot]"
    _gate
    [ "$status" -eq 1 ]
}

@test "dependencies: the bot list equals spec-gate's, so the two skips cannot drift" {
    _bots() {
        awk '/^_is_dependency_bot\(\)/ { on=1; next } on && /^}/ { exit } on' "$1" |
            grep -oE '"[^"]+\[bot\]"' | sort
    }
    spec=$(_bots "$SCRIPTS_DIR/check-spec-gate.sh")
    knowledge=$(_bots "$SCRIPTS_DIR/check-knowledge-gate.sh")
    [ -n "$spec" ]
    [ "$spec" = "$knowledge" ]
}

@test "release-please footer: a release PR passes on the section its footer carries" {
    footer=$(jq -r '."pull-request-footer" // empty' "$BATS_TEST_DIRNAME/../release-please-config.json")
    [ -n "$footer" ]
    SDD_PR_BODY=":robot: I have created a release *beep* *boop*
---


## [0.60.0](https://github.com/mlorentedev/dotfiles/compare/v0.59.0...v0.60.0) (2026-09-26)


### Bug Fixes

* **mem:** a fix ([#1726](https://github.com/mlorentedev/dotfiles/issues/1726))

---
$footer"
    export SDD_PR_BODY
    _gate
    [ "$status" -eq 0 ]
}
