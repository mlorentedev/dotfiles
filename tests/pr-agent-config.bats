#!/usr/bin/env bats
# TOOL-013 (#786): the PR-Agent reviewer's configuration.
#
# These assert the DECISIONS, not the syntax. Each one encodes a choice that
# would be easy to undo in a passing-looking edit, and expensive to discover
# afterwards — a reviewer that quietly rubber-stamps, or that quietly reads
# credential material.

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CFG="$REPO/.pr_agent.toml"
    WF="$REPO/.github/workflows/pr-agent.yml"
}

refute_grep() {
    local pattern="$1" file="$2"
    if grep -qE "$pattern" "$file"; then
        printf 'expected NOT to find /%s/ in %s, but it is there:\n' "$pattern" "$file" >&2
        grep -nE "$pattern" "$file" >&2
        return 1
    fi
}

@test "pr-agent: the config and workflow both exist" {
    [ -f "$CFG" ]
    [ -f "$WF" ]
}

@test "pr-agent: never sends sensitive/ to an inference endpoint" {
    # The load-bearing assertion in this file. sensitive/ holds age ciphertext
    # and the DR escrow; encrypted or not, credential material is not shipped to
    # a model to be reviewed. The path filter runs before the model call, which
    # is the only point at which the diff leaves this infrastructure.
    grep -q '"sensitive/\*\*"' "$CFG"
}

@test "pr-agent: does not fall back to a latency-optimised model" {
    # harness/reviewer-pool.json excludes qwen3.6 from the adversarial-review
    # pool by name — "a reviewer that PASSes cheaply is worse than no gate,
    # because it converts the gate into a green checkmark". #786's draft proposed
    # exactly that model as a fallback here, which would leave two files in this
    # repo holding opposite policies on who may review.
    #
    # Safe because GUARD-002 exists: a failed review shows as `declined` and the
    # PR goes red, so absence is loud. A cheap fallback would trade a loud
    # absence for a quiet rubber stamp.
    grep -qE '^fallback_models = \[\]' "$CFG"
    refute_grep '^fallback_models = \[[^]]' "$CFG"
}

@test "pr-agent: the reviewing model is the one measured, not the fast one" {
    grep -q 'model = "openai/deepseek-v4-flash"' "$CFG"
}

@test "pr-agent: the token budget is stated, because NaN models are not in LiteLLM's registry" {
    # Without this LiteLLM assumes a small default and silently truncates the
    # diff — a reviewer reading half a change reports on half a change.
    grep -qE '^custom_model_max_tokens = [0-9]+' "$CFG"
    grep -qE '^max_model_tokens = [0-9]+' "$CFG"
}

@test "pr-agent: response language is English per English-only durable record policy" {
    grep -q 'response_language = "en-US"' "$CFG"
}

@test "pr-agent: AGENTS.md enters the review prompt" {
    # The repo's behavioural SSOT becomes review criteria for free: standing
    # orders, English-only, no-auto-merge, the shell compatibility table.
    #
    # Asserted as membership, not as the exact literal. The original form was
    # `grep -q 'repo_context_files = ["AGENTS.md"]'`, which pinned the whole list
    # rather than the property it names, so ADDING a second context file broke a
    # guard whose subject had not changed. A guard that fails when its property
    # still holds trains people to edit the guard, which is how a real one gets
    # weakened later.
    run python3 -c "
import sys, tomllib
files = tomllib.load(open('$CFG', 'rb'))['config']['repo_context_files']
sys.exit(0 if 'AGENTS.md' in files else 1)
"
    [ "$status" -eq 0 ]
}

@test "pr-agent: the workflow pins the action to an immutable commit, not a tag" {
    # The old form of this test asserted `@v[0-9]` while its own title said "not a
    # moving ref" — and a tag IS a moving ref: it belongs to upstream, and whoever
    # can move it decides what runs in a job holding NAN_API_KEY and
    # `pull-requests: write`, on a PUBLIC repository. The test encoded the weaker
    # property its name disclaimed, which is why a tag pin survived review.
    #
    # 40 hex characters, checked as such rather than by pattern-matching a version:
    # the point is immutability, and only a commit id has it.
    local ref
    ref="$(grep -oE 'uses: The-PR-Agent/pr-agent@[0-9a-zA-Z._-]+' "$WF" | head -1 | cut -d@ -f2)"
    [ -n "$ref" ] || { printf 'the workflow does not reference the action at all\n' >&2; return 1; }
    if ! printf '%s' "$ref" | grep -qE '^[0-9a-f]{40}$'; then
        printf 'action pinned to %q — expected a 40-character commit id\n' "$ref" >&2
        return 1
    fi
    # The comment beside it must still say which release that commit is, or the pin
    # becomes unreadable and the next bump is made blind.
    grep -qE '@[0-9a-f]{40}\s+# v[0-9]' "$WF"
}

@test "pr-agent: the comment trigger is gated on repository membership" {
    # PUBLIC repository: any account can comment on a pull request, and
    # issue_comment runs in the BASE repo context WITH secrets. Without this gate a
    # stranger typing a slash command spends the inference budget and exercises
    # `pull-requests: write` on demand.
    grep -q 'github.event.comment.author_association' "$WF"
    for assoc in OWNER MEMBER COLLABORATOR; do
        grep -q "$assoc" "$WF"
    done
}

@test "pr-agent: the endpoint is NaN, and no OpenAI credential is referenced" {
    grep -q 'OPENAI__API_BASE: https://api.nan.builders/v1' "$WF"
    grep -q 'secrets.NAN_API_KEY' "$WF"
    # The `openai/` model prefix and OPENAI__ env names are LiteLLM transport
    # selectors, not a provider choice. Nothing here should reach for a real
    # OpenAI key.
    refute_grep 'secrets.OPENAI_API_KEY' "$WF"
}

@test "pr-agent: the workflow takes exactly one secret" {
    # #1025: the spec-review path injects the WHOLE registry to authenticate one
    # model, so one broken item mapping takes down authentication for
    # everything. This path must not inherit that shape — the failure surface
    # stays one key wide.
    local n
    n=$(grep -cE '\$\{\{ *secrets\.[A-Z_]+ *\}\}' "$WF")
    [ "$n" -eq 2 ] || {
        printf 'expected exactly 2 secret references (GITHUB_TOKEN + NAN_API_KEY), found %s:\n' "$n" >&2
        grep -nE '\$\{\{ *secrets\.[A-Z_]+ *\}\}' "$WF" >&2
        return 1
    }
}

@test "pr-agent: NAN_API_KEY is declared as a CI consumer of this repo" {
    # Without this, `dotf secrets sync ci` will not deliver the key and the
    # workflow authenticates with an empty string.
    python3 -c "
import sys, yaml
d = yaml.safe_load(open('$REPO/secrets/registry.yaml'))
n = [s for s in d['secrets'] if s['id'] == 'NAN_API_KEY'][0]
sys.exit(0 if 'ci:mlorentedev/dotfiles' in n['consumers'] else 1)
"
}

@test "pr-agent: draft PRs are skipped" {
    # A draft is by definition not asking to be read yet; reviewing it spends
    # inference on a change the author has not finished making.
    grep -q 'draft == false' "$WF"
}

@test "pr-agent: a comment cannot cancel an in-flight review" {
    # The defect this encodes (#1040): a PR comment is an issue_comment whose
    # issue.number IS the PR number. With the group keyed on the number alone, a
    # comment-triggered run landed in the same concurrency group as the running
    # pull_request run and cancel-in-progress killed it. PR-Agent is a Docker
    # action; CodeRabbit's auto-summary comment reliably arrives during the build.
    #
    # Measured before the fix on #1037 and #1038: both cancelled mid-build, both
    # workflows green, zero reviews. It had been broken since the hour it merged
    # and nothing noticed, because the only symptom is a cancelled run.
    #
    # Asserted on the group EXPRESSION rather than on behaviour, because nothing
    # here can run GitHub's scheduler. What it protects is the discriminator:
    # remove github.event_name from the key and the two event types collide again.
    grep -qE '^\s*group:.*github\.event_name' "$WF"
}

@test "pr-agent: both PR-number sources stay in the concurrency key" {
    # The event_name suffix above only helps while the key still identifies the
    # PR. A "simplification" that drops either source re-breaks it differently:
    # without issue.number every slash-command run shares one group, without
    # pull_request.number every push does.
    local group
    group=$(grep -E '^\s*group:' "$WF")
    printf '%s\n' "$group" | grep -q 'github.event.pull_request.number'
    printf '%s\n' "$group" | grep -q 'github.event.issue.number'
}

@test "pr-agent: every event the workflow fires on is handled by something" {
    # #1053. Two event lists in one file that must agree, and nothing compared
    # them: the workflow asked for `synchronize`, PR-Agent's pr_actions default
    # excludes it, so every push started a runner, built the Docker action and
    # reviewed nothing. Measured on #1048 — 4 pull_request runs, 1 artifact.
    #
    # The cost was never inference; pushes make no model calls. It was that the
    # run reported `review: SUCCESS`, which reads as "your new commits were
    # reviewed" and means "the action declined to act". GUARD-002 exists because
    # that green is worse than a red one.
    #
    # The invariant is NOT set equality any more, and the change is a widening of
    # the same finding rather than a weakening of it. Upstream routes
    # `synchronize` down a separate path — "handle_push_trigger = false # when
    # true, handle synchronize events using push_commands" — so under equality a
    # correct configuration is unrepresentable, and the obvious repair (adding
    # synchronize to pr_actions) is a setting that asserts nothing.
    #
    # What must hold is what #1053 actually found: nothing in the workflow's
    # trigger list may go unhandled. Every type is handled by pr_actions, except
    # synchronize, which is handled if and only if handle_push_trigger is true.
    run python3 -c "
import json, sys, yaml
d = yaml.safe_load(open('$WF'))
types = set(d[True]['pull_request']['types'])
step = next(s for s in d['jobs']['review']['steps'] if 'pr-agent' in s.get('uses', ''))
env = step['env']
actions = set(json.loads(env['github_action_config.pr_actions']))
push_on = str(env.get('github_action_config.handle_push_trigger', 'false')).lower() == 'true'

handled = set(actions)
if push_on:
    handled.add('synchronize')
unhandled = sorted(types - handled)
if unhandled:
    print('the workflow fires on events nothing handles: ' + ', '.join(unhandled))
    sys.exit(1)
# The reverse direction still matters: a handler for an event the workflow never
# fires on is dead configuration that reads as coverage.
orphan = sorted(actions - types)
if orphan:
    print('pr_actions handles events the workflow never fires: ' + ', '.join(orphan))
    sys.exit(1)
if 'synchronize' in types and not push_on:
    print('synchronize fires but handle_push_trigger is off: every push builds and reviews nothing')
    sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# Enabling the push path without naming its commands silently reinstates
# `describe`, whose default is ['/describe', '/review'] — and describe rewrites
# the PR body, which this repo turned OFF as a decision twelve lines above. A
# default that quietly re-enables what you disabled elsewhere is the same defect
# one layer down.
#
# Asserted as EXACT equality with ["/review"], after a reviewer pointed out that
# the first version only rejected /describe — so it passed on `[]` and on
# ["/improve"]. `[]` is the dangerous one: the push trigger would fire, run
# nothing, and this guard would stay green while the feature it protects was
# entirely inert. A guard whose green survives the death of its subject is the
# defect this whole file exists to catch, written into the file itself.
#
# Exact rather than "contains /review" on purpose: adding a command to the push
# path costs a second inference call on every push, which is a decision worth
# forcing through this line rather than letting it arrive as an edit nobody
# weighs.
@test "pr-agent: the push path runs exactly /review, no more and no less" {
    run python3 -c "
import json, sys, yaml
d = yaml.safe_load(open('$WF'))
step = next(s for s in d['jobs']['review']['steps'] if 'pr-agent' in s.get('uses', ''))
env = step['env']
if str(env.get('github_action_config.handle_push_trigger', 'false')).lower() != 'true':
    sys.exit(0)
raw = env.get('github_action_config.push_commands')
if raw is None:
    print('handle_push_trigger is on with no push_commands: describe returns by default')
    sys.exit(1)
cmds = [c.strip() for c in json.loads(raw)]
if cmds != ['/review']:
    print(f'push_commands is {cmds!r}, want exactly [\'/review\']')
    if '/describe' in cmds:
        print('  /describe rewrites the PR body, turned off deliberately')
    if not cmds:
        print('  an empty list makes the push trigger fire and do nothing')
    sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "pr-agent: describe does not rewrite the PR body" {
    # A model rewriting a body that carries hand-written measurement tables,
    # merge orders and before/after evidence destroys the part worth reading.
    # Off as a decision; pinned so it cannot drift back on a default.
    grep -q 'github_action_config.auto_describe: "false"' "$WF"
}

# The reviewer cannot read a file that is not in repo_context_files, so an
# instruction naming one asserts nothing. This was real: extra_instructions told
# the reviewer to consult the prohibited-pattern table in .claude/CLAUDE.md while
# only AGENTS.md was in context. The guard is mechanical — every repo-relative
# path mentioned in extra_instructions must also be loaded.
@test "pr-agent: every file the instructions name is a file the reviewer can read" {
    run python3 - "$CFG" <<'PY'
import re, sys, tomllib

cfg = tomllib.load(open(sys.argv[1], "rb"))
loaded = set(cfg["config"]["repo_context_files"])
instructions = cfg["pr_reviewer"]["extra_instructions"]

# Repo-relative paths: a dotted name containing a '/' or a leading '.', which is
# how this repo writes them (AGENTS.md, .claude/CLAUDE.md, specs/<id>/).
named = set(re.findall(r"(?:^|\s)((?:\.[\w.-]+/)?[\w.-]+\.(?:md|json|toml|sh|yml))", instructions))
missing = sorted(n for n in named if n not in loaded)
if missing:
    print("named but not loaded: " + ", ".join(missing))
    sys.exit(1)
PY
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The point of the harness pass is that it runs on EVERY PR without being asked.
# An edit that turns it into an opt-in ("when relevant", "if applicable") would
# leave the section looking present while it silently stops firing.
@test "pr-agent: the harness compliance pass is unconditional" {
    grep -q 'HARNESS COMPLIANCE' "$CFG"
    grep -q 'Report it even when everything passes' "$CFG"
    refute_grep 'HARNESS COMPLIANCE.*(if |when relevant|where applicable)' "$CFG"
}

# The reason this tool was adopted is inline comments on the diff — the half
# CodeRabbit's free tier withholds on private repos. That claim sat in a
# `[pr_reviewer]` comment for the tool's whole life while dual publishing was at
# its default of -1 (disabled), and 0 inline comments were posted across #1042,
# #1047 and #1051. Pinned as a decision, not a value: any threshold in [0-10]
# publishes inline; -1 or a missing section silently stops.
@test "pr-agent: inline suggestions are actually enabled, not merely claimed" {
    run python3 -c "
import sys, tomllib
cfg = tomllib.load(open('$CFG', 'rb'))
s = cfg.get('pr_code_suggestions')
if s is None:
    print('no [pr_code_suggestions] section: improve runs on defaults, inline disabled'); sys.exit(1)
t = s.get('dual_publishing_score_threshold', -1)
if not (0 <= t <= 10):
    print(f'dual_publishing_score_threshold={t} does not publish inline'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The linters own style in this repo and extra_instructions forbids restating
# them. A score floor of 0 (the upstream default) publishes everything, which
# reintroduces exactly that noise through a different door.
@test "pr-agent: suggestions below the style line are not published" {
    run python3 -c "
import sys, tomllib
s = tomllib.load(open('$CFG', 'rb')).get('pr_code_suggestions', {})
t = s.get('suggestions_score_threshold', 0)
sys.exit(0 if 1 <= t <= 8 else 1)
"
    [ "$status" -eq 0 ]
}

# The two halves must move together or the change makes things worse: excluding
# release PRs from the reviewer WITHOUT exempting them from the attestation gate
# produces a PR with no reviewer output that nothing can ever clear — #1061's
# unreachable state, manufactured deliberately.
#
# This guard spans two files because the coupling does. Neither file can express
# it alone, and a guard that lived in either would pass while the other half was
# removed — which is exactly what happened when this was first written: deleting
# the reviewer exclusion broke nothing.
@test "pr-agent: excluding release PRs from review is paired with a gate exemption" {
    # The exclusion lives in the workflow's job-level `if`, NOT in .pr_agent.toml's
    # ignore_pr_source_branches. That key is applied on upstream's webhook path and
    # is inert for the Action entrypoint, so it read like a decision and asserted
    # nothing (#1073, measured on #1085 — a release-please-- head ref that carried
    # a "## PR Reviewer Guide" comment anyway). Only a gate the reviewed tool
    # cannot ignore is worth pairing against.
    #
    # Detected with grep rather than inside the python below, because the pattern
    # carries both quote characters and embedding it in a double-quoted heredoc
    # mangles them.
    local has_exclusion=no
    if grep -q "startsWith(github.event.pull_request.head.ref, 'release-please--')" \
        "$REPO/.github/workflows/pr-agent.yml"; then
        has_exclusion=yes
    fi

    run python3 -c "
import json, sys
reg = json.load(open('$REPO/harness/review-attestation.json'))
exempt = reg.get('exempt', {}).get('signatures', [])

has_ignored = '$has_exclusion' == 'yes'
rp_sig = next((s for s in exempt if s.get('name') == 'release-please'), None)
expected_files = {'.release-please-manifest.json', 'CHANGELOG.md', 'versions.conf'}
has_sig = rp_sig is not None and set(rp_sig.get('files', [])) == expected_files

if has_ignored != has_sig:
    if has_ignored:
        print('the reviewer skips release PRs the gate still demands a review for:')
        print('  the workflow if: excludes release-please-- head refs')
        print('  no release-please signature in the registry -> those PRs can never go green')
    else:
        print('the gate exempts release PRs the reviewer still reviews:')
        print('  exempt.signatures = ' + repr([s.get('name') for s in exempt]))
        print('  the workflow if: does not exclude release-please-- head refs')
    sys.exit(1)
if not has_ignored or not has_sig:
    print('expected the workflow branch exclusion and the 3-file signature to be declared')
    sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}
