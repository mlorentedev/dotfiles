---
tags: [spec, verification, templates]
created: "2026-08-16"
---

# Verification - TOOL-013-pr-agent-reviewer

## Evidence for the #2010 update

- Implementation commit: `54811ad3c9ffc8ef896a3bad07b0856015e98247`
  on #2010; [review triage](https://github.com/mlorentedev/dotfiles/pull/2010#issuecomment-6001736990)
  disposes both PR-Agent findings.
- AC5: `pr-agent-config.bats` confirms the Action is pinned to a commit; the
  v0.47.0 pin resolves at `The-PR-Agent/pr-agent`.
- AC8: BATS cases for the executing workflow pin, the remote Git blob identities,
  and the absence of copied versions passed. A synthetic changed blob SHA failed
  with `pr_agent/agent/pr_agent.py changed since the approved upstream contract`;
  a synthetic invalid token failed with HTTP 401 and `cannot verify ...`.
- AC9: BATS confirms one Action, `fail_on_tool_errors: "true"`, retained internal
  fallbacks, no `continue-on-error`, and a guard bound to the Action outcome.
  The previous unconditional second Action is gone. Live publication behavior
  awaits CI on the pushed commit.
- AC1-AC4, AC6-AC7: existing behavior and earlier checks remain in place; this
  update does not claim fresh live-review or secrets-delivery evidence.

## Test status

- Git Bash on Windows: `bats --filter "extension list is read|filter derives|pinned upstream|ambiguous failures|no expression parses|push gate does not duplicate" tests/pr-agent-config.bats tests/pr-agent-push-gate.bats` -> 6/6 passed.
- `bats tests/pr-agent-config.bats tests/pr-agent-model-preflight.bats tests/pr-agent-push-gate.bats` -> exit 1 locally (103 cases). Git Bash paths embedded in Python strings are not readable by Windows Python, and `zsh` is absent; this is not evidence of a Linux CI regression.
- `shellcheck -S warning scripts/pr-agent-push-gate.sh scripts/pr-agent-model-preflight.sh` -> exit 0.
- `actionlint -color=false -ignore 'unexpected key "queue"' .github/workflows/pr-agent.yml` -> exit 0. Local actionlint reports `concurrency.queue` without this ignore on both the old and new workflow.
- CI Linux on the new commit: pending until push. Do not infer a green result from local checks.

## Decisions made during implementation

- A step failure does not distinguish a model outage from a tool error in this
  Action. Fail closed rather than retrying an Action that may already have
  published; the model fallback stays inside its single attempt.
- Compare Git blob identities of the upstream files the gate relies on instead
  of copying the Action version into the gate. The workflow at
  `github.workflow_sha` supplies the effective runtime pin.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no: the operational update procedure is in `docs/runbooks/guide-pr-agent-reviewer.md`.
- [ ] ADR-worthy decision for the repo's `docs/adr/`? no: the review-bounding decision remains ADR-040.
- [ ] New pattern candidate for `00_meta/patterns/`? no: this is a repository-specific dependency contract.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-013-pr-agent-reviewer/` -> `specs/archive/TOOL-013-pr-agent-reviewer/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Independent adversarial review passes for the final contract
