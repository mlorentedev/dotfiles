#!/usr/bin/env bats
# Self-test for tests/lib/both-shells.bash.

load 'lib/both-shells'

setup() {
    SCRATCH="$BATS_TEST_TMPDIR"
    printf '%s\n' 'printf "path-head=%s\n" "${PATH%%:*}"' > "$SCRATCH/probe.sh"
}

# zsh reads ~/.zshenv for every script it runs, and the deployed one sources
# path.sh, which puts mise's shims and ~/.local/bin ahead of PATH. So a test
# that ran a guard with PATH=/usr/bin:/bin to hide dotf hid it from bash only:
# zsh found dotf, resolved the real vault, and the two shells disagreed on the
# owner's Mac while CI, which has no ~/.zshenv, stayed green (#1043's test).
# A comparison of the two shells must give both the environment the test set.
@test "run_both_shells: zsh does not read the user's startup files" {
    mkdir -p "$SCRATCH/home"
    printf '%s\n' 'export PATH="/from/zshenv:$PATH"' > "$SCRATCH/home/.zshenv"
    HOME="$SCRATCH/home" PATH="/usr/bin:/bin" run_both_shells "$SCRATCH/probe.sh"
    [ "$status" -eq 0 ] || false
    [ "$output" = "path-head=/usr/bin" ] || false
}

@test "run_both_shells: a real difference between the shells still fails" {
    printf '%s\n' 'if [ -n "${ZSH_VERSION:-}" ]; then echo zsh; else echo bash; fi' > "$SCRATCH/differs.sh"
    run run_both_shells "$SCRATCH/differs.sh"
    [ "$status" -ne 0 ] || false
}
