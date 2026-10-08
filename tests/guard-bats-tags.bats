#!/usr/bin/env bats
# Guard for the bats tag comments that select the OS-sensitive tier (PLAT-001d).
#
# bats reads a tag only from an exact comment form, and ignores a near miss
# without a word: `# bats test_tag=...`, `# bats tests_tags=...`, a tag list with
# a space in it. A mistyped tag therefore does not fail, it quietly leaves a test
# out of the tier it was meant for, and the macOS leg passes green on a smaller
# suite than anyone believes it runs. This is the silent-failure shape the repo
# already guards in versions.conf (the `# mise: cli` marker) and in the @test
# name lint. Two things are asserted: every tag comment is spelled exactly, and
# the tier is not empty.
#
# bats scans every file it runs for these comments, so a test file must never
# contain one in a heredoc or a string; fixtures assemble the line instead.

KNOWN_TAGS="os-sensitive"

setup() {
    TESTS="$BATS_TEST_DIRNAME"
}

# Every comment that is, or is a near miss of, a bats tag line, in every suite
# but this one, as file:line:text.
tag_comments() {
    local f
    for f in "$TESTS"/*.bats; do
        [ "$f" -ef "$BATS_TEST_FILENAME" ] && continue
        awk -v f="$f" '
            /^[[:space:]]*#[[:space:]]*bats[_[:space:]]*[A-Za-z_]*[Tt][Aa][Gg]/ { printf "%s:%d:%s\n", f, NR, $0 }
        ' "$f"
    done | sort -u
}

@test "the guard sees the tag comments, so a clean result means something" {
    run tag_comments
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}

@test "every bats tag comment is spelled exactly, with only known tags" {
    local bad="" line text list tag
    while IFS= read -r line; do
        [ -n "$line" ] || continue
        text="${line#*:*:}"
        case "$text" in
            "# bats file_tags="*|"# bats test_tags="*) ;;
            *) bad="$bad$line (not an exact file_tags/test_tags comment)"$'\n'; continue ;;
        esac
        list="${text#*=}"
        case "$list" in
            ''|*[[:space:]]*|,*|*,|*,,*) bad="$bad$line (empty, spaced or malformed tag list)"$'\n'; continue ;;
        esac
        for tag in $(printf '%s' "$list" | tr ',' ' '); do
            case " $KNOWN_TAGS " in
                *" $tag "*) ;;
                *) bad="$bad$line (unknown tag '$tag'; add it to KNOWN_TAGS here on purpose)"$'\n' ;;
            esac
        done
    done < <(tag_comments)
    [ -z "$bad" ] || { printf '%s' "$bad" >&2; return 1; }
}

@test "a test_tags comment sits directly above the @test it tags" {
    local f bad=""
    for f in "$TESTS"/*.bats; do
        [ "$f" -ef "$BATS_TEST_FILENAME" ] && continue
        bad="$bad$(awk -v f="$f" '
            tagged && $0 !~ /^@test / { printf "%s:%d: test_tags is not followed by an @test\n", f, NR }
            { tagged = ($0 ~ /^# bats test_tags=/) }
        ' "$f")"
    done
    [ -z "$bad" ] || { printf '%s\n' "$bad" >&2; return 1; }
}

@test "the os-sensitive tier selects tests, and bats agrees with the comments" {
    run bats --count --filter-tags os-sensitive "$TESTS"/*.bats
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+$ ]] || false
    [ "$output" -ge 100 ]
}
