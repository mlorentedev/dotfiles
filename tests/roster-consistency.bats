#!/usr/bin/env bats
# Guard for HARNESS-046: ROSTER.md and the persona definitions must not drift.
#
# scripts/check-roster-consistency.py reads both from the VAULT, the source of
# record, because checking the generated harness/agents/ copy would pass
# whenever the generator faithfully rendered a wrong definition. CI has no
# vault, so the guard is wired twice:
#
#   * the cases here run everywhere, CI included: they pin the comparison
#     logic against a throwaway vault and a stubbed `dotf`;
#   * roster-consistency-real.bats drives the real `dotf` parser and the real
#     vault wherever they exist, and skips by name where they do not.
#
# Until this file existed the script sat in the spec folder and ran only when
# someone remembered it, which is how it went red on a harmless reordering of
# curator's skills with nobody noticing (HARNESS-046 review).

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    GUARD="$DOTFILES_DIR/scripts/check-roster-consistency.py"
    VAULT="$BATS_TEST_TMPDIR/vault"
    STUB_BIN="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$VAULT/00_meta/agents/definitions" "$STUB_BIN"
    # `dotf harness resolve-skills <file>` prints the record's flow sequence.
    # The stub reads the same `skills: [a, b, c]` line the real parser would.
    cat > "$STUB_BIN/dotf" <<'STUB'
#!/usr/bin/env bash
[ "$1 $2" = "harness resolve-skills" ] || exit 2
sed -n 's/^skills:[[:space:]]*//p' "$3"
STUB
    chmod +x "$STUB_BIN/dotf"
}

# roster <row>... : the invocable table, one markdown row per argument
roster() {
    {
        printf '| Role | Phase | Skills |\n|---|---|---|\n'
        for row in "$@"; do printf '%s\n' "$row"; done
    } > "$VAULT/00_meta/agents/ROSTER.md"
}

# definition <name> <flow sequence>
definition() {
    mkdir -p "$VAULT/00_meta/agents/definitions/$1"
    printf -- '---\nkind: invocable\nskills: %s\n---\n\nbody\n' "$2" \
        > "$VAULT/00_meta/agents/definitions/$1/AGENT.md"
}

run_guard() {
    run env VAULT_PATH="$VAULT" PATH="$STUB_BIN:$PATH" python3 "$GUARD"
}

@test "roster: the same skills in a different order are consistent" {
    roster '| **curator** | Crystallize | handoff, crystallize, insights |'
    definition curator '[crystallize, insights, handoff]'
    run_guard
    [ "$status" -eq 0 ]
}

@test "roster: a skill missing from the definition is drift, naming the role" {
    roster '| **curator** | Crystallize | handoff, crystallize, insights |'
    definition curator '[crystallize, handoff, vault-doctor]'
    run_guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"curator: skills diverge"* ]] || false
}

@test "roster: a skill listed twice is drift, not a match" {
    roster '| **curator** | Crystallize | handoff, crystallize, insights |'
    definition curator '[crystallize, handoff, insights, handoff]'
    run_guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"curator: skills diverge"* ]] || false
}

@test "roster: a role bundling fewer than three skills fails" {
    roster '| **architect** | Decide | read-all-adrs, architecture-session |'
    definition architect '[read-all-adrs, architecture-session]'
    run_guard
    [ "$status" -eq 1 ]
    [[ "$output" == *"architect: bundles 2 skill(s)"* ]] || false
}
