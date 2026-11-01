#!/usr/bin/env bats
# Parity oracle for `dotf harness refresh` (CLI-026 slice 7): the Go refresh
# must write what scripts/compile-harness.sh --refresh writes, byte for byte,
# before the script is deleted. CI has no vault, so setup builds one back from
# the committed checkout: each enforced record under a heading whose slug is its
# manifest source, and each skill and agent record with its generated_* stamp
# removed, which is the vault source it was copied from. Both engines then
# refresh their own copy of the checkout from that vault.
#
# The Go side runs through `dotf harness refresh`, so the vault is a clone level
# with a bare upstream: the command refuses any other (#2162).
#
# This file goes with compile-harness.sh: it has no oracle once the script is
# deleted.

load 'lib/dotf-bin'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    BIN_DIR="${BATS_FILE_TMPDIR:-/tmp/bats_refresh_parity_$$}/bin"
    dotf_bin_resolve "$BIN_DIR/dotf"
    command -v jq >/dev/null 2>&1 || skip "jq is required by the bash oracle"

    FX="$BATS_TEST_TMPDIR/fx"
    mkdir -p "$FX/scripts"
    cp -R "$REPO/harness" "$FX/"
    cp "$REPO/scripts/compile-harness.sh" "$FX/scripts/"
    local target
    while IFS= read -r target; do
        mkdir -p "$FX/$(dirname "$target")"
        cp "$REPO/$target" "$FX/$target"
    done < <(jq -r '.targets[].file' "$REPO/harness/manifest.json")
    build_vault
}

git_q() { git -c user.name=t -c user.email=t@t -c init.defaultBranch=master -c commit.gpgsign=false "$@"; }

# build_vault writes the vault the committed records were refreshed from.
build_vault() {
    VAULT="$BATS_TEST_TMPDIR/vault"
    local upstream="$BATS_TEST_TMPDIR/upstream.git" id src pat slug kind sub rec
    git_q init -q --bare "$upstream"
    git_q clone -q "$upstream" "$VAULT" 2>/dev/null
    mkdir -p "$VAULT/00_meta/patterns"
    while IFS=$'\t' read -r id src; do
        pat="${src%%#*}"; slug="${src#*#}"
        # A level-1 heading: every heading a record holds is deeper than the one
        # it was cut under, so none of them ends this section early.
        printf '# %s\n' "$(printf '%s' "$slug" | tr '-' ' ')" >>"$VAULT/00_meta/patterns/$pat"
        cat "$FX/harness/enforced/$id.md" >>"$VAULT/00_meta/patterns/$pat"
    done < <(jq -r '.enforced[] | "\(.id)\t\(.source)"' "$FX/harness/manifest.json")
    jq -r '.triggers[].pattern // empty' "$FX/harness/triggers.json" | while IFS= read -r pat; do
        [ -f "$VAULT/00_meta/patterns/$pat.md" ] || printf '# %s\n' "$pat" >"$VAULT/00_meta/patterns/$pat.md"
    done
    for kind in skills:SKILL.md agents:AGENT.md; do
        sub="$(jq -r ".${kind%%:*}.vault_subpath" "$FX/harness/manifest.json")"
        mkdir -p "$VAULT/$sub"
        for rec in "$FX/harness/${kind%%:*}"/*/; do
            # No trailing slash: BSD cp copies a dir/ argument's contents, not the dir.
            cp -R "${rec%/}" "$VAULT/$sub/"
            perl -ni -e 'print unless /^generated(_from|_sha)?: /' "$VAULT/$sub/$(basename "$rec")/${kind#*:}"
        done
    done
    git_q -C "$VAULT" add -A
    git_q -C "$VAULT" commit -q -m vault
    git_q -C "$VAULT" push -q origin HEAD 2>/dev/null
}

# vault_commit records an edit to the vault and pushes it, so the Go side
# still finds the vault level with its upstream.
vault_commit() {
    git_q -C "$VAULT" add -A
    git_q -C "$VAULT" commit -q -m edit
    git_q -C "$VAULT" push -q origin HEAD 2>/dev/null
}

# refresh_both refreshes a copy of the fixture with each engine and fails
# unless they agree on the exit status and, when both succeed, on every byte
# and file mode they leave. The shared status is left in $status.
refresh_both() {
    local b="$BATS_TEST_TMPDIR/bash" g="$BATS_TEST_TMPDIR/go" bs gs
    rm -rf "$b" "$g"
    cp -R "$FX" "$b"
    cp -R "$FX" "$g"
    rm -rf "$g/scripts"
    HARNESS_REPO_ROOT="$b" VAULT_PATH="$VAULT" LC_ALL=C bash "$b/scripts/compile-harness.sh" --refresh \
        >"$BATS_TEST_TMPDIR/bash.out" 2>&1 && bs=0 || bs=$?
    "$DOTF_BIN" harness refresh --repo "$g" --vault "$VAULT" >"$BATS_TEST_TMPDIR/go.out" 2>&1 && gs=0 || gs=$?
    rm -rf "$b/scripts"
    if [ "$bs" -ne 0 ] && [ "$gs" -eq 0 ]; then
        # The Go command reports a failed render as an error (exit 1).
        printf 'bash failed (%s), go passed\n' "$bs"; cat "$BATS_TEST_TMPDIR/bash.out"; return 1
    fi
    if [ "$bs" -eq 0 ] && [ "$gs" -ne 0 ]; then
        printf 'go failed (%s), bash passed\n' "$gs"; cat "$BATS_TEST_TMPDIR/go.out"; return 1
    fi
    if [ "$bs" -eq 0 ]; then
        diff -r "$b" "$g" || { printf 'trees differ (< bash, > go)\n'; return 1; }
        diff <(cd "$b" && find . -type f -perm -u+x | LC_ALL=C sort) \
             <(cd "$g" && find . -type f -perm -u+x | LC_ALL=C sort) \
            || { printf 'executable bits differ\n'; return 1; }
    fi
    status="$bs"
}

@test "refresh parity: the vault the checkout came from refreshes it unchanged on both" {
    refresh_both
    [ "$status" -eq 0 ]
    diff -r "$FX/harness" "$BATS_TEST_TMPDIR/go/harness"
    diff "$FX/AGENTS.md" "$BATS_TEST_TMPDIR/go/AGENTS.md"
}

@test "refresh parity: section edits, a deeper heading, a new and a removed record agree" {
    local pats="$VAULT/00_meta/patterns" skills="$VAULT/00_meta/skills"
    perl -0pi -e 's/^(# 6 attribution policy\n)/$1\n### a deeper heading stays in the body\n- added rule\n/m' "$pats/pattern-git-workflow.md"
    mkdir -p "$skills/zz-new/scripts"
    printf -- '---\nid: zz-new\ntype: skill\nstatus: active\ncreated: x\nowner: t\nname: zz-new\ndescription: a new skill\n---\n\nbody\n' >"$skills/zz-new/SKILL.md"
    printf '#!/bin/sh\necho hi\n' >"$skills/zz-new/scripts/run.sh"
    chmod +x "$skills/zz-new/scripts/run.sh"
    rm -rf "$skills/handoff"
    printf '\none more line\n' >>"$VAULT/00_meta/agents/definitions/builder/AGENT.md"
    vault_commit
    refresh_both
    [ "$status" -eq 0 ]
    grep -qx -- '- added rule' "$BATS_TEST_TMPDIR/go/AGENTS.md"
    [ -x "$BATS_TEST_TMPDIR/go/harness/skills/zz-new/scripts/run.sh" ]
    [ ! -e "$BATS_TEST_TMPDIR/go/harness/skills/handoff" ]
}

# assert_go_wrote_nothing holds the Go side to the shell's failure plus one
# guarantee the shell lacked: a failed refresh leaves the checkout as it was.
assert_go_wrote_nothing() {
    diff -r -x scripts "$FX" "$BATS_TEST_TMPDIR/go"
}

@test "refresh parity: a section renamed in the vault fails on both, and Go writes nothing" {
    perl -0pi -e 's/^# 6 attribution policy$/# renamed/m' "$VAULT/00_meta/patterns/pattern-git-workflow.md"
    vault_commit
    refresh_both
    [ "$status" -ne 0 ]
    assert_go_wrote_nothing
}

@test "refresh parity: a trigger naming a pattern the vault lacks fails on both, and Go writes nothing" {
    # A trigger pattern no enforced record is cut from, so only the trigger
    # check can fail on it.
    local pat
    pat="$(jq -r --slurpfile m "$FX/harness/manifest.json" \
        '[.triggers[].pattern // empty | select((. + ".md") as $p | [$m[0].enforced[].source | split("#")[0]] | index($p) | not)][0]' \
        "$FX/harness/triggers.json")"
    [ -n "$pat" ] && [ "$pat" != null ]
    rm "$VAULT/00_meta/patterns/$pat.md"
    vault_commit
    refresh_both
    [ "$status" -ne 0 ]
    assert_go_wrote_nothing
}

@test "refresh parity: a vault skill without its required frontmatter fails on both, and Go writes nothing" {
    perl -ni -e 'print unless /^owner:/' "$VAULT/00_meta/skills/handoff/SKILL.md"
    vault_commit
    refresh_both
    [ "$status" -ne 0 ]
    assert_go_wrote_nothing
}
