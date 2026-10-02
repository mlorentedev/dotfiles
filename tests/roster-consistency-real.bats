#!/usr/bin/env bats
# Real-dependency coverage for tests/roster-consistency.bats, which stubs
# `dotf harness resolve-skills`. These cases use the real parser, and the last
# one the real vault: the drift HARNESS-046 guards against lives there, and CI
# has no vault, so that case runs wherever one resolves.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    GUARD="$DOTFILES_DIR/scripts/check-roster-consistency.py"
    dotf harness resolve-skills --help >/dev/null 2>&1 \
        || skip "no dotf with \`harness resolve-skills\` on PATH"
}

@test "roster: the real parser reads both skills forms the same way" {
    vault="$BATS_TEST_TMPDIR/vault"
    mkdir -p "$vault/00_meta/agents/definitions/curator" "$vault/00_meta/agents/definitions/builder"
    printf '| Role | Phase | Skills |\n|---|---|---|\n| **curator** | Crystallize | handoff, crystallize, insights |\n| **builder** | Build | test, debug, lint |\n' \
        > "$vault/00_meta/agents/ROSTER.md"
    printf -- '---\nkind: invocable\nskills: [crystallize, insights, handoff]\n---\n\nbody\n' \
        > "$vault/00_meta/agents/definitions/curator/AGENT.md"
    printf -- '---\nkind: invocable\nskills:\n  - id: lint\n    enforce: warn\n  - id: test\n    enforce: block\n  - id: debug\n    enforce: warn\n---\n\nbody\n' \
        > "$vault/00_meta/agents/definitions/builder/AGENT.md"
    run env VAULT_PATH="$vault" python3 "$GUARD"
    [ "$status" -eq 0 ]
}

@test "roster: the live vault's roster and definitions agree" {
    local vault="${VAULT_PATH:-}"
    [ -n "$vault" ] || vault="$(dotf env path VAULT_PATH 2>/dev/null || true)"
    [ -n "$vault" ] && [ -f "$vault/00_meta/agents/ROSTER.md" ] \
        || skip "no vault resolves here; roster-consistency.bats pins the logic"
    run env VAULT_PATH="$vault" python3 "$GUARD"
    [ "$status" -eq 0 ]
}
