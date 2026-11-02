#!/usr/bin/env bats
# test-windows runs only the `windows` tier of the bats files its step names
# (#2059 N1). Most tests in those files grep or jq the checkout, and Linux
# `test` runs them already; on Windows they cost ~0.17 s each in Git Bash and
# proved nothing new.
#
# A filter fails quietly: an untagged test that runs pwsh leaves the Windows
# job without a word, and a filter that matches nothing passes green on zero
# tests (lesson 309). This file holds both ends:
# - the step filters on the tag, over a non-empty list of files;
# - in those files, every test that runs pwsh carries the tag;
# - a file that tags a test is in the list, or its tagged tests never run there;
# - bats selects at least the tests the comments tag.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CI="$REPO/.github/workflows/ci.yml"
    STEP=$(awk '/name: Run PowerShell bats subset/,/^$/' "$CI" | grep -vE '^[[:space:]]*#')
    FILES=$(printf '%s\n' "$STEP" | grep -oE 'tests/[A-Za-z0-9_.-]+\.bats' | sort -u)
}

# Tests in a file that run pwsh without the tag, as file:line: name. A test
# runs pwsh when its body calls require_pwsh (tests/lib/pwsh.bash, the gate
# every such test passes) or invokes pwsh with a flag. A bare word "pwsh" in a
# grep pattern is not an invocation.
untagged_pwsh_tests() {
    awk -v f="$1" '
        /^@test / { name = $0; line = NR; tagged = (prev ~ /^# bats test_tags=([^ ]*,)?windows(,[^ ]*)?$/); body = 1; hit = 0 }
        body && /(require_pwsh|(^|[[:space:]])pwsh[[:space:]]+-)/ { hit = 1 }
        body && /^}/ { if (hit && !tagged) printf "%s:%d: %s\n", f, line, name; body = 0 }
        { prev = $0 }
    ' "$1"
}

@test "ci: the Windows bats step runs only the windows tier, over the files it names" {
    [ -n "$STEP" ] || { echo "no 'Run PowerShell bats subset' step in ci.yml" >&2; return 1; }
    printf '%s\n' "$STEP" | grep -qE 'bats" --filter-tags windows \\$'
    [ "$(printf '%s\n' "$FILES" | grep -c .)" -ge 1 ]
}

@test "ci: every test that runs pwsh in a Windows-step file is tagged windows" {
    local f bad=""
    for f in $FILES; do
        [ -f "$REPO/$f" ] || { bad="$bad$f: named by the step but missing"$'\n'; continue; }
        bad="$bad$(untagged_pwsh_tests "$REPO/$f")"
    done
    [ -z "$bad" ] || { printf 'untagged, so test-windows would not run them:\n%s\n' "$bad" >&2; return 1; }
}

@test "ci: a file that tags a test windows is one the Windows step runs" {
    local f rel bad=""
    for f in "$REPO"/tests/*.bats; do
        grep -qE '^# bats test_tags=([^ ]*,)?windows(,[^ ]*)?$' "$f" || continue
        rel="tests/${f##*/}"
        printf '%s\n' "$FILES" | grep -qxF "$rel" || bad="$bad$rel"$'\n'
    done
    [ -z "$bad" ] || { printf 'tagged windows but not in the step, so never run on Windows:\n%s' "$bad" >&2; return 1; }
}

@test "ci: bats selects every test the windows tags mark, and not zero" {
    local f tagged=0 args=()
    for f in $FILES; do
        args+=("$REPO/$f")
        tagged=$((tagged + $(grep -cE '^# bats test_tags=([^ ]*,)?windows(,[^ ]*)?$' "$REPO/$f")))
    done
    [ "$tagged" -ge 1 ]
    run bats --count --filter-tags windows "${args[@]}"
    [ "$status" -eq 0 ]
    [ "$output" -eq "$tagged" ] || { echo "bats selects $output, the comments tag $tagged" >&2; return 1; }
}
