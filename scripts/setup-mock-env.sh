#!/usr/bin/env bash
# setup-mock-env.sh - Build the ~/.dotfiles mock the bats suite runs against in CI.
#
# Shared by the Linux `test` and the `test-macos` jobs in ci.yml (#2152). It
# replaced an inline step that discarded its own failures:
#
#     age-keygen -o ~/.config/age/key.txt 2>/dev/null
#     cp -r sensitive/* ~/.dotfiles/sensitive/ 2>/dev/null || true
#
# The suite treats a missing key or fixture as a reason to skip, so a setup
# failure there turned into absence and the leg still read as a pass. Here
# every promise the step makes is asserted, and a broken one fails the job.
#
# Run from the repository root. Copying scripts/ copies this file into the
# mock too; that is harmless and keeps the copy a plain mirror of scripts/.
set -euo pipefail

fail() {
    printf '::error::setup-mock-env: %s\n' "$1" >&2
    exit 1
}

[ -f versions.conf ] && [ -d scripts ] && [ -d sensitive ] \
    || fail "run from the repository root (versions.conf, scripts/ and sensitive/ expected in $(pwd))"
command -v age-keygen >/dev/null 2>&1 || fail "age-keygen is not on PATH; install age before this step"

mkdir -p "$HOME/.dotfiles/sensitive" "$HOME/.dotfiles/scripts" "$HOME/.config/age" \
    || fail "could not create the mock directories under $HOME"

# age-keygen refuses to overwrite an existing key; a rerun keeps the first one.
key="$HOME/.config/age/key.txt"
[ -s "$key" ] || age-keygen -o "$key" || fail "age-keygen could not write $key"
[ -s "$key" ] || fail "age-keygen left no key at $key"

# Every failure goes through fail(), so the step always carries an annotation;
# `set -e` alone would stop the script with plain stderr and no ::error:: line.
cp versions.conf "$HOME/.dotfiles/" || fail "could not copy versions.conf"
cp -r scripts/. "$HOME/.dotfiles/scripts/" || fail "could not copy scripts/"
cp -r sensitive/. "$HOME/.dotfiles/sensitive/" || fail "could not copy sensitive/"

for dir in scripts sensitive; do
    # -quit, not `| grep -q .`: under pipefail grep's early exit would SIGPIPE
    # find and report a full directory as empty.
    [ -n "$(find "$HOME/.dotfiles/$dir" -mindepth 1 -print -quit)" ] \
        || fail "$HOME/.dotfiles/$dir is empty after the copy"
done

printf 'mock environment ready: %s, %s\n' "$key" "$HOME/.dotfiles"
