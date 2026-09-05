#!/usr/bin/env bats
# GUARD (#1514): every lesson has its own number, and every lesson file is
# reachable from the index.
#
# WHAT THIS CAUGHT
# ----------------
# Three collisions reached `main` from parallel agent sessions, and all three
# were found by a human noticing, never by a check:
#
#   lesson-256  two files, in main since 2026-09-01, unnoticed for four days
#   lesson-269  two files, ninety minutes apart on 2026-09-05, fixed by #1512
#   lesson-268  two files, landed by #1515 on 2026-09-05 -- during the very
#               session that was arguing about the other two
#
# Nothing here is a race that careful authors avoid. Two sessions pick the next
# free number from the same tree at the same time; neither can see the other's
# unpushed work, and `main` accepts both because no assertion spans them. The
# third one landed in a PR that had just been triaged file-by-file by an agent
# who was looking at Go code and reviewer comments, which is the argument
# against "be more careful" as the fix.
#
# WHY FOUR ASSERTIONS, AND WHY THE FIRST TWO TURNED OUT TO BE ONE SIGNAL
# ----------------------------------------------------------------------
# Measured on `main` at 4e2e2c4: the set of duplicate-numbered files and the set
# of files missing from `_index.md` were IDENTICAL -- exactly two files, the same
# two. That is not a coincidence, it is the shape of the event: a session writes
# the file, takes a number that is already gone, and its index line either never
# arrives or loses the same race the number did.
#
# So the orphan check is not a second nicety bolted on. It catches the same
# mistake one step earlier, at the point where it is still a missing line rather
# than a lost lesson -- and a lesson absent from the index is invisible, since
# the index is the only surface anything reads.
#
# WHY IT RUNS AGAINST THE REAL TREE
# ---------------------------------
# No fixture. The bug is a property of the actual `docs/lessons/` directory, and
# a check that only ever passes on a synthetic clean tree would have passed every
# day of the four that lesson-256 sat duplicated in main.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    LESSONS_DIR="$DOTFILES_DIR/docs/lessons"
    INDEX="$LESSONS_DIR/_index.md"
}

@test "guard: no two lesson files share a number" {
    [ -d "$LESSONS_DIR" ] || skip "docs/lessons/ not present"

    # `ls | grep -oE` rather than a glob: an unmatched glob aborts the whole
    # compound command under zsh's default NOMATCH, and this repo has been bitten
    # by that exact silence before.
    dupes="$(ls "$LESSONS_DIR" | grep -oE '^lesson-[0-9]+' | sort | uniq -d)"

    if [ -n "$dupes" ]; then
        printf 'Lesson numbers used more than once:\n%s\n\n' "$dupes"
        printf 'The colliding files:\n'
        for n in $dupes; do
            ls "$LESSONS_DIR" | grep -E "^${n}-" | sed 's/^/  /'
        done
        printf '\nRenumber the LATER one to the next free number, update its\n'
        printf 'frontmatter id and its heading, and add its line to _index.md.\n'
    fi

    [ -z "$dupes" ]
}

@test "guard: every lesson file is listed in _index.md" {
    [ -d "$LESSONS_DIR" ] || skip "docs/lessons/ not present"
    [ -f "$INDEX" ] || skip "_index.md not present"

    orphans=""
    for f in "$LESSONS_DIR"/lesson-*.md; do
        [ -e "$f" ] || continue
        base="$(basename "$f")"
        if ! grep -q "$base" "$INDEX"; then
            orphans="$orphans$base"$'\n'
        fi
    done

    if [ -n "$orphans" ]; then
        printf 'Lesson files missing from _index.md:\n%s\n' "$orphans"
        printf 'The index is the only surface anything reads, so a lesson absent\n'
        printf 'from it is a lesson nobody will find.\n'
    fi

    [ -z "$orphans" ]
}

@test "guard: every _index.md lesson link points at a file that exists" {
    [ -f "$INDEX" ] || skip "_index.md not present"

    missing=""
    # Link targets only: the table also carries titles containing brackets.
    while IFS= read -r target; do
        [ -n "$target" ] || continue
        if [ ! -f "$LESSONS_DIR/$target" ]; then
            missing="$missing$target"$'\n'
        fi
    done < <(grep -oE '\(lesson-[^)]+\.md\)' "$INDEX" | tr -d '()')

    if [ -n "$missing" ]; then
        printf 'Index links with no file behind them:\n%s\n' "$missing"
        printf 'A renumbering that moved the file without moving its link leaves\n'
        printf 'the index pointing at nothing, which reads as "no such lesson".\n'
    fi

    [ -z "$missing" ]
}

# Added because fixing the collision above broke one: lesson-264 carried
# [[lesson-256-hooks-reload-...]], and renumbering that file to 271 left the
# wikilink pointing at nothing. The repair for a collision is a rename, so
# dangling wikilinks are not a separate hazard -- they are the predictable
# second-order effect of the first three assertions being acted on.
@test "guard: every lesson wikilink resolves to a lesson file" {
    [ -d "$LESSONS_DIR" ] || skip "docs/lessons/ not present"

    # TWO conventions are in use and both are legitimate: [[lesson-212]] names a
    # lesson by number, [[lesson-268-full-slug]] names the file. Asserting only
    # the second reported the first as dangling -- a guard calling correct state
    # broken, which is the same defect as one calling broken state correct.
    # Found by running against the real tree; a fixture carrying one convention
    # would have passed.
    dangling=""
    while IFS= read -r target; do
        [ -n "$target" ] || continue
        if [ -f "$LESSONS_DIR/$target.md" ]; then
            continue                                  # full-slug form
        fi
        if ls "$LESSONS_DIR" | grep -qE "^${target}-"; then
            continue                                  # number-only form
        fi
        dangling="$dangling$target"$'\n'
    done < <(grep -rhoE '\[\[lesson-[^]]+\]\]' "$LESSONS_DIR" 2>/dev/null | tr -d '[]' | sort -u)

    if [ -n "$dangling" ]; then
        printf 'Wikilinks pointing at no lesson file:\n%s\n' "$dangling"
        printf 'Most likely a renumbering that moved the file and left its inbound\n'
        printf 'links behind. Grep for the old slug before renaming, not after.\n'
    fi

    [ -z "$dangling" ]
}
