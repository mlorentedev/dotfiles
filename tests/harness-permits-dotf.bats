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
    cmds="$(printf '%s' "$line" | sed -E 's/^[^[]*\[//; s/\].*$//; s/,/ /g')"
    [ -n "$cmds" ] || false
    for c in $cmds; do
        probes | grep -q "^$c " || {
            printf 'allowed_commands names %s but probes() has no invocation of it\n' "$c" >&2
            return 1
        }
    done
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

@test "opencode: the session level denies no bash (only the plan agent does)" {
    local f="$REPO/ai/opencode/opencode.jsonc" bash_rule
    # The file is JSONC: full-line comments only, which is how it is kept.
    bash_rule="$(grep -vE '^[[:space:]]*//' "$f" | jq -r '(.permission // {}) | if type == "object" then (.bash // "unset") else . end')" || false
    case "$bash_rule" in
        deny) echo "opencode.jsonc denies bash at the session level" >&2; false ;;
    esac
}

@test "copilot: config.json persists no command deny (those are per-session --deny-tool)" {
    local keys
    keys="$(jq -r 'keys[]' "$REPO/ai/copilot/config.json" | grep -iE 'den(y|ied)|permission|tool' | grep -vx 'deniedUrls')" || keys=""
    if [ -n "$keys" ]; then
        printf 'ai/copilot/config.json grew a permission key; probe it here first:\n%s\n' "$keys" >&2
        return 1
    fi
}

@test "pi: settings.json has no permission model to deny with" {
    local keys
    keys="$(jq -r 'keys[]' "$REPO/ai/pi/settings.json" | grep -iE 'den(y|ied)|permission|allow')" || keys=""
    if [ -n "$keys" ]; then
        printf 'ai/pi/settings.json grew a permission key; probe it here first:\n%s\n' "$keys" >&2
        return 1
    fi
}
