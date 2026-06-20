#!/usr/bin/env bats
# REFACTOR-013: version_gte is the shared minimum-version comparator used by the
# install gates in setup-linux.sh. versions.conf pins are MINIMUMS: upgrade when
# installed < pin, leave a newer install untouched (no downgrade). These tests
# pin the semver semantics so a future edit can't silently reintroduce the
# exact-match (== / !=) behavior that downgraded newer tools.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    source "$DOTFILES_DIR/scripts/utils.sh"
}

@test "version_gte: equal versions are satisfied" {
    run version_gte "1.16.2" "1.16.2"
    [ "$status" -eq 0 ]
}

@test "version_gte: a newer install satisfies the minimum (no downgrade)" {
    run version_gte "1.17.0" "1.16.2"
    [ "$status" -eq 0 ]
}

@test "version_gte: an older install does NOT satisfy the minimum (needs upgrade)" {
    run version_gte "1.15.9" "1.16.2"
    [ "$status" -eq 1 ]
}

@test "version_gte: compares numerically, not lexically (1.10.0 >= 1.9.0)" {
    run version_gte "1.10.0" "1.9.0"
    [ "$status" -eq 0 ]
}

@test "version_gte: differing component counts (1.2 < 1.2.0)" {
    run version_gte "1.2" "1.2.0"
    [ "$status" -eq 1 ]
}

@test "version_gte: empty minimum means any version is OK (no pin)" {
    run version_gte "0.1.0" ""
    [ "$status" -eq 0 ]
}

@test "version_gte: empty installed means needs install" {
    run version_gte "" "1.16.2"
    [ "$status" -eq 1 ]
}

@test "version_gte: two-digit minor/patch sort correctly (1.22.22 >= 1.22.9)" {
    run version_gte "1.22.22" "1.22.9"
    [ "$status" -eq 0 ]
}
