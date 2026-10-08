#!/usr/bin/env bats
# No document tells the reader to clone the repository INTO ~/.dotfiles. That is
# the deploy target setup writes into, and setup refuses to run from it (#695),
# so a doc that says so fails the reader on the first command. The runbook
# ai-tools-setup.md said exactly that until this guard existed.

load 'lib/refute'

@test "no doc clones the repository into ~/.dotfiles" {
    local root="$BATS_TEST_DIRNAME/.."
    local f
    for f in "$root"/README.md "$root"/docs/runbooks/*.md; do
        refute_grep 'git clone [^ ]*dotfiles(\.git)?[[:space:]]+(~|\$HOME)/\.dotfiles([[:space:]]|$)' "$f"
        refute_grep '(^|[[:space:]])cd (~|\$HOME)/\.dotfiles[[:space:]]*&&[[:space:]]*(git pull|\./setup)' "$f"
    done
}
