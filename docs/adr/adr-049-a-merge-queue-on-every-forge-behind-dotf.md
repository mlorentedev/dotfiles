---
id: "ADR-049-a-merge-queue-on-every-forge-behind-dotf"
type: adr
status: proposed
owner: manu
date: "2026-10-10"
supersedes: []
extends: [adr-047-standing-merge-grant-opt-in-per-repository]
issue: mlorentedev/dotfiles#2246
tags: [architecture, decision, forge, ci, merge-queue, iac]
created: "2026-10-10"
---

# ADR-049: Every repository lands through a merge queue, on GitHub, GitLab and Gitea, behind one `dotf` interface

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

The owner works on three forges, and has repositories with pull requests on each (owner, 2026-10-10). What each
offers was checked on 2026-10-10:

| Forge | Native queue | Without a paid plan |
|---|---|---|
| GitHub | Merge queue, available only to repositories that an organization owns | Free organization, public repositories |
| GitLab | Merge trains, Premium and Ultimate tiers only | marge-bot, a maintained community fork (v1.1.0, 2026-02), BSD-3 |
| Gitea / Forgejo | None; go-gitea/gitea#36392 is an unaccepted proposal | gitea-mq (Gitea and GitHub, MIT, needs PostgreSQL), or shunt |

No free tool covers all three forges. Zuul has drivers for GitHub and GitLab but none for Gitea, and it is
heavy to operate.

dotfiles is the canonical project for every other repository, and every development agent uses `dotf`
(owner, 2026-10-10).

## Decision

1. **Every repository with pull requests lands through a merge queue.** `strict` comes off where the queue is
   on, because the queue gives the same guarantee without the update cascade.
2. **Agents land through one interface, whatever the forge.** `dotf pr land <n>` enqueues: on GitHub it adds
   the PR to the merge queue, on GitLab it assigns the merge request to marge-bot, and on Gitea it applies
   gitea-mq's label. A forge adapter sits behind the command. No agent updates a branch or calls a forge's merge
   API directly.
3. **The engine is the forge's own where one exists, and a sidecar only where it does not.**
   - **GitHub:** the native merge queue. The repositories move to a free organization.
   - **Gitea:** gitea-mq.
   - **GitLab:** marge-bot, or merge trains if the instance has Premium.

   Each sidecar is deployed on kubelab as code (Helm), next to the forge it serves.
4. **The queue is declared as code in dotfiles.** `forge/` declares, per repository:
   - the queue (merge method, build concurrency, group size, "only merge non-failing");
   - the required checks.

   `dotf forge … apply/check` converges each forge idempotently, a second run reports `changed=0`, and
   `dotf doctor` reports drift. This extends GUARD-017 from classic branch protection to the queue, through a
   driver per forge.
5. **CI has two lanes, published as templates from dotfiles:**
   - **Pull-request lane:** lint, unit tests and the review, fast, on every push.
   - **Queue lane:** the expensive jobs (Windows, macOS, integration), only on the queue's candidate. That is
     `merge_group` on GitHub, `gitea-mq/*` branches on Gitea, and marge-bot's pipeline on GitLab.

   Every required check reports in both lanes, or the queue waits for it forever. The LLM review never runs in
   the queue lane. The templates ship as reusable workflows for GitHub (which Gitea Actions also reads) and as
   an `include:` for GitLab, so other repositories consume them instead of copying them.
6. **Enqueueing is the supervised act.** The queue lands a PR only after it was deliberately enqueued, under
   the same conditions as a merge today: the user authorized that PR, or the ADR-047 grant holds. The rule in
   the git-workflow pattern ("merge is a supervised action, never a queued automatic one") changes in the vault
   to say so. That is an owner edit, because it changes the agents' own rules. `allow_auto_merge` stays
   `false`: the native GitHub queue does not need it.

## Consequences

- **The GitHub repositories move to an organization.** Only the owner can create it (GitHub has no API for a
  free organization) and approve each transfer. Git and web redirects survive a transfer. These do not, and
  are part of the move:
  - GitHub App installations (pr-agent, CodeRabbit) are reinstalled on the organization;
  - collaborators become outside collaborators unless teams are created;
  - `add-to-project.yml` looks up the user's Project.

  Whether a user-level Project accepts items from organization repositories is not verified. It must be
  checked before the first transfer, and if the answer is no, the board moves too.
- **The first day of the queue is the dangerous one.** `review-attestation`, `spec-gate` and
  `knowledge-gate` trigger only on pull-request, comment and workflow-run events today. Without a
  `merge_group` trigger, they never report on the candidate, and nothing lands. The rollout adds those
  triggers before it turns the queue on, and the GUARD-017 preflight (a check must have reported before it is
  required) covers the queue lane too.
- **kubelab carries the Gitea and GitLab queues.** When the cluster is down, PRs on those forges do not land.
  GitHub does not depend on it, which is the deciding reason against gitea-mq on GitHub.
- **The rollout goes repository by repository:** dotfiles first, measured on head commits per merge and time
  to `main` against the numbers above, then kubelab, hive and the rest.
- GUARD-017's out-of-scope line on rulesets and merge queues is withdrawn, because rulesets are free for public
  repositories. The other layers are rows on the epic, specified when the WIP limit allows:
  - the `dotf pr land` adapters;
  - the CI templates;
  - the sidecar deployments.

## Alternatives rejected

- **Drop `strict` and revert on breakage.** It needs no tooling. With several landers in parallel, two PRs that
  pass alone and break `main` together become routine, and a revert is itself a change that waits for CI.
- **gitea-mq for GitHub too.** That would be one engine for two forges, with no transfer. On GitHub it turns
  on `allow_auto_merge`, which ADR-047 and the git-workflow pattern forbid. It also needs the Administration
  permission, a bypass on the ruleset it creates, and an inbound webhook, so kubelab would have to accept
  traffic from the internet and would become the merge path of every repository.
- **Mergify, Aviator, Trunk or Graphite.** They work only on GitHub and are paid, or free with conditions.
- **Zuul for every forge.** It has no Gitea driver, and it is a CI system to operate, not a queue to adopt.
- **Serial landing with `dotf pr land` alone.** This is today's state. Its lock is local to one machine, so it
  does not hold once several agents and people land at the same time.
