#!/usr/bin/env bats
# The self-deploy trigger is converge's `self-deploy` step on every OS (OPS-053,
# #2502). Both setup twins used to install it, gated on DOTFILES_AUTODEPLOY: a
# systemd timer from setup-linux.sh and a Scheduled Task from setup-windows.ps1,
# each with its own unit files and bats. #2693 deleted them. A block that came
# back would install a second, unmarked trigger the step reports as foreign, and
# put logic back into the setup scripts (OPS-053 review round 1, F5).

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
}

@test "neither setup script installs a self-deploy trigger" {
    for f in setup-linux.sh setup-windows.ps1; do
        [ -f "$REPO/$f" ] || { echo "$f is missing" >&2; false; }
        run bash -c "tr -d '\r' < '$REPO/$f' | grep -v '^[[:space:]]*#' | grep -nE 'DOTFILES_AUTODEPLOY|DotfilesSelfUpdate|dotfiles-selfupdate'"
        [ "$status" -eq 1 ] || { echo "$f still installs the trigger: $output" >&2; false; }
    done
}

@test "the setup-era unit files stay deleted" {
    for u in dotfiles-selfupdate.service dotfiles-selfupdate.timer; do
        [ ! -e "$REPO/systemd/$u" ] || { echo "systemd/$u is back" >&2; false; }
    done
}

@test "converge registers the self-deploy step" {
    grep -qF 'selfDeploy{' "$REPO/cli/internal/converge/records.go" || false
}
