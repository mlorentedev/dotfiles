#!/usr/bin/env bash
# Shared case runner for the crystallize golden corpus (CLI-021 / #672).
#
# ONE runner, used by BOTH capture.sh and knowledge-crystallize-golden.bats. That
# is deliberate: if capture and verify normalised differently the goldens would be
# meaningless, and the drift would be invisible because both sides would still be
# self-consistent. Sharing the function makes the divergence unrepresentable.
#
# The shell is the oracle (see ORACLE for the pinned revisions). Nothing here may
# "improve while translating" — a bug in the twin is reproduced faithfully and
# ticketed separately, per the CLI-021 proposal's "Out of scope".

# gc_run_case CASE_DIR OUT_DIR [IMPL]
#   Builds an isolated $HOME, materialises the case's fixtures, runs IMPL
#   (default: the shell oracle), and writes normalised artefacts to OUT_DIR:
#       stdout     merged stdout+stderr, normalised
#       exit       the exit status
#       memory.md  every resulting MEMORY.md, concatenated with a header per file
gc_run_case() {
    local case_dir="$1" out_dir="$2"
    local impl="${3:-$GC_ORACLE_SH}"

    local sandbox fake_home
    sandbox=$(mktemp -d)
    # No dashes anywhere in the path: the script's decode_path() reverses its
    # encoding with `tr '-' '/'`, so a dash in the sandbox path would make the
    # project undecodable and silently change which branch the test exercises.
    case "$sandbox" in
        *-*) printf 'gc_run_case: sandbox path contains a dash: %s\n' "$sandbox" >&2; return 1 ;;
    esac
    fake_home="$sandbox/h"
    mkdir -p "$fake_home"

    # Isolation is a claim that needs its own assertion (docs/lessons.md,
    # 2026-08-09): this script family nearly wrote through a symlinked "sandbox"
    # last session. A fixture tree must never contain a link out.
    if [ -n "$(find "$case_dir" -type l 2>/dev/null)" ]; then
        printf 'gc_run_case: fixture tree contains a symlink: %s\n' "$case_dir" >&2
        return 1
    fi

    local -a projects=()
    if [ -d "$case_dir/projects" ]; then
        local p
        for p in "$case_dir"/projects/*; do
            [ -d "$p" ] || continue
            projects+=("$(basename "$p")")
            # Fixtures are named input.md, never MEMORY.md: GUARD-001 forbids an
            # agent-memory filename anywhere outside the vault, and a fixture that
            # has to be committed with --no-verify is a fixture that disables a
            # guard. The sandbox is where it becomes MEMORY.md.
            _gc_place_project "$fake_home" "$(basename "$p")" "$p/input.md"
        done
    elif [ -f "$case_dir/input.md" ]; then
        projects+=(demo)
        _gc_place_project "$fake_home" demo "$case_dir/input.md"
    else
        # No fixture at all: exercises the "no MEMORY.md found" path. The project
        # directory still has to exist, because main does `cd "$PROJECT_DIR"`.
        projects+=(demo)
        mkdir -p "$fake_home/Projects/demo"
    fi

    # Orphans: a memory directory whose encoded name decodes to nothing on disk
    # (a project deleted, or a key from another machine). --all must report it as
    # skipped rather than processed — the `processed = found - skipped` arithmetic
    # is exactly what a naive port gets wrong.
    if [ -f "$case_dir/orphans" ]; then
        local orphan
        while IFS= read -r orphan; do
            [ -n "$orphan" ] || continue
            mkdir -p "$fake_home/.claude/projects/$orphan/memory"
            printf '# Orphan memory\n' > "$fake_home/.claude/projects/$orphan/memory/MEMORY.md"
        done < "$case_dir/orphans"
    fi

    local -a args=()
    if [ -f "$case_dir/args" ]; then
        # Fixture-authored, not user input; word-splitting is the point.
        # shellcheck disable=SC2207
        args=($(cat "$case_dir/args"))
    else
        args=("$fake_home/Projects/demo")
    fi

    mkdir -p "$out_dir"

    local today rc=0
    today=$(date +%Y-%m-%d)

    # `runs` > 1 pins IDEMPOTENCE: the corpus keeps the LAST run's stdout and the
    # final file, so a second pass that duplicates a section or relocates the
    # handoff block shows up as a golden diff. (BUG-060's second seed case.)
    local runs=1
    [ -f "$case_dir/runs" ] && runs=$(cat "$case_dir/runs")

    local raw="$sandbox/raw.out"
    local n=1
    while [ "$n" -le "$runs" ]; do
        rc=0
        # `|| rc=$?` rather than relying on the caller's flags — the same lesson
        # the reconciler cost us (docs/lessons.md, 2026-08-09).
        HOME="$fake_home" bash "$impl" "${args[@]}" >"$raw" 2>&1 || rc=$?
        n=$((n + 1))
    done

    printf '%s\n' "$rc" > "$out_dir/exit"
    _gc_normalize "$fake_home" "$today" < "$raw" > "$out_dir/stdout"

    # Concatenate every resulting MEMORY.md so multi-project (--all) cases are
    # covered by the same artefact as single-project ones.
    : > "$out_dir/memory.md"
    local name mf
    for name in "${projects[@]}"; do
        mf="$fake_home/.claude/projects/$(printf '%s' "$fake_home/Projects/$name" | tr '/' '-')/memory/MEMORY.md"
        [ -f "$mf" ] || continue
        printf '===== %s =====\n' "$name" >> "$out_dir/memory.md"
        _gc_normalize "$fake_home" "$today" < "$mf" >> "$out_dir/memory.md"
    done

    rm -rf "$sandbox"
    return 0
}

# Place FIXTURE as project NAME's MEMORY.md inside FAKE_HOME, using the same path
# encoding Claude Code uses ('/' -> '-').
_gc_place_project() {
    local fake_home="$1" name="$2" fixture="$3"
    local proj="$fake_home/Projects/$name" encoded mem_dir
    mkdir -p "$proj"
    encoded=$(printf '%s' "$proj" | tr '/' '-')
    mem_dir="$fake_home/.claude/projects/$encoded/memory"
    mkdir -p "$mem_dir"
    [ -f "$fixture" ] && cp "$fixture" "$mem_dir/MEMORY.md"
    return 0
}

# Normalise the two things that legitimately vary between runs, and nothing else.
#
# The date is normalised at COMPARE time, never baked into the golden: a golden
# carrying a capture-day literal would go red at the next midnight. Fixture inputs
# deliberately carry old dates (2025-01-01) so an unchanged line cannot normalise
# to <TODAY> and mask a script that failed to write.
_gc_normalize() {
    local fake_home="$1" today="$2"
    local home_key
    home_key=$(printf '%s' "$fake_home" | tr '/' '-')
    sed -e 's/\x1b\[[0-9;]*m//g' \
        -e "s|${home_key}|<HOMEKEY>|g" \
        -e "s|${fake_home}|<HOME>|g" \
        -e "s|${today}|<TODAY>|g"
}
