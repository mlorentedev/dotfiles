---
tags: [spec, tasks]
created: "2026-10-21"
---

# Tasks - TOOL-031-pr-agent-reusable-workflow

> TDD order. One task = one focused commit.
>
> Markers: `[P]` means no dependency on an unchecked task; `[AC<n>]` means the task serves acceptance criterion n.
>
> Repos:
> - **D**: dotfiles (this spec, `cli/` and the workflow)
> - **C**: `mlorentedev/ci-workflows`
> - **H**: hive

## Setup

- [x] Branch `feat/pr-agent-reusable-workflow` in a dotfiles worktree. #2893 is claimed and In Progress.
- [ ] `proposal.md` complete; owner reviews the `[AGENT-DRAFT]` criteria and R2
- [x] Comment on #2217 (CLI-101) and kubelab#2295 that TOOL-031 carries them

## Implementation

### A1. D: `dotf pr route` (three stacked PRs, each under the 300 executable-LOC cap)

A1 is about 600 executable lines, so it ships as three PRs, each based on the previous one.

- **A1a: the pieces the route reads and writes.**
  - [x] [AC1] The shared pool parser reads the `pr_agent` blocks and `pr_agent_risk` (`spec.ParseReviewerPool`).
  - [x] [AC1] The allowlist, with a price row and ceiling per model, and drift tests against the script's `ALLOWED_ANTHROPIC` and the bats `prices` dict.
  - [x] [AC1] `--pool embedded|<path>` with no fallback, and a test that the embedded copy equals `harness/reviewer-pool.json`.
  - [x] [AC1] `Probe` and `Verdict`. The key travels only in the `x-api-key` header, and no redirect is followed with any client.
  - [x] [AC1] `Result`, with its outputs, summary and notice.
- **A1b: the route.**
  - [ ] [AC1] `Route`: the weighted draw, the risk route, the overrides and the refusals, with the bats cases ported (`route_test.go`).
  - [ ] [AC1] A differential test: the script and `Route` on the same inputs and the same httptest server give identical outputs, warnings and probes.
- **A1c: the command.**
  - [ ] [AC1] `dotf pr route` (`--changed-lines`, `--labels`, `--pool`, `--json`, `--print-allowlist`), with the GitHub outputs and step summary written when `GITHUB_OUTPUT` is set. Exit 2 on a refusal.

### A2 to A4. D: `preflight`, `guard`, `gate` (one PR each)

- [x] [P] [AC2] `dotf pr preflight`: port the `pr-agent-model-preflight.bats` cases to Go (FAIL), implement (PASS), keep the output names
- [ ] [P] [AC2] `dotf pr guard`: port the `pr-agent-publish-guard.bats` cases (FAIL), implement (PASS)
- [ ] [P] [AC2] `dotf pr gate`: port the `pr-agent-push-gate.bats` cases, including the upstream-contract read (FAIL), implement (PASS)

### A5. D: the workflow switches (one PR)

- [ ] [AC3] `tests/pr-agent-config.bats`, test first:
  - no `./scripts/pr-agent-*` invocation remains;
  - the job builds `dotf` from the PR head before the first decision;
  - the output names are unchanged.

  Expected: FAIL.
- [ ] [AC3] Change `.github/workflows/pr-agent.yml` (PASS), then delete the four scripts and their bats suites
- [ ] [AC3] Make one live run on a PR that changes a weight and link it in `verification.md`
- [ ] Release: the owner merges the release-please PR, which ships `dotf pr …` in a tagged version

### B. C: the thin reusable workflow (blocked on R5's repo)

- [ ] [AC4] Bats tests first:
  - `on: workflow_call` only, with the declared secrets and no `inherit`;
  - no branch literal and no `github.workflow_sha`;
  - a `dotf_version` input;
  - an install step that verifies a checksum.

  Expected: FAIL.
- [ ] [AC4] Write `.github/workflows/pr-agent.yml`, derived from D's A5 workflow:
  - the triggers, permissions and workflow-level concurrency are dropped;
  - own files come from `job.workflow_ref` at `job.workflow_sha`;
  - `dotf` is installed at `inputs.dotf_version` (R3);
  - every decision step is `dotf pr …`.

  Expected: PASS.
- [ ] [AC4] `actions/push-gate`: the composite runs `dotf pr gate` and fails the step on error. Test first.
- [ ] [AC4] [AC6] Inputs `declare_unreviewed` and `skip_branch_prefixes`, with kubelab's declare tests (`tests/test_pr_agent_workflow.py:411-640, 485-527`) ported as bats
- [ ] [AC6] Add caller fixtures `tests/fixtures/callers/{kubelab,dotfiles,hive}.yml`, plus a bats case that each fixture's `with:` and `secrets:` match the declared interface
- [ ] C: README (inputs, secrets, caller template with the `ci:` consumer step in dotfiles `secrets/registry.yaml` and `dotf secrets sync ci`, pinning rule) and a CI workflow (bats and actionlint); tag `v0.1.0`

### C/D. H: pilot

- [ ] [AC5] H PR-C (draft): add `.github/workflows/pr-agent-shared.yml`.
  - It is pinned to C's `v0.1.0` SHA and passes `pool: embedded` and `dotf_version`.
  - The caller owns triggers, permissions and concurrency.
  - A pin-shape test is added.
- [ ] [AC5] On one PR, both jobs run, and the summary names the member, the pool source and the dotf version
- [ ] D: once PR-C is open, add `ci:mlorentedev/hive` to the consumers of `PR_AGENT_ANTHROPIC_API_KEY` in `secrets/registry.yaml`, with its bats test, in the shape of #2508. The owner then runs `dotf secrets sync ci` (R5).
- [ ] [AC5] Tabulate seven days of runs in `verification.md`
- [ ] [AC5] H PR-D: remove the old job. Record that `review` is not a required check (`gh api repos/mlorentedev/hive/branches/master/protection`).

## Closing

- [ ] Every acceptance criterion is covered by a test or a recorded run
- [ ] #2217 is closed by A5, #1570 by B with the AC6 evidence, and hive#503 by PR-D. ts-bridge-91 is told when the pilot is green (ts-bridge#474).
- [ ] `verification.md` filled; promotion lines answered
- [ ] Independent adversarial review (not the implementer) before archive
