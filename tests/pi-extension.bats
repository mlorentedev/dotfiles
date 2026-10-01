#!/usr/bin/env bats

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export EXT_PATH="$DOTFILES_DIR/ai/pi/extensions/dynamic-compaction.ts"
}

@test "Pi dynamic-compaction extension exists" {
    [ -f "$EXT_PATH" ]
}

@test "Pi dynamic-compaction registers agent_before_settle hook" {
    grep -q 'pi.on("agent_before_settle"' "$EXT_PATH"
}

@test "Pi dynamic-compaction computes 40% threshold" {
    grep -q 'usage.contextWindow \* 0.40' "$EXT_PATH"
}

@test "Pi dynamic-compaction calls ctx.compact" {
    grep -q 'ctx.compact(' "$EXT_PATH"
}

@test "Pi dynamic-compaction handles null usage tokens gracefully" {
    grep -q 'usage.tokens === null' "$EXT_PATH"
}
