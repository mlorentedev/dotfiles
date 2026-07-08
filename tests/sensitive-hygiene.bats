#!/usr/bin/env bats
# GUARD-002 (#669): sensitive/env-mapping.conf was retired twice (#587/#601,
# then silently resurrected by #659) with no guard against a third comeback.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "sensitive/env-mapping.conf does not exist (retired; registry.yaml is the mapping SSOT)" {
    [ ! -f "$DOTFILES_DIR/sensitive/env-mapping.conf" ]
}

@test "no tracked file references the nonexistent docs/SECRETS.md" {
    # Pathspec exclusions: this file itself names the string as its own
    # search pattern (would otherwise fail on itself), and docs/lessons.md
    # discusses the incident in prose -- a historical mention is not a live
    # dead reference, and the guard's job is to catch the latter.
    run git -C "$DOTFILES_DIR" grep -l "docs/SECRETS.md" -- . \
        ':!tests/sensitive-hygiene.bats' ':!docs/lessons.md'
    [ "$status" -ne 0 ]
}

# --- #687: operational-script secret/policy hygiene (audit C8/C26/C27) ---
# All guards scope to '*.sh': the forbidden literals appear in this .bats file
# and in docs/audits/ as prose/patterns, so scoping to shell scripts keeps the
# guard from matching its own trigger string (the #710 lesson) without brittle
# per-path exclusions.

@test "no shell script enables gh pr merge --auto (auto-merge is forbidden repo-wide)" {
    # C8: auto-merge lands a PR the instant CI is green, bypassing human review.
    run git -C "$DOTFILES_DIR" grep -lE 'gh pr merge[^|]*--auto' -- '*.sh'
    [ "$status" -ne 0 ]
}

@test "no shell script passes a secret to gh secret set via --body (argv leak)" {
    # C26: --body puts the secret in /proc/<pid>/cmdline; pipe via stdin instead.
    run git -C "$DOTFILES_DIR" grep -lE 'gh secret set[^|]*--body' -- '*.sh'
    [ "$status" -ne 0 ]
}

@test "no shell script puts an auth token on curl argv via -H Authorization: Bearer" {
    # C26: token on argv is world-readable; use curl -K - (config via stdin).
    run git -C "$DOTFILES_DIR" grep -l -e '-H "Authorization: Bearer' -- '*.sh'
    [ "$status" -ne 0 ]
}

@test "age-encrypt-decrypt.sh sets umask 077 before writing .dec plaintext" {
    # C27: .dec are plaintext secrets; umask 077 forces 600, never world-readable.
    grep -q 'umask 077' "$DOTFILES_DIR/scripts/age-encrypt-decrypt.sh"
}

@test "age-standalone.sh sets umask 077 before writing .dec plaintext" {
    grep -q 'umask 077' "$DOTFILES_DIR/scripts/age-standalone.sh"
}
