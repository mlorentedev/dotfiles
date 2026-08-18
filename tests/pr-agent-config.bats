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

@test "pr-agent: AGENTS.md enters the review prompt" {
    # The repo's behavioural SSOT becomes review criteria for free: standing
    # orders, English-only, no-auto-merge, the shell compatibility table.
    grep -q 'repo_context_files = \["AGENTS.md"\]' "$CFG"
}

@test "pr-agent: the workflow pins the action to a tag, not a moving ref" {
    grep -qE 'uses: The-PR-Agent/pr-agent@v[0-9]' "$WF"
    refute_grep 'uses: The-PR-Agent/pr-agent@(main|master|latest)' "$WF"
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

@test "pr-agent: the workflow's trigger list agrees with PR-Agent's own pr_actions" {
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
    # Asserted as set equality, not substring presence: a list that gains an
    # event the other lacks is the drift, in either direction.
    local wf_types pr_actions
    wf_types=$(python3 -c "
import yaml, json
d = yaml.safe_load(open('$WF'))
print('\n'.join(sorted(d[True]['pull_request']['types'])))
")
    pr_actions=$(python3 -c "
import yaml, json
d = yaml.safe_load(open('$WF'))
env = d['jobs']['review']['steps'][-1]['env']
print('\n'.join(sorted(json.loads(env['github_action_config.pr_actions']))))
")
    if [ "$wf_types" != "$pr_actions" ]; then
        printf 'workflow types and github_action_config.pr_actions disagree:\n' >&2
        diff <(printf '%s\n' "$wf_types") <(printf '%s\n' "$pr_actions") >&2
        return 1
    fi
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
