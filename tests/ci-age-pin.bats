#!/usr/bin/env bats
# The CI age install must honour versions.conf, on every OS.
#
# It did not, for months, and nothing said so: versions.conf declared
# AGE_VERSION=1.3.1 while the Linux job ran `apt-get install age` and got 1.1.1.
# Windows tested the pinned version, Linux tested whatever Ubuntu shipped, from
# one line of the same manifest — for the tool that decrypts every secret in
# this repo. A pin that one of two consumers ignores is not a pin.
#
# Asserted on the workflow text, because nothing here can run GitHub's runner.
# What is protected is that the version comes from the manifest rather than
# from a distro's opinion.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CI="$REPO/.github/workflows/ci.yml"
}

@test "ci: no job installs age from a distro package manager" {
    # The failure is silent by construction — apt succeeds, the suite is green,
    # and it exercised a different binary than the one declared.
    if grep -nE 'apt-get install[^\n]*[^-a-z]age([ "'"'"']|$)' "$CI"; then
        printf 'age must come from the pinned release, not apt — see AGE_VERSION.\n' >&2
        return 1
    fi
}

@test "ci: both age installs resolve the version from versions.conf" {
    # One per OS. Neither may hardcode a version: a literal here and a value in
    # versions.conf is the same two-copies problem the manifest exists to end.
    local n
    n=$(grep -cE 'AGE_VERSION' "$CI")
    [ "$n" -ge 2 ] || { printf 'expected >=2 AGE_VERSION references in ci.yml, found %s\n' "$n" >&2; return 1; }

    if grep -nE 'releases/download/v[0-9]+\.[0-9]+\.[0-9]+' "$CI"; then
        printf 'a hardcoded age version in a download URL — read it from versions.conf\n' >&2
        return 1
    fi
}

@test "ci: the linux install verifies what it got against the pin" {
    # Downloading the right URL is not the same as running the right binary: a
    # release could be retagged, a tarball could change layout, and a silent
    # PATH shadow would leave an older age first. The assertion is what makes
    # the pin observable rather than merely intended.
    grep -q 'age --version' "$CI"
    grep -qE 'AGE_VERSION#v|expected \$\{AGE_VERSION' "$CI"
}
