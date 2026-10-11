#!/usr/bin/env bats
# A script someone invokes as `./thing.sh` must carry the executable bit in git.
#
# Incident (2026-09-01, #1411): setup-linux.sh went 100755 -> 100644 and reached
# main. `install.sh:52` is `exec ./setup-linux.sh "$@"` -- the bootstrap entry
# point -- and README documents the same invocation, so a fresh clone failed at
# the first step with exit 126. Machines already set up noticed nothing.
#
# Since PLAT-001b the bootstrap is `install.sh` -> `dotf converge`, whose
# legacy-setup step runs the checkout's setup-linux.sh as a program, with no
# interpreter in front of it: the same exit 126 on a lost bit.
#
# CI could not see it and would not have. `tests/Dockerfile.integration` runs
# `bash setup-linux.sh` with an explicit interpreter, which works at any mode, so
# the integration job exercises the script without ever exercising the path that
# broke. The contrast lived inside the repo the whole time: install.sh was still
# 100755.
#
# How the bit was lost is worth naming, because the mechanism is one both the
# mutation-testing convention here and any ad-hoc edit will hit again:
#
#     grep -v 'pattern' file.sh > /tmp/x && mv /tmp/x file.sh
#
# The redirection creates a NEW file at the umask (644) and `mv` installs it over
# the original, dropping the bit. Restoring the content afterwards with
# `cp backup file.sh` does not bring it back -- `cp` onto an existing file keeps
# the DESTINATION's mode, which is already 644. So the content restores
# byte-for-byte and the mode does not, which is precisely why nobody notices.

# bats file_tags=os-sensitive

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

# The list is DERIVED from the invokers, never hand-maintained. A hand-written
# list cannot catch the entry point nobody thought to add to it -- the same
# reasoning as the merge-policy parity guard in claude-settings-template.bats.
#
# Two exclusions, both found by review on this very PR and both the same class of
# mistake the guard exists to catch:
#
#   - `docs/lessons/` is PRUNED. Lesson 250 is prose ABOUT this invocation and
#     quotes `./setup-linux.sh`; leaving it in meant the derivation fed itself
#     from documentation. Measured: replacing the real invocations in install.sh
#     and README with `bash setup-linux.sh` left the "guard actually finds the
#     entry point" test GREEN, sourced entirely from that lesson. A guard a
#     history file can satisfy is not a guard.
#   - Only TRACKED files survive. Prose also mentions placeholders like `./x.sh`,
#     which is not a script and must never enter the list.
_invoked_scripts() {
    cd "$DOTFILES_DIR" || return 1
    # No shell glob anywhere: zsh's NOMATCH would abort the whole command on an
    # unmatched pattern. grep and find resolve their own patterns.
    {
        grep -rhoE '(^|[^a-zA-Z0-9_./-])\./[a-zA-Z0-9_./-]+\.sh' install.sh README.md 2>/dev/null
        find docs -name '*.md' -not -path 'docs/lessons/*' -exec \
            grep -hoE '(^|[^a-zA-Z0-9_./-])\./[a-zA-Z0-9_./-]+\.sh' {} + 2>/dev/null
    } \
        | grep -oE '\./[a-zA-Z0-9_./-]+\.sh' \
        | sed 's|^\./||' \
        | sort -u \
        | while IFS= read -r p; do
            git -C "$DOTFILES_DIR" ls-files --error-unmatch "$p" >/dev/null 2>&1 && printf '%s\n' "$p"
        done
}

@test "every script invoked as ./x.sh is executable in git's index" {
    local file_path mode missing="" seen=0
    # Self-defending: an empty derivation would make this pass vacuously, and
    # relying on a SEPARATE test to notice that is exactly the coupling a
    # reviewer flagged. Assert the premise here too.
    while IFS= read -r file_path; do
        [ -n "$file_path" ] || continue
        seen=$((seen + 1))
        mode="$(git -C "$DOTFILES_DIR" ls-files -s -- "$file_path" | awk '{print $1}')"
        # The INDEX, not the filesystem: the index is what a fresh clone
        # materialises from, and it is what a commit records. A working tree can
        # be executable while the thing that ships is not.
        if [ "$mode" != "100755" ]; then
            missing="$missing $file_path($mode)"
        fi
    done < <(_invoked_scripts)

    if [ "$seen" -eq 0 ]; then
        printf 'the derivation found no invoked scripts at all -- this test would pass vacuously\n' >&2
        return 1
    fi
    if [ -n "$missing" ]; then
        printf 'invoked as ./ but not executable in the index:%s\n' "$missing" >&2
        return 1
    fi
}

@test "the guard actually finds the bootstrap entry point" {
    # Without this, a regex that silently matched nothing would make the test
    # above pass on an empty list -- a guard that passes on the broken thing is
    # the defect the guard exists to prevent. setup-linux.sh is invoked by
    # install.sh and documented in README, so it must appear.
    _invoked_scripts | grep -qx 'setup-linux.sh'
}

@test "converge runs setup-linux.sh as a program, the invocation this guard protects" {
    # Pins the premise. The bootstrap reaches the script through converge's
    # legacy-setup step (cli/internal/converge/legacy.go), which executes the
    # path itself rather than `bash <path>`. If that ever gains an interpreter,
    # the guard above is still correct but no longer load-bearing, and whoever
    # changed it should see this fail and re-read the comment at the top.
    #
    # ANCHORED to the start of a line, so a commented-out call cannot satisfy
    # it: evidence that merely MENTIONS the invocation is not evidence that it
    # happens.
    legacy="$DOTFILES_DIR/cli/internal/converge/legacy.go"
    grep -qE '^[[:space:]]*c = exec\.Command\(cmd\)' "$legacy"
    grep -qF 'return "setup-linux.sh"' "$legacy"
}
