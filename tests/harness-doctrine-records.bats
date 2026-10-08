#!/usr/bin/env bats
# Guards on the REAL doctrine records under harness/enforced/. The HARNESS-056
# cases in tests/compile-harness.bats prove the full-only MECHANISM on a
# synthetic fixture, and they stayed green while the real auto-merge record was
# one `--refresh` away from losing its markers (#1525): #1495 added them to the
# generated record but not to its vault source. The second case needs the real
# vault, and CI has none, so it runs wherever one resolves.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SCRIPT="$DOTFILES_DIR/scripts/compile-harness.sh"
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
        printf 'then --refresh. Editing only the record is reverted by the next refresh.\n' >&2
        return 1
    fi
}

@test "--refresh against the real vault leaves the working tree's records unchanged" {
    # A record edited without its vault source is reverted by the next
    # --refresh. That happened twice (#1490, #1525). Re-rendering every record
    # from the vault must therefore be a no-op. A vault ahead of this checkout
    # turns this red on purpose: the records have drifted from their source.
    local vault="${VAULT_PATH:-}"
    [ -n "$vault" ] || vault="$(dotf env path VAULT_PATH 2>/dev/null || true)"
    [ -n "$vault" ] && [ -d "$vault/00_meta/patterns" ] \
        || skip "no vault resolves here; compile-harness.bats pins the logic"

    local copy="$BATS_TEST_TMPDIR/repo" target targets
    mkdir -p "$copy"
    cp -R "$DOTFILES_DIR/harness" "$copy/harness"
    targets="$(jq -r '.targets[].file' "$DOTFILES_DIR/harness/manifest.json")"
    [ -n "$targets" ]
    while IFS= read -r target; do
        mkdir -p "$copy/$(dirname "$target")"
        cp "$DOTFILES_DIR/$target" "$copy/$target"
    done <<< "$targets"

    run env HOME="$BATS_TEST_TMPDIR/home" HARNESS_REPO_ROOT="$copy" VAULT_PATH="$vault" \
        "$SCRIPT" --refresh
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; return 1; }

    run diff -r "$DOTFILES_DIR/harness" "$copy/harness"
    [ "$status" -eq 0 ] || { printf 'records drift from the vault (run --refresh):\n%s\n' "$output" >&2; return 1; }
    while IFS= read -r target; do
        run diff "$DOTFILES_DIR/$target" "$copy/$target"
        [ "$status" -eq 0 ] || { printf '%s drifts from the vault:\n%s\n' "$target" "$output" >&2; return 1; }
    done <<< "$targets"
}
