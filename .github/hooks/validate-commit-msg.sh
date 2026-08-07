#!/usr/bin/env sh
# Validate the commit SUBJECT against the Conventional Commits form this repo
# actually uses and release-please consumes: `type(scope)!: subject`.
#
# POSIX sh on purpose — tests/validate-commit-msg.bats asserts this parses under
# sh, bash AND zsh. `#!/usr/bin/env sh` rather than `#!/bin/sh`: pre-commit's
# `language: script` resolves an `env` shebang against PATH, while a literal
# /bin/sh does not exist on native Windows, so the hook could not run there at
# all (#794).
#
# The type stays permissive (`[a-z]+`) rather than an allow-list, because the
# repo also uses non-release types such as `wip:`. The fix is that the scope and
# the breaking-change `!` are now accepted: the previous `^[a-z]+: .+` rejected
# `feat(tmux):`, `docs(spec):`, `fix(setup):` and every other scoped commit on
# main. It only ever appeared to pass because PRs are squash-merged on GitHub,
# where this local hook never runs (#794).

MSG_FILE="$1"

if [ -z "$MSG_FILE" ] || [ ! -f "$MSG_FILE" ]; then
    echo "[ERROR] validate-commit-msg: no commit message file passed (got '$MSG_FILE')." >&2
    exit 1
fi

# Validate the SUBJECT only. The previous version grepped the whole message, so
# `^` matched at any line start and a conforming body line could rescue a
# malformed subject.
SUBJECT=$(head -n 1 "$MSG_FILE")

# git generates these; they are not authored subjects and carry no type.
case "$SUBJECT" in
    "Merge "* | 'Revert "'* | "fixup!"* | "squash!"*) exit 0 ;;
esac

if ! printf '%s\n' "$SUBJECT" | grep -Eq '^[a-z]+(\([a-zA-Z0-9._/-]+\))?!?: .+'; then
    echo "[ERROR] Commit subject must be a Conventional Commit: type(scope)!: subject" >&2
    echo "" >&2
    echo "  type   required, lowercase (feat, fix, docs, chore, refactor, test, wip, ...)" >&2
    echo "  scope  optional, in parentheses (tmux, spec, setup, ...)" >&2
    echo "  !      optional, marks a breaking change" >&2
    echo "  :      followed by a single space, then a non-empty subject" >&2
    echo "" >&2
    echo "  Valid: feat(tmux): add a ~/.tmux.conf.local override seam" >&2
    echo "         docs: clarify the vault path cascade" >&2
    echo "         feat(api)!: drop the v1 endpoint" >&2
    echo "" >&2
    echo "Subject was: $SUBJECT" >&2
    exit 1
fi
