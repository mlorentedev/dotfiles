#!/usr/bin/env bats
# SDD-038: the archive-on-merge half of the Discipline Gate.
#
# A PR that CLOSES an issue must archive the active spec tracking it. The SDD
# lifecycle's terminal step was dropped systematically under delivery speed, and
# every un-archived spec widens the SPEC_FLOOR alibi bypass: a large PR can touch
# ten lines of an unrelated shipped spec and satisfy the gate.
#
# The check is keyed on GitHub's closing keywords, never on "a spec was touched".
# `Refs #N` legitimately leaves a spec active when the issue's work continues.

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    GATE="$SCRIPTS_DIR/check-spec-gate.sh"
    REPO_FIXTURE="$(mktemp -d "/tmp/bats_archivegate_XXXXXX")"
    cd "$REPO_FIXTURE" || exit 1
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    echo "seed" > seed.txt
    git add seed.txt
    git commit -q -m "seed"
    # A real origin, so the cross-repo discrimination is exercised rather than
    # sidestepped: qualified references are only ours if the slug matches.
    git remote add origin https://github.com/mlorentedev/dotfiles.git
    # Every test starts with a clean env: a leaked label or body from the shell
    # would silently change which branch of the gate is exercised.
    unset SDD_LABELS SDD_PR_BODY SDD_PR_AUTHOR
}

teardown() {
    cd / || true
    [ -z "${REPO_FIXTURE:-}" ] || rm -rf "$REPO_FIXTURE"
}

# An active spec on main tracking $2 (a raw `issue:` frontmatter value), so the
# base ref carries the issue -> spec linkage the gate resolves.
seed_active_spec() {
    local id="$1" issue_value="$2"
    mkdir -p "specs/$id"
    cat > "specs/$id/proposal.md" <<EOF
---
id: "$id"
type: spec
status: implementing
issue: $issue_value
---

# $id
EOF
    git add -A
    git commit -q -m "add spec $id"
}

# Move it to specs/archive/, i.e. what \`dotf spec archive\` does.
archive_spec() {
    local id="$1"
    mkdir -p specs/archive
    git mv "specs/$id" "specs/archive/$id"
    git add -A
    git commit -q -m "archive $id"
}

start_feature() { git checkout -q -b feature; }

# A change small enough that the LOC gate itself never fires, so a failure can
# only come from the archive check.
tiny_change() {
    echo "tweak" >> seed.txt
    git add -A
    git commit -q -m "tiny change"
}

run_gate() { run "$GATE" --base-ref main --head-ref feature; }

@test "AC1: closing an issue whose active spec is not archived fails the gate" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-001-demo"* ]]
    # The message has to be actionable, not just a refusal.
    [[ "$output" == *"dotf spec archive"* ]]
}

@test "AC2: the same PR passes once the spec folder is archived" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    archive_spec FOO-001-demo
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC3: 'Refs #N' does not trigger the check" {
    # PR #765 is the real case: it says `Refs #748` because the issue's remaining
    # work moved to a follow-up, so its spec must stay active.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Refs #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC3: an issue number mentioned in prose does not trigger the check" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="This is unrelated to #123, see also #999."

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC3: 'Part of #N' does not trigger the check" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Part of #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC4: the 'repo#N' frontmatter shape is matched" {
    # Three shapes exist in-tree today; all must resolve.
    seed_active_spec FOO-002-demo '"dotfiles#161"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Fixes #161"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-002-demo"* ]]
}

@test "AC4: a bare unquoted numeric frontmatter value is matched" {
    seed_active_spec FOO-003-demo '479'
    start_feature
    tiny_change
    export SDD_PR_BODY="Resolves #479"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-003-demo"* ]]
}

@test "AC4: a full GitHub issue URL in the closing reference is matched" {
    seed_active_spec FOO-004-demo '"mlorentedev/dotfiles#321"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes https://github.com/mlorentedev/dotfiles/issues/321"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-004-demo"* ]]
}

@test "AC4: closing keywords are matched case-insensitively" {
    seed_active_spec FOO-005-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="CLOSES #123"

    run_gate
    [ "$status" -eq 1 ]
}

@test "AC5: a closing reference to another repo is ignored" {
    # `Closes owner/other#123` must not archive THIS repo's spec 123.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes otherowner/otherrepo#123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC6: the check fires below the LOC threshold" {
    # A three-line PR can still end an issue's life, so the archive check must
    # not sit behind the TOTAL_LOC < THRESHOLD early exit.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 1 ]
}

@test "AC6: skip-sdd does not skip the archive check" {
    # skip-sdd asserts "this change needs no spec", which says nothing about
    # whether an existing spec's work is finished.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_LABELS="skip-sdd"
    export SDD_PR_BODY=$'Closes #123\n\n## SDD skip rationale\n\nmechanical rename'

    run_gate
    [ "$status" -eq 1 ]
}

@test "AC7: skip-archive with a non-empty rationale passes" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_LABELS="skip-archive"
    export SDD_PR_BODY=$'Closes #123\n\n## Archive skip rationale\n\nfollow-up work continues under #124'

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC7: skip-archive without a rationale fails" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change
    export SDD_LABELS="skip-archive"
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Archive skip rationale"* ]]
}

@test "AC8: an empty PR body is a clean pass" {
    # Local pre-push runs have no PR body; the gate must not fail on absence.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    tiny_change

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC8: closing an issue with no matching active spec is a clean pass" {
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes #999"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC8: a spec with no issue frontmatter is not enforced" {
    # 28 of 44 active specs predate the issue: field. A gate that failed on data
    # it cannot link would block PRs for a problem the author did not create.
    mkdir -p specs/FOO-006-nolink
    printf -- '---\nid: "FOO-006-nolink"\ntype: spec\n---\n\n# FOO-006\n' \
        > specs/FOO-006-nolink/proposal.md
    git add -A
    git commit -q -m "spec without issue link"
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC8: an already-archived spec does not re-trigger the check" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    archive_spec FOO-001-demo
    start_feature
    tiny_change
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "AC1: a spec created and closed in the same PR must still be archived" {
    # Reading only the base ref would miss this: the spec does not exist on main
    # yet. It is exactly the "created, shipped, never archived" pattern #670
    # exists to stop, so the head ref is checked too.
    start_feature
    seed_active_spec FOO-008-samepr '"mlorentedev/dotfiles#301"'
    export SDD_PR_BODY="Closes #301"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-008-samepr"* ]]
}

@test "AC1: multiple closing references are all checked" {
    seed_active_spec FOO-007-a '"mlorentedev/dotfiles#201"'
    seed_active_spec FOO-007-b '"mlorentedev/dotfiles#202"'
    start_feature
    archive_spec FOO-007-a
    export SDD_PR_BODY=$'Closes #201\nCloses #202'

    # The first is archived, the second is not -> still a violation.
    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"FOO-007-b"* ]]
    [[ "$output" != *"FOO-007-a"* ]]
}

# BUG-050: the two halves of the gate were mutually unsatisfiable for a PR that
# is over the LOC threshold AND closes its own issue. Archiving satisfied
# archive-on-merge and broke the Discipline Gate (specs/archive/* is deliberately
# not an active-spec touch, #397); not archiving did the reverse. The only ways
# through were the escape hatches, so the normal path had become the escaped one.

# A production change over the 50 LOC threshold, so BOTH halves are live at once
# — the only configuration in which they conflict.
big_change() {
    seq 1 60 > payload.sh
    git add -A
    git commit -q -m "big change"
}

@test "BUG-050: a large PR that archives its own spec satisfies both halves" {
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    big_change
    archive_spec FOO-001-demo
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 0 ]
    [[ "$output" == *"spec folder touched"* ]]
}

@test "BUG-050: an archive move alone is far under SPEC_FLOOR, so it cannot count by LOC" {
    # Why the mandated archive sets the touch outright instead of feeding
    # SPEC_LOC: a real archive renders as ~4 LOC against a floor of 10, so
    # counting its lines would leave the gate exactly as unsatisfiable.
    seed_active_spec FOO-001-demo '"mlorentedev/dotfiles#123"'
    start_feature
    big_change
    archive_spec FOO-001-demo
    export SDD_PR_BODY="Closes #123"

    run "$GATE" --base-ref main --head-ref feature --explain
    [ "$status" -eq 0 ]
    [[ "$output" == *"Active-spec LOC (added+removed): 0"* ]]
    [[ "$output" == *"Specs archived for archive-on-merge"* ]]
    [[ "$output" == *"FOO-001-demo"* ]]
}

@test "BUG-050: a gratuitous archive-move earns no spec touch (#397 intact)" {
    # The red direction. Credit is reachable only through the `issue:`
    # frontmatter of a spec whose issue THIS PR closes — archiving something
    # unrelated must remain worthless, or the fix reopens the bypass #397 shut.
    seed_active_spec BAR-002-other '"mlorentedev/dotfiles#456"'
    start_feature
    big_change
    archive_spec BAR-002-other
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}

@test "BUG-050: a spec created and archived in the same PR counts" {
    # Invisible to the ACTIVE map at base (it does not exist yet) and at head
    # (it is already archived), so the linkage can only come from the archived
    # tree at head.
    start_feature
    big_change
    seed_active_spec FOO-003-new '"mlorentedev/dotfiles#123"'
    archive_spec FOO-003-new
    export SDD_PR_BODY="Closes #123"

    run_gate
    [ "$status" -eq 0 ]
}

@test "BUG-050: closing nothing leaves an archive move worthless as before" {
    # No closing keyword means no mandate, so the archived path must not become
    # a free spec touch for any large PR that happens to move a spec.
    seed_active_spec BAR-002-other '"mlorentedev/dotfiles#456"'
    start_feature
    big_change
    archive_spec BAR-002-other
    export SDD_PR_BODY="Refs #456"

    run_gate
    [ "$status" -eq 1 ]
    [[ "$output" == *"Discipline Gate"* ]]
}
