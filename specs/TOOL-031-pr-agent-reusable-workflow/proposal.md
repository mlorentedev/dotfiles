---
id: "TOOL-031-pr-agent-reusable-workflow"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-21"
issue: "mlorentedev/dotfiles#2893"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, pr-agent, reusable-workflows, dotf]
template_version: "1.0"
wip_override: "Owner-approved 2026-10-22: unblocks the PR-Agent reviewer pool for hive#503 and ts-bridge wave D6 (12 active, limit 10, 2026-10-21)"
---

# TOOL-031: PR-Agent's decisions in `dotf pr`, served by a reusable workflow

## Why

<!-- from issue #2893: ci(pr-agent): serve the reviewer pool as a reusable workflow in ci-workflows, piloted in hive -->

A repository without the reviewer pool sends every PR-Agent review to NaN alone. NaN allows five concurrent requests per model, shared by pi, qq, dotfiles, kubelab and hive. Under load, PR-Agent hangs and publishes nothing. In hive on 2026-10-21/22, three reviews were published across five PRs, and about eight other runs hit the 12-minute timeout (hive#503).

The pool fixes this (AI-045 AC9, #1923: a preflight, a weighted draw, Haiku when NaN does not answer, a publish guard). Today its logic is four bash scripts that run only in dotfiles' workflow, and they travel as copies. kubelab's copy has already drifted (#1570); a copy into hive would be a third.

The owner settled the shape on 2026-10-18 (kubelab#2295): "The draw moves into `dotf` … `scripts/pr-agent-route.sh` is then deleted, not copied … one two-repo spec covering `dotf review route`, the reusable workflow and the pod consumer". That decision rejected shipping the bash route elsewhere, because it would leave three readers of the pool in two languages. CLI-101 (#2217) already scoped the port of all four scripts as one arc. On 2026-10-22 the owner confirmed that this spec is that one (row D3 of #1843).

## What

ADR-042's classes place each piece. No new rule is needed.

1. **The decisions move into Go: `dotf pr route|preflight|guard|gate` (class 3: one binary everywhere, CLI-101).**
   - `route` ports `pr-agent-route.sh`: the weighted draw, the Sonnet risk route, and the Anthropic allowlist with a price row and a worst-review ceiling for every allowed model. A pool that names a model outside the allowlist fails before any request (ADR-046 §5, the hard requirement in kubelab#2295).
   - `preflight`, `guard` and `gate` port the other three scripts, with the same outputs contract the workflow reads.
   - Each command prints a `--json` result and, in CI, the same job-summary lines the scripts print today.
   - The bats suites become Go tests with the same cases, including an exact share at every point of the total weight.
   - The four scripts are deleted. kubelab's forge pod can run the same command, pinned to `DOTF_VERSION` (TOOL-117).
2. **The pool has one source: dotfiles' `harness/reviewer-pool.json`.** `dotf` embeds a copy at build time.
   - **Explicit source.** `route` reads the pool the caller names, either `--pool embedded` or `--pool <path>`. There is no implicit fallback: an absent file never turns into a build-time default without anyone noticing (the reason `model_map.go` refuses an embedded fallback).
   - **Job summary.** It names the source and the dotf version.
   - **Consumers.** A consumer with no local pool, such as hive, passes `embedded`, so its dotf pin is also its pool pin. HARNESS-180 (#2439) still delivers the file to repositories that run `dotf spec review`.
3. **A thin reusable workflow in `mlorentedev/ci-workflows` (class 1), plus the push gate as a composite action (class 2).**
   - **What it does.** The workflow installs `dotf` at the version the caller passes and runs the PR-Agent action. It calls `dotf pr …` for each decision.
   - **Its own commit.** It checks out its own files through `job.workflow_sha` and `job.workflow_ref`, never `github.workflow_sha`.
   - **Secrets.** `NAN_API_KEY` is required and `PR_AGENT_ANTHROPIC_API_KEY` is optional. Each is declared, never `secrets: inherit`.
   - **Caller.** The caller owns triggers, permissions and workflow-level concurrency. The repository-wide job lane stays inside the job.
   - **kubelab's differences, as inputs.** `declare_unreviewed` (default off) puts the label and the rationale from the caller's `escape.*` on an ignored-only PR. It is idempotent and clears a stale label. `skip_branch_prefixes` covers branches such as `deploy/` and `promote/`.
   - **Artifact.** The `pr-number` artifact (`pr-number.txt`) is uploaded on every run.
4. **dotfiles keeps testing its own changes.** Its workflow builds `dotf` from the PR's head, so a PR that changes the route runs its own version (CLI-101's requirement). Consumers use a released, pinned version.
5. **hive is the pilot, in two PRs.** The first adds a caller pinned by SHA next to the old job. After a week with both outputs, the second removes the old job. hive's `review` check is not required, so ADR-042 decision 6's protection steps are moot here. Two PRs are kept for the side-by-side comparison (owner decision, 2026-10-22).

`#1570` (kubelab's design in the canon) closes with the reusable workflow and its kubelab-shaped fixture. It does not close through a port into dotfiles' local `pr-agent.yml`. The port would reverse #1618 for dotfiles: the canon chose not to declare ignored-only PRs, because review-attestation "still needs CodeRabbit or a human" (`pr-agent.yml:344-349`). Every other ported item changes nothing that dotfiles runs today.

## Out of scope

- **Serialising repositories against NaN's five slots.** GitHub concurrency groups are per repository. What moves load off a single model is the draw, the preflight and the Haiku route, and the Haiku route costs money in every repository that holds the key.
- **The forge pod consuming `dotf pr route`.** That is kubelab's TOOL-117 (kubelab#2295 item 2) and #2467. This spec only keeps the command usable there: it needs no GitHub context, and it takes the changed lines and labels as flags.
- **#1618's policy question for dotfiles.** `declare_unreviewed` defaults to off, so dotfiles keeps today's behaviour.
- **The attestation side of kubelab's design.** It waits on kubelab#2593 (DELIVERY-004).
- **Migrating dotfiles' own caller, kubelab and ts-bridge.** That is wave D6 (ts-bridge#474).
- **HARNESS-180's delivery and drift check for `dotf spec review` pools** (#2439).

## Risks / open questions

- **R1: size and order.** Five commands' worth of port is far past the 300-LOC cap, so it splits into PRs:
  - A1: `route`, with the allowlist, prices and ceiling;
  - A2: `preflight`;
  - A3: `guard`;
  - A4: `gate`;
  - A5: dotfiles' workflow calls `dotf pr …` built from head, and the scripts are deleted.

  A dotf release follows A5, then B (ci-workflows), then C and D (hive). Relief for hive#503 arrives only after the release.
- **R2: embedded pool versus HARNESS-180.** The owner's 10-18 text names HARNESS-180 as the delivery mechanism. This spec adds `--pool embedded` for consumers that have no file. Each choice is explicit and named in the job summary. The owner reviews it with the criteria. [AGENT-DRAFT — review before archive]
- **R3: installing `dotf` in CI.** The workflow needs a pinned binary on `ubuntu-latest`. A checksum-verified release download is preferred to `setup-go` plus a build, which would add both to every review. Check `dotf`'s release assets and any installer precedent before PR-B.
- **R4: `vars.PR_AGENT_PROVIDER` in a called workflow** is not asserted. It becomes a smoke row on hive's first run, and an input if that row fails.
- **R5: owner actions.**
  - Create the repository: `gh repo create mlorentedev/ci-workflows --public`.
  - Decide whether hive gets `PR_AGENT_ANTHROPIC_API_KEY` (a cost decision). Without the key, hive draws over NaN only, and AC5's Haiku clause cannot be observed.

## Acceptance criteria

[AGENT-DRAFT — review before archive]

- [ ] **AC1 (route).** `dotf pr route` matches the bash route on the same inputs:
  - every point of the total weight draws the member that owns it;
  - a PR at or above `min_changed_lines`, or carrying the label, goes to the `route: risk` member;
  - an Anthropic model outside the allowlist, or without a price row and a ceiling, fails before any probe;
  - the key travels on stdin, never on argv.

  The ported Go tests pass, and so does a differential test against the last bash version.
- [ ] **AC2 (preflight, guard, gate).** Each command reproduces its script's cases as Go tests. Its outputs keep the names the workflow reads.
- [ ] **AC3 (dotfiles).** dotfiles' `pr-agent.yml` calls `dotf pr …` built from the PR's head, and the four scripts are gone. A PR that changes the route exercises its own version, shown by one live run linked in `verification.md`.
- [ ] **AC4 (reusable workflow).** Bats tests in `ci-workflows`, plus actionlint, assert each of these:
  - the workflow declares `on: workflow_call`, with `NAN_API_KEY` required and `PR_AGENT_ANTHROPIC_API_KEY` optional;
  - it contains no `secrets: inherit`, no branch literal and no `github.workflow_sha`;
  - `dotf` is installed at the caller's version, with a checksum check;
  - the push-gate action fails the step when the gate errors;
  - `declare_unreviewed` on writes the label and the section, idempotently, and off prints the notice only.
- [ ] **AC5 (hive pilot).**
  - PR-C pins the reusable workflow by full SHA, and one PR runs both jobs; the new summary names the drawn member, the pool source and the dotf version.
  - For seven days after PR-C merges, every run names its member, and every run whose NaN preflight fails is reviewed by Haiku (R5).
  - PR-D removes the old job.
- [ ] **AC6 (fleet).** Caller fixtures shaped like kubelab (`declare_unreviewed: true`, `deploy/` and `promote/` skips) and like dotfiles pass the reusable workflow's tests with no change to it. This closes #1570 and shows that D6 can migrate unchanged.

## References

- Bitácora:
  - mlorentedev/dotfiles#2893 (row D3 of epic #1843), which also closes #2217 (CLI-101) and #1570.
  - Consumers: hive#503 and ts-bridge#474.
  - Related: #2439 (HARNESS-180), #1618.
- Owner decision of 2026-10-18: mlorentedev/kubelab#2295 (TOOL-117), last comment.
- ADRs: ADR-042 (shared CI is referenced, not copied), ADR-046 (Anthropic reviewers; §5 allowlist), ADR-040 (push gate), ADR-054 (container action pinned by digest).
- `cli/internal/harness/model_map.go`: why an implicit embedded fallback is refused.
- GitHub docs, contexts reference (`job.workflow_sha`, `job.workflow_ref`): https://docs.github.com/en/actions/reference/workflows-and-actions/contexts
