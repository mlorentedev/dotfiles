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

## Evidence for the archive (2026-10-09)

Read against `main` at `11c56bc0`. All nine `features.json` verifications exit 0 there.

- AC1: the reviewer finds real defects on live PRs, through NaN. On #2039 (DOCS-020) a PR-Agent round
  found two, both applied: a numbering-prefix rule that stripped `3-` from `3-2-1 backup rule`, and an
  index left stale when its directory emptied (DOCS-020 `verification.md`). On #2211, run 37887978561
  reviewed with `openai/mimo-v2.6-flash` and raised three findings; one was applied in `17553abc`
  ([triage](https://github.com/mlorentedev/dotfiles/pull/2211#issuecomment-6074931691)). The
  findings arrive inline since #2325 (`inline_key_issues = true`, merged 2026-10-11T02:25:20Z as
  `7186bc0a`), from the same `/review` call #1107 kept to one per PR. Every PR-Agent review in the
  window that followed, read on 2026-10-11 with
  `gh api repos/mlorentedev/dotfiles/pulls/<n>/reviews` and `.../pulls/<n>/comments`, filtered to
  `github-actions[bot]`:

  | PR | Review id | State | Head | Submitted | Inline comments |
  |---|---|---|---|---|---|
  | #2326 | 5481615000 | COMMENTED | `3df52387` | 02:30:22Z | 2 |
  | #2327 | 5481624169 | COMMENTED | `44e8809d` | 02:34:41Z | 1 |
  | #2329 | 5481712753 | COMMENTED | `e3086b93` | 03:18:07Z | 1 |
  | #2332 | 5481717209 | COMMENTED | `47fa3db2` | 03:20:18Z | 1 |
  | #2334 | 5481759720 | COMMENTED | `71df0865` | 03:41:06Z | 2 |
  | #2335 | 5481828782 | COMMENTED | `5d2ddab3` | 04:07:07Z | 2 |

  A finding with no line (path "PR description") stays in the summary comment: #2333's only finding
  was one, and that PR has no review object. `/improve` still posts suggestions on demand. Recorded
  on #786 ([comment](https://github.com/mlorentedev/dotfiles/issues/786#issuecomment-6105247189)).
- AC2: `.pr_agent.toml` names `mimo-v2.6-flash` with `deepseek-v4-flash` as fallback; `qwen3.6` is
  `model_weak`, which the review never uses.
- AC3: `ignore.glob` keeps `sensitive/**` out of the call (f3).
- AC4: holds on `main`: the workflow reads one inference secret, `NAN_API_KEY`, and its step fails
  naming the remedy when that secret is absent (`pr-agent.yml`, `HAS_NAN_API_KEY`). AI-045 (#2188,
  held for the owner's billing review) adds a second credential on purpose and records the reversal.
- AC5, AC8: the #2010 evidence above stands; the pin is v0.47.0.
- AC6: live, both halves. Run 37736957032 (#2174, NaN timed out): the step "Fail if no review was
  published" failed the job. Run 37888015495 (#2211, before the review published):
  `review-attestation` reported `[FAIL] pending` on the PR head.
- AC7: `repo_context_files = ["AGENTS.md", ".claude/CLAUDE.md"]` (f7).
- AC9: holds on `main`: one Action, `fail_on_tool_errors`, the fallback inside its attempt. AI-045
  (#2188) adds an attempt on a second provider by recorded decision.
- Blocked tasks: #983 and #939 are both closed.
- No earlier `review.md` exists for this spec, in git history or in the Linux worktrees.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? no: the operational update procedure is in `docs/runbooks/guide-pr-agent-reviewer.md`.
- [x] ADR-worthy decision for the repo's `docs/adr/`? no: the review-bounding decision remains ADR-040.
- [x] New pattern candidate for `00_meta/patterns/`? no: this is a repository-specific dependency contract.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-013-pr-agent-reviewer/` -> `specs/archive/TOOL-013-pr-agent-reviewer/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Independent adversarial review passes for the final contract
