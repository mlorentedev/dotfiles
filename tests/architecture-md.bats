#!/usr/bin/env bats
# Drift guard for docs/architecture.md (CLI-002, #336): the declared tree
# must track the real one, or the doc rots into fiction.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export ARCH_MD="$DOTFILES_DIR/docs/architecture.md"
}

@test "docs/architecture.md exists" {
    [[ -f "$ARCH_MD" ]]
}

@test "README links docs/architecture.md" {
    grep -qF 'docs/architecture.md' "$DOTFILES_DIR/README.md"
}

@test "every real top-level dir is declared in the architecture table" {
    local missing="" d name
    for d in "$DOTFILES_DIR"/*/; do
        name=$(basename "$d")
        grep -qF "\`$name/\`" "$ARCH_MD" || missing="$missing $name"
    done
    if [[ -n "$missing" ]]; then
        echo "top-level dirs missing from docs/architecture.md table:$missing"
        return 1
    fi
}

@test "architecture.md declares the cli/ standard layout (cmd + internal)" {
    grep -qF 'cmd/dot' "$ARCH_MD"
    grep -qF 'internal/' "$ARCH_MD"
}

@test "architecture.md points to the language boundary SSOT instead of duplicating it" {
    grep -qF 'Language Boundary' "$ARCH_MD"
    grep -qF 'AGENTS.md' "$ARCH_MD"
}
