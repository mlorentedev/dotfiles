#!/usr/bin/env bats
# TOOL-030 AC5 (#2635): no harness config the repo ships denies, wholesale, a
# command the autopilot envelope allows. The dialects differ (ADR-048), so each
# harness is one @test reading its own deploy source; a command denied in one
# dialect and not the others would otherwise drift apart unseen.
#
# "Wholesale" is the point. A subcommand deny such as Claude's
# `Bash(*dotf secrets show*)` is deliberate and passes; `Bash(dotf:*)`,
# `Bash(*dotf*)` or `command(dotf*)` denies the whole command and fails.
#
# The guard reads the repo's sources. A deny a machine holds locally (the
# deploy unions lists and never narrows, ADR-048 ¶5) is dotf doctor's to
# report, not this file's. When #1852 gives harness/capability-map.json a
# session-level section, these leaves are what it renders, and this guard
# moves there.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    AUTOPILOT_SKILL="${AUTOPILOT_SKILL:-$REPO/harness/skills/autopilot/SKILL.md}"
    CLAUDE_SETTINGS="${CLAUDE_SETTINGS:-$REPO/ai/claude/settings.json}"
    AGY_SETTINGS="${AGY_SETTINGS:-$REPO/ai/agy/settings.json}"
    OPENCODE_CFG="${OPENCODE_CFG:-$REPO/ai/opencode/opencode.jsonc}"
    PI_SETTINGS="${PI_SETTINGS:-$REPO/ai/pi/settings.json}"
    COPILOT_CONFIG="${COPILOT_CONFIG:-$REPO/ai/copilot/config.json}"
    COPILOT_SCAN_ROOT="${COPILOT_SCAN_ROOT:-$REPO}"
}

# The envelope's allowed_commands, read from the skill that defines it, so the
# list has one home. Printed as a JSON array.
envelope_commands() {
    sed -n 's/^[[:space:]]*allowed_commands:[[:space:]]*\[\([^]]*\)\].*/\1/p' "$AUTOPILOT_SKILL" \
        | head -n 1 | tr -d ' ' | jq -Rc 'split(",") | map(select(length > 0))'
}

# jq: true when the pattern inside a deny entry denies $c as a whole command.
# shellcheck disable=SC2016 # a jq program: $c is jq's variable, not the shell's
WHOLESALE='def wholesale($c): . == "*" or test("^\\*?[[:space:]]*" + $c + "(\\*|:\\*|[[:space:]]+\\*)?$");'

# Fails naming every (command, entry) pair a list of deny patterns denies.
assert_none_denied() { # $1 harness, $2 JSON array of deny patterns
    run jq -nr --argjson cmds "$(envelope_commands)" --argjson deny "$2" "$WHOLESALE"'
        [ $cmds[] as $c | $deny[] | select(wholesale($c)) | "\($c): \(.)" ] | .[]'
    [ "$status" -eq 0 ] || { echo "jq failed: $output"; false; }
    [ -z "$output" ] || { echo "$1 denies an envelope command wholesale:"; echo "$output"; false; }
}

@test "the envelope's allowed_commands is read from the autopilot skill" {
    run envelope_commands
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    run jq -e --argjson c "$output" -n '$c | length > 0 and all(test("^[a-z][a-z0-9-]*$"))'
    [ "$status" -eq 0 ] || { echo "allowed_commands not found or not plain command words in $AUTOPILOT_SKILL"; false; }
}

@test "claude: no permissions.deny entry denies an envelope command" {
    deny="$(jq -c '[.permissions.deny[]? | select(startswith("Bash(") and endswith(")")) | .[5:-1]]' "$CLAUDE_SETTINGS")"
    assert_none_denied claude "$deny"
}

@test "agy: no permissions.deny entry denies an envelope command" {
    deny="$(jq -c '[.permissions.deny[]? | select(startswith("command(") and endswith(")")) | .[8:-1]]' "$AGY_SETTINGS")"
    assert_none_denied agy "$deny"
}

@test "opencode: bash is not denied for an envelope command, except in the plan agent" {
    # opencode.jsonc carries whole-line comments and trailing commas, read as
    # tests/nan-context-windows.bats reads it. `plan` is the documented
    # read-only agent (edit and bash denied); any other agent denying bash fails.
    run python3 - "$OPENCODE_CFG" <<'PY'
import json, re, sys
src = "".join(l for l in open(sys.argv[1]) if not l.lstrip().startswith("//"))
cfg = json.loads(re.sub(r",(\s*[}\]])", r"\1", src))
def denied(perm):
    if perm == "deny":
        return ["*"]
    if not isinstance(perm, dict):
        return []
    bash = perm.get("bash")
    if bash == "deny":
        return ["*"]
    if isinstance(bash, dict):
        return [p for p, v in bash.items() if v == "deny"]
    return []
out = {"(top level)": denied(cfg.get("permission"))}
for name, agent in (cfg.get("agent") or {}).items():
    if name != "plan":
        out[name] = denied(agent.get("permission"))
print(json.dumps(out))
PY
    [ "$status" -eq 0 ] || { echo "$output"; false; }
    scopes="$output"
    for scope in $(printf '%s' "$scopes" | jq -r 'keys[] | @base64'); do
        name="$(printf '%s' "$scope" | base64 --decode)"
        assert_none_denied "opencode $name" "$(printf '%s' "$scopes" | jq -c --arg k "$name" '.[$k]')"
    done
}

@test "copilot: config.json persists no command deny, and no shipped wrapper passes --deny-tool for an envelope command" {
    # Copilot keeps command denies per session (--deny-tool), not in
    # config.json (ADR-048 ¶4), so the surface is any file that passes the flag.
    run jq -e '[paths | map(tostring) | join(".") | select(test("(?i)den(y|ied).*tool|tool.*den(y|ied)"))] | length == 0' "$COPILOT_CONFIG"
    [ "$status" -eq 0 ] || { echo "config.json declares a command deny key; extend this guard to read it"; false; }
    # A quoted spec is read to its closing quote, so a scoped deny such as
    # `shell(git push --force)` is not truncated to a wholesale-looking `git`.
    run git -C "$COPILOT_SCAN_ROOT" grep -h -o -I -E -e "--deny-tool[= ]+(\"[^\"]*\"|'[^']*'|[^'\" ]+)" -- . ':!tests/harness-permits-dotf.bats'
    # 1 is "no match"; anything else is git failing to look, which must not read as "nothing denied".
    [ "$status" -le 1 ] || { echo "git grep failed in $COPILOT_SCAN_ROOT (exit $status): $output"; false; }
    deny="$(printf '%s\n' "$output" | sed -E "s/^--deny-tool[= ]+//; s/^[\"']//; s/[\"']\$//; s/^shell\\(//; s/\\)\$//" | jq -Rc '[inputs, . | select(length > 0)] | unique')"
    assert_none_denied copilot "$deny"
}

@test "pi: settings.json declares no permission model, the gap ADR-048 records" {
    # pi runs every tool without asking. The day its settings grow a deny list
    # this test fails, and the list must be read here like the others.
    run jq -e 'has("permissions") or has("permission") or has("deny") | not' "$PI_SETTINGS"
    [ "$status" -eq 0 ] || { echo "pi settings.json now carries permissions; read its deny list in this guard"; false; }
}
