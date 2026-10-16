---
id: "CLI-127-lane-reconcile"
type: spec
status: draft # draft | implementing | verifying | archived
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

[AGENT-DRAFT — review before archive] Parallel lanes are coordinated by prose and memory, and the thread gets lost: on 2026-10-15, 16 dotfiles worktrees existed, herdr showed 4, 10 had no lease, and 5 finished ones were still on disk (#2546). After this run's cleanup, 2026-10-16, `dotf worktree list` still shows 4 UNADOPTED, 1 DIRTY and 1 merged worktree held by a live session. Every one of those was found by hand. ADR-050 decision 4 makes the controller code: a level-triggered `status | plan | apply`, so the agent in main judges only what the plan marks ambiguous.

## What

[AGENT-DRAFT — review before archive] Three verbs, one per PR:

1. `dotf lane status [--json]`. A read-only join of `git worktree list`, `.dotf-worktree.json`, the forge (REST pulls, never the GraphQL quota), a process scan for liveness and the herdr snapshot when present. It prints one row per worktree: ticket, branch, PR and its state, agent, lease and a **derived** phase (`Pending`, `Running`, `Review`, `Landing`, `Landed`, `Reaped`, `Stalled`, `Orphan`, `Unadopted`). Phases are never stored. herdr absent is a supported state, not an error.
2. `dotf lane plan`. From `status` and `.dotf/lane.json`, it prints the actions a reconcile would take: adopt, reap, flag stalled or orphan, start the next queued item, and the landing order. It mutates nothing and marks each action `auto` or `ambiguous`.
3. `dotf lane apply`. It runs only the `auto` actions, through the existing verbs (`worktree adopt`, `worktree sweep`/`done`, `lane next`, `pr land`), and prints `changed=N`. A second run on the same state prints `changed=0`.

## Out of scope

[AGENT-DRAFT — review before archive]
- The atomic claim (`refs/dotf/claims/*` or `claim/*`): that is #2551. `status` lists claims only once #2551 defines them.
- A daemon, cron, bus or state store: ADR-050 rejects each one. `apply` runs on demand only.
- The headless tier (#2553), `initrepo` scaffolding (#2552), and anything in `cli/internal/worktree` beyond its exported API (CLI-118 owns it).

## Risks / open questions

[AGENT-DRAFT — review before archive]
- **MUST resolve before tasks.md:** where `ambiguous` begins. Proposal: anything that removes work is `ambiguous` unless sweep's fail-closed gates (merged PR, clean tree, expired lease, no live process) all hold. That covers a DIRTY worktree, a merged worktree with a live session, and an UNADOPTED one with no PR.
- **MUST resolve:** whether `apply` may adopt an UNADOPTED worktree on its own. Adoption writes a lease that makes the worktree reapable later, so it is not neutral.
- Liveness is a process scan on cwd (as `worktree done` does) plus transcript age through `worktree.RecentSession` (#2556), which fails closed on an unreadable store. CLI-118 is extending it to the non-Claude stores, so do not reimplement it (#2550 comment 6205806273). Both signals are heuristics, and a false "dead" reaps a live session. Default to "alive" when unsure.
- herdr 0.9.3 cannot isolate a second server: `HERDR_SOCKET_PATH`, `HERDR_HOME` and `HERDR_CONFIG_PATH` move only the sockets. A second server restores the owner's `session.json` and resumes every agent, measured by CLI-118 on 2026-10-16. `status` only reads the snapshot of the running server, and no test or live run starts a herdr server. Only a different `HOME` isolates.
- Paths held by CLI-118 (`cli/internal/worktree/session*.go`, `cli/internal/herdr/**`, `cli/internal/deploy/**`) are consumed through their exported API only.
- Forge rate limits with N worktrees: use one `pulls?state=all&head=` request per branch, or one list request, and cache it for the run.

## Acceptance criteria

[AGENT-DRAFT — review before archive]
- [ ] AC1: `dotf lane status --json` on a fixture of git, lease, forge and process state returns the expected phase for each of the 9 phases. One table test per phase, with herdr absent in at least one row.
- [ ] AC2: `dotf lane plan` on the same fixture prints the expected actions, each marked `auto` or `ambiguous`, and the fixture's files and refs are byte-identical afterwards.
- [ ] AC3: `dotf lane apply` executes only `auto` actions through the existing verbs (the test asserts the verb calls), reports `changed=N`, and a second run on the resulting state reports `changed=0`.
- [ ] AC4: A worktree whose cwd holds a live process, or whose transcript `RecentSession` reports as recent, is never reaped by `apply`, whatever its PR state. Covered by a test and by one live run on this machine against the already-running herdr (read only). The live run never starts a second herdr server.

## References

- Epic: #2546 (CLI-123). Gate: #2550.
- ADR: `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md` (decisions 1, 3 and 4). Also ADR-032 (herdr is the cockpit, not an executor) and ADR-049 (the merge queue).
- Prior lanes: #2555 (lease on `lane next`), #2558 (`triage-queue --pr`), #2567 (prompts land on the scoped gate).
