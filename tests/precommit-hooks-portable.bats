#!/usr/bin/env bats
# Every `language: script` hook in .pre-commit-config.yaml is executed by
# pre-commit through the script's SHEBANG, and on Windows pre-commit resolves
# that shebang as a literal path: `#!/bin/bash` becomes
#   Executable `/bin/bash` not found
# and the commit is refused. The precedent guard covers one hook
# (validate-commit-msg.bats); scripts/check-bats-names.sh escaped it and blocked
# the first commit attempted from the Windows work box (2026-08-27). This guard
# derives the hook list from the config, so the next hook is covered by
# construction rather than by remembering.

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export CONFIG="$DOTFILES_DIR/.pre-commit-config.yaml"
}

# Prints the entry script of every hook declared with `language: script`
# (first token of `entry:`, so an entry carrying arguments still resolves).
script_hook_entries() {
    awk '
        /^[[:space:]]*-[[:space:]]*id:/ { entry = "" }
        /^[[:space:]]*entry:/ { entry = $2 }
        /^[[:space:]]*language:[[:space:]]*script/ { if (entry != "") print entry }
    ' "$CONFIG"
}

@test "the config declares script hooks, so the guard below checks something" {
    run script_hook_entries
    [[ $status -eq 0 ]]
    [[ -n "$output" ]]
}

@test "every script hook resolves its interpreter via env so pre-commit can run it on Windows" {
    bad=()
    while IFS= read -r entry; do
        [[ -n "$entry" ]] || continue
        first="$(head -n 1 "$DOTFILES_DIR/$entry")"
        [[ "$first" == '#!/usr/bin/env '* ]] || bad+=("$entry: $first")
    done < <(script_hook_entries)
    if (( ${#bad[@]} > 0 )); then
        printf 'hook script with a shebang pre-commit cannot resolve on Windows:\n'
        printf '  %s\n' "${bad[@]}"
        return 1
    fi
}
