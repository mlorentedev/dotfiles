---
id: "ADR-050-one-lane-contract-for-every-repo-agent-and-forge"
type: adr
status: proposed
owner: manu
date: "2026-10-15"
supersedes: []
extends: [adr-032-cross-harness-agent-orchestration, adr-043-a-wip-limit-on-active-specs, adr-047-standing-merge-grant-opt-in-per-repository, adr-049-a-merge-queue-on-every-forge-behind-dotf]
issue: mlorentedev/dotfiles#2546
tags: [architecture, decision, lane, worktree, orchestration, forge]
created: "2026-10-15"
---

# ADR-050: One lane contract for every repository, agent, forge and OS, reconciled from git and the forge

## Context

Parallel agent sessions (herdr panes, worktrees, pull requests) are coordinated by prose and memory, and the thread gets lost. Measured on 2026-10-15 (#2546):

- 16 dotfiles worktrees existed, and herdr's sidebar showed 4. 10 of the 16 had no dotf metadata, so `dotf worktree sweep` never reaps them, and 5 were finished but still on disk.
- `dotf lane next` created its worktree through `herdr worktree create` and wrote no lease, so every lane worktree was unadopted. Fixed in #2555 (#2547).
- The landing queue is a convention. The `pr land` lock serializes but is not FIFO: #2527 lost three CI rounds. `triage-queue` answered for the whole repo, so one untriaged PR gated every session: #2542 held up #2539, and on the run that wrote this ADR the repo-wide queue exited 1 on #2556 and then #2557 while the PR about to land was clean. Fixed in #2558 (#2549), which adds `--pr N`.
- Three queue declarations exist with no single source: `.dotf/lane.json`, the autopilot skill's board query, and `.agent/unattended.yml`.
- Only dotfiles has the machinery. kubelab's AGENTS.md has no `dotf` references, and no other repository has `.dotf/lane.json` or `.github/merge-grant.yml`.

The model below was agreed with the owner on 2026-10-15. It follows level-triggered reconciliation as Kubernetes does it, with no new infrastructure. The owner reviewed the prior art in that session: openai/symphony (SPEC §7–9) and sortie-ai/sortie (a reconcile tick), OrchestratorInc/agent-orchestrator (facts stored, status derived), max-sixty/worktrunk (integration checks). Two earlier vault studies point the same way: `10_projects/dotfiles/research/2026-08-20-agent-orchestration-evidence-report.md` (axis is the workflow phase, routing is a declared map, no message bus; read with its `-review` companion) and `10_projects/iris/research/iris-scheduler-research.md` §1.

## Decision

1. **Conventions.**
   - Branch `<type>/<issue>-<slug>`. The issue number joins ticket, branch, worktree, PR and claim without guessing.
   - Worktree `<WORKTREE_ROOT>/<repo>/<branch-slug>` (CLI-118, #2503).
   - herdr workspace label `<repo>#<issue> <title>`.
   - Intent lives in `.dotf-worktree.json`: ticket, agent, session, template. The field names are CLI-118 PR 4's. Until they land, `lane next` writes the agent in `creator` (#2555).
   - Phases `Pending`, `Running`, `Review`, `Landing`, `Landed`, `Reaped`, plus `Stalled`, `Orphan` and `Unadopted`. They are derived on read and never stored.
2. **One queue source.** Order lives in the committed `.dotf/lane.json`. The autopilot skill's board query and the `queue` block of `.agent/unattended.yml` retire into it. The envelope keeps its scope, budget and authority.
   - **Claim** is an atomic compare-and-swap on a deterministic ref: `git push --force-with-lease=<ref>: <sha>:<ref>`, whose empty expected value makes the server refuse a ref that already exists. The forge assignee is set after the claim wins, for people.
   - **Open: which ref.** (a) a custom namespace (`refs/dotf/claims/<issue>`): invisible in the forge UI, outside branch protection, triggers no workflow, but a forge may refuse a non-branch ref. (b) a `claim/<issue>` branch: accepted everywhere and visible, but it falls under branch-protection patterns, fires `push` workflows and crowds the branch list and herdr's view. It is decided by pushing each form with an empty expected value to a scratch repository on GitHub and on Gitea, then pushing it again to confirm the refusal. #2551 does that before it implements the claim.
3. **Done is judged from git and the forge:** the PR merged, or its head an ancestor of the base, squash-aware. A process scan and the transcript's age say whether a session is still live. A herdr pane decides nothing (ADR-032: herdr is the cockpit, not an executor).
4. **The controller is `dotf lane status | plan | apply`** (#2550). It is idempotent like `converge`: a second `apply` reports `changed=0`. It runs on demand, with no daemon, bus, scheduler or store of its own. `plan` prints actions and changes nothing. `apply` runs the existing verbs (`worktree adopt`, `worktree sweep`, `lane next`, `pr land`). **The operator is the agent in the main checkout**, of any harness. It runs the plan and decides only what the plan marks ambiguous.
5. **Two tiers, one contract.** Interactive lanes (herdr panes) and headless workers (`dotf agent run`, later iris as a backend, #2553) read the same queue, claim the same way and land through the same merge queue (ADR-049).
6. **Distribution.** `dotf initrepo` scaffolds `.dotf/lane.json`, the prompt template and a common AGENTS.md section in every repository (#2552). kubelab goes first. The same `dotf` runs on every OS, and per-repository facts live in the repository.

## Consequences

- **The queue and the claim live in each repository's git and forge.** A session that can push can coordinate; nothing else has to be running. A forge outage stops claims and landings, as it already stops CI.
- **Facts are stored and phases derived.** A wrong phase is fixed in the derivation once, not by rewriting stored state on every worktree.
- **The repeatable decisions move into code.** The agent in main runs `plan` and judges only what the plan marks ambiguous. That is the lesson of #1072, where a standing instruction was followed 1 time in 16.
- **The first two pieces have landed.** #2555 leases every worktree `lane next` starts, and #2558 scopes the triage gate to the PR being landed. The rest is #2550–#2553, in that order, and needs a spec slot under ADR-043.
- **Two follow-ups sit outside dotfiles' code.** The autopilot skill and `.dotf/lane-prompt.md` should call `dotf pr triage-queue --pr N` before `dotf pr land N`, once a release ships it (#2546).
- **The vault gets a read-only projection** of lane state, regenerable from git and the forge, never written as the record.

## Alternatives rejected

- **A message bus as the record.** It needs a service on every machine and a second truth to reconcile with git. The 2026-08-20 evidence report concluded that routing is a declared map and that no message bus is needed.
- **A director model, an LLM holding the queue and assigning work.** Instructions held by a model are followed unreliably: TOOL-022 (#1072) measured one compliant review in sixteen. The controller is code, and the model judges only what the plan marks ambiguous.
- **A daemon or cron reconciler.** CLI-075 left unattended background polling out of the worktree lifecycle, keeping sweep on-demand, at `/handoff` and as the preflight of `dotf agent run`. A reconciler run on demand inherits that, and a stale view costs one `status` call.
- **Concurrent landers.** Four `pr land --update-branch` processes on 2026-10-07 each updated every other PR after every merge (lesson 344). Landing stays serial, through ADR-049's queue.
- **Auto-merge.** ADR-047 and ADR-049 keep `allow_auto_merge=false`. A claim decides who works on an item, not whether the result merges.
- **The vault as the coordination store.** It is one contended branch that every session commits to, it is not reachable from CI or from another forge's runners, and it would put task state in the store (ADR-018). It gets the read-only projection instead.
