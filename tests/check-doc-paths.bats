#!/usr/bin/env bats
# Guard for #916: an instruction file must not name a repo path that is gone.
#
# #907 committed .claude/CLAUDE.md while it named seven files that no longer
# existed, and two sessions in two days acted on one of them. Nothing in the
# repo checked it: docs-drift.bats checks content sync between two copies,
# check-md-escapes.sh checks corruption, compile-harness --check checks harness
# drift. None of them ask whether a path in prose still resolves.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    GUARD="$DOTFILES_DIR/scripts/check-doc-paths.sh"
    SCRATCH="$BATS_TMPDIR/check-doc-paths"
    mkdir -p "$SCRATCH"
}

teardown() {
    rm -rf "$SCRATCH"
}

# The files this guard governs: those that tell an agent what to do. Historical
# records (docs/lessons.md) are deliberately absent — they name retired scripts
# on purpose, and that is correct.
instruction_files() {
    printf '%s\n' \
        ".claude/CLAUDE.md" \
        "AGENTS.md" \
        "README.md" \
        "ai/claude/CLAUDE.md" \
        "ai/copilot/copilot-instructions.md" \
        ".github/copilot-instructions.md"
}

@test "check-doc-paths.sh exists and is executable" {
    [ -x "$GUARD" ]
}

@test "check-doc-paths: every instruction file's repo paths resolve [#916]" {
    local failed=0 f
    while IFS= read -r f; do
        [ -f "$DOTFILES_DIR/$f" ] || continue
        if ! (cd "$DOTFILES_DIR" && "$GUARD" "$f"); then
            failed=$((failed + 1))
        fi
    done < <(instruction_files)
    [ "$failed" -eq 0 ]
}

@test "check-doc-paths: catches a path that does not exist [#916]" {
    printf 'See `scripts/definitely-not-here.sh` for details.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 1 ]
    [[ "$output" == *"definitely-not-here.sh"* ]]
}

@test "check-doc-paths: catches a glob that matches nothing [#916]" {
    printf 'Skills live in `ai/skills/*/SKILL.md` today.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 1 ]
    [[ "$output" == *"matches nothing"* ]]
}

@test "check-doc-paths: accepts a path that exists [#916]" {
    printf 'The library is `scripts/utils.sh` and the manifest `versions.conf`.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 0 ]
}

# The false-positive cases below are the reason this guard is usable at all. An
# earlier revision flagged all of them, which on AGENTS.md meant 13 bogus
# failures — and a guard that cries wolf gets deleted rather than obeyed.
@test "check-doc-paths: ignores non-path tokens that merely look pathish [#916]" {
    {
        printf 'Model `opencode-go/qwen3.6-plus` is pinned.\n'
        printf 'Never use `&>/dev/null` in a script.\n'
        printf 'Per-agent files live in `ai/<agent>/` directories.\n'
        printf 'Encrypt to `sensitive/KEYNAME.secret.age`.\n'
        printf 'Deployed to `~/.claude/skills/` at setup.\n'
        printf 'Resolve via `$VAULT_PATH/00_meta/patterns/x.md`.\n'
        printf 'Any `.sh` file must run under both shells.\n'
        printf 'The store index is `_index.md` over there.\n'
        printf 'See `https://example.com/a/b.md` for context.\n'
    } > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 0 ]
}

@test "check-doc-paths: an ALL-CAPS filename with a known extension is still checked [#916]" {
    # SKILL.md / AGENTS.md / MEMORY.md are real files, not placeholders. An
    # earlier revision skipped every ALL-CAPS segment and so silently ignored
    # the dead `ai/skills/*/SKILL.md` reference it was written to catch.
    printf 'Records live at `harness/skills/nope-not-real/SKILL.md`.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 1 ]
}

@test "check-doc-paths: usage error without arguments [#916]" {
    run "$GUARD"
    [ "$status" -eq 2 ]
}
