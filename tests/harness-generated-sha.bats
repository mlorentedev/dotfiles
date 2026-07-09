#!/usr/bin/env bats
# Guard for #673/GOV-004: every HARNESS GENERATED region carries a truncated
# sha256 of its own between-markers content in the BEGIN marker. That sha is
# emitted by scripts/compile-harness.sh (sha_of = `sha256sum | cut -c1-16`) so a
# reader WITHOUT the vault can spot a hand-edited or corrupted generated block.
# Nothing enforced it before — the marker could silently go stale. This asserts
# each marker's declared sha equals the recomputed content sha.
#
# Region CONTENT drift vs the committed harness/enforced/ records is a separate,
# stronger check already run in CI (`compile-harness.sh --check`, ci.yml). This
# guard adds integrity of the self-describing sha marker itself.
#
# Note (#710 self-match): the BEGIN-marker pattern is ANCHORED at column 0 (^).
# This test file never starts a line with the marker text (references above are
# prose / assignments), so `git grep` never lists this file.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    BEGIN_RE='^<!-- BEGIN HARNESS GENERATED \(sha256:'
}

@test "every HARNESS GENERATED block's sha256 marker matches its content (#673)" {
    cd "$DOTFILES_DIR"
    local failed=0 f declared recomputed
    for f in $(git grep -lE "$BEGIN_RE" 2>/dev/null); do
        declared="$(grep -oE 'sha256:[0-9a-f]{16}' "$f" | head -1 | cut -d: -f2)"
        recomputed="$(awk '/^<!-- BEGIN HARNESS GENERATED/{inb=1;next} /^<!-- END HARNESS GENERATED -->/{inb=0} inb{print}' "$f" | sha256sum | cut -c1-16)"
        if [ "$declared" != "$recomputed" ]; then
            echo "DRIFT: $f declared=$declared recomputed=$recomputed — re-run scripts/compile-harness.sh" >&2
            failed=1
        fi
    done
    [ "$failed" -eq 0 ]
}

@test "the sha guard is not vacuously green: at least one marker exists (#673)" {
    cd "$DOTFILES_DIR"
    local n
    n="$(git grep -lE "$BEGIN_RE" 2>/dev/null | wc -l)"
    [ "$n" -ge 1 ]
}
