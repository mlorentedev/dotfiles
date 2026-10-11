#!/usr/bin/env bats
# No document tells the reader to clone the repository INTO ~/.dotfiles, or to
# pull and run setup from there. That is the deploy target setup writes into,
# and setup refuses to run from it (#695), so a doc that says so fails the
# reader on the first command. The runbook ai-tools-setup.md (since replaced by
# guide-agent-provisioning.md) said exactly that
# until this guard existed.

load 'lib/refute'

docs() {
    local root="$BATS_TEST_DIRNAME/.."
    printf '%s\n' "$root"/README.md "$root"/docs/runbooks/*.md
}

# Prints each line of $1 where a `cd` into the deploy target is followed, on the
# same line or the next one, by a pull or a setup run (#2158 review).
pull_or_setup_hits() {
    awk '
        function bad(s) { return s ~ /git pull|setup-linux\.sh|install\.sh/ }
        pending { if (bad($0)) print FILENAME ":" NR ": " $0; pending = 0 }
        /(^|[[:space:];&])cd[[:space:]]+"?(~|\$HOME)\/\.dotfiles"?([[:space:];&]|$)/ {
            if (bad($0)) print FILENAME ":" NR ": " $0; else pending = 1
        }' "$1"
}

@test "no doc clones the repository into ~/.dotfiles" {
    local f
    while IFS= read -r f; do
        refute_grep 'git clone [^ ]*dotfiles(\.git)?[[:space:]]+"?(~|\$HOME)/\.dotfiles"?([[:space:]]|$)' "$f"
    done < <(docs)
}

@test "no doc pulls or runs setup from ~/.dotfiles" {
    local f hits
    while IFS= read -r f; do
        hits=$(pull_or_setup_hits "$f")
        [ -z "$hits" ] || { printf '%s\n' "$hits" >&2; false; }
    done < <(docs)
}

@test "the pull-or-setup detector catches every spelling it claims to, and nothing else" {
    local f="$BATS_TEST_TMPDIR/doc.md" spelling
    for spelling in 'cd ~/.dotfiles && git pull && ./setup-linux.sh' \
                    'cd "$HOME/.dotfiles" && bash setup-linux.sh' \
                    'cd $HOME/.dotfiles; ./install.sh' \
                    $'cd ~/.dotfiles\ngit pull'; do
        printf '%s\n' "$spelling" > "$f"
        [ -n "$(pull_or_setup_hits "$f")" ] || { printf 'missed: %s\n' "$spelling" >&2; false; }
    done
    for spelling in 'cd ~/Projects/dotfiles && ./install.sh' 'ls ~/.dotfiles' \
                    $'cd ~/.dotfiles\nls'; do
        printf '%s\n' "$spelling" > "$f"
        [ -z "$(pull_or_setup_hits "$f")" ] || { printf 'false hit: %s\n' "$spelling" >&2; false; }
    done
}
