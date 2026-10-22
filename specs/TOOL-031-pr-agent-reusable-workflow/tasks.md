---
tags: [spec, tasks]
created: "2026-10-21"
---

# Tasks - TOOL-031-pr-agent-reusable-workflow

> TDD order: each task is one focused commit.
>
> Markers:
> - `[P]`: depends on no unchecked task.
> - `[AC<n>]`: serves acceptance criterion n.
>
> Repositories:
> - **D**: dotfiles (this spec).
> - **C**: `mlorentedev/ci-workflows` (a local clone at `~/Projects/ci-workflows` until the remote exists).
> - **H**: hive.

## Setup

- [x] Branch `feat/pr-agent-reusable-workflow` in a dotfiles worktree; #2893 claimed and In Progress
- [ ] `proposal.md` complete; the owner has reviewed the `[AGENT-DRAFT]` criteria
- [ ] Owner: `gh repo create mlorentedev/ci-workflows --public --description "Shared CI referenced by SHA (ADR-042)"` (R5)
- [ ] D: the spec PR opened as a draft once 0.77.0 is out (R4)

## Implementation

### B1. C: scripts, pool and their tests (moved, behaviour unchanged)

- [ ] [P] [AC2] C:
  - Run `git init`.
  - Copy `scripts/pr-agent-{model-preflight,route,publish-guard,push-gate}.sh` from D at a recorded SHA.
  - Copy their bats tests (`pr-agent-route.bats`, `pr-agent-publish-guard.bats`, `pr-agent-model-preflight.bats`, `pr-agent-push-gate.bats`) and fixtures.
  - Run `bats tests/`. Expected: the same pass count as in D.
- [ ] [AC2] C: create `pr-agent/pool.json`.
  - Content: D's `harness/reviewer-pool.json`, reduced to `id`, `vendor` and `pr_agent` per member, plus `pr_agent_risk` and kubelab's `forge_rank`.
  - Point route.sh's `POOL_FILE` default at it.
  - Test first: a bats case asserting that every member with `pr_agent.weight > 0` has a model and that the draw ignores `forge_rank`. FAIL, then PASS.
- [ ] [AC2] C: `.github/workflows/ci.yml` runs bats and actionlint, with actions pinned by SHA

### B2. C: the reusable workflow

- [ ] [AC2] Test first (`tests/pr-agent-workflow.bats`). It asserts:
  - the only trigger is `on: workflow_call`;
  - secret `NAN_API_KEY` is required and `PR_AGENT_ANTHROPIC_API_KEY` is `required: false`;
  - there is no `secrets: inherit` and no `github.workflow_sha`;
  - there is no branch literal (`main`, `master`, `release-please--`).

  Expected: FAIL (the file does not exist yet).
- [ ] [AC2] Create `.github/workflows/pr-agent.yml` from D's workflow. Tests PASS.
  - Drop `on:`, the workflow-level `concurrency` and `permissions`.
  - Keep the job lane.
  - Check out the repository parsed from `job.workflow_ref`, at `job.workflow_sha` (R1).
- [ ] [AC2] Test, then change: both Anthropic attempts set `LITELLM__CUSTOM_LLM_PROVIDER: anthropic`
- [ ] [AC2] Test, then change: upload `pr-number` / `pr-number.txt` on every run, numeric check first
- [ ] [AC2] Test, then change: the upstream-contract read uses the `job.workflow_*` pair. The remedy text stops naming `dotfiles/secrets/registry.yaml` and names the caller's secret instead.

### B3. C: push-gate action, kubelab inputs, caller fixtures

- [ ] [AC3] `actions/push-gate`, test first: the composite step exits non-zero when the gate script errors (FAIL). Implement it (PASS), then use it in the workflow.
- [ ] [AC1] Port kubelab's reviewable and declare tests (`kubelab tests/test_pr_agent_workflow.py:411-640, 485-527`) as bats for input `declare_unreviewed`. FAIL, then implement, then PASS.
  - On: label and section are written, idempotently, and a stale label is cleared.
  - Off: the notice only.
- [ ] [AC6] Input `skip_branch_prefixes` (default `release-please--`). Test it with a kubelab-shaped list that includes `deploy/` and `promote/`.
- [ ] [AC6] Add `tests/fixtures/callers/{kubelab,dotfiles,hive}.yml`, plus a bats case asserting that each fixture's `with:` and `secrets:` match the workflow's declared inputs and secrets, with no unknown key.
- [ ] C: README (inputs, secrets, the caller template, the pinning rule). Tag `v0.1.0`.

### C/D. H: pilot

- [ ] [AC4] H PR-C (draft): add `.github/workflows/pr-agent-shared.yml` next to the existing job.
  - The caller is pinned to C's `v0.1.0` SHA.
  - The caller owns triggers, permissions and concurrency.
  - `tests/test_pr_agent_workflow.py` gains a test of the pin's shape.
- [ ] [AC4] H PR-C: on one PR, both jobs run and the job summary names the drawn member. Link the run in `verification.md`.
- [ ] Owner: `dotf secrets sync ci` for `PR_AGENT_ANTHROPIC_API_KEY` to hive (R5). This is a cost decision.
- [ ] [AC5] After PR-C merges, tabulate seven days of hive runs in `verification.md`: the drawn member and the Haiku fallbacks.
- [ ] [AC5] H PR-D: remove the old job and its tests. `verification.md` records that `review` is not a required check (`gh api repos/mlorentedev/hive/branches/master/protection`).

## Closing

- [ ] Every acceptance criterion is covered by a test or a recorded run
- [ ] `features.json` verifications are not vacuous
- [ ] B3's PR closes #1570, with the AC6 evidence
- [ ] PR-D closes hive#503
- [ ] ts-bridge-91 told the pilot is green (ts-bridge#474)
- [ ] `verification.md` filled in, with the promotion lines answered, including whether to amend ADR-042 (R6)
- [ ] An independent adversarial review (not by the implementer) before archive
