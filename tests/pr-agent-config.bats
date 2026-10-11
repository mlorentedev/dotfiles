#!/usr/bin/env bats
# TOOL-013 (#786): the PR-Agent reviewer's configuration.
#
# These assert the DECISIONS, not the syntax. Each one encodes a choice that
# would be easy to undo in a passing-looking edit, and expensive to discover
# afterwards — a reviewer that quietly rubber-stamps, or that quietly reads
# credential material.

load 'lib/refute'

setup() {
    REPO="$BATS_TEST_DIRNAME/.."
    CFG="$REPO/.pr_agent.toml"
    WF="$REPO/.github/workflows/pr-agent.yml"
    # The publication guard's logic, shared by the final guard and the probe that
    # gates the fallback outside NaN (AI-045 AC9).
    GUARD="$REPO/scripts/pr-agent-publish-guard.sh"
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
    #
    # This asserted an EMPTY list until #1107, and that assertion has been
    # narrowed rather than deleted. The empty list was never the point; not
    # falling back to a model the pool excludes was. #1107 measured a per-model
    # concurrency limit (8 concurrent to deepseek -> 5x200 + 3x429, while mimo
    # answered 200 during that saturation), which makes a same-tier second NaN
    # model a genuine fallback rather than a cheaper one. What must stay
    # impossible is naming one of the excluded models here.
    refute_grep 'fallback_models = \[[^]]*qwen3\.6' "$CFG"
    refute_grep 'fallback_models = \[[^]]*gemma4' "$CFG"
}

@test "pr-agent: the reviewing model is reasoning-class, not the fast one" {
    # This pinned `deepseek-v4-flash` by name until #1149 moved CI to its own
    # model lane. Pinning one id made a routing change look like a regression,
    # so the assertion now names the PROPERTY that actually matters and the
    # excluded models by name — the same shape the fallback test above already
    # uses, and the same reason.
    #
    # The property: the reviewing model must be reasoning-class. The line
    # reviewer-pool.json draws is context window plus a mandatory reasoning
    # chain — deepseek-v4-flash and mimo-v2.6-flash at 1M sit on one side, qwen3.6
    # and gemma4 at 262K on the other. A reviewer that PASSes cheaply is worse than
    # no gate.
    run grep -E '^model = "openai/(deepseek-v4-flash|mimo-v2\.6-flash)"' "$CFG"
    [ "$status" -eq 0 ]

    # And the excluded ones stay impossible in the primary slot, not only in the
    # fallback list. The old test could not catch this, because asserting one
    # allowed id says nothing about which ids are forbidden.
    refute_grep '^model = "openai/qwen3\.6"' "$CFG"
    refute_grep '^model = "openai/gemma4"' "$CFG"
}

@test "pr-agent: CI and the spec-review gate do not share a model" {
    # #1149. NaN's concurrency limit is per MODEL, not per API key, so CI and
    # `dotf spec review` drawing on the same id share one bucket of five.
    # Measured 2026-08-21: a spec review died with
    # `429: deepseek-v4-flash concurrency limit` because PR-Agent was reviewing
    # the very PR that review had to clear. CI starved the gate its own PR
    # needed to pass.
    #
    # Two models are two independent buckets, so the fix is separation and this
    # asserts it stays separated. The pool's PRIMARY is what matters: that is
    # what `dotf spec review` reaches for first.
    pool_primary="$(python3 -c "
import json
d = json.load(open('$REPO/harness/reviewer-pool.json'))
print(next(e['model'] for e in d['pool'] if e['role'] == 'primary'))
")"
    [ -n "$pool_primary" ]
    refute_grep "^model = \"openai/${pool_primary}\"" "$CFG"
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

@test "pr-agent: the workflow takes only its declared secrets, one inference credential per attempt" {
    # #1025: the spec-review path injects the WHOLE registry to authenticate one
    # model, so one broken item mapping takes down authentication for
    # everything. This path must not inherit that shape: each secret arrives by
    # name, and each PR-Agent step holds exactly one inference credential, so
    # the NaN attempt never sees the Anthropic key, nor an Anthropic attempt NaN's.
    #
    # Counted as distinct NAMES, not as references: the model preflight (AI-045)
    # hands the same NAN_API_KEY to its own step, and the pool draw the Anthropic
    # key to its probe, which are second uses of one credential, not new ones.
    local names
    names=$(grep -oE '\$\{\{ *secrets\.[A-Z_]+ *\}\}' "$WF" | grep -oE 'secrets\.[A-Z_]+' | sort -u | tr '\n' ' ')
    [ "$names" = "secrets.GITHUB_TOKEN secrets.NAN_API_KEY secrets.PR_AGENT_ANTHROPIC_API_KEY " ] || {
        printf 'expected exactly GITHUB_TOKEN, NAN_API_KEY and PR_AGENT_ANTHROPIC_API_KEY, found: %s\n' "$names" >&2
        grep -nE '\$\{\{ *secrets\.[A-Z_]+ *\}\}' "$WF" >&2
        return 1
    }
    run python3 -c "
import re, sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))['jobs']['review']['steps']
for s in steps:
    held = sorted(set(re.findall(r'secrets\.([A-Z_]+)', str(s.get('env', {})))) - {'GITHUB_TOKEN'})
    if held:
        print(s.get('id'), ' '.join(held))
" "$WF"
    [ "$status" -eq 0 ]
    # `credential` holds only a presence test (`!= ''`), never the value.
    [ "${lines[0]%$'\r'}" = "credential NAN_API_KEY" ]
    [ "${lines[1]%$'\r'}" = "models NAN_API_KEY" ]
    [ "${lines[2]%$'\r'}" = "route PR_AGENT_ANTHROPIC_API_KEY" ]
    [ "${lines[3]%$'\r'}" = "pr_agent_anthropic_first PR_AGENT_ANTHROPIC_API_KEY" ]
    [ "${lines[4]%$'\r'}" = "pr_agent NAN_API_KEY" ]
    [ "${lines[5]%$'\r'}" = "pr_agent_anthropic_second PR_AGENT_ANTHROPIC_API_KEY" ]
    [ "${#lines[@]}" -eq 6 ]
}

@test "pr-agent: PR_AGENT_ANTHROPIC_API_KEY is declared for this repo's CI only" {
    # Without the consumer, `dotf secrets sync ci` never delivers the key and the
    # fallback is skipped with a warning. Only this repository: the other
    # workflows adopt the fallback in their own pull requests, and a credential
    # synced where nothing reads it only widens the surface. Purpose-named, never
    # ANTHROPIC_API_KEY, which Claude Code and the SDKs read on their own.
    run python3 -c "
import yaml
d = yaml.safe_load(open('$REPO/secrets/registry.yaml'))
s = [s for s in d['secrets'] if s['id'] == 'PR_AGENT_ANTHROPIC_API_KEY'][0]
print(s['consumers'], s['expose'], s['plane'])
print(any(s.get('expose', {}).get('env') == 'ANTHROPIC_API_KEY' for s in d['secrets']))
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "['ci:mlorentedev/dotfiles'] {'env': 'PR_AGENT_ANTHROPIC_API_KEY'} app" ]
    [ "${lines[1]}" = "False" ]
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
# weighs. And incremental (TOOL-023): a push is reviewed for the commits since
# the previous review, never in full again, UNLESS the push gate's own mode
# output says the baseline is not one it can vouch for (a forged marker, a
# rebase, or no previous review at all) — then it is /review, in full,
# because PR-Agent's own baseline pick for `-i` has no author check either.
#
# push_commands is now a template driven by steps.push_gate.outputs.mode
# rather than a bare literal, so this reads out every quoted array the
# template can produce and checks each one, rather than json.loads-ing the
# whole value (which would just throw on a template string).
@test "pr-agent: the push path runs /review -i only when the gate's mode says incremental, /review otherwise" {
    run python3 -c "
import json, re, sys, yaml
d = yaml.safe_load(open('$WF'))
step = next(s for s in d['jobs']['review']['steps'] if 'pr-agent' in s.get('uses', ''))
env = step['env']
if str(env.get('github_action_config.handle_push_trigger', 'false')).lower() != 'true':
    sys.exit(0)
raw = env.get('github_action_config.push_commands')
if raw is None:
    print('handle_push_trigger is on with no push_commands: describe returns by default')
    sys.exit(1)
raw = str(raw)
allowed = {('/review -i',), ('/review',)}
if '\${{' not in raw:
    # A plain literal always reviews the same way regardless of the gate's
    # verdict, which only the incremental command could ever have meant.
    cmds = tuple(c.strip() for c in json.loads(raw))
    if cmds != ('/review -i',):
        print(f'push_commands is {cmds!r}, want exactly (\'/review -i\',)')
        sys.exit(1)
    sys.exit(0)
if 'steps.push_gate.outputs.mode' not in raw:
    print(f'push_commands is a template that does not read the push gate mode: {raw!r}')
    sys.exit(1)
literals = re.findall(r\"'(\[[^]]*\])'\", raw)
if not literals:
    print(f'push_commands template has no quoted command array to check: {raw!r}')
    sys.exit(1)
branches = [tuple(c.strip() for c in json.loads(lit)) for lit in literals]
bad = [b for b in branches if b not in allowed]
if bad:
    print(f'push_commands branch(es) {bad!r} are not one of {sorted(allowed)!r}')
    if any('/describe' in b for b in bad):
        print('  /describe rewrites the PR body, turned off deliberately')
    if any(not b for b in bad):
        print('  an empty list makes the push trigger fire and do nothing')
    sys.exit(1)
if set(branches) != allowed:
    print(f'push_commands only ever produces {sorted(set(branches))!r}, missing the other mode\'s branch')
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
    grep -q 'on every review' "$CFG"
    refute_grep 'HARNESS COMPLIANCE.*(if |when relevant|where applicable)' "$CFG"
}

# The review is a YAML document under PR-Agent's fixed schema, and only what
# the schema declares survives it (#2287). An instruction to "open" the review
# with a section, or to report passes, asks for content no field carries: the
# model dropped the section on 5 of 7 reviews and once wrote it above the
# mapping, losing the `review:` root and the whole review. The pass reports its
# FAILs through key_issues_to_review, the one field every review has that
# carries a file and lines.
@test "pr-agent: the harness compliance pass reports inside the review schema" {
    run python3 - "$CFG" <<'PY'
import re, sys, tomllib

cfg = tomllib.load(open(sys.argv[1], "rb"))
instructions = cfg["pr_reviewer"]["extra_instructions"]
flat = " ".join(instructions.split())
problems = []
if "key_issues_to_review" not in flat:
    problems.append("the instructions do not name key_issues_to_review as the pass's home")
for pattern in (r"\b[Oo]pen (every|each|the) review", r"\b([Bb]egin|[Ss]tart) (every|each|the) review",
                r"even when everything passes", r"\b(with|as|in) an? (\w+ )?section\b"):
    if re.search(pattern, flat):
        problems.append(f"asks for content outside the schema: /{pattern}/")
if cfg["pr_reviewer"].get("num_max_findings", 3) < 5:
    problems.append("num_max_findings below 5: a compliance FAIL would crowd out a defect")
if problems:
    print("\n".join(problems))
    sys.exit(1)
PY
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The reason this tool was adopted is inline comments on the diff (TOOL-013 AC1),
# the half CodeRabbit's free tier withholds on private repos. The previous guard
# asserted [pr_code_suggestions] dual publishing, which configures `/improve` --
# a command the automatic path stopped running in #1107 -- so it stayed green
# while the automatic review posted no inline comment at all (archive review F1).
# This one reads the setting the automatic path consults, and checks that
# `review`, the command that reads it, runs automatically on every PR-Agent step.
# Whether `improve` also runs is #1107's cost decision, not this guard's: turning
# it back on adds a second inline path and removes none.
@test "pr-agent: the automatic review publishes its findings inline" {
    run python3 - "$CFG" "$WF" <<'PY'
import sys, tomllib, yaml
cfg = tomllib.load(open(sys.argv[1], 'rb'))
if cfg.get('pr_reviewer', {}).get('inline_key_issues') is not True:
    print('[pr_reviewer] inline_key_issues is not true: upstream defaults it to false, so no finding is posted inline')
    sys.exit(1)
wf = yaml.safe_load(open(sys.argv[2]))
steps = [s for job in wf['jobs'].values() for s in job.get('steps', [])
         if str(s.get('uses', '')).startswith('The-PR-Agent/pr-agent@')]
if not steps:
    print('no PR-Agent step found'); sys.exit(1)
for s in steps:
    env = s.get('env', {})
    if env.get('github_action_config.auto_review') != 'true':
        print('a PR-Agent step does not run review automatically, so inline_key_issues guards nothing'); sys.exit(1)
PY
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# `/improve` still runs when asked for on a PR, and an asked-for run should
# publish inline too: -1, the upstream default, disables that entirely.
@test "pr-agent: an asked-for /improve publishes its suggestions inline" {
    run python3 -c "
import sys, tomllib
s = tomllib.load(open('$CFG', 'rb')).get('pr_code_suggestions')
if s is None:
    print('no [pr_code_suggestions] section: improve runs on defaults, inline disabled'); sys.exit(1)
t = s.get('dual_publishing_score_threshold', -1)
if not (0 <= t <= 10):
    print(f'dual_publishing_score_threshold={t} does not publish inline'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# Declaring [ignore] glob replaces upstream's list instead of extending it, and
# upstream's list at the pin is ['vendor/**'] (TOOL-013 archive review, F3).
@test "pr-agent: the ignore list keeps upstream's vendor/** default" {
    run python3 -c 'import sys,tomllib; g=tomllib.load(open(sys.argv[1],"rb"))["ignore"]["glob"]; sys.exit(0 if "vendor/**" in g else 1)' "$CFG"
    [ "$status" -eq 0 ] || { echo "declaring [ignore] glob dropped upstream's vendor/** default" >&2; false; }
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

# #1107: the fallback exists to dodge a PER-MODEL concurrency limit (measured:
# 8 concurrent to deepseek gives 5x200 + 3x429, while mimo answers 200 during
# that saturation). A fallback on the SAME model, or an empty list, does not
# solve what this was added for.
@test "pr-agent: the fallback is a second NaN model, not the primary again" {
    run python3 -c "
import tomllib, sys
cfg = tomllib.load(open('$CFG', 'rb'))['config']
fb = cfg.get('fallback_models', [])
primary = cfg.get('model')
if not fb:
    print('fallback_models is empty: a saturated primary leaves no reviewer (#1107)'); sys.exit(1)
if primary in fb:
    print('the fallback repeats the primary, so it shares its concurrency bucket'); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# The pool excludes the latency-optimised daily models BY NAME because "a
# reviewer that PASSes cheaply is worse than no gate". A fallback naming one of
# them would put two files in this repo holding opposite policies — the
# contradiction review-attestation.json already flags on #786.
@test "pr-agent: the fallback is not a model the reviewer pool excludes by name" {
    run python3 -c "
import tomllib, json, sys
fb = tomllib.load(open('$CFG', 'rb'))['config'].get('fallback_models', [])
pool = open('$REPO/harness/reviewer-pool.json').read()
bad = [m for m in fb if m.split('/')[-1] in ('qwen3.6', 'gemma4')]
if bad:
    print('fallback names a model the pool excludes by name: ' + repr(bad)); sys.exit(1)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

# #1107: PR-Agent swallowed a RateLimitError into a clean exit, so six PRs in one
# session carried a green `review` job and no review. ADR-032 forbids exactly
# that — "queues or escalates, NEVER degrades silently" — so the job must fail
# when it published nothing.
@test "pr-agent: the workflow fails when no review was published" {
    grep -q 'name: Fail if no review was published' "$REPO/.github/workflows/pr-agent.yml" \
        || { echo "the no-review guard step is gone; a silent degrade is back (#1107)" >&2; false; }
}

# The heading is declared once, in the reviewer registry, and read by three
# consumers now: the attestation classifier, `dotf pr triage-queue`, and this
# guard. Restating it in the workflow would rebuild the two-file agreement
# nobody checks that the registry exists to prevent.
@test "pr-agent: the no-review guard reads its marker from the registry, not a literal" {
    run grep -c 'contents/harness/review-attestation.json' "$GUARD"
    [ "$status" -eq 0 ] && [ "$output" -ge 1 ] \
        || { echo "the guard no longer reads the marker from the registry" >&2; false; }

    # And it must read the BASE ref: this repository is public, so a PR able to
    # supply the marker would redefine it to something it does post.
    grep -q 'BASE_REF:' "$REPO/.github/workflows/pr-agent.yml" \
        || { echo "the guard must resolve the registry at the base ref, not the PR head" >&2; false; }
}

@test "pr-agent: fork PRs are excluded from the automatic path" {
    # A fork PR runs with an empty secrets store and a read-only token, so the
    # review cannot run and the guard would fail every fork PR for no reason.
    grep -q "github.event.pull_request.head.repo.fork == false" "$WF"
}

@test "pr-agent: the guard only counts comments authored by github-actions[bot]" {
    # On a public repository anyone can paste the marker text into a comment.
    grep -q 'select(.user.login == "github-actions\[bot\]"' "$GUARD"
}

@test "pr-agent: the guard binds the marker to this run's start stamp" {
    grep -q 'id: start' "$WF"
    grep -q 'STARTED: \${{ steps.start.outputs.started }}' "$WF"
    grep -q '(.updated_at >= $started)' "$GUARD"
}

@test "pr-agent: a missing NAN_API_KEY fails before the reviewer runs, naming the remedy" {
    grep -q "HAS_NAN_API_KEY: \${{ secrets.NAN_API_KEY != '' }}" "$WF"
    grep -q "dotf secrets sync ci --repo" "$WF"
}

@test "pr-agent: the guard reads the marker from the PR head when the base has no registry yet" {
    grep -q 'read_markers "${BASE_REF}"' "$GUARD"
    grep -q 'read_markers "${HEAD_SHA}"' "$GUARD"
}

@test "pr-agent: the guard counts comments across all pages, not per page" {
    grep -q -- '/comments" --paginate)' "$GUARD"
    grep -q 'jq -s --arg started' "$GUARD"
}

@test "pr-agent: the model travels in the workflow env so bootstrap PRs do not fall back to upstream defaults" {
    # PR-Agent reads .pr_agent.toml from the DEFAULT branch, so the model a PR
    # declares travels in the workflow, which is read from the PR head. Since
    # AI-045 it travels through the preflight and the pool draw.
    grep -q 'NAN_MODEL: ${{ steps.models.outputs.model }}' "$WF"
    grep -q 'NAN_FALLBACKS: ${{ steps.models.outputs.fallbacks }}' "$WF"
    grep -q 'CONFIG__MODEL: ${{ steps.route.outputs.nan_model }}' "$WF"
    grep -q 'CONFIG__FALLBACK_MODELS: ${{ steps.route.outputs.nan_fallbacks }}' "$WF"
}

@test "pr-agent: the chain the preflight probes equals the toml's, model and fallbacks" {
    # Two declarations of one chain, because the two are read from different
    # refs. This keeps them one chain: a model changed in one place only fails
    # here, not as a PR reviewed by a model nobody declared.
    run python3 -c "
import json, sys, tomllib, yaml
cfg = tomllib.load(open('$CFG', 'rb'))['config']
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
env = next(s for s in steps if s.get('id') == 'models')['env']
bad = []
if env['DECLARED_MODEL'] != cfg['model']:
    bad.append('model: workflow %s, toml %s' % (env['DECLARED_MODEL'], cfg['model']))
if json.loads(env['DECLARED_FALLBACK_MODELS']) != cfg.get('fallback_models', []):
    bad.append('fallbacks: workflow %s, toml %s' % (env['DECLARED_FALLBACK_MODELS'], cfg.get('fallback_models')))
print('; '.join(bad)); sys.exit(1 if bad else 0)
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
}

@test "pr-agent: the preflight and the draw run wherever PR-Agent runs, and NaN down hands over to Anthropic" {
    run _step_if "s.get('id') == 'models'"
    [ "$status" -eq 0 ] || { echo "no step with id models (AI-045)" >&2; false; }
    local models_if
    models_if=$(printf '%s' "$output" | tr -s ' \n' ' ')
    grep -q 'scripts/pr-agent-model-preflight.sh' "$WF"
    # The draw runs wherever the preflight does, whatever the preflight's outcome.
    run _step_if "s.get('id') == 'route'"
    local route_if
    route_if=$(printf '%s' "$output" | tr -s ' \n' ' ')
    [[ "$route_if" == "!cancelled() && "*"$models_if"* ]] \
        || { printf 'preflight if: %s\ndraw if: %s\n' "$models_if" "$route_if" >&2; false; }
    [[ "$route_if" != *"steps.models.outcome"* ]] || false
    # No declared model answered: NaN is down as a whole. The failure must not
    # end the job before the draw, and the final guard must still run to report it.
    run python3 -c "
import yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
m = next(s for s in steps if s.get('id') == 'models')
r = next(s for s in steps if s.get('id') == 'route')
print(m.get('continue-on-error', False), r['env']['NAN_OUTCOME'])
print(r['env']['NAN_PREFLIGHT_EXIT'])
print('echo \"exit=\${rc}\" >> \"\$GITHUB_OUTPUT\"' in m['run'], m['run'].rstrip().endswith('exit \"\$rc\"'))
"
    # continue-on-error leaves `outcome` failure for exit 1 and exit 2 alike, so
    # the code itself travels to the draw, which stops on a broken setup.
    [ "${lines[0]}" = 'True ${{ steps.models.outcome }}' ]
    [ "${lines[1]}" = '${{ steps.models.outputs.exit }}' ]
    [ "${lines[2]}" = 'True True' ]
    run _step_if "s.get('name') == 'Fail if no review was published'"
    [[ "$output" != *"steps.models.outcome"* ]] || false
}

@test "pr-agent: the publication guard is skipped after a credential failure" {
    grep -q 'id: credential' "$WF"
    grep -q "steps.credential.outcome != 'failure'" "$WF"
}

@test "pr-agent: no line exceeds the fleet yamllint limit of 130 characters" {
    [ "$(awk 'length > 130' "$WF" | wc -l)" -eq 0 ]
}

@test "pr-agent: the head-ref fallback also works for issue_comment runs" {
    grep -q 'HEAD_SHA=$(gh_api "repos/${GITHUB_REPOSITORY}/pulls/${PR_NUMBER}" --jq' "$GUARD"
}

@test "pr-agent: the guard reads the registry from the default branch, never from base.ref" {
    run grep -c 'BASE_REF: \${{ github.event.pull_request.base.ref' "$WF"
    [ "$output" = "0" ]
    grep -q 'BASE_REF: \${{ github.event.repository.default_branch }}' "$WF"
}

@test "pr-agent: the head-ref fallback never takes the marker text from the PR" {
    grep -q "markers='\\[\"PR Reviewer Guide\"\\]'" "$GUARD"
    run grep -c 'markers=$(read_markers "${HEAD_SHA}")' "$GUARD"
    [ "$output" = "0" ]
}

# #1618: a PR whose every file is in [ignore] reaches PR-Agent as an empty diff.
# It used to fail the job with a false NaN diagnosis, on every DR escrow refresh.
# These run the REAL matcher: the Python between `<<'PY'` and `PY` in the
# workflow's reviewable step, fed this repository's real .pr_agent.toml.
_reviewable_kept() { # $1 = newline-separated file list; prints how many survive [ignore]
    local py
    py=$(awk "/<<'PY'/{f=1; next} /^ *PY\$/{f=0} f" "$WF" | sed 's/^          //')
    [ -n "$py" ] || { echo "the reviewable step's matcher is gone from $WF" >&2; return 1; }
    CFG="$(cat "$CFG")" FILES="$1" python3 -c "$py"
}

@test "pr-agent: an escrow-only PR is recognised as having nothing PR-Agent may read (#1618)" {
    run _reviewable_kept $'sensitive/dr/bitwarden-export.age\nsensitive/dr/escrow-manifest.json'
    [ "$status" -eq 0 ] && [ "$output" = "0" ] \
        || { echo "escrow-only PR: expected 0 reviewable files, got '$output'" >&2; false; }
}

@test "pr-agent: one reviewable file is enough to run the review (#1618)" {
    run _reviewable_kept $'sensitive/dr/escrow-manifest.json\ncli/internal/secrets/bw.go'
    [ "$status" -eq 0 ] && [ "$output" = "1" ]
    run _reviewable_kept $'specs/X-1/review-transcript.jsonl\nspecs/X-1/review.md'
    [ "$status" -eq 0 ] && [ "$output" = "1" ]
}

@test "pr-agent: an unanswerable reviewable check falls through to running PR-Agent (#1618)" {
    # An empty file list must not read as "nothing to review": that would skip a
    # review on an API hiccup. The matcher refuses it and the step keeps PR-Agent on.
    run _reviewable_kept ""
    [ "$status" -ne 0 ]
    grep -q 'if: steps.reviewable.outputs.reviewable != '"'false'" "$WF"
    grep -q "steps.reviewable.outputs.reviewable != 'false'" "$WF"
}

# #1417: PR-Agent also drops files by type before the model sees them
# (`is_valid_file` in pr_agent/algo/language_handler.py): its bad_extensions
# list and an exact lockfile list. A `.gitignore`-only PR (#1977) or a
# lockfile-only dependabot bump reached it as an empty diff and went red.
_upstream_ext='[bad_extensions]
default = ["gitignore", "lock", "png"]'
# The shape of upstream's is_valid_file: two literals the step reads by name.
_upstream_handler='def is_valid_file(filename, bad_extensions=None):
    auto_generated_files_exact = {"uv.lock", "go.sum"}
    auto_generated_suffixes = (".min.js", ".js.map")
    computed = sorted(bad_extensions)
    return True'

@test "pr-agent: a PR of file types PR-Agent never reads is recognised as empty (#1417)" {
    UPSTREAM_EXT="$_upstream_ext" UPSTREAM_HANDLER="$_upstream_handler" \
        run _reviewable_kept $'.gitignore\nsite/uv.lock\ngo.sum\nassets/Logo.PNG\nweb/app.min.js'
    [ "$status" -eq 0 ] && [ "$output" = "0" ] \
        || { echo "type-filtered PR: expected 0 reviewable files, got '$output'" >&2; false; }
}

@test "pr-agent: a readable file next to type-filtered ones keeps the review (#1417)" {
    UPSTREAM_EXT="$_upstream_ext" UPSTREAM_HANDLER="$_upstream_handler" \
        run _reviewable_kept $'.gitignore\nscripts/utils.sh'
    [ "$status" -eq 0 ] && [ "$output" = "1" ]
}

@test "pr-agent: a failed read of either upstream list narrows the skip, never widens it (#1417)" {
    # Both lists are fetched at run time. Whichever read fails contributes nothing.
    UPSTREAM_EXT="" UPSTREAM_HANDLER="$_upstream_handler" run _reviewable_kept $'.gitignore\ngo.sum'
    [ "$status" -eq 0 ] && [ "$output" = "1" ]
    UPSTREAM_EXT="$_upstream_ext" UPSTREAM_HANDLER="" run _reviewable_kept $'.gitignore\ngo.sum'
    [ "$status" -eq 0 ] && [ "$output" = "1" ]
    UPSTREAM_EXT="" UPSTREAM_HANDLER="" run _reviewable_kept $'.gitignore\ngo.sum'
    [ "$status" -eq 0 ] && [ "$output" = "2" ]
}

@test "pr-agent: the extension list is read at the same commit the action is pinned to (#1417)" {
    local pins
    pins=$(grep -o 'The-PR-Agent/pr-agent@[0-9a-f]\{40\}' "$WF" | sort -u)
    # An empty $pins still prints one line, so the count alone passes with no pin.
    [ -n "$pins" ] || { echo "the action is not pinned to a commit" >&2; false; }
    [ "$(printf '%s\n' "$pins" | wc -l)" -eq 1 ]
    refute_grep '^ *PR_AGENT_REF: [0-9a-f]{40}' "$WF"
    grep -qF 'WORKFLOW_SHA: ${{ github.workflow_sha }}' "$WF"
}

@test "pr-agent: the filter derives its upstream ref from the executing action pin" {
    run python3 -c "
import sys, yaml
steps = yaml.safe_load(open(sys.argv[1]))['jobs']['review']['steps']
filter_step = next(s for s in steps if s.get('id') == 'reviewable')
print(filter_step['env'].get('WORKFLOW_SHA') == '\${{ github.workflow_sha }}')
print('PR_AGENT_REF=' in filter_step['run'])
print('The-PR-Agent/pr-agent@' in filter_step['run'])
print('PR_AGENT_REF' not in filter_step['env'])
" "$WF"
    [ "$status" -eq 0 ]
    [ "${lines[0]%$'\r'}" = "True" ]
    [ "${lines[1]%$'\r'}" = "True" ]
    [ "${lines[2]%$'\r'}" = "True" ]
    [ "${lines[3]%$'\r'}" = "True" ]
}

@test "pr-agent: the audited upstream contract covers every source and is bound to the action pin" {
    local contract="$REPO/harness/pr-agent-upstream-contract.json" ref file_path audited
    [ -s "$contract" ] || { echo "missing audited upstream contract" >&2; false; }
    for file_path in action.yaml pr_agent/settings/configuration.toml \
        pr_agent/agent/pr_agent.py pr_agent/algo/comment_identity.py \
        pr_agent/algo/file_filter.py pr_agent/algo/inline_comment_dedup.py \
        pr_agent/algo/review_finding_state.py \
        pr_agent/git_providers/github_provider.py pr_agent/servers/github_action_runner.py \
        pr_agent/tools/pr_reviewer.py; do
        jq -e --arg path "$file_path" '.files[$path] | test("^[0-9a-f]{40}$")' \
            "$contract" >/dev/null || { echo "missing audited source: $file_path" >&2; return 1; }
    done
    ref=$(sed -n 's|.*uses: The-PR-Agent/pr-agent@\([0-9a-f]\{40\}\).*|\1|p' "$WF" | sort -u)
    [[ "$ref" =~ ^[0-9a-f]{40}$ ]] || { echo "action pin is not unique" >&2; false; }
    audited=$(jq -r '.audited_ref' "$contract")
    [ "$audited" = "$ref" ] || {
        echo "action pin $ref is not the audited ref $audited: re-audit harness/pr-agent-upstream-contract.json" >&2
        false
    }
}

# The network leg (#2021). A blob SHA at an immutable commit cannot change, and
# the test above binds the pin to the audited ref, so this comparison can only
# find something on a diff that moves the pin or edits the contract. CI sets
# DOTF_TEST_NETWORK=1 on exactly those PRs and on every push to main; the
# default local run stays hermetic.
@test "pr-agent: pinned upstream review contracts match their approved source identities" {
    [ "${DOTF_TEST_NETWORK:-}" = 1 ] \
        || skip "network leg: set DOTF_TEST_NETWORK=1 to compare the contract with upstream"
    local contract="$REPO/harness/pr-agent-upstream-contract.json" ref file_path expected actual
    [ -s "$contract" ] || { echo "missing audited upstream contract" >&2; false; }
    ref=$(sed -n 's|.*uses: The-PR-Agent/pr-agent@\([0-9a-f]\{40\}\).*|\1|p' "$WF" | sort -u)
    [[ "$ref" =~ ^[0-9a-f]{40}$ ]] || { echo "action pin is not unique" >&2; false; }
    if [ -n "${DOTF_TEST_GH_TOKEN:-}" ]; then
        export GH_TOKEN="$DOTF_TEST_GH_TOKEN"
    fi
    while IFS=$'\t' read -r file_path expected; do
        expected=${expected%$'\r'}
        actual=$(gh api "repos/The-PR-Agent/pr-agent/contents/$file_path?ref=$ref" --jq '.sha') \
            || { echo "cannot verify $file_path at $ref" >&2; return 1; }
        [ "$actual" = "$expected" ] \
            || { echo "$file_path changed since the approved upstream contract" >&2; return 1; }
    done < <(jq -r '.files | to_entries[] | [.key, .value] | @tsv' "$contract")
}

@test "pr-agent: no ignore glob relies on negation, which PR-Agent does not implement" {
    # fnmatch.translate makes a leading '!' a literal character, so a '!path'
    # entry re-includes nothing and silently hides nothing either (#1618).
    run python3 -c 'import sys,tomllib; g=tomllib.load(open(sys.argv[1],"rb"))["ignore"]["glob"]; bad=[x for x in g if x.startswith("!")]; print(bad); sys.exit(1 if bad else 0)' "$CFG"
    [ "$status" -eq 0 ] || { echo "negated globs are inert in PR-Agent: $output" >&2; false; }
}

# TOOL-023: a push is reviewed only past a threshold of new commits, and the
# decision is made BEFORE PR-Agent starts, by scripts/pr-agent-push-gate.sh, so
# that a skipped push is not mistaken by the guard for a failed inference.
# The three PR-Agent steps, in execution order: Anthropic drawn first, NaN,
# Anthropic after NaN.
PA_IDS="pr_agent_anthropic_first pr_agent pr_agent_anthropic_second"

_step_if() { # $1 = python expression selecting a step; prints its `if:`
    python3 -c "
import sys, yaml
d = yaml.safe_load(open('$WF'))
steps = d['jobs']['review']['steps']
s = next(s for s in steps if $1)
print(s.get('if', ''))
"
}

@test "pr-agent: a push runs the push gate first, and only a push does" {
    run _step_if "s.get('id') == 'push_gate'"
    [ "$status" -eq 0 ] || { echo "no step with id push_gate" >&2; false; }
    [[ "$output" == *"github.event.action == 'synchronize'"* ]] || false
    grep -q 'scripts/pr-agent-push-gate.sh' "$WF"
    # The job has no full checkout; the gate script, the model preflight
    # (AI-045, which runs on every event) and the publication guard (AC9) are
    # the only files checked out.
    run _step_if "'checkout' in s.get('uses', '') and 'pr-agent-push-gate' in str(s.get('with', {}))"
    [ "$status" -eq 0 ] || { echo "no sparse checkout of the gate script" >&2; false; }
    run python3 -c "
import yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
co = next(s for s in steps if 'checkout' in s.get('uses', ''))
print(co['with']['sparse-checkout'].split())
"
    [ "$output" = "['harness/reviewer-pool.json', 'scripts/pr-agent-push-gate.sh', 'scripts/pr-agent-model-preflight.sh', 'scripts/pr-agent-route.sh', 'scripts/pr-agent-publish-guard.sh']" ]
}

@test "pr-agent: below the push gate's threshold, neither PR-Agent nor the guard runs" {
    # The gate sits on the draw, and every PR-Agent step runs only on the draw's
    # output, directly or through a probe that does.
    run _step_if "s.get('id') == 'route'"
    [[ "$output" == *"steps.push_gate.outputs.run != 'false'"* ]] \
        || { echo "the draw still runs on a push the gate skipped: $output" >&2; false; }
    run python3 -c "
import yaml
steps = {s.get('id'): s for s in yaml.safe_load(open('$WF'))['jobs']['review']['steps']}
for sid in '$PA_IDS'.split() + ['published_after_anthropic', 'published_after_nan']:
    cond = steps[sid]['if']
    probes = [p for p in ('published_after_anthropic', 'published_after_nan') if 'steps.%s.' % p in cond]
    ok = 'steps.route.outputs.' in cond or all('steps.route.outputs.' in steps[p]['if'] for p in probes) and probes
    print(sid, bool(ok))
"
    [ "$status" -eq 0 ]
    [[ "$output" != *"False"* ]] || { echo "$output" >&2; false; }
    run _step_if "s.get('name') == 'Fail if no review was published'"
    [[ "$output" == *"steps.push_gate.outputs.run != 'false'"* ]] \
        || { echo "the guard would fail a push the gate skipped: $output" >&2; false; }
}

@test "pr-agent: the registry declares the incremental Guide heading as a review marker" {
    run jq -r '.reviewers[] | select(.login == "github-actions") | .review_markers[]' \
        "$REPO/harness/review-attestation.json"
    [[ "$output" == *"## PR Reviewer Guide"* ]] || false
    [[ "$output" == *"## Incremental PR Reviewer Guide"* ]] || false
}

@test "pr-agent: the guard accepts every marker the registry declares, not only the first" {
    run grep -c 'review_markers\[0\]' "$GUARD"
    [ "$output" = "0" ]
    grep -q -- '--argjson markers' "$GUARD"
    grep -q 'any($markers\[\]' "$GUARD"
}

# AI-045: timeouts must not be mislabeled as concurrency (#1107); failures
# can also be tool errors. Report the actual outcome and selected model.
@test "pr-agent: a cancelled or failed Action reports its outcome and model, for the last attempt that ran" {
    # ATTEMPTS lists every PR-Agent step in execution order with the model it
    # ran, so the guard can judge the last one that ran.
    run python3 -c "
import re, yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
guard = next(s for s in steps if s.get('name') == 'Fail if no review was published')
pa = [s for s in steps if 'pr-agent' in s.get('uses', '')]
listed = [l.split() for l in guard['env']['ATTEMPTS'].strip().splitlines()]
print([re.sub(r'.*steps\.(\w+)\.outcome.*', r'\1', l[0] + l[1]) for l in listed] == [s['id'] for s in pa])
models = [s['env']['CONFIG__MODEL'] for s in pa]
print([' '.join(l[3:]) for l in listed] == models)
print(guard['env']['ROUTE_NOTE'], guard['run'].strip())
print(guard['env']['ROUTE_OUTCOME'])
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "True" ]
    [ "${lines[1]}" = "True" ]
    [ "${lines[2]}" = '${{ steps.route.outputs.note }} ./scripts/pr-agent-publish-guard.sh' ]
    # A failed draw must reach the guard, or it reads as a pool with no answer.
    [ "${lines[3]}" = '${{ steps.route.outcome }}' ]
    grep -q '    cancelled|failure)' "$GUARD"
}

# AI-045: the guard blamed every silent run on concurrency (#1107). Run
# 36680454975 disproved that as the only cause: deepseek-v4-flash took a 42K-token
# review and PR-Agent exited cleanly after 12 minutes with nothing published.
# The message must name both measured causes, so a reader does not chase the
# wrong one.
@test "pr-agent: the no-review guard names both measured causes, not concurrency alone" {
    grep -q '#1107' "$GUARD"
    grep -q 'non-streamed answer.*#1858\|#1858.*non-streamed' "$GUARD" \
        || grep -A2 'non-streamed answer' "$GUARD" | grep -q '#1858'
}

# A failed Action is not a classified model failure: a second run could publish
# the same review twice. The guard must turn the swallowed step outcome red.
@test "pr-agent: at most two bounded attempts, and the second needs a measured absence of the first" {
    run python3 -c "
import yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
pa = [s for s in steps if 'pr-agent' in s.get('uses', '')]
for s in pa:
    print(s.get('id'), s.get('timeout-minutes'), s.get('continue-on-error', False), ' '.join(s['if'].split()))
ids = [s.get('id') for s in steps]
print(ids.index('route') < ids.index('pr_agent_anthropic_first') < ids.index('published_after_anthropic')
      < ids.index('pr_agent') < ids.index('published_after_nan') < ids.index('pr_agent_anthropic_second'))
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "pr_agent_anthropic_first 8 True !cancelled() && steps.route.outputs.first == 'anthropic'" ]
    [ "${lines[1]}" = "pr_agent 12 True !cancelled() && (steps.route.outputs.first == 'nan' || steps.published_after_anthropic.outputs.published == 'false')" ]
    [ "${lines[2]}" = "pr_agent_anthropic_second 8 True !cancelled() && steps.published_after_nan.outputs.published == 'false'" ]
    [ "${lines[3]}" = "True" ]
    # Each probe runs only after the first attempt it measures, toward the other provider.
    run _step_if "s.get('id') == 'published_after_anthropic'"
    [[ "$output" == *"steps.route.outputs.first == 'anthropic'"*"steps.route.outputs.second == 'nan'"* ]] || false
    run _step_if "s.get('id') == 'published_after_nan'"
    [[ "$output" == *"steps.route.outputs.first == 'nan'"*"steps.route.outputs.second == 'anthropic'"* ]] || false
}

# The fallback runs on one answer only: the probe measured that nothing was
# published. An Action that failed may still have published, and an API that
# failed answers nothing; neither may run a second Action (#1923, AI-045 AC9).
@test "pr-agent: ambiguous failures fail closed without invoking a second Action" {
    # Both probe steps run the real guard script in probe mode, and the second
    # attempt is keyed on its literal `false`: `true`, `unknown` and an empty
    # answer (the probe crashed) never run it.
    run python3 -c "
import yaml
steps = {s.get('id'): s for s in yaml.safe_load(open('$WF'))['jobs']['review']['steps']}
for p in ('published_after_anthropic', 'published_after_nan'):
    print(steps[p]['run'].strip())
print(steps['pr_agent']['if'].count(\"steps.published_after_anthropic.outputs.published == 'false'\"),
      steps['pr_agent_anthropic_second']['if'].count(\"steps.published_after_nan.outputs.published == 'false'\"))
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = './scripts/pr-agent-publish-guard.sh --probe --output "$GITHUB_OUTPUT"' ]
    [ "${lines[1]}" = './scripts/pr-agent-publish-guard.sh --probe --output "$GITHUB_OUTPUT"' ]
    [ "${lines[2]}" = "1 1" ]
    # PR-Agent's own tool errors still fail the attempt rather than pass silently.
    run python3 -c "
import yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
print(' '.join(s['env'].get('github_action_config.fail_on_tool_errors') for s in steps if 'pr-agent' in s.get('uses', '')))
"
    [ "$output" = "true true true" ]
}

# Run 36812454370: a merge commit from "update branch" is a push the gate does
# not review, so the preflight was skipped and its outputs were empty. GitHub
# evaluates a step's `env` before its `if`, so `fromJSON('')` in the retry's env
# threw although the step would have been skipped, and the job failed. Parsing a
# step output inside an expression fails whenever its producer is skipped; the
# preflight emits a scalar instead.
@test "pr-agent: no expression parses a step output with fromJSON" {
    run grep -nE 'fromJSON\(\s*steps\.' "$WF"
    [ "$status" -eq 1 ]
}

# AI-045 AC6 (#1858, #1923): every attempt streams its NaN call. Non-streamed, a
# review-sized request sat silent for 12 minutes with `ai_timeout: 120` in the
# effective config and no retry logged (run 36826726168): the timeout bounds the
# gap between reads, and a held connection never trips it. Streamed, the same
# timeout measures the gap between chunks, and NaN's edge stops cutting the
# answer at about 125 s.
#
# PR-Agent forces streaming only when BOTH halves match: the request's
# `custom_llm_provider` equals `force_streaming_custom_llm_provider`, and its
# `api_base` contains one of the substrings. The substring is checked against
# the base URL the step really sends, so moving NaN's endpoint fails here
# instead of silently turning streaming off.
@test "pr-agent: the NaN attempt streams its calls, and an Anthropic attempt carries no NaN transport" {
    run python3 -c "
import json, yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
for s in [s for s in steps if 'pr-agent' in s.get('uses', '')]:
    e = s['env']
    if 'OPENAI__API_BASE' not in e:
        print(s['id'], 'no-nan', [k for k in e if k.startswith(('OPENAI__', 'LITELLM__'))])
        continue
    subs = json.loads(e.get('LITELLM__FORCE_STREAMING_API_BASE_SUBSTRINGS', '[]'))
    print(s['id'],
          e.get('LITELLM__CUSTOM_LLM_PROVIDER') == 'openai',
          e.get('LITELLM__FORCE_STREAMING_CUSTOM_LLM_PROVIDER') == 'openai',
          bool(subs) and all(x in e['OPENAI__API_BASE'] for x in subs))
"
    [ "$status" -eq 0 ]
    # A forced `openai` provider would send the Anthropic model through NaN's transport.
    [ "${lines[0]}" = "pr_agent_anthropic_first no-nan []" ]
    [ "${lines[1]}" = "pr_agent True True True" ]
    [ "${lines[2]}" = "pr_agent_anthropic_second no-nan []" ]
    [ "${#lines[@]}" -eq 3 ]
}

# AI-045 AC9: glm5.3-flash at its default effort thinks until NaN closes the
# stream and publishes nothing (measured 2026-10-08). PR-Agent sends an effort
# to a model LiteLLM does not know only when it is listed, so the list and the
# level are what keep it a reviewer.
@test "pr-agent: the NaN attempt lowers glm5.3-flash's effort, and lists only chain members" {
    run python3 -c "
import json, yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
by = {s.get('id'): s for s in steps}
e = by['pr_agent']['env']
listed = json.loads(e.get('CONFIG__ADDITIONAL_REASONING_EFFORT_MODELS', '[]'))
m = by['models']['env']
chain = [m['DECLARED_MODEL']] + json.loads(m['DECLARED_FALLBACK_MODELS'])
print(e.get('CONFIG__REASONING_EFFORT'), listed)
print(all(any(c == x or c.endswith('/' + x) for c in chain) for x in listed))
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "low ['glm5.3-flash']" ]
    [ "${lines[1]}" = "True" ]
}

# A job timeout shorter than its Action step would hide the Action's own bound.
@test "pr-agent: the job outlives both bounded Actions and the probes before them" {
    # At most two attempts run, one per provider. The worst pair, plus the
    # preflight's probes (PREFLIGHT_TIMEOUT, 90 s per NaN model), the draw's
    # (30 s), three minutes of setup (31 s measured, run 37879933613), and a
    # minute for each step after the attempts that calls the GitHub API (two
    # publication measurements and the final guard: up to three calls each, with
    # 5 s and 10 s retry sleeps), must fit inside the job.
    run python3 -c "
import json, math, yaml
job = yaml.safe_load(open('$WF'))['jobs']['review']
by = {s.get('id'): s for s in job['steps']}
pair = by['pr_agent']['timeout-minutes'] + max(by[i]['timeout-minutes'] for i in ('pr_agent_anthropic_first', 'pr_agent_anthropic_second'))
chain = 1 + len(json.loads(by['models']['env']['DECLARED_FALLBACK_MODELS']))
probes = math.ceil((90 * chain + 30) / 60)
after = len([s for s in job['steps'] if s.get('id') in ('published_after_anthropic', 'published_after_nan')
             or s.get('name') == 'Fail if no review was published'])
print(after == 3 and job['timeout-minutes'] >= pair + probes + 3 + after, job['timeout-minutes'], pair, probes, after)
"
    [ "$status" -eq 0 ]
    [[ "$output" == "True "* ]] || { echo "$output" >&2; false; }
}

# AI-045 AC9 (#1923): what the key outside NaN may spend. The Claude Console
# cannot restrict a key to a model, so the allowlist is ALLOWED_ANTHROPIC in
# scripts/pr-agent-route.sh, and this test is what makes it a rule rather than a
# line anyone can edit: every allowed model needs a price row below, and its
# worst review under the caps must stay under that row's ceiling.
@test "pr-agent: the Anthropic attempts run the routed model, allowlisted, with bounded input and output" {
    run python3 -c "
import json, re, yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
by = {s.get('id'): s for s in steps}
a1, a2 = by['pr_agent_anthropic_first'], by['pr_agent_anthropic_second']
e = a1['env']
routed = '\${{ steps.route.outputs.anthropic_model }}'
# The model and its effort are the route's outputs, never a literal.
print(e['CONFIG__MODEL'] == e['CONFIG__MODEL_WEAK'] == routed,
      e['CONFIG__REASONING_EFFORT'] == '\${{ steps.route.outputs.anthropic_effort }}')
# Both copies are one attempt: identical in every key and the action pin.
print(a1['env'] == a2['env'] and a1['uses'] == a2['uses'] and a1['timeout-minutes'] == a2['timeout-minutes'])
# The allowlist, read from the script that enforces it.
m = re.search(r'^ALLOWED_ANTHROPIC=\"([^\"]*)\"$', open('$REPO/scripts/pr-agent-route.sh').read(), re.M)
allowed = m.group(1).split()
print(allowed)
# Every Anthropic model the pool gives PR-Agent is allowed, with a known effort.
pool = json.load(open('$REPO/harness/reviewer-pool.json'))['pool']
ant = [p['pr_agent'] for p in pool if p.get('pr_agent', {}).get('model', '').startswith('anthropic/')]
print(all(a['model'] in allowed and a.get('reasoning_effort', 'medium') in ('low', 'medium', 'high') for a in ant))
print(json.loads(e['CONFIG__FALLBACK_MODELS']))
# Temperature is never sent and adaptive thinking reaches every allowed model:
# PR-Agent's built-in pattern matches neither id.
nt = json.loads(e['CONFIG__NO_TEMPERATURE_MODELS'])
print(all(a in nt and a.split('/', 1)[1] in nt for a in allowed),
      sorted(json.loads(e['CONFIG__CLAUDE_ADAPTIVE_THINKING_MODELS_OVERRIDE'])) == sorted(allowed),
      e['CONFIG__ENABLE_CLAUDE_ADAPTIVE_THINKING'])
# The prompt cap equals the NaN attempt's, so every pool member reviews the same
# diff (the owner chose parity over Haiku's 100,000-token price step, 2026-10-08).
nan = by['pr_agent']['env']
print(e['CONFIG__MAX_MODEL_TOKENS'] == e['CONFIG__CUSTOM_MODEL_MAX_TOKENS'] == nan['CONFIG__CUSTOM_MODEL_MAX_TOKENS'])
# LiteLLM's 4,096 default for an unknown Claude model would cut a thinking review short.
print(e['DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS'])
# The worst review the caps allow, per allowed model, at its highest prices per
# million tokens (in, out) with Claude's tokenizer ~30% larger, stays under the
# ceiling. Haiku 5.5: its rates above 100,000 prompt tokens. Sonnet 5.5: no
# price step. A model without a row, or a cap raised past a ceiling, fails here.
prices = {'anthropic/claude-haiku-5-5': (0.50, 2.50, 0.25),
          'anthropic/claude-sonnet-5-5': (2.00, 10.00, 1.00)}
def worst(model):
    i, o, _ = prices[model]
    return int(e['CONFIG__MAX_MODEL_TOKENS']) * 1.3 * i * 1e-6 + int(e['DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS']) * o * 1e-6
print(all(a in prices and worst(a) < prices[a][2] for a in allowed))
# The call is not streamed, so ai_timeout bounds the whole answer; it must end
# inside the step, or the step's kill would hide PR-Agent's own timeout.
print(int(e['CONFIG__AI_TIMEOUT']) < a1['timeout-minutes'] * 60)
# Every model id .pr_agent.toml pins (model, model_weak, any later one) is a NaN
# id. Each needs its own override here, or a call to it leaves Anthropic for NaN.
import tomllib
cfg = tomllib.load(open('$REPO/.pr_agent.toml', 'rb'))['config']
pinned = sorted(k for k, v in cfg.items() if k.startswith('model') and isinstance(v, str) and '/' in v)
print(pinned, all(e.get('CONFIG__' + k.upper()) == routed for k in pinned))
# The guard names the model each Anthropic attempt ran, from the same output.
guard = next(s for s in steps if s.get('run', '').strip() == './scripts/pr-agent-publish-guard.sh')
lines = guard['env']['ATTEMPTS'].strip().splitlines()
print(lines[0].endswith(routed) and lines[2].endswith(routed))
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    [ "${lines[0]}" = "True True" ]
    [ "${lines[1]}" = "True" ]
    [ "${lines[2]}" = "['anthropic/claude-haiku-5-5', 'anthropic/claude-sonnet-5-5']" ]
    [ "${lines[3]}" = "True" ]
    [ "${lines[4]}" = "[]" ]
    [ "${lines[5]}" = "True True true" ]
    [ "${lines[6]}" = "True" ]
    [ "${lines[7]}" = "32000" ]
    [ "${lines[8]}" = "True" ]
    [ "${lines[9]}" = "True" ]
    [ "${lines[10]}" = "['model', 'model_weak'] True" ]
    [ "${lines[11]}" = "True" ]
}

# The pool and the preflight name one NaN chain: a member the pool weighs but the
# preflight never probes would never be in the draw, and a model the preflight
# probes with no pool entry would never be drawn first.
@test "pr-agent: the pool's NaN members for PR-Agent are exactly the chain the preflight probes" {
    run python3 -c "
import json, yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
e = next(s for s in steps if s.get('id') == 'models')['env']
declared = [e['DECLARED_MODEL']] + json.loads(e['DECLARED_FALLBACK_MODELS'])
pool = json.load(open('$REPO/harness/reviewer-pool.json'))['pool']
nan = [p['pr_agent']['model'] for p in pool if p.get('pr_agent', {}).get('model', '').startswith('openai/')]
print(sorted(nan) == sorted(declared), sorted(nan))
"
    [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; false; }
    [[ "$output" == "True "* ]] || { echo "$output" >&2; false; }
}

# The fallback reviews what the NaN attempt would have: same action, same pin,
# same event handling. A setting changed on one step only fails here.
@test "pr-agent: every attempt shares the action pin and every review setting" {
    run python3 -c "
import yaml
steps = yaml.safe_load(open('$WF'))['jobs']['review']['steps']
pa = [s for s in steps if 'pr-agent' in s.get('uses', '')]
pick = lambda s: {k: v for k, v in s['env'].items()
                  if k.startswith('github_action_config.')
                  or k in ('CONFIG__PUBLISH_OUTPUT_PROGRESS', 'CONFIG__OUTPUT_RUN_DETAILS')}
print(len(pa), len({s['uses'] for s in pa}))
print(all(pick(s) == pick(pa[0]) for s in pa), len(pick(pa[0])))
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "3 1" ]
    [ "${lines[1]}" = "True 11" ]
}

# AI-045 AC10 (#1923, option 3): this repository runs one review at a time. NaN
# limits concurrency per model and per key, shared with pi, qq, hive and the
# archive gate, so parallel PRs reviewed together exhaust the bucket that each of
# them needs. The group is repository-wide and lives on the JOB, because the
# workflow-level group is per PR with `cancel-in-progress: true` (a push
# supersedes the review of the push before it), and GitHub refuses `queue: max`
# beside `cancel-in-progress: true`.
#
# `queue: max` is load-bearing. Without it a group holds one running and ONE
# pending job, and a newer pending job cancels the older one: three PRs at once
# would lose a review, and the publish guard would never see it because the job
# never ran.
@test "pr-agent: reviews queue one at a time across the repository, and none is dropped" {
    run python3 -c "
import yaml
wf = yaml.safe_load(open('$WF'))
c = wf['jobs']['review'].get('concurrency') or {}
print(c.get('group'), c.get('cancel-in-progress'), c.get('queue'))
print(wf['concurrency']['cancel-in-progress'])
"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = 'pr-agent-review-${{ github.repository }} False max' ]
    # The per-PR supersession above stays as it was.
    [ "${lines[1]}" = "True" ]
}
