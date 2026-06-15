#!/usr/bin/env bats
# Guard (incident -> guard, CLI-014): every embedded template under
# cli/internal/initrepo/templates/ MUST be git-tracked.
#
# A template present on disk but gitignored embeds fine locally via //go:embed
# (which reads the working tree, not the index) yet is ABSENT in a fresh CI
# checkout -> `dotf init` panics at runtime with `open templates/X: file does
# not exist`, and the binary still builds (embed of a whole dir never requires a
# specific file). The original incident: the CLAUDE.md pointer template was
# swallowed by the root .gitignore `CLAUDE.md` rule and only surfaced on the
# first CI run. This guard catches the whole class locally, before CI.

setup() {
    export REPO_DIR="$BATS_TEST_DIRNAME/.."
    export TMPL_DIR="cli/internal/initrepo/templates"
}

@test "every embed template is git-tracked (gitignored template -> absent in fresh checkout)" {
    cd "$REPO_DIR"
    [ -d "$TMPL_DIR" ]
    local f rel untracked=""
    for f in "$TMPL_DIR"/*; do
        [ -f "$f" ] || continue
        rel="$TMPL_DIR/$(basename "$f")"
        git ls-files --error-unmatch "$rel" >/dev/null 2>&1 || untracked="$untracked $rel"
    done
    if [ -n "$untracked" ]; then
        echo "Untracked embed templates (present on disk, absent in a fresh checkout):$untracked"
        return 1
    fi
}

@test "staticFiles claude-md mapping is present (regression: CLAUDE.md source name is gitignored)" {
    # The source template must NOT be literally CLAUDE.md (the root .gitignore
    # CLAUDE.md rule would drop it); it is committed as claude-md and mapped here.
    grep -qF '{"claude-md", "CLAUDE.md"}' "$REPO_DIR/cli/internal/initrepo/scaffold.go"
    [ -f "$REPO_DIR/$TMPL_DIR/claude-md" ]
    [ ! -f "$REPO_DIR/$TMPL_DIR/CLAUDE.md" ]
}
