---
id: "OPS-023-bitacora-board-resilience"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#809"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-023-bitacora-board-resilience

## Why

<!-- from issue #809: OPS-023: add-to-project silently drops issues from the bitácora on any API failure -->

`add-to-project.yml` drops items off the bitácora board whenever an API call
fails, and says nothing. Both paths failed hard with no retry, and nobody watches
an Actions run fired by an `issues: opened` event — **a missing ticket looks
exactly like a ticket that was never filed.**

Five confirmed instances across three days and two repos, the most recent while
this very ticket was being filed. The root cause (migrated here from #774, now
closed as superseded) is that GitHub bills the 5,000-point GraphQL pool to the
**account**, not the token: an interactive session draining it takes CI's board
writes down with it. Per-purpose PATs (#321) cannot fix that, and the only real
isolation — a GitHub App installation token — is closed off by ADR-031, since
Apps cannot write to a user-owned Projects v2 board.

With isolation unavailable, the design has to be *degrade without losing data*.

## What

1. **Classify failures three ways** in `add-to-project.yml` instead of failing
   hard on everything:
   - **primary rate limit → soft-fail, no retry.** The reset can be an hour away;
     a job sleeping toward it burns a runner and still loses. Green job, loud
     `::warning::`, and the item is left for the reconciler.
   - **transient (5xx, network, secondary limit) → retry with backoff.**
   - **anything else → red.** A bad token, a renamed field or a deleted board
     must never look like the handled case.
2. **Unify both event types on one GraphQL call.** The issues path used
   `actions/add-to-project@v2.0.0`, which cannot be wrapped in a retry without
   re-running the whole job or adding a third-party retry action — a new
   supply-chain dependency in a repo that SHA-pins its actions (#692). The
   content node ID is already in the event payload, so unifying costs nothing and
   also deletes the PR path's separate node-ID lookup (one fewer call against the
   pool in question).
3. **`bitacora-rollout.sh --backfill-only`** — step 4 alone. Provisioning
   (linking, uploading a secret, pushing workflow files) stays a deliberate human
   act and must never run on a timer. The flag also removes the age-key
   dependency, which is what makes the reconciler runnable in CI at all.
4. **`bitacora-reconcile.yml`** — daily at 04:43 UTC. Heals whatever the
   event-driven path dropped. Healed drift is a `::notice::` count; the
   *reconciler itself* failing opens a deduplicated labelled issue and goes red,
   mirroring `pat-expiry.yml` (OPS-009).
5. **Test coverage**, which this machinery had none of — the reason a drop could
   stay invisible for three days.

## Out of scope

- **The multi-repo rollout.** This PR changes the canonical copy; propagating it
  is an operational act against other repositories and needs an explicit go.
- The `item-list` cache wrapper and `rateLimit` introspection carried over from
  #774. Both are real, neither is on the drop path; they belong in a follow-up.
- Migrating the board to an org (#640), the only true isolation.

## Acceptance criteria

- [x] **AC1** `--backfill-only` ensures every open issue and PR is on the board.
- [x] **AC2** `--backfill-only` performs no provisioning: no project link, no
      secret upload, no workflow push.
- [x] **AC3** `--backfill-only` never attempts an age decrypt, so it runs in CI
      where the key does not exist.
- [x] **AC4** A full run still provisions — the flag narrows scope, it does not
      change the tool.
- [x] **AC5** The add workflow keeps rate-limit and non-rate-limit failures on
      separate branches, and carries no blanket `continue-on-error`.
- [x] **AC6** The add workflow reads the node ID from the payload and no longer
      uses `actions/add-to-project`.
- [x] **AC7** The reconciler validates its `workflow_dispatch` repo-name input
      before word-splitting it into a command line.
- [x] **AC8** The reconciler is scheduled, and goes loud (issue + red) when it
      cannot run.

## Risks / open questions

- **The reconciler could become the next drain.** It sweeps every open item of
  every repo. Accepted deliberately: `item-add` is cheap, it runs once daily
  off-peak, and it soft-skips on rate limit so it can never spin against an
  exhausted pool. `workflow_dispatch` takes explicit repo args for a narrower run.
- **The healer needs the thing it heals.** A reconciler failure files an issue,
  which is itself added to the board by the workflow under repair. That is
  acceptable: the issue exists in the backlog regardless of whether it reaches
  the board, which is strictly better than today's silence.
