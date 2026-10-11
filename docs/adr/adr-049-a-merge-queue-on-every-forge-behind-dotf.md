---
id: "ADR-049-a-merge-queue-on-every-forge-behind-dotf"
type: adr
status: proposed
owner: manu
date: "2026-10-10"
supersedes: []
extends: [adr-047-standing-merge-grant-opt-in-per-repository]
issue: mlorentedev/dotfiles#2249
tags: [architecture, decision, forge, ci, merge-queue, iac]
created: "2026-10-10"
---

# ADR-049: Every repository lands through one merge queue, gitea-mq on kubelab's prod VPS, behind `dotf pr land`

## Context

Each repository protects its default branch with "require branches to be up to date" (`strict: true` in
`forge/branch-protection.json`). That rule exists to stop two pull requests that pass alone from breaking
`main` together. Its cost grows with the number of PRs open at once: every merge leaves every other PR behind,
and each one has to merge `main` in and run its whole CI again. Several agents and people now work on these
repositories in parallel, so that cost is the bottleneck of the whole development loop.

Measured on dotfiles over 52 hours (2026-10-08T01Z to 2026-10-10T05Z, #2246):

- 302 PR head commits ran CI for 61 merges, about 5 per merge, at about 20.5 runner-minutes each. The median
  wall time is 8 minutes and p90 is 17. The long pole is `test-windows` (6.1 min median, 8.5 p90).
- 43 of those head commits were update-branch merges of `main`. They triggered 361 runs and used about 880
  runner-minutes.
- `review-attestation` ran 1,537 times.

The repository's current mitigation is to land one PR at a time and update only the next one (`dotf pr land`,
lesson 344, CLI-098). It removes some of the update cascade. It does not survive several landers working at
once, and its lock is local to one machine.

The established answer is a merge queue: Graydon Hoare's "Not Rocket Science" rule, as implemented by bors,
Zuul, Chromium's CQ, Uber's SubmitQueue and GitHub's merge queue. A PR is tested once, against `main` plus the
PRs ahead of it, right before it lands. Nobody updates a branch, and the expensive jobs run only on the queue's
candidate.

The owner's repositories with pull requests live on GitHub and on a self-hosted Gitea. GitLab is not deployed
(kubelab ADR-050 D3 names it only as a future adapter). What each offers was checked on 2026-10-10:

| Forge | Native queue | Without a paid plan |
|---|---|---|
| GitHub | Merge queue, available only to repositories that an organization owns | Free organization with public repositories, or a sidecar |
| Gitea / Forgejo | None; go-gitea/gitea#36392 is an unaccepted proposal | gitea-mq, or shunt |
| GitLab | Merge trains, Premium and Ultimate tiers only | marge-bot |

[gitea-mq](https://github.com/Mic92/gitea-mq) is the only free queue that serves GitHub and Gitea from one
process. It is MIT-licensed Go, needs PostgreSQL, and publishes a multi-arch image (`ghcr.io/mic92/gitea-mq`).
Reading its source (2026-10-10) settled how it lands a PR on GitHub:

- **Default mode** arms the forge's auto-merge, so it needs `allow_auto_merge=true`. With the App's
  Administration permission, its auto-setup turns that setting on.
- **Label mode** enqueues a PR that carries the merge label. When the PR's run on the `gitea-mq/<pr>` branch
  passes, gitea-mq merges it itself through the API (`finalizeLabeledMerge`, `internal/poller/poller.go`), with
  the repository's default merge method. No auto-merge is involved.
- **Batch mode** (`GITEA_MQ_BATCH_MAX` ≠ 1) fast-forwards the target branch and lands merge commits, so a
  squash-only repository keeps the batch size at 1.

kubelab has a place to run it (measured 2026-10-10):

- The prod VPS (Hetzner CAX21, ARM64) is always on and public.
- It already receives github.com webhooks (Argo CD, n8n) and runs pr-agent, a webhook service of the same
  shape (HMAC, Traefik, SOPS secrets).
- It has a shared PostgreSQL with per-application tenants.
- Its namespace quota leaves 1,568 Mi of requests and 1,792 Mi of limits free at steady state. In the worst
  case (CronJobs and every rolling deploy at once) it leaves 832 Mi of requests and 128 Mi of limits.

dotfiles is the canonical project for every other repository, and every development agent uses `dotf`
(owner, 2026-10-10).

## Decision

1. **Every repository with pull requests lands through a merge queue.** `strict` comes off where the queue is
   on, because the queue gives the same guarantee without the update cascade.
2. **One engine for GitHub and Gitea: gitea-mq in label mode, with a batch size of 1.** One process serves both
   forges.
   - The GitHub App gets no Administration permission. Its auto-setup is therefore skipped, and
     `allow_auto_merge` stays `false`.
   - Repositories are squash-only, so the default merge method it uses is a squash.
3. **It runs on kubelab's prod VPS** as a prod-overlay service modelled on pr-agent, with a PostgreSQL tenant
   provisioned on the deploy path. It passes kubelab ADR-028's 3 AM test because the GitHub repositories it
   serves are always on. Deployment is kubelab#2156.
4. **Agents land through one interface.** `dotf pr land <n>` checks the same conditions as today, adds the
   merge label, and returns. It does not hold the session while the queue runs: an agent hands the PR over,
   as it does with CI today. `dotf pr land --status <n>` reads where the PR is (queued, testing, merged or
   ejected with gitea-mq's reason) for an agent whose next step depends on the merge. No agent updates a
   branch or calls a forge's merge API directly. A GitLab adapter (marge-bot or merge trains) is added behind the same command only if a GitLab
   instance comes into use.
5. **The queue's half of the protection is declared as code in dotfiles.** `forge/` declares, per repository:
   - the allowed merge methods;
   - `strict`;
   - the required checks, including `gitea-mq`.

   `dotf forge … apply/check` converges each forge idempotently, a second run reports `changed=0`, and
   `dotf doctor` reports drift. The required `gitea-mq` check that its auto-setup would have added through a
   ruleset is added here, through the classic branch protection that `forge/` already manages. That works on
   every plan, public or private, so no ruleset is needed. This extends GUARD-017.
6. **CI has two lanes, published as templates from dotfiles:**
   - **Pull-request lane:** lint, unit tests and the review, fast, on every push.
   - **Queue lane:** the expensive jobs (Windows, macOS, integration), on `push` to `gitea-mq/**` branches.

   gitea-mq waits on the merge branch for every check that branch protection requires, so every required
   check reports in the queue lane, or the queue waits for it forever. The checks that judge the PR itself
   (`review-attestation`, `spec-gate`, `knowledge-gate`) are not re-run there, and the LLM review never runs
   there. Their queue-lane job passes only if the same check passed on the PR head that the candidate merges.
   That head is the merge commit's second parent, so the check reads it from the commit and never from the
   branch name: a branch named to match would otherwise grant the check (git-workflow, "branch-name
   matching"). GitHub still enforces the PR-lane checks on the PR head when gitea-mq merges it. The templates are reusable workflows, which Gitea Actions also reads, so other repositories
   consume them instead of copying them.
7. **Enqueueing is the supervised act.** A PR is labelled only under the conditions that authorize a merge
   today: the user authorized that PR, or the ADR-047 grant holds.
   - The rule in the git-workflow pattern ("merge is a supervised action, never a queued automatic one")
     changes in the vault to say so. That is an owner edit, because it changes the agents' own rules, and it
     lands before any agent enqueues.
   - `allow_auto_merge` stays `false` on every repository.
8. **dotfiles pilots it (#2249).** Other repositories follow one at a time, and only once the pilot is stable.

## Consequences

- **The VPS is on the merge path of every repository.** While it is down, CI still runs but nothing lands. The
  break-glass is the owner merging by hand. The service gets an Uptime Kuma probe and a Grafana alert.
- **A young dependency is on that path.** gitea-mq's first commit is from 2026-02-09, it has one release
  (`v0.1.0`), and one author wrote most of it. That author runs the GitHub backend on their own repositories.
  The image is pinned by digest, and the code is small and MIT-licensed, so a fork is the fallback.
- **The shared PostgreSQL is the tight resource**, not the service: its limit is 256 Mi, half of it
  `shared_buffers`. gitea-mq gets a small connection pool, and Postgres is measured before and after the tenant
  lands.
- **Throughput is serial:** one PR per queue-lane run. That is about 4 to 7 merges an hour against a measured
  1.2. `GITEA_MQ_SKIP_QUEUE_IF_UP_TO_DATE` lands a PR already on top of `main` without a second run.
- **The queue lane must exist before the check is required.** `review-attestation`, `spec-gate` and
  `knowledge-gate` trigger only on pull-request, comment and workflow-run events today. The rollout makes each
  report on `gitea-mq/**` branches before `gitea-mq` becomes required, and the GUARD-017 preflight (a check
  must have reported before it is required) covers the queue lane too.
- **Repositories become squash-only.** The merge and rebase buttons go away.
- **The Gitea queue works only while Gitea does.** Gitea runs on the on-demand Beelink, so the queue adds no
  availability there; it adds the same serialization.
- **The rollout is measured.** On dotfiles, head commits per merge and time to `main` are compared with the
  numbers above before any other repository moves.

## Alternatives rejected

- **GitHub's native merge queue, with gitea-mq only for Gitea.** It needs no service of ours for GitHub, and
  GitHub runs it. But it is available only to organization-owned repositories, so every GitHub repository
  would move to an organization. GitHub App installations would have to be reinstalled, collaborators would
  become outside collaborators, and whether the user-level Project board accepts organization issues is
  unverified. It would also mean two engines and two adapters behind `dotf pr land`.
- **gitea-mq in default or batch mode.** Default mode needs `allow_auto_merge=true`, which ADR-047 and the
  git-workflow pattern forbid. Batch mode lands merge commits and makes the App a bypass actor that pushes to
  `main`.
- **Drop `strict` and revert on breakage.** It needs no tooling. With several landers in parallel, two PRs that
  pass alone and break `main` together become routine, and a revert is itself a change that waits for CI.
- **A queue written into `dotf` and run as a CI job.** It needs no service, but it means maintaining a
  home-grown queue, with the same bypass to push to `main`.
- **Mergify, Aviator, Trunk or Graphite.** They work only on GitHub and are paid, or free with conditions.
- **Zuul.** It has no Gitea driver, and it is a CI system to operate, not a queue to adopt.
- **Serial landing with `dotf pr land` alone.** This is today's state. Its lock is local to one machine, so it
  does not hold once several agents and people land at the same time.
