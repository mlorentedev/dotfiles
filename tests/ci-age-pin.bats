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
    #
    # Line continuations are JOINED first, because grep is line-based and the
    # multi-line form is the idiomatic one for a multi-package apt install:
    #
    #     apt-get install -y --no-install-recommends \
    #         age \
    #         zsh
    #
    # A per-line matcher never sees `install` and `age` together there and
    # reports clean. tests/Dockerfile.integration is written in exactly that
    # shape, so it is the likely form of any regression. Found while checking a
    # reviewer's claim on #1059: the claim itself (that a flagless `apt-get
    # install age` escapes) was wrong — the regex backtracks and catches it —
    # but testing the claim surfaced this, which is real.
    # Comments are dropped BEFORE matching. The step this guard protects carries
    # a comment explaining the defect, which names `apt-get install age` in prose
    # — and the first version of this test tripped on it. A guard that fires on
    # documentation ABOUT the thing rather than the thing is the same false
    # positive GUARD-002 hit, and it trains people to weaken the guard.
    local joined
    joined=$(grep -vE '^[[:space:]]*#' "$CI" | awk '{ if (/\\$/) { printf "%s", substr($0,1,length-1) } else { print } }')
    if printf '%s\n' "$joined" | grep -nE 'apt-get[[:space:]]+install\b[^|;&]*\bage\b'; then
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

@test "setup-linux.sh: age install pins AGE_VERSION rather than fetching latest" {
    local setup_sh="$REPO/setup-linux.sh"
    grep -q 'AGE_VERSION' "$setup_sh"
    grep -q 'FiloSottile/age/releases/download/v' "$setup_sh"
    local age_section
    age_section="$(sed -n '/Installing age/,/age already installed/p' "$setup_sh")"
    if echo "$age_section" | grep -q 'filippo.io/age/latest' || echo "$age_section" | grep -q 'releases/latest'; then
        printf 'setup-linux.sh still fetches latest age instead of pinned AGE_VERSION\n' >&2
        return 1
    fi
}

@test "integration: Dockerfile.integration installs pinned AGE_VERSION release rather than apt" {
    local dockerfile="$REPO/tests/Dockerfile.integration"
    grep -q 'ARG AGE_VERSION' "$dockerfile"
    grep -q 'FiloSottile/age/releases/download/v\${AGE_VERSION}' "$dockerfile"
    if grep -vE '^[[:space:]]*#' "$dockerfile" | grep -E 'apt-get install.*age\b'; then
        printf 'Dockerfile.integration still installs age from apt\n' >&2
        return 1
    fi
}
