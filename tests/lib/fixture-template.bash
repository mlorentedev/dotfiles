#!/usr/bin/env bash
# Build a fixture once per bats file, copy it per test.  Load with: load 'lib/fixture-template'
#
# WHY THIS EXISTS
# ---------------
# Five files ran `git init`, a config and a seed commit in every setup().  On a
# Mac that is about 100 ms a test against 6 ms for a copy: 9 of the 30 seconds
# compile-harness.bats took (CI-012 N3, #2059).  Building the tree once in
# setup_file and copying it keeps each test's fixture its own, so no test can
# see what another wrote.
#
# THE RULE
# --------
#   fixture_template_build <name> <builder>   (setup_file)
#       Runs the function <builder> in a new empty directory, in a new bash with
#       -e and -u, then makes the whole tree read-only.  The builder sees its own
#       definition and exported variables, nothing else: it is a self-contained
#       recipe.  A failing builder fails the build and prints its output: a
#       half-built template would otherwise be copied into every test.
#   fixture_template_copy <name> <dest>       (setup)
#       Copies the template to <dest>, which must not exist, and makes the copy
#       writable.  A template that was never built is a failure, not an empty
#       directory.
#   fixture_template_release                  (teardown_file)
#       Gives write access back, so bats can delete BATS_FILE_TMPDIR.
#
# The template is read-only on purpose (the CI-004 rule for shared fixtures): a
# test that writes into it by mistake fails instead of changing what every test
# after it starts from.

_fixture_template_dir() { printf '%s/template-%s' "$BATS_FILE_TMPDIR" "$1"; }

fixture_template_build() {
    local dir log rc=0
    dir="$(_fixture_template_dir "$1")"
    log="$dir.log"
    mkdir "$dir" || { echo "fixture: template $1 already exists at $dir" >&2; return 1; }
    # A new bash, not a subshell. A subshell inherits "errexit ignored" from any
    # caller under `||`, `&&`, `if` or `!`, and bash 3.2 and 5 disagree on which
    # subshells do, so a builder failing mid-way could pass. A new process cannot
    # inherit it. -u too: the builder gets only its own definition, so a variable
    # the caller did not export fails here instead of expanding to nothing.
    # shellcheck disable=SC2016 # $1 and $2 are the new shell's arguments
    "$BASH" -euc "$(declare -f "$2")"'
        cd "$1"; "$2"' _ "$dir" "$2" > "$log" 2>&1 || rc=$?
    if [ "$rc" -ne 0 ]; then
        echo "fixture: building template $1 with $2 failed (exit $rc):" >&2
        cat "$log" >&2
        return 1
    fi
    chmod -R a-w "$dir"
}

fixture_template_copy() {
    local dir
    dir="$(_fixture_template_dir "$1")"
    [ -d "$dir" ] || { echo "fixture: template $1 was never built (no $dir)" >&2; return 1; }
    # cp -a into an existing directory nests the template inside it.
    [ ! -e "$2" ] || { echo "fixture: $2 already exists; the copy of template $1 needs a new path" >&2; return 1; }
    cp -a "$dir" "$2" && chmod -R u+w "$2"
}

fixture_template_release() {
    local d
    for d in "$BATS_FILE_TMPDIR"/template-*; do
        [ -d "$d" ] || continue
        chmod -R u+w "$d"
    done
}
