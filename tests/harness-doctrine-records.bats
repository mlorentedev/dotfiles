#!/usr/bin/env bats
# Guards on the REAL doctrine records under harness/enforced/. The Go tests in
# cli/internal/harness prove the full-only MECHANISM on a synthetic fixture, and
# its shell predecessor stayed green while the real auto-merge record was one
# refresh away from losing its markers (#1525): #1495 added them to the
# generated record but not to its vault source. The second case needs the real
# vault, and CI has none, so it runs wherever one resolves.

load 'lib/dotf-bin'

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
}

@test "no-auto-merge: the auto-merge exception sits inside a full-only region" {
    # The markers keep a region out of the capped compact payload that agy and
    # codex receive instead of the constitution. That payload must carry the
    # prohibition, never the exception: an agent that knows an exception
    # exists may try to qualify for it.
    local record="$DOTFILES_DIR/harness/enforced/no-auto-merge.md" begin end exception
    begin="$(grep -n -m1 -F '<!-- full-only:begin -->' "$record" | cut -d: -f1)"
    end="$(grep -n -m1 -F '<!-- full-only:end -->' "$record" | cut -d: -f1)"
    exception="$(grep -n -m1 -F 'One exception' "$record" | cut -d: -f1)"
    if [ -z "$begin" ] || [ -z "$end" ] || [ -z "$exception" ] \
        || [ "$begin" -ge "$exception" ] || [ "$exception" -ge "$end" ]; then
        printf '%s must wrap its "One exception" paragraph in full-only markers\n' "$record" >&2
        printf '(begin=%s exception=%s end=%s). Without them the exception ships\n' \
            "${begin:-none}" "${exception:-none}" "${end:-none}" >&2
        printf 'into the capped payload next to the prohibition it qualifies.\n' >&2
        printf 'Fix the vault source (00_meta/patterns/pattern-git-workflow.md),\n' >&2
        printf 'then dotf harness refresh. Editing only the record is reverted by the next refresh.\n' >&2
        return 1
    fi
}

@test "the committed records match what a refresh would render from the real vault" {
    # A record edited without its vault source is reverted by the next refresh.
    # That happened twice (#1545, #1525). `dotf harness check --against-vault`
    # renders every record from the vault without writing anything, and fails on
    # either side: a vault ahead of this checkout, or a record edited by hand.
    local vault="${VAULT_PATH:-}"
    [ -n "$vault" ] || vault="$(dotf env path VAULT_PATH 2>/dev/null || true)"
    [ -n "$vault" ] && [ -d "$vault/00_meta/patterns" ] \
        || skip "no vault resolves here; the Go tests pin the logic"
    dotf_bin_resolve "${BATS_FILE_TMPDIR}/dotf-doctrine"

    run "$DOTF_BIN" harness check --repo-root "$DOTFILES_DIR" --against-vault --vault "$vault"
    [ "$status" -eq 0 ] || { printf 'records drift from the vault:\n%s\n' "$output" >&2; return 1; }
}
