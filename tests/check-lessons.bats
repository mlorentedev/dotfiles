#!/usr/bin/env bats
# scripts/check-lessons.sh against a fixture index larger than a pipe buffer.
#
# The per-file membership test was `printf '%s\n' "$targets" | grep -qxF` under
# `set -o pipefail`. grep -q exits at its first match while printf may still be
# writing; printf then dies of SIGPIPE, the pipeline reports 141, and a lesson
# that IS indexed is reported missing. On the real index (about 20 KB) that
# failed 3 runs in 40, naming a different, correctly indexed lesson each time.
# Past 64 KB, printf cannot finish before grep exits, so this case fails on
# every run of the old code rather than on a scheduling accident.

setup() {
    FIX="$BATS_TEST_TMPDIR/repo"
    mkdir -p "$FIX/scripts" "$FIX/docs/lessons"
    cp "$BATS_TEST_DIRNAME/../scripts/check-lessons.sh" "$FIX/scripts/"
}

@test "check-lessons: an indexed lesson is found in an index larger than a pipe buffer" {
    local pad i name
    pad=$(printf 'x%.0s' {1..200})
    {
        echo '# Lessons'
        for i in $(seq -w 1 400); do
            name="lesson-$i-$pad.md"
            : > "$FIX/docs/lessons/$name"
            printf '| [%s](%s) | 2026-09-25 |  |\n' "$i" "$name"
        done
    } > "$FIX/docs/lessons/_index.md"
    [ "$(wc -c < "$FIX/docs/lessons/_index.md")" -gt 65536 ]

    run "$FIX/scripts/check-lessons.sh"
    [ "$status" -eq 0 ]
    [[ "$output" == *"OK (400 lessons)"* ]]
}

@test "check-lessons: a lesson missing from the index is still reported" {
    echo '# Lessons' > "$FIX/docs/lessons/_index.md"
    : > "$FIX/docs/lessons/lesson-001-orphan.md"

    run "$FIX/scripts/check-lessons.sh"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not in docs/lessons/_index.md: lesson-001-orphan.md"* ]]
}
