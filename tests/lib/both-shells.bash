#!/usr/bin/env bash
# Run a script under zsh and bash and require them to agree.  Load with:
#   load 'lib/both-shells'
#
# WHY THIS EXISTS
# ---------------
# Every script in the repo must run under bash and zsh (.claude/CLAUDE.md), and
# the incompatibilities that matter most fail silently: zsh leaves BASH_REMATCH
# empty without `setopt BASH_REMATCH`, indexes arrays from 1, and ties `path` to
# $PATH.  A suite that only calls the script directly runs it under its shebang,
# bash, so nothing it asserts says anything about zsh.

# run_both_shells <script> [args...]: run the script under zsh, then bash, and
# fail when their status or output differ.  Leaves bash's result in $status and
# $output for the caller's assertions.  The script's environment comes from the
# caller (`VAR=value run_both_shells ...`, or an export in the test).
#
# zsh runs with -f (NO_RCS): every zsh reads ~/.zshenv, and the deployed one
# sources path.sh, which puts ~/.local/bin ahead of PATH. Without -f a test that
# hid dotf with PATH=/usr/bin:/bin hid it from bash only, and the two shells
# disagreed on a configured machine while CI stayed green. bash reads no
# startup file for a script unless BASH_ENV is set.
# shellcheck disable=SC2154  # bats' run assigns status and output
run_both_shells() {
    local script="$1"
    shift
    run zsh -f "$script" "$@"
    local zsh_status="$status" zsh_output="$output"
    run bash "$script" "$@"
    if [ "$status" != "$zsh_status" ] || [ "$output" != "$zsh_output" ]; then
        printf 'bash (%s) and zsh (%s) disagree\n--- bash\n%s\n--- zsh\n%s\n' \
            "$status" "$zsh_status" "$output" "$zsh_output"
        return 1
    fi
}
