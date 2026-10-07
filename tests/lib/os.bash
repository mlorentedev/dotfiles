#!/usr/bin/env bash
# Portability helpers for tests that must run on Linux and macOS.  Load with:
#   load 'lib/os'
#
# WHY THIS EXISTS
# ---------------
# The suite was written on Linux, where `stat -c`, `sha256sum` and GNU `sed` are
# simply there.  On macOS they are BSD tools with different flags, or absent
# (sha256sum only exists from macOS 26 and then only in /sbin).  A test that
# reaches for the GNU spelling fails there for a reason that has nothing to do
# with what it tests, and one that swallows the error passes for the same
# reason.  Each helper below is the one spelling that works on both.

# sha256sum is a function on a machine that only has shasum, so a test (and the
# subshell it opens) writes the same `<hash>  <file>` lines on either OS.
if ! command -v sha256sum >/dev/null 2>&1; then
    sha256sum() { shasum -a 256 "$@"; }
fi

# file_mode <path>: the octal permission bits, e.g. 600.
file_mode() {
    stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

# require_gnu_sed: skip, naming the reason, where sed is not GNU sed.  For a test
# that executes a CI step which only ever runs on ubuntu-latest, whose script
# uses GNU-only sed syntax (\b, the I flag).  Call it as the first line of the
# test, never inside `run`, where a skip cannot reach bats.
require_gnu_sed() {
    sed --version 2>/dev/null | grep -q 'GNU sed' ||
        skip "the step under test runs on ubuntu-latest and needs GNU sed; this sed is BSD"
}
