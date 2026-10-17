#!/usr/bin/env bats
# GUARD: no harness config denies the commands an unattended run is allowed to
# use (TOOL-030, AC5; ADR-048 decision 4).
#
# Every harness speaks its own permission dialect, and nothing compared them: a
# deny written for one tool (`Bash(*dotf secrets show*)`) matched prose inside a
# commit body and refused a harmless call (BUG-152, #2634). The autopilot
# envelope names the commands a run may use (`allowed_commands` in
# harness/skills/autopilot/SKILL.md); this file asserts that none of them, in a
# representative harmless form, is denied by any harness config.
#
# One @test per harness, in that harness's dialect:
#   claude    permissions.deny   Bash(prefix:*) is a word-boundary prefix, any
#                                other Bash(glob) matches the whole command line
#   agy       permissions.deny   command(glob)
#   opencode  permission         a session-level bash rule; the `plan` agent's
#                                deny is a mode, not the session
#   copilot   (none)             command deny is per-session --deny-tool, so the
#                                config must not grow a persisted one unprobed
#   pi        (none)             no permission model
# A harness that cannot express a command deny is CHECKED for staying that way,
# not given a copy of the others' rules: the day it grows one, this file fails
# until the new dialect is probed too.
#
# The matcher is exercised against a config that DOES deny dotf, so a guard that
# stopped matching would fail here instead of passing on an empty result.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    PROBE_DIR="$BATS_TEST_TMPDIR"
}

# One harmless invocation per envelope command, plus the write surface TOOL-030
# adds. Bodies come from a file: the command line carries no prose.
probes() {
    printf '%s\n' \
        'dotf doctor --quick' \
        'dotf pr triage-queue --pr 1' \
        'dotf forge issue comment 1 --body-file /tmp/body.md' \
        'dotf forge pr create --draft --body-file /tmp/body.md' \
        'gh issue view 1' \
        'gh pr create --draft --body-file /tmp/body.md' \
        'git status' \
        'git push -u origin feat/x' \
        'make verify'
}

# Does the Bash-style pattern $1 (the text inside `Bash(...)`) match command $2?
# `prefix:*` is the word-boundary form; anything else is a glob on the whole line.
claude_pattern_matches() {
    local pat="$1" cmd="$2" base
    case "$pat" in
        *:\*)
            base="${pat%:\*}"
            [ "$cmd" = "$base" ] && return 0
            case "$cmd" in "$base "*) return 0 ;; esac
            return 1
            ;;
    esac
    # shellcheck disable=SC2254 # the pattern is data, a glob on purpose
    case "$cmd" in $pat) return 0 ;; esac
    return 1
}

# Prints each deny entry of dialect $1 ("Bash" or "command") in file $2, the text
# between the parentheses, one per line. A read that fails is an error, never an
# empty list.
deny_entries() {
    local kind="$1" file="$2"
    jq -er --arg k "$kind" '
        (.permissions.deny // [])[]
        | select(startswith($k + "("))
        | .[($k | length) + 1 : -1]
    ' "$file" || [ $? -eq 4 ]
}

# Prints "<deny pattern> <probe>" for every probe a deny of dialect $1 in file $2
# matches. Empty output means nothing is denied.
denied_probes() {
    local kind="$1" file="$2" pat cmd entries
    entries="$(deny_entries "$kind" "$file")" || return 2
    while IFS= read -r cmd; do
        while IFS= read -r pat; do
            [ -n "$pat" ] || continue
            if claude_pattern_matches "$pat" "$cmd"; then
                printf '%s  <-  %s\n' "$pat" "$cmd"
            fi
        done <<EOF
$entries
EOF
    done <<EOF
$(probes)
EOF
}

# The deny-free assertion every dialect test ends on.
assert_nothing_denied() {
    local denied="$1" what="$2"
    if [ -n "$denied" ]; then
        printf '%s denies a command an unattended run may use:\n%s\n' "$what" "$denied" >&2
        return 1
    fi
}

@test "envelope: every allowed command has a probe here" {
    local line cmds c
    line="$(grep -m1 '^[[:space:]]*allowed_commands:' "$REPO/harness/skills/autopilot/SKILL.md")" || false
    cmds="$(printf '%s' "$line" | sed -E 's/^[^[]*\[//; s/\].*$//' | tr ',' '\n' | tr -d ' ')"
    [ -n "$cmds" ] || false
    while IFS= read -r c; do
        [ -n "$c" ] || continue
        probes | grep -q "^$c " || {
            printf 'allowed_commands names %s but probes() has no invocation of it\n' "$c" >&2
            return 1
        }
    done <<EOF
$cmds
EOF
}

@test "matcher: a deny of dotf is seen in the claude dialect (the guard is not vacuous)" {
    printf '%s' '{"permissions":{"deny":["Bash(dotf:*)","Bash(*body-file*)"]}}' >"$PROBE_DIR/deny-dotf.json"
    run denied_probes Bash "$PROBE_DIR/deny-dotf.json"
    [ "$status" -eq 0 ]
    case "$output" in *"dotf forge issue comment"*) ;; *) echo "$output" >&2; false ;; esac
    case "$output" in *"gh pr create"*) ;; *) echo "$output" >&2; false ;; esac
}

@test "matcher: a deny of dotf is seen in the agy dialect (the guard is not vacuous)" {
    printf '%s' '{"permissions":{"deny":["command(dotf:*)"]}}' >"$PROBE_DIR/deny-agy.json"
    run denied_probes command "$PROBE_DIR/deny-agy.json"
    [ "$status" -eq 0 ]
    case "$output" in *"dotf doctor --quick"*) ;; *) echo "$output" >&2; false ;; esac
}

@test "matcher: the word boundary holds (dotf:* does not match dotfiles-sync)" {
    claude_pattern_matches 'dotf:*' 'dotf doctor'
    run claude_pattern_matches 'dotf:*' 'dotfiles-sync run'
    [ "$status" -eq 1 ]
}

@test "claude: no deny in ai/claude/settings.json matches an envelope command" {
    local denied
    denied="$(denied_probes Bash "$REPO/ai/claude/settings.json")" || false
    assert_nothing_denied "$denied" "ai/claude/settings.json"
}

@test "agy: no deny in ai/agy/settings.json matches an envelope command" {
    local denied
    denied="$(denied_probes command "$REPO/ai/agy/settings.json")" || false
    assert_nothing_denied "$denied" "ai/agy/settings.json"
}

# Counts the `deny` values a session-level opencode permission puts on bash,
# whichever form the file uses: `"permission": "deny"`, `{"bash": "deny"}` or the
# pattern object `{"bash": {"dotf *": "deny"}}`. The file is JSONC with
# full-line comments only, which is how it is kept. A file that cannot be read
# or parsed is an error, never a count of zero.
opencode_bash_denies() {
    local stripped
    [ -f "$1" ] || { echo "no such file: $1" >&2; return 2; }
    stripped="$(grep -vE '^[[:space:]]*//' "$1")" || { echo "nothing left of $1 after comments" >&2; return 2; }
    printf '%s\n' "$stripped" | jq -er '
        [ (.permission? // empty)
          | (if type == "string" then . elif type == "object" then (.bash? // empty) else empty end)
          | (if type == "object" then .[] else . end)
          | select(. == "deny") ] | length'
}

# Prints the keys of the JSON object in $1 that match the extended regex $2,
# minus any named in $3. Reading the file is checked on its own: a pipe from jq
# into grep would report grep's status, so a missing file read as "no keys".
config_keys_matching() {
    local all
    all="$(jq -r 'keys[]' "$1")" || { echo "cannot read $1" >&2; return 2; }
    [ -n "$all" ] || { echo "$1 has no keys" >&2; return 2; }
    printf '%s\n' "$all" | grep -iE "$2" | grep -vx "${3:-^$}" || true
}

@test "matcher: opencode's deny forms are all counted, and an unreadable file is an error" {
    local f="$PROBE_DIR/oc.jsonc"
    printf '%s' '{"permission":"deny"}' >"$f"
    [ "$(opencode_bash_denies "$f")" -eq 1 ]
    printf '%s' '{"permission":{"bash":"deny"}}' >"$f"
    [ "$(opencode_bash_denies "$f")" -eq 1 ]
    printf '%s' '{"permission":{"bash":{"*":"deny","git *":"allow"}}}' >"$f"
    [ "$(opencode_bash_denies "$f")" -eq 1 ]
    printf '%s' '{"permission":{"edit":"deny","bash":"allow"}}' >"$f"
    [ "$(opencode_bash_denies "$f")" -eq 0 ]
    run opencode_bash_denies "$PROBE_DIR/absent.jsonc"
    [ "$status" -eq 2 ]
    run config_keys_matching "$PROBE_DIR/absent.json" 'permission'
    [ "$status" -eq 2 ]
}

@test "opencode: the session level denies no bash (only the plan agent does)" {
    local n
    n="$(opencode_bash_denies "$REPO/ai/opencode/opencode.jsonc")" || false
    if [ "$n" -ne 0 ]; then
        echo "ai/opencode/opencode.jsonc denies bash at the session level" >&2
        return 1
    fi
}

@test "copilot: config.json persists no command deny (those are per-session --deny-tool)" {
    local keys
    keys="$(config_keys_matching "$REPO/ai/copilot/config.json" 'den(y|ied)|permission|tool' deniedUrls)" || false
    if [ -n "$keys" ]; then
        printf 'ai/copilot/config.json grew a permission key; probe it here first:\n%s\n' "$keys" >&2
        return 1
    fi
}

@test "pi: settings.json has no permission model to deny with" {
    local keys
    keys="$(config_keys_matching "$REPO/ai/pi/settings.json" 'den(y|ied)|permission|allow')" || false
    if [ -n "$keys" ]; then
        printf 'ai/pi/settings.json grew a permission key; probe it here first:\n%s\n' "$keys" >&2
        return 1
    fi
}
