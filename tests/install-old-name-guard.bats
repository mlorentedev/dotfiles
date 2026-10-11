#!/usr/bin/env bats
# AC7 of PLAT-001b: the installers live at the root as install.{sh,ps1}, and
# the root install.sh no longer clones and runs setup. No live file may still
# send a reader to the old names or the old flow's knob, or a fresh machine
# follows a path that no longer exists.
#
# The historical records keep the old names on purpose: ADRs and audits,
# lessons, specs (live ones describe the migration they deliver) and the
# changelog describe what was true when they were written.

setup() {
    REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

# The names are split so this file never matches itself.
OLD_NAMES="install""-dotf|DOTFILES""_SKIP_SETUP"

@test "no live file names the old installers or the clone-and-setup knob" {
    # Preflight: a broken git grep would answer "no match", which reads as a
    # clean tree. Prove it finds a name that is there before believing it.
    git -C "$REPO" grep -q -E 'install_dotf' -- install.sh

    run git -C "$REPO" grep -n -E "$OLD_NAMES" -- \
        ':!docs/adr/' ':!docs/lessons/' ':!specs/' ':!CHANGELOG.md'
    # git grep exits 1 when nothing matches; anything else is a failed search.
    [ "$status" -eq 1 ] || {
        printf 'live references to the old installers:\n%s\n' "$output"
        false
    }
}

@test "the root installers exist and the old paths are gone" {
    [ -f "$REPO/install.sh" ]
    [ -f "$REPO/install.ps1" ]
    [ ! -e "$REPO/scripts/install""-dotf.sh" ]
    [ ! -e "$REPO/scripts/install""-dotf.ps1" ]
}
