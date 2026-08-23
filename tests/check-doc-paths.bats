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
    # The probe case (below) stages a fabricated instruction file in the REAL
    # working tree — it has to, because discovery reads the git index. Cleanup
    # lives here rather than at the end of that test because bats aborts a test
    # body at the first failing command: an unguarded `git add -N` that failed
    # (an index.lock from a concurrent session is enough) would skip the
    # cleanup and strand a file claiming a dead path in a tree the whole suite
    # shares. teardown always runs; test-body ordering does not.
    if [ -n "${PROBE_REL:-}" ]; then
        git -C "$DOTFILES_DIR" rm -q --cached "$PROBE_REL" 2>/dev/null || true
        rm -rf "$DOTFILES_DIR/$(dirname "$PROBE_REL")"
        PROBE_REL=""
    fi
}

# The files this guard governs: those that tell an agent what to do.
# DISCOVERED, not enumerated. Three consecutive adversarial-review rounds each
# found a different instruction file missing from a hand-written list
# (`cli/AGENTS.md` and `ai/hermes/AGENTS.md` in round 2, `ai/agy/AGY.md` in
# round 3). A list maintained by hand rots exactly the way the docs this guard
# watches rot, so enumerating it a fourth time would patch the instance and
# leave the class. Anything matching the naming contract is governed the day it
# lands; excluding one requires saying so below, with a reason.
#
# Round 5 found README.md itself breaking that promise: every other pattern
# matched anywhere in the tree, but README.md was anchored to the repo root
# only (`^README\.md$`), undocumented and not among the exclusions below —
# leaving `cli/README.md` and `ai/nan/README.md` ungoverned with real dead
# paths, one of them `scripts/healthcheck.sh`, this guard's own founding
# incident (#916). Folded into the same alternation as everything else.
instruction_files() {
    git -C "$DOTFILES_DIR" ls-files '*.md' \
        | grep -E '(^|/)(AGENTS\.md|CLAUDE\.md|AGY\.md|GEMINI\.md|copilot-instructions\.md|README\.md)$' \
        | grep -vE '^harness/|^specs/|^docs/'
}
# Exclusions above, each for a stated reason:
#   harness/  — generated records; the vault source is the thing to fix, and
#               `compile-harness.sh --check` already guards their rendering
#   specs/    — per-feature proposals, frozen once archived; they describe a
#               change at a moment, not standing instructions
#   docs/     — historical records (docs/lessons.md names retired scripts on
#               purpose); see this file's header

# Absolute paths of the same set, for checks that grep file contents directly.
# The two lists were separate and drifted immediately: `cli/AGENTS.md` and
# `ai/hermes/AGENTS.md` are instruction files by this suite's own criterion and
# were governed by neither. One source, two consumers.
governed_files() {
    local f
    while IFS= read -r f; do
        printf '%s\n' "$DOTFILES_DIR/$f"
    done < <(instruction_files)
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
#
# Honest limit, found by mutation during the DOCS-013 adversarial review: three
# of these shapes (`~/…`, `$VAR/…`, `https://…`) stay rejected even with the
# leading-character filter removed, because is_repo_rooted rejects them too.
# This case pins the OUTCOME for all nine, not the mechanism for those three.
@test "check-doc-paths: ignores non-path tokens that merely look pathish [#916]" {
    {
        printf 'Model `opencode-go/qwen3.6-plus` is pinned.\n'
        printf 'Never use `&>/dev/null` in a script.\n'
        printf 'Per-agent files live in `ai/<agent>/` directories.\n'
        printf 'Encrypt to `sensitive/KEYNAME.secret.age`.\n'
        printf 'Deployed to `~/.claude/skills/` at setup.\n'
        printf 'Resolve via `$SOME_VAR/00_meta/patterns/x.md`.\n'
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

@test "check-doc-paths: a newly added instruction file is governed automatically [#916]" {
    # The point of discovery over enumeration. Three review rounds each found a
    # different file missing from a hand-written list; this asserts the fourth
    # cannot happen silently. `git ls-files` reads the index, so a staged file
    # counts — which is what matters, since a new instruction file arrives
    # staged in the same commit as the code it describes.
    #
    # PROBE_REL is set BEFORE anything is created, so teardown can clean up even
    # if the very next line fails. Do not inline the cleanup here.
    PROBE_REL="ai/probe-agent/AGENTS.md"
    mkdir -p "$DOTFILES_DIR/ai/probe-agent"
    printf 'Run `scripts/definitely-not-here.sh` first.\n' > "$DOTFILES_DIR/$PROBE_REL"
    git -C "$DOTFILES_DIR" add -N "$PROBE_REL"

    instruction_files | grep -qx "$PROBE_REL"
    run bash -c "cd '$DOTFILES_DIR' && '$GUARD' '$PROBE_REL'"
    [ "$status" -eq 1 ]
}

@test "check-doc-paths: a nested README.md is governed, not only the root one [#916]" {
    # Round 5: AGENTS.md/CLAUDE.md/AGY.md/GEMINI.md/copilot-instructions.md all
    # matched anywhere in the tree, but README.md was anchored to root only.
    # cli/README.md was sitting ungoverned with a dead reference to
    # scripts/healthcheck.sh — the exact file whose staleness in
    # .claude/CLAUDE.md was this guard's own founding incident.
    instruction_files | grep -qx "cli/README.md"
}

@test "check-doc-paths: discovery floor - each governed pattern has a known real member [#916]" {
    # Round 5, second finding: mutating the discovery alternation (e.g.
    # dropping AGY.md) left all tests green — nothing pinned that the
    # discovered set actually contains a member of each pattern. Membership
    # assertions, not an exact list or count: an exact list resurrects the
    # hand-maintained list round 3 replaced with discovery, and would rot the
    # same way. GEMINI.md has no real member in this repo today, so it is not
    # asserted here — add its line the day one exists.
    local found
    found="$(instruction_files)"
    printf '%s\n' "$found" | grep -qx "AGENTS.md"
    printf '%s\n' "$found" | grep -qx "ai/agy/AGY.md"
    printf '%s\n' "$found" | grep -qx ".claude/CLAUDE.md"
    printf '%s\n' "$found" | grep -qx "ai/copilot/copilot-instructions.md"
    printf '%s\n' "$found" | grep -qx "README.md"
}

@test "check-doc-paths: a file under an excluded prefix is not discovered [#916]" {
    # The exclusions carry reasons but no real file exercises them today, so
    # without this they are prose. A regression that dropped the exclusion would
    # pull generated harness records and frozen specs into the governed set.
    PROBE_REL="harness/probe-agent/AGENTS.md"
    mkdir -p "$DOTFILES_DIR/harness/probe-agent"
    printf 'Generated record naming `scripts/gone.sh`.\n' > "$DOTFILES_DIR/$PROBE_REL"
    git -C "$DOTFILES_DIR" add -N "$PROBE_REL"

    local discovered
    discovered="$(instruction_files)"
    run grep -qxF "$PROBE_REL" <<<"$discovered"
    [ "$status" -eq 1 ]
}

@test "check-doc-paths: auto-discovers instruction files when run without arguments [#1021]" {
    run "$GUARD"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "check-doc-paths: OK AGENTS.md" ]]
    [[ "$output" =~ "check-doc-paths: OK ai/claude/CLAUDE.md" ]]
}

@test "check-doc-paths: rejects a token that escapes the repo root [#916]" {
    # `scripts/../../<something real>` passes is_repo_rooted (first segment is
    # `scripts`) and used to resolve — and be reported OK — outside the repo.
    # A false negative, which is the failure mode this guard exists to avoid.
    printf 'Escaping: `scripts/../../dotfiles/README.md` is outside.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 1 ]
    [[ "$output" == *"escapes the repo root"* ]]
}

@test "check-doc-paths: a .. token that is not repo-rooted stays ignored [#916]" {
    # The traversal check must not fire on tokens the guard promises to ignore.
    # Ordering it before the rooted gate turned a false-negative fix into a
    # false positive on `not-a-real-dir/../x.md`.
    printf 'Unrelated: `not-a-real-toplevel/../other/thing.md` here.\n' > "$SCRATCH/doc.md"
    run "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 0 ]
}

@test "check-doc-paths: the source builtin needs ./ to read versions.conf under zsh [#916]" {
    # The DOCS-013 review found `. versions.conf` in .claude/CLAUDE.md, which
    # resolves to an empty version under zsh: a slashless argument to `.` is
    # searched on $PATH only, and unlike bash, zsh does not fall back to the
    # cwd. Pin the working form, and pin that the broken one really is broken —
    # otherwise this regresses the moment someone "simplifies" the path.
    local good bad
    good="$(zsh -c "cd '$DOTFILES_DIR' && . ./versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null)"
    [ -n "$good" ]

    # The broken form yields an empty string, not an error the caller notices.
    bad="$(zsh -c "cd '$DOTFILES_DIR' && . versions.conf && printf '%s' \"\$GOLANGCI_LINT_VERSION\"" 2>/dev/null || true)"
    [ -z "$bad" ]
}

@test "check-doc-paths: no instruction file sources a config without ./ [#916]" {
    # Class-level guard: a bare `. <file>` in a documented command is a zsh trap.
    #
    # Comment lines are excluded, and that exclusion is load-bearing rather than
    # convenient: .claude/CLAUDE.md documents the anti-pattern by name ("`. ./x`,
    # not `. x`"), so a grep that cannot tell an instruction from a warning about
    # that instruction fires on the very text teaching people to avoid it. The
    # first draft of this test did exactly that.
    # `(^|[^./a-zA-Z0-9_-])` — the alternation is the fix for a real miss: the
    # first draft required a delimiter character immediately before `. file`,
    # so a source line flush-left inside a ```bash fence — the exact shape the
    # original bug took — matched nothing and the test stayed green through two
    # mutations. A guard blind to the canonical form of its own bug is theatre.
    local hits f
    hits=""
    while IFS= read -r f; do
        [ -f "$f" ] || continue
        hits="$hits$(grep -nE '(^|[^./a-zA-Z0-9_-])\. [a-zA-Z_][a-zA-Z0-9_.-]*\.(conf|sh|zsh)' "$f" \
            | grep -v ':[[:space:]]*#' | sed "s|^|$f:|" || true)"
    done < <(governed_files)
    if [ -n "$hits" ]; then
        echo "bare source (needs ./) in an instruction file:" >&2
        echo "$hits" >&2
    fi
    [ -z "$hits" ]
}

@test "check-doc-paths: catches a missing \$VAULT_PATH path when vault is present [#1043]" {
    local fake_vault="$SCRATCH/vault"
    mkdir -p "$fake_vault/00_meta/skills"
    echo "content" > "$fake_vault/00_meta/skills/real.md"

    cat <<'EOF' > "$SCRATCH/doc.md"
Read `$VAULT_PATH/00_meta/skills/missing.md` for details.
EOF

    run env VAULT_PATH="$fake_vault" "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "referenced vault path does not exist" ]]
}

@test "check-doc-paths: accepts an existing \$VAULT_PATH path when vault is present [#1043]" {
    local fake_vault="$SCRATCH/vault"
    mkdir -p "$fake_vault/00_meta/skills"
    echo "content" > "$fake_vault/00_meta/skills/real.md"

    cat <<'EOF' > "$SCRATCH/doc.md"
Read `$VAULT_PATH/00_meta/skills/real.md` for details.
EOF

    run env VAULT_PATH="$fake_vault" "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 0 ]
}

@test "check-doc-paths: skips vault checks cleanly when VAULT_PATH is unset or non-existent [#1043]" {
    cat <<'EOF' > "$SCRATCH/doc.md"
Read `$VAULT_PATH/00_meta/skills/missing.md` for details.
EOF

    run env -u VAULT_PATH PATH="/usr/bin:/bin" "$GUARD" "$SCRATCH/doc.md"
    [ "$status" -eq 0 ]
}

