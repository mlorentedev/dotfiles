---
id: "TOOL-031-pr-agent-reusable-workflow"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-21"
issue: "mlorentedev/dotfiles#2893"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, pr-agent, reusable-workflows]
template_version: "1.0"
wip_override: "Owner-approved 2026-10-22: unblocks the PR-Agent reviewer pool for hive#503 and ts-bridge wave D6 (12 active, limit 10, 2026-10-21)"
---

# TOOL-031: PR-Agent reviewer as a reusable workflow

## Why

<!-- from issue #2893: ci(pr-agent): serve the reviewer pool as a reusable workflow in ci-workflows, piloted in hive -->

A repository without the reviewer pool sends every PR-Agent review to NaN alone. NaN allows five concurrent requests per model, shared by pi, qq, dotfiles, kubelab and hive. Under load, PR-Agent hangs and publishes nothing. In hive on 2026-10-21/22, three reviews were published across five PRs, and about eight other runs hit the 12-minute timeout (hive#503).

The pool fixes this (AI-045 AC9, #1923: a preflight, a weighted draw, Haiku when NaN does not answer, a publish guard), but it travels only as copies. kubelab's copy has already drifted (#1570), and a copy into hive would be a third. ADR-042 decides that shared CI is referenced from `mlorentedev/ci-workflows`, not copied. This spec is row D3 of #1843.

## What

1. **A reusable workflow in `mlorentedev/ci-workflows` that carries both designs.**
   - **Base.** The canonical dotfiles workflow: the push gate, the digest-pinned image, the draft read from the live PR, the comment concurrency key, the pool draw, the Anthropic attempts and the guard.
   - **kubelab additions, behind inputs.**
     - `declare_unreviewed` (default `false`): an ignored-only PR gets the `escape.label` label and the `escape.section` rationale read from the caller's `harness/review-attestation.json` at the default branch. Setting it is idempotent, and the label is read back after writing. A stale label is cleared once the PR becomes reviewable.
     - `skip_branch_prefixes`: `release-please--`, `deploy/`, `promote/` and so on.
   - **Provider override.** Both Anthropic attempts set `LITELLM__CUSTOM_LLM_PROVIDER: anthropic`. kubelab's `.pr_agent.toml` sets a `[litellm]` provider that otherwise leaks into them (#2302). dotfiles sets it per step (`pr-agent.yml:638`), so it is unaffected today.
   - **Reviewable parser.** The canon's parser stays (tomllib, `[ignore].regex`, PR-Agent's file filter) and keeps failing open: when it cannot read `.pr_agent.toml`, it reviews (owner decision, 2026-10-22).
   - **#1570 closes here.** It closes with this workflow, not with a port into dotfiles' local copy (see Sequence).
2. **The workflow ships its own logic and data.**
   - It contains the four scripts, the push gate as a composite action that fails closed (ADR-040, ADR-042 decision 1), and the **PR-Agent pool**. The pool holds the members' `pr_agent` blocks, `pr_agent_risk`, and kubelab's `forge_rank` as a key the CI draw ignores (owner decision, 2026-10-22).
   - It checks these files out from its own repository at its own commit (R1), never from the caller.
3. **An explicit interface.**
   - **Secrets.** `NAN_API_KEY` is required and `PR_AGENT_ANTHROPIC_API_KEY` is optional. Each is declared, never `secrets: inherit` (decision 4).
   - **Caller.** The caller owns triggers, permissions and workflow-level concurrency (decision 5). The repository-wide job lane stays inside the reusable job, keyed on the caller's repository.
   - **Artifact.** The run uploads the artifact `pr-number` (file `pr-number.txt`) on every run, kubelab's contract. Each repository's attestation gate adopts it when that repository migrates.
4. **hive is the pilot, in two PRs.** The first adds a caller pinned by SHA, next to the old job. After a week with both outputs, the second removes the old job. hive's `review` check is not required, so ADR-042 decision 6's protection steps are moot here; the evidence goes in `verification.md`. Two PRs are kept for the side-by-side comparison (owner decision, 2026-10-22).

**Placement rule (owner question, 2026-10-22).** Reusable logic, and the data it reads at runtime, live in `ci-workflows`. Per-repository policy that a `dotf` command reads stays in the repository that applies it. `harness/reviewer-pool.json` mixes the two today:
- `dotf` reads `id`, `runner`, `provider`, `model`, `role`, `secret_id`, `auth`, `vendor` and `signs` (`cli/internal/spec/review_launch.go:71-84`), and never `pr_agent`.
- The CI scripts read only `.pr_agent.*` and `.pr_agent_risk` (`scripts/pr-agent-route.sh:126-145, 253`).

The PR-Agent half moves and the spec-reviewer half stays. Neither repository reads the other at runtime.

### Sequence, and why #1570 lands in ci-workflows

ADR-042 decision 7 ports kubelab's design into the canonical workflow before the pilot. Ported into dotfiles' local `pr-agent.yml`, every item changes nothing dotfiles runs today:
- **`Declare unreviewed`** would reverse #1618. The canon chose not to declare ignored-only PRs: "review-attestation still needs CodeRabbit or a human" (`pr-agent.yml:344-349`). As a caller input it stays off for dotfiles and on for kubelab.
- **The stale-label clear** fires only where that label is set.
- **The provider override** is already per step in dotfiles.
- **The artifact name** in dotfiles agrees with its own gate today; it needs the fleet name only when dotfiles becomes a caller.

So the port would land in a file that D6 deletes, in a repository frozen for 0.77.0. The single design that decision 7 asks for is the ci-workflows workflow. kubelab's pytest pins become bats tests there, not in dotfiles as #1570 suggested. This follows decision 7's intent, not its letter.

## Out of scope

- **Serialising repositories against NaN's five slots.** GitHub concurrency groups are per repository, so nothing here queues hive, kubelab and dotfiles against each other. What moves load off one model is the draw, the preflight and the Haiku route, and that route costs money in each repository that holds the key.
- **#1618's open policy question:** what attestation means for an ignored-only PR in dotfiles. `declare_unreviewed` defaults to off, so dotfiles keeps today's behaviour.
- **The attestation side of kubelab's design.** It waits on kubelab#2593 (DELIVERY-004), which narrows the staging-deploy exemption. kubelab can drop `deploy/` and `promote/` from its input when that lands.
- **Migrating dotfiles, kubelab and ts-bridge to the caller.** That is wave D6 (ts-bridge#474). Until dotfiles migrates, its local workflow and pool keep their `pr_agent` blocks, so both copies exist for that window. Removing them is part of dotfiles' D6 migration.
- **kubelab's forge render** (`tests/test_pr_agent_config_render.py:155-200`) reading `forge_rank` from the shared pool instead of its local copy (D6).
- **spec-gate, board workflows and the other ADR-042 classes** (D4 and later).

## Risks / open questions

- **R1 (resolved 2026-10-22): how the called workflow finds its own commit.** GitHub's contexts reference says `job.workflow_sha` and `job.workflow_ref` refer, for jobs defined in a reusable workflow, to "the reusable workflow file". The reference gives an example that checks out the called workflow's own source with `job.workflow_sha`.
  - The checkout parses the repository from `job.workflow_ref` (`owner/repo/.github/workflows/<file>@<ref>`), so a fork checks out itself.
  - The upstream-contract read moves to the same pair. Today it reads `${GITHUB_REPOSITORY}@${github.workflow_sha}` (`pr-agent.yml:282, 297-303`), which in a called workflow would read the caller.
  - A bats test fails if `github.workflow_sha` appears in the reusable workflow.
- **R2: `vars.PR_AGENT_PROVIDER` inside a called workflow** is not asserted either way. It becomes a smoke row on hive's first run, and an input if that row fails.
- **R3: PR-B is an initial import of about 830 lines of YAML, about 950 lines of script, the pool and the tests.** It exceeds the 300-LOC cap. It splits into B1 (scripts, pool and their bats, moved), B2 (the reusable workflow and its bats) and B3 (the composite action, the kubelab inputs and the AC6 fixtures). Each is reviewed on its own.
- **R4: CI freeze.** Nothing is pushed to dotfiles until 0.77.0 is out. This spec's own PR (the spec folder) waits for it too.
- **R5: owner actions gate PR-B and the Haiku half of AC5.**
  - Create the repository (`gh repo create mlorentedev/ci-workflows --public`).
  - Sync `PR_AGENT_ANTHROPIC_API_KEY` to hive.

  Without the key, hive draws over NaN members only, and AC5's Haiku clause cannot be observed.
- **R6: ADR-042 has no row for this split.** Its four classes cover neither data shipped with a reusable workflow nor per-repository policy a `dotf` command reads. Whether the placement rule amends ADR-042 is a promotion question at archive.

## Acceptance criteria

[AGENT-DRAFT — review before archive]

- [ ] **AC1.** In ci-workflows, `declare_unreviewed` behaves as follows:
  - On, an ignored-only fixture PR gets the label and the section, a second run leaves one label, and a stale label is removed once the PR is reviewable.
  - Off, the run prints the notice only and writes nothing.

  Bats tests pin each case.
- [ ] **AC2.** The reusable workflow:
  - declares `on: workflow_call`, with `NAN_API_KEY` required and `PR_AGENT_ANTHROPIC_API_KEY` optional;
  - contains no `secrets: inherit`, no hard-coded branch name, and no `github.workflow_sha`;
  - reads its scripts and pool from `job.workflow_sha`;
  - sets `LITELLM__CUSTOM_LLM_PROVIDER: anthropic` on both Anthropic attempts;
  - uploads `pr-number` / `pr-number.txt` on every run.

  Bats tests pin each point, and actionlint passes.
- [ ] **AC3.** The push-gate composite action fails the step when its script errors. A bats test pins it.
- [ ] **AC4.** hive PR-C calls the reusable workflow pinned by a full SHA, and `check-actions-pinned.sh` passes on it. On one hive PR, the old job and the new caller both run, and the new job summary names the drawn member.
- [ ] **AC5.** For seven days after PR-C merges:
  - every hive review run names its drawn member in the job summary;
  - every run whose NaN preflight fails is reviewed by Haiku (needs R5's key).

  PR-D then removes the old job.
- [ ] **AC6.** A kubelab-shaped caller fixture (`declare_unreviewed: true`, `deploy/` and `promote/` skips) and a dotfiles-shaped one (push gate, attestation artifact) pass the reusable workflow's tests with no change to the workflow. This shows #1570 is closed and that D6 can migrate unchanged.

## References

- Bitácora: mlorentedev/dotfiles#2893 (row D3 of epic #1843). Precondition: #1570. Consumers: hive#503 and ts-bridge#474. Related: #1618.
- ADR-042 (shared CI is referenced, not copied), ADR-040 (push gate), ADR-054 (container action pinned by digest).
- AI-045 AC9 / #1923 (the pool). TOOL-013 and TOOL-023 (archived PR-Agent specs).
- kubelab: `.github/workflows/pr-agent.yml`, `tests/test_pr_agent_workflow.py`, kubelab#2593 (DELIVERY-004).
- GitHub docs, contexts reference (`job.workflow_sha`, `job.workflow_ref`): https://docs.github.com/en/actions/reference/workflows-and-actions/contexts
