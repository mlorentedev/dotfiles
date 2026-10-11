#!/usr/bin/env bats
# macOS ships bash 3.2 as /bin/bash, and ADR-003 holds every shell script to it.
# compile-harness.sh used mapfile (bash 4): on a Mac its refresh, deploy and
# check all died at the first one, setup downgraded that to a warning, and no
# skill reached any agent (PLAT-001 W3, #2013). Linux CI runs bash 5, so nothing
# there could see it. This guard is static, so it runs on every OS: it fails on a
# bash-4-only construct in any POSIX shell file that bash executes.

# bats file_tags=os-sensitive

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# Not in bash 3.2: mapfile/readarray, associative arrays, namerefs and declare -g
# (4.2), case-changing and @-operator expansions (4.4), [[ -v ]] (4.2), the |& and
# &>> redirections, coproc, and the ;;& / ;& case fallthroughs (a ;& only at the
# end of a case arm, so a bracket expression like [^|;&] is not one).
BASH4_ONLY='(^|[^[:alnum:]_])(mapfile|readarray|coproc)([^[:alnum:]_]|$)|(declare|local|typeset)[[:space:]]+-[[:alpha:]]*[Ang]|\$\{[[:alpha:]_][[:alnum:]_]*(\[[^]]*\])?(,,?|\^\^?|@[[:alpha:]])\}|\[\[[[:space:]]+-v[[:space:]]|&>>|\|&|;;&|[^;[];&[[:space:]]*$'

# Every shell file bash runs: the scripts dir, the root bootstrap scripts, the
# global git hooks and their library, the bash rc file, and the test suite itself
# (bats sources every .bats file with the bash on PATH). That is 3.2 on a Mac
# only with /bin first: the catalog installs brew's bash 5 ahead of it (#2202),
# so CI's macOS leg runs `PATH="/bin:$PATH" ./scripts/run-bats.sh --expect-bash 3`,
# and the same line reproduces it locally. .zsh/ is
# zsh-only, and this file is skipped because its pattern table names every
# construct on purpose.
bash_files() {
    local f
    for f in "$DOTFILES_DIR"/scripts/*.sh "$DOTFILES_DIR"/setup-linux.sh \
             "$DOTFILES_DIR"/install.sh "$DOTFILES_DIR"/.bashrc \
             "$DOTFILES_DIR"/git-hooks/* "$DOTFILES_DIR"/git-hooks/lib/*.sh \
             "$DOTFILES_DIR"/tests/*.bats "$DOTFILES_DIR"/tests/*.bash; do
        [[ -f "$f" ]] || continue
        [[ "$f" -ef "$BATS_TEST_FILENAME" ]] && continue
        case "$(head -n 1 "$f")" in
            '#!'*zsh*) continue ;;
        esac
        printf '%s\n' "$f"
    done
}

# Prints file:line: text for each code line (comments stripped) that matches.
offenders() {
    local f
    while IFS= read -r f; do
        sed 's/[[:space:]]#.*$//; s/^[[:space:]]*#.*$//' "$f" |
            grep -nE "$BASH4_ONLY" | sed "s|^|${f#"$DOTFILES_DIR"/}:|"
    done
}

@test "the guard scans files, so a clean result means something" {
    run bash_files
    [[ $status -eq 0 ]] || false
    [[ $(printf '%s\n' "$output" | grep -c 'compile-harness.sh') -eq 1 ]] || false
    [[ $(printf '%s\n' "$output" | grep -c 'tests/compile-harness.bats') -eq 1 ]] || false
    [[ $(printf '%s\n' "$output" | grep -c 'bash32-portable.bats') -eq 0 ]] || false
    [[ $(printf '%s\n' "$output" | wc -l) -gt 20 ]] || false
}

@test "the pattern catches each bash-4-only construct and spares bash 3.2 code" {
    for bad in 'mapfile -t ids < <(jq .)' 'readarray x' 'declare -A map' \
               'local -n ref=$1' 'echo ${name,,}' 'echo ${name^^}' 'cmd &>> log' \
               'cmd |& tee' 'coproc foo' 'declare -g VAR=x' 'local -A eol_of text_of' \
               '[[ -v name ]]' 'echo ${name@Q}' '    a) echo x ;&'; do
        printf '%s\n' "$bad" | grep -qE "$BASH4_ONLY" || { echo "missed: $bad"; return 1; }
    done
    for ok in 'read_array ids < <(jq .)' 'local -a list' 'echo ${name}' \
              'cmd >>log 2>&1' 'x="mapfiles"' 'declare -r CONST=1' \
              "grep -E 'install[^|;&]*age'" 'echo "${name:-x}"' 'a) echo x ;;'; do
        if printf '%s\n' "$ok" | grep -qE "$BASH4_ONLY"; then echo "false positive: $ok"; return 1; fi
    done
}

@test "no shell file bash runs uses a construct bash 3.2 lacks (ADR-003)" {
    run offenders < <(bash_files)
    if [[ -n "$output" ]]; then
        printf 'bash-4-only constructs (macOS /bin/bash is 3.2):\n%s\n' "$output"
        return 1
    fi
}
