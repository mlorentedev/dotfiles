#!/usr/bin/env bats
# test-windows restores the pinned pi CLI on a pull request instead of
# reinstalling it from npm, and only main writes that cache.
#
# CI-010 (#2047) cached the pi CLI: ~130 s of every PR's test-windows (p50
# 508 s, 2026-10-07). #2059 added the three npm CLIs `dotf tools install`
# pinned in packages.json (bw, opencode, copilot); #2438 moved them to mise's
# npm backend, which installs under mise's data directory, so the cache holds
# pi only and its key no longer names packages.json or the installer.
#
# What these tests hold:
# - the key names every input, so a PR that changes any of them misses and
#   runs the fresh install itself: the runner image, PI_VERSION and
#   setup-windows.ps1;
# - the match is exact (no restore-keys), so a stale install never stands in;
# - only a push to main saves, as the job's last step, so the cache is always a
#   fresh install from a fully green run, never one restored and re-saved;
# - a npm tool that packages.json pins again must join the cache or be named as
#   left out.
#
# Asserted on the workflow text, because nothing here can run GitHub's runner.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CI="$REPO/.github/workflows/ci.yml"
    # The npm CLI cache steps only, comments dropped so a comment naming the old
    # form cannot satisfy or trip a check.
    STEPS=$(awk '/name: Resolve the npm CLI cache key/,/^  integration:/' "$CI" | grep -vE '^[[:space:]]*#')
    RESTORE=$(printf '%s\n' "$STEPS" | awk '/uses: actions\/cache\/restore@/,/key:/')
    SAVE=$(printf '%s\n' "$STEPS" | awk '/name: Save the npm CLIs/,/key:/')
}

@test "ci: the npm CLI cache steps exist, restore before setup and save as the job's last step" {
    [ -n "$STEPS" ] || { echo "no 'Resolve the npm CLI cache key' step in ci.yml" >&2; return 1; }
    # Scoped to the test-windows job, comments dropped: a comment or a
    # same-named step in another job must not reorder or pad this list.
    order=$(awk '/^  test-windows:/,/^  integration:/' "$CI" | grep -vE '^[[:space:]]*#' |
        grep -E 'name: (Resolve the npm CLI cache key|Restore the pinned npm CLIs|Run setup-windows.ps1|Post-setup doctor gate|Run PowerShell bats subset|Save the npm CLIs)' |
        sed 's/^ *- name: //')
    expected=$(printf '%s\n' \
        'Resolve the npm CLI cache key' \
        'Restore the pinned npm CLIs (pull requests; a lookup on main)' \
        'Run setup-windows.ps1 end-to-end (PS 5.1 -> BUG-005 re-exec)' \
        'Post-setup doctor gate (TEST-003)' \
        'Run PowerShell bats subset (Git Bash, pinned bats)' \
        'Save the npm CLIs that main installed')
    [ "$order" = "$expected" ] || { printf 'step order:\n%s\n' "$order" >&2; return 1; }
    after=$(printf '%s\n' "$STEPS" | awk '/name: Save the npm CLIs/{f=1;next} f' | grep -cE '^[[:space:]]*- (name|uses):' || true)
    [ "$after" -eq 0 ] || { echo "$after step(s) after the save: a later failure would not stop a cache write" >&2; return 1; }
}

@test "ci: the npm CLI cache key names the runner image, PI_VERSION and setup-windows.ps1" {
    printf '%s\n' "$STEPS" | grep -q "PI_VERSION="
    for part in "runner.os" "steps.picli.outputs.image" "steps.picli.outputs.version" "hashFiles('setup-windows.ps1')"; do
        printf '%s\n' "$RESTORE" | grep -E 'key: npm-clis-' | grep -qF "$part" || { echo "restore key lacks $part" >&2; return 1; }
        printf '%s\n' "$SAVE" | grep -E 'key: npm-clis-' | grep -qF "$part" || { echo "save key lacks $part" >&2; return 1; }
    done
}

@test "ci: the npm CLI restore is an exact match and downloads nothing on main" {
    printf '%s\n' "$RESTORE" | grep -qE 'uses: actions/cache/restore@[0-9a-f]{40}'
    if printf '%s\n' "$STEPS" | grep -q 'restore-keys'; then
        echo "restore-keys would let a stale install stand in for the pinned one" >&2
        return 1
    fi
    printf '%s\n' "$RESTORE" | grep -qE "lookup-only: \\$\\{\\{ github.event_name == 'push' \\}\\}"
}

@test "ci: only a push to main saves the npm CLI cache, and only on a miss" {
    printf '%s\n' "$SAVE" | grep -qE 'uses: actions/cache/save@[0-9a-f]{40}'
    cond=$(printf '%s\n' "$SAVE" | grep -E '^[[:space:]]*if:')
    # Every step's HARNESS-041 guard already names `push` inside an OR, which
    # admits a PR. Drop it, then require `push` as a conjunct of its own.
    narrowed=${cond/"(github.event_name == 'push' || needs.changes.outputs.code == 'true')"/}
    printf '%s\n' "$narrowed" | grep -qF "&& github.event_name == 'push' &&" || { echo "the save can run outside a push to main: $cond" >&2; return 1; }
    printf '%s\n' "$narrowed" | grep -qF "steps.picli-cache.outputs.cache-hit != 'true'"
    # A status function replaces the implicit success(): with one, a red or
    # cancelled main run would write the cache every PR then restores.
    if printf '%s\n' "$cond" | grep -qE '(always|failure|cancelled)\(\)'; then
        echo "the save must not run after a failed step: $cond" >&2
        return 1
    fi
}

@test "ci: the npm CLI cache holds pi and its shims, not the whole npm prefix" {
    paths=$(printf '%s\n' "$RESTORE" | awk '/path: \|/{f=1;next} /key:/{f=0} f' | sed 's/^[[:space:]]*//')
    [ "$(printf '%s\n' "$paths" | wc -l | tr -d ' ')" -eq 4 ] || { printf 'paths:\n%s\n' "$paths" >&2; return 1; }
    printf '%s\n' "$paths" | grep -qxF "\${{ steps.picli.outputs.prefix }}\\node_modules\\@earendil-works\\pi-coding-agent" || { echo "missing package pi" >&2; return 1; }
    for shim in pi pi.cmd pi.ps1; do
        printf '%s\n' "$paths" | grep -qxF "\${{ steps.picli.outputs.prefix }}\\$shim" || { echo "missing shim $shim" >&2; return 1; }
    done
    saved=$(printf '%s\n' "$SAVE" | awk '/path: \|/{f=1;next} /key:/{f=0} f' | sed 's/^[[:space:]]*//')
    [ "$saved" = "$paths" ] || { printf 'save paths differ from restore paths:\n%s\n' "$saved" >&2; return 1; }
}

@test "ci: a npm tool packages.json pins is cached" {
    # The catalog holds no npm tool since #2438 moved bw, copilot, opencode and
    # yarn to mise; one that returns must join the cache, so the cache cannot
    # fall behind the pins unnoticed.
    #
    # With no npm tool the query has nothing to match, so a typo in it, or a
    # renamed `source.type`, would read as "none" too. Hold it to the catalog's
    # shape first: every tool carries a string type, and every type is one the
    # loader knows, so the select below is a query against a field that exists.
    [ "$(jq -r '.tools | length' "$REPO/packages.json")" -gt 0 ] || { echo "packages.json lists no tool: the jq path is wrong" >&2; return 1; }
    jq -e 'all(.tools[]; (.source.type | type == "string") and (.source.type | IN("system", "npm", "github-release", "github-archive", "uv-tool")))' "$REPO/packages.json" >/dev/null ||
        { echo "a catalog tool has no known source.type: update this test with the loader" >&2; return 1; }
    pkgs=$(jq -r '.tools[] | select(.source.type == "npm") | .name + " " + .source.package' "$REPO/packages.json")
    [ -z "$pkgs" ] && return 0
    while read -r name pkg; do
        win=$(printf '%s' "$pkg" | tr '/' '\\')
        printf '%s\n' "$RESTORE" | grep -qF "node_modules\\$win" || { echo "packages.json pins $pkg ($name) but the cache does not hold it" >&2; return 1; }
    done <<EOF2
$pkgs
EOF2
}
