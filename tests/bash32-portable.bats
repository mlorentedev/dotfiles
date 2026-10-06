#!/usr/bin/env bats
# macOS ships bash 3.2 as /bin/bash, and ADR-003 holds every shell script to it.
# compile-harness.sh used mapfile (bash 4): on a Mac its refresh, deploy and
# check all died at the first one, setup downgraded that to a warning, and no
# skill reached any agent (PLAT-001 W3, #2013). Linux CI runs bash 5, so nothing
# there could see it. This guard is static, so it runs on every OS: it fails on a
# bash-4-only construct in any POSIX shell file that bash executes.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# bash-4+ only: mapfile/readarray, associative arrays and namerefs, case-changing
# expansions, the |& and &>> redirections, coproc and the ;;& / ;& case fallthroughs.
BASH4_ONLY='(^|[^[:alnum:]_])(mapfile|readarray|coproc)([^[:alnum:]_]|$)|(declare|local|typeset)[[:space:]]+-[[:alpha:]]*[An]|\$\{[[:alpha:]_][[:alnum:]_]*(\[[^]]*\])?(,,?|\^\^?)\}|&>>|\|&|;;&|;&'

# Every shell file bash runs: the scripts dir, the root bootstrap scripts, the
# global git hooks and their library, and the bash rc file. .zsh/ is zsh-only.
bash_files() {
    local f
    for f in "$DOTFILES_DIR"/scripts/*.sh "$DOTFILES_DIR"/setup-linux.sh \
             "$DOTFILES_DIR"/install.sh "$DOTFILES_DIR"/.bashrc \
             "$DOTFILES_DIR"/git-hooks/* "$DOTFILES_DIR"/git-hooks/lib/*.sh; do
        [[ -f "$f" ]] || continue
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
    [[ $status -eq 0 ]]
    [[ $(printf '%s\n' "$output" | grep -c 'compile-harness.sh') -eq 1 ]]
    [[ $(printf '%s\n' "$output" | wc -l) -gt 20 ]]
}

@test "the pattern catches each bash-4-only construct and spares bash 3.2 code" {
    for bad in 'mapfile -t ids < <(jq .)' 'readarray x' 'declare -A map' \
               'local -n ref=$1' 'echo ${name,,}' 'echo ${name^^}' 'cmd &>> log' \
               'cmd |& tee' 'coproc foo'; do
        printf '%s\n' "$bad" | grep -qE "$BASH4_ONLY" || { echo "missed: $bad"; return 1; }
    done
    for ok in 'read_array ids < <(jq .)' 'local -a list' 'echo ${name}' \
              'cmd >>log 2>&1' 'x="mapfiles"' 'declare -r CONST=1'; do
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
