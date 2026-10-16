---
id: "CLI-127-lane-reconcile"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-15"
issue: "mlorentedev/dotfiles#2550"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "Owner chose option B on 2026-10-16 (#2550 park, epic #2546): CLI-127 is the next item of the ADR-050 lane contract and starts over the ADR-043 limit until an active spec archives. (12 active, limit 10, 2026-10-15)"
---

# CLI-127-lane-reconcile

> **Naming**: file lives at `<repo>/specs/CLI-127-lane-reconcile/proposal.md`. `CLI-127-lane-reconcile` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #2550: CLI-127: dotf lane status, plan and apply -->

Parallel lanes are coordinated by prose and memory, and the thread gets lost: on 2026-10-15, 16 dotfiles worktrees existed, herdr showed 4, 10 had no lease, and 5 finished ones were still on disk (#2546). After this run's cleanup, 2026-10-16, `dotf worktree list` still shows 4 UNADOPTED, 1 DIRTY and 1 merged worktree held by a live session. Every one of those was found by hand. ADR-050 decision 4 makes the controller code: a level-triggered `status | plan | apply`, so the agent in main judges only what the plan marks ambiguous.

## What

Three verbs, one per PR:

1. `dotf lane status [--json]`. A read-only join of `git worktree list`, `.dotf-worktree.json`, the forge (REST pulls, never the GraphQL quota), a process scan for liveness and the herdr snapshot when present. It prints one row per worktree: ticket, branch, PR and its state, agent, lease and a **derived** phase (`Pending`, `Running`, `Review`, `Landing`, `Landed`, `Reaped`, `Stalled`, `Orphan`, `Unadopted`). Phases are never stored. herdr absent is a supported state, not an error.
2. `dotf lane plan`. From `status` and `.dotf/lane.json`, it prints the actions a reconcile would take, each in one of three classes (decision D1): `auto`, `pending` (with the time it becomes due) or `ambiguous` (with the exact command that would resolve it). It also prints the landing order and the head PR's four-condition check. It mutates nothing.
3. `dotf lane apply`. It runs only the `auto` actions, through the existing verbs (`worktree sweep` for reaps, `lane next` for the next queued item), and prints `changed=N`. A second run on the same state prints `changed=0`. It never adopts (D2) and never lands (D3).

## Out of scope

- The atomic claim (`refs/dotf/claims/*` or `claim/*`): that is #2551. `status` lists claims only once #2551 defines them.
- A daemon, cron, bus or state store: ADR-050 rejects each one. `apply` runs on demand only.
- The headless tier (#2553), `initrepo` scaffolding (#2552), and anything in `cli/internal/worktree` beyond its exported API (CLI-118 owns it).

## Risks / open questions

- Liveness is a process scan on cwd (as `worktree done` does) plus transcript age through `worktree.RecentSession` (#2556), which fails closed on an unreadable store. CLI-118 is extending it to the non-Claude stores, so do not reimplement it (#2550 comment 6205806273). Both signals are heuristics, and a false "dead" reaps a live session. Default to "alive" when unsure.
- herdr 0.9.3 cannot isolate a second server: `HERDR_SOCKET_PATH`, `HERDR_HOME` and `HERDR_CONFIG_PATH` move only the sockets. A second server restores the owner's `session.json` and resumes every agent, measured by CLI-118 on 2026-10-16. `status` only reads the snapshot of the running server, and no test or live run starts a herdr server. Only a different `HOME` isolates.
- Paths held by CLI-118 (`cli/internal/worktree/session*.go`, `cli/internal/herdr/**`, `cli/internal/deploy/**`) are consumed through their exported API only.
- Forge rate limits with N worktrees: use one `pulls?state=all&head=` request per branch, or one list request, and cache it for the run.

## Decisions

Decided with the owner on 2026-10-16, on the advisor's recommendation.

- **D1, the three classes.** An action is `auto` when an existing verb accepts it without `--force`. That means a reap whose `sweep` gates all hold (merged PR, clean tree, `reap_ok` with an expired lease, no process inside the worktree, no recent session), or starting the next open queued item with `lane next`. It is `pending` when the only gate missing is time: the lease has not expired yet, or the minimum age has not been reached. `plan` prints when it becomes due and asks nothing. Everything else that `status` flags is `ambiguous`: DIRTY, merged with a live session, Stalled, Orphan, UNADOPTED. Each `ambiguous` row carries the one command that resolves it, such as `dotf worktree done --force <path>` or `dotf worktree adopt <path>`, so the human's judgement is a yes or no, not design work.
- **D2, adoption is never `auto`.** `worktree.Adopt` writes `ReapOK: true`, as `add` and `lane next`'s lease do, so adopting starts the clock towards a reap; it is not neutral. `plan` proposes `dotf worktree adopt <path>` as `ambiguous` when the branch (`<type>/<issue>-<slug>`) or a PR makes the intent derivable. Otherwise it reports the worktree with no proposal: the controller records what git says and never invents intent (ADR-050 decision 1).
- **D3, landing is never `auto`.** The standing merge grant covers a PR the agent opened. `apply` run from main and landing another lane's PR would widen that, and a widening of the agent's authority is prepared, never landed, by the agent (#2178). `plan` prints the landing order and the head PR's four-condition check. The lane that opened the PR, or the owner, runs `dotf pr triage-queue --pr N && dotf pr land N`.

## Acceptance criteria

- [ ] AC1: `dotf lane status --json` on a fixture of git, lease, forge and process state returns the expected phase for each of the 9 phases. One table test per phase, with herdr absent in at least one row.
- [ ] AC2: `dotf lane plan` on the same fixture prints the expected actions, each classed `auto`, `pending` (with its due time) or `ambiguous` (with its resolving command). The fixture's files and refs are byte-identical afterwards.
- [ ] AC3: `dotf lane apply` executes only `auto` actions through `worktree sweep` and `lane next` (the test asserts the verb calls), never `worktree adopt` or `pr land`, and reports `changed=N`. A second run on the resulting state reports `changed=0`.
- [ ] AC4: A worktree whose cwd holds a live process, or whose transcript `RecentSession` reports as recent, is never reaped by `apply`, whatever its PR state. Covered by a test and by one live run of the read-only verbs (`status`, `plan`) on this machine, against the already-running herdr: every worktree with a live session is classed `ambiguous`. The live run never runs `apply` and never starts a herdr server.

## References

- Epic: #2546 (CLI-123). Gate: #2550.
- ADR: `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md` (decisions 1, 3 and 4). Also ADR-032 (herdr is the cockpit, not an executor) and ADR-049 (the merge queue).
- Prior lanes: #2555 (lease on `lane next`), #2558 (`triage-queue --pr`), #2567 (prompts land on the scoped gate).
