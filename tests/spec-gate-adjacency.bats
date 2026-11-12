#!/usr/bin/env bats
# Tests for the advisory adjacency report in scripts/check-spec-gate.sh
# (HARNESS-063). The report names OTHER open issues that reference a file this
# PR changes — the signal that was missing when #851 merged while #849, which
# described the shape the fix did not cover, sat open on the board.

load 'lib/both-shells'
load 'lib/dotf-bin'
load 'lib/spec-gate-parity'
load 'lib/refute'

setup() {
    SCRIPTS_DIR="$BATS_TEST_DIRNAME/../scripts"
    dotf_bin_resolve "$BATS_FILE_TMPDIR/dotf" || return 1
    REPO_FIXTURE="/tmp/bats_adjacency_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$REPO_FIXTURE"
    cd "$REPO_FIXTURE" || exit 1
    git init -q -b main
    git config user.email test@test
    git config user.name test
    git config commit.gpgsign false
    echo "seed" > seed.txt
    git add seed.txt
    git commit -q -m "seed"
    git checkout -q -b feature
}

teardown() {
    cd / || true
    rm -rf "$REPO_FIXTURE"
}

_commit() {
    git add -A
    git commit -q -m "${1:-change}"
}

# The worked example: PR #851's two changed files.
_replay_851_diff() {
    mkdir -p scripts
    printf 'line %d\n' {1..30} > scripts/knowledge-crystallize.sh
    printf 'line %d\n' {1..30} > scripts/knowledge-crystallize.ps1
    _commit "replay of PR 851"
}

# The adjacency feed is a TSV of "<number><TAB><title + body, newlines flattened>",
# produced by the workflow so the script itself stays offline. This fixture keeps
# #849's REAL formatting: the full path appears only inside an inline code span,
# and the bare basename only in the title.
_write_849_feed() {
    printf '%s\t%s\n' 849 \
'BUG-059: knowledge-crystallize.sh cannot stamp a YAML block-scalar MEMORY.md and would corrupt it `scripts/knowledge-crystallize.sh` cannot stamp any `MEMORY.md` that stores its content inside a YAML `content: |` block scalar.' \
        > "$REPO_FIXTURE/adjacency.tsv"
}

@test "adjacency: replaying PR 851 against an open #849 flags it" {
    _replay_851_diff
    _write_849_feed
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature \
            --adjacency-issues "$REPO_FIXTURE/adjacency.tsv"
    [[ "$output" == *"849"* ]] || false
    [[ "$output" == *"knowledge-crystallize"* ]] || false
}

@test "adjacency: matches a path that appears only inside an inline code span" {
    # Guards the polarity trap: _strip_markdown_code() exists so the CLOSING-ref
    # matcher ignores code spans. Reusing it here would delete the only mention
    # of the path in this issue's body and silently kill the whole check.
    _replay_851_diff
    printf '%s\t%s\n' 777 \
        'An unrelated-looking title with no filename at all, but the body cites `scripts/knowledge-crystallize.ps1` in a code span.' \
        > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" == *"777"* ]] || false
}

@test "adjacency: an issue naming no changed file is not reported" {
    _replay_851_diff
    printf '%s\t%s\n' 900 'OPS-001: rotate the age key on the Windows box, see sensitive/env-mapping.conf' > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"900"* ]] || false
}

@test "adjacency: the issues this PR closes are excluded from its own report" {
    _replay_851_diff
    printf '%s\t%s\n' 850 'BUG-060: crystallize appends past the handoff block in scripts/knowledge-crystallize.sh' > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"#850"* ]] || false
}

@test "adjacency: spec-template and doc filenames do not drag in the backlog" {
    # Measured against the live backlog: matching the whole diff reported 37
    # issues, 3 with signal. The other 34 matched `proposal.md`, `tasks.md` or
    # `docs/lessons.md` — files nearly every PR touches. Only files this gate
    # already counts as production are matched.
    mkdir -p specs/SDD-999-test docs scripts
    printf 'proposal line %d\n' {1..12} > specs/SDD-999-test/proposal.md
    printf 'lesson line %d\n' {1..12} > docs/lessons.md
    printf 'line %d\n' {1..60} > scripts/real-change.sh
    _commit "spec + lesson + one production file"
    {
        printf '%s\t%s\n' 111 'SDD-039: spec-tree hygiene — decide the scaffold-only specs, see proposal.md'
        printf '%s\t%s\n' 222 'HARNESS-024: enforce lessons capture in docs/lessons.md at handoff'
        printf '%s\t%s\n' 333 'BUG-999: scripts/real-change.sh mishandles the second input shape'
    } > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"111"* ]] || false
    [[ "$output" != *"222"* ]] || false
    [[ "$output" == *"333"* ]] || false
}

@test "adjacency: the report is advisory and cannot change the exit status" {
    _replay_851_diff
    _write_849_feed
    SDD_PR_BODY="Closes #850" \
        spec_gate_both --base-ref main --head-ref feature
    local without_status="$status"
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature \
            --adjacency-issues "$REPO_FIXTURE/adjacency.tsv"
    [ "$status" -eq "$without_status" ]
}

@test "adjacency: a missing feed file degrades to silence, never to an error" {
    # The workflow writes the feed only when a token is present. No token means
    # no file, and the gate must behave exactly as it did before this change.
    _replay_851_diff
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues /nonexistent/feed.tsv
    [ "$status" -ne 2 ]
    [[ "$output" != *"Adjacent open issues"* ]] || false
}

@test "adjacency: with no flag the output matches the stored golden" {
    # Characterization test (#672): adding --adjacency-issues support did not
    # silently change the script's existing offline output when the flag goes
    # unused. The golden was captured from the pre-change script on this exact
    # fixture; it is a file, not origin/main, so the test cannot skip itself in
    # a shallow clone and does not move when main does (#2303). The documented
    # SDD_PR_BODY escape hatch (#854/BUG-061) is stripped from the live output:
    # it is an intentional, in-scope addition to this path, not a regression.
    _replay_851_diff
    SDD_PR_BODY="Closes #850" \
        spec_gate_both --base-ref main --head-ref feature --explain
    local stripped
    stripped=$(printf '%s\n' "$output" \
        | sed '/^       If this already archived a spec/,/^       be set by hand for a one-off check/d' | cat -s)
    [ "$stripped" = "$(cat "$BATS_TEST_DIRNAME/fixtures/spec-gate/no-adjacency-flag.golden")" ]
}

@test "adjacency: a generic basename in a subdirectory does not drag in the backlog" {
    mkdir -p build
    printf 'line %d\n' {1..60} > build/Makefile
    _commit "nested Makefile"
    printf '%s\t%s\n' 444 'DX-001: the Makefile is slow to start' > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"444"* ]] || false
}

@test "adjacency: a basename inside a longer word is not a match" {
    # `scripts/config` must not flag an issue that only says "configuration".
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/config
    _commit "config"
    {
        printf '%s\t%s\n' 555 'OPS-002: the configuration of the runner is wrong'
        printf '%s\t%s\n' 556 'OPS-003: the config file is read twice'
    } > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"555"* ]] || false
    [[ "$output" == *"556"* ]] || false
}

@test "adjacency: a dot in a basename matches only a dot" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/a.b.sh
    _commit "dotted name"
    printf '%s\t%s\n' 666 'BUG-1: aXb.sh fails' > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" != *"666"* ]] || false
}

@test "adjacency: an issue citing the source path of a renamed file is reported" {
    # HARNESS-178: a rename lists the file under both names, so an issue that
    # still cites the old path surfaces.
    git checkout -q main
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/old-name.sh
    _commit "old name on main"
    git checkout -q feature
    git reset -q --hard main
    git mv scripts/old-name.sh scripts/new-name.sh
    _commit "rename"
    printf '%s\t%s\n' 777 'BUG-2: scripts/old-name.sh drops the last line' > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" == *"777"* ]] || false
}

@test "adjacency: a percent sign in a path is escaped in the CI annotation" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > 'scripts/100%.sh'
    _commit "percent"
    printf '%s\t%s\n' 888 'BUG-3: scripts/100%.sh breaks' > feed.tsv
    GITHUB_ACTIONS=true SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" == *"::warning title=Adjacent open issues::#888 (scripts/100%25.sh)"* ]] || false
}

@test "adjacency: a pipe in a path is escaped in the step-summary table" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > 'scripts/a|b.sh'
    _commit "pipe"
    printf '%s\t%s\n' 999 'BUG-4: scripts/a|b.sh breaks' > feed.tsv
    GITHUB_STEP_SUMMARY="$REPO_FIXTURE/summary.md" SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    grep -qF '| #999 | `scripts/a\|b.sh` |' "$REPO_FIXTURE/summary.md"
}

# Issue titles are writable by anyone who can open an issue (#3389). The feed is
# line-based, so a newline cannot pass, but a carriage return can: it would end
# a workflow command early and start another on the runner's side.
@test "adjacency: control characters in an issue title reach no output" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/tool.sh
    _commit "tool"
    printf '%s\t%s\n' 901 $'BUG-5: scripts/tool.sh breaks\r::error::injected\033[31m' > feed.tsv
    GITHUB_ACTIONS=true GITHUB_STEP_SUMMARY="$REPO_FIXTURE/summary.md" SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" == *"#901"* ]] || false
    [[ "$output" != *$'\r'* ]] || false
    [[ "$output" != *$'\033'* ]] || false
    refute_grep $'[\r\033]' "$REPO_FIXTURE/summary.md"
    grep -qF 'breaks ::error::injected [31m' "$REPO_FIXTURE/summary.md"
}

# The runner trims leading whitespace before it reads a workflow command, so the
# report's indentation does not protect a title that starts with one (#3389).
@test "adjacency: workflow-command syntax in an issue title is broken on stdout" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/tool.sh
    _commit "tool"
    {
        printf '%s\t%s\n' 903 '::error::scripts/tool.sh broke'
        printf '%s\t%s\n' 904 'BUG-7: scripts/tool.sh ##[error]legacy'
    } > feed.tsv
    SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    [[ "$output" == *"#903"* && "$output" == *"#904"* ]] || false
    printf '%s\n' "$output" > out.txt
    refute_grep '^[[:space:]]*::' out.txt
    refute_grep_fixed '##[' out.txt
}

# Escaping lengthens the cell, so clipping after it could cut an entity in half
# and the page would show "&l". Clip the title first, then escape it.
@test "adjacency: a summary cell is clipped before it is escaped" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/tool.sh
    _commit "tool"
    printf '%s\t%s\n' 905 "scripts/tool.sh $(printf 'x%.0s' {1..70})<b>tail" > feed.tsv
    GITHUB_STEP_SUMMARY="$REPO_FIXTURE/summary.md" SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    grep -qE 'x&lt;b \|$' "$REPO_FIXTURE/summary.md"
}

@test "adjacency: angle brackets in an issue title are escaped in the step summary" {
    mkdir -p scripts
    printf 'line %d\n' {1..60} > scripts/tool.sh
    _commit "tool"
    printf '%s\t%s\n' 902 'BUG-6: scripts/tool.sh <img src=x onerror=alert(1)> breaks' > feed.tsv
    GITHUB_STEP_SUMMARY="$REPO_FIXTURE/summary.md" SDD_PR_BODY="Closes #850" \
        spec_gate_both \
            --base-ref main --head-ref feature --adjacency-issues feed.tsv
    grep -qF '&lt;img src=x onerror=alert(1)&gt; breaks' "$REPO_FIXTURE/summary.md"
    refute_grep_fixed '<img' "$REPO_FIXTURE/summary.md"
}

@test "adjacency: --help documents the flag" {
    spec_gate_both --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"--adjacency-issues"* ]] || false
}
