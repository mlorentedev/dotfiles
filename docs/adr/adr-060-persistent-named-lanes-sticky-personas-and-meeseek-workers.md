---
id: "ADR-060-persistent-named-lanes-sticky-personas-and-meeseek-workers"
type: adr
status: accepted
owner: manu
date: "2026-11-07"
issue: "mlorentedev/dotfiles#3255"
extends: [adr-050-one-lane-contract-for-every-repo-agent-and-forge, adr-057-project-workspace-boundary-and-worktree-roots]
tags: [architecture, decision, worktrees, lanes, personas, meeseek, harness, cross-platform]
created: "2026-11-07"
---

# ADR-060: Persistent named lanes with in-place park/recycle, sticky persona gate, and `meeseek-<role>-<fruit>` workers

## Status

Accepted 2026-11-07.

**Amended 2026-11-12:** what each lane is for, how many there are, and how a session reaches one are now decided. See *Amendment 2026-11-12 (lanes as the default, #3432)*.

## Context

Operating across Herdr, Orca ADE, GitHub Copilot CLI, `tmux`, and bare terminals (`claude`, `agy`, `pi`, `opencode`) across Windows (`EGW-LEN029`), macOS (`macmini`), and Linux (`ace2`) exposed four structural failure modes:

1. **Unbounded branch-per-directory worktree sprawl & Windows directory locks:** Creating a new directory for every branch accumulated 35 worktrees on disk, exceeded Windows 260-char path budgets (`ADR-057`), and caused `git worktree remove` to fail with `Permission denied` whenever a shell or background hook (`board-pickup.sh`) held the directory open.
2. **Main-thread `no-role` gap (95.1%) & mid-session persona thrashing:** `dotf harness gate` only resolved personas when `--role` or `agent_type` was supplied, leaving 95.1% of main-thread calls unscoped (`cli/internal/cmd/agent_auto.go:18`). Naive path-based persona switching on every tool call thrashes the active persona when a `builder` session edits `docs/adr/` or `specs/` to satisfy the `## Knowledge` Definition of Done.
3. **Opaque ephemeral subagents & `pi` pool fallback halt:** Subagent dispatches lacked deterministic, human-memorable identifiers across logs and claims, and `piBackend` in `cli/internal/agent/backends.go:23` omitted `--provider nan`, causing `classifyProcess` to mark missing pool credentials as `StatusTaskFailed` instead of `StatusPoolUnavailable`.
4. **Cross-machine PR triage blindness vs `no-attribution`:** Triaging a PR from a second machine/OS lacked environment/lane context on the PR while AI/persona attribution remains strictly forbidden in Git/GitHub/Gitea artifacts.

## Decision

1. **Layer 1 — 3 Persistent Named Lanes per Repo (`$WORKTREE_ROOT/<repo>/<lane>`):**
   - Provision `feat-lane-primary`, `feat-lane-secondary`, and `fast-track-lane` (plus `main` as read-only coordinator).
   - **In-Place Park/Recycle (`dotf lane park`, `dotf worktree done`, `dotf worktree sweep`):** Never delete a named lane directory (`reap_ok: false`). After verifying a clean working tree and `branchLanding` tip SHA containment in `origin/main` or `LandedHeads`, detach `HEAD` at `origin/main`, delete the merged local branch (logging its 40-char SHA), remove `.dotf-lane-prompt.md`, release `refs/dotf/claims/<issue>`, set `.dotf-worktree.json` to `state: "PARKED"` (`issue: 0`, `persona: ""`), and archive `memory/threads/<branch>.md` to `memory/archive/` while pruning `MEMORY.md` under `LockHandoffMemory`.
2. **Layer 2 — Sticky Primary Persona Gate (`$STATE_DIR/gate/<scope>.persona.json`):**
   - Exempt read-only discovery tools (`Read`, `Grep`, `Glob`, `LS`, `read_file`, `grep_search`, `find_by_name`, `list_dir`, `view`) from blocking.
   - Persist a sticky primary persona per `(scope, branch)` using atomic temp-file replacement serialized by `atomicWriteMu` and bounded Windows rename retry (15 attempts). Once bound for a branch, editing `docs/adr/` or `specs/` in the same session does not flip-flop the persona.
3. **Layer 3 — Ephemeral Fruit Workers (`meeseek-<role>-<fruit>`):**
   - Allocate worker IDs sequentially without replacement from a session-seeded shuffled permutation of a 64-fruit dictionary (`meeseek-reviewer-avocado`, `meeseek-builder-mango`, etc.), exported via `DOTF_WORKER_ID` and resolved to `<role>` in `loadGatePersona()`.
   - Pass `--provider nan` to `pi` in `backends.go` and classify `No matching provider is authenticated` as `StatusPoolUnavailable`.
4. **Layer 4 — Cross-OS Forge Provenance:**
   - Emit `Environment: <machine-id> (<goos>/<goarch>) · <lane>` in PR bodies and triage comments (`no-attribution` compliant), storing full agent/persona/worker provenance only in `refs/dotf/claims/<issue>` and `memory/threads/<branch>.md`.

## Consequences

- Worktree count per repository is bounded to at most 4 directories (`main` + 3 named lanes), eliminating `MAX_PATH` overflow and Windows directory-deletion locks.
- Any session on any OS can immediately identify which machine, OS, and lane produced a PR and resume full thread state via `dotf mem resume`.

## Amendment 2026-11-12 (lanes as the default, #3432)

Five days after acceptance, no repository carried `.dotf/lanes.json` and no machine had a lane directory. Every session still worked in a branch-per-directory worktree, about 40 of them for dotfiles alone. The CLI shipped, but nothing ever routed a session into a lane. This amendment settles the three questions that left the decision unexecuted.

1. **Three typed working lanes, and three is the cap on purpose.**

   | Directory | Holds |
   |---|---|
   | `main` | The read-only coordinator: triage, review, launching sessions |
   | `feat-lane-primary` | One large change (a spec) |
   | `feat-lane-secondary` | A second large change in parallel |
   | `fast-track-lane` | Quick fixes, hotfixes, and answering other sessions |

   `RouteBranchToLane` already routes by the branch's type: `feat`, `refactor`, `perf` and `build` go to the two feature lanes, and any other type goes to `fast-track-lane`, falling back to `feat-lane-secondary`. When every lane a branch may take is ACTIVE, it answers "WIP cap reached". That answer is the policy, not a defect to route around: it is ADR-043's finish-before-starting rule enforced at the directory level. The count stays the code's `CanonicalLanes`. A marker-driven count was considered and rejected: it buys per-repository flexibility nobody asked for, and it turns the cap into a setting each repository can raise.

2. **`dotf worktree add` coexists with lanes.** In a repository carrying `.dotf/lanes.json` it still creates the worktree and prints which lane to use instead. Refusing would break the detached worktrees `dotf spec review` creates, and the worktrees that exist until the sweep. Detached and review worktrees stay outside the lanes.

3. **A session reaches a lane through `dotf`, never by where it was opened.** A session's working directory is fixed when it starts, so no hook, skill or rule can move it afterwards. The two entrances are `dotf lane next` (queue-driven) and `dotf lane switch <branch>` (branch-driven), and both must be agent-, OS- and machine-agnostic:
   - Every harness in the agnostic set gets a launcher once a real launch is observed (the CLI-119 rule).
   - `lane next` gets a backend that needs no herdr.
   - Every harness's session start reports the lane it is in, or that the repository runs lanes. It informs and never relocates.
   - An autopilot run takes a free lane. When none is free, the run ends rather than opening a directory.

4. **Activation is committed, then converged.** A repository opts in with a PR that adds `.dotf/lanes.json`. Each machine then creates its lanes and arms the guard in a converge step that reads the committed marker and reaches `changed=0` on re-run. A run or a person running `dotf lane init` by hand is not the activation path, because it arms the guard for every worktree of the repository on that machine without a reviewed change.

The word "lane" belongs to this ADR. AGENTS.md also uses it for implementation tiers (Answer, Inline, Fast-Track, Full SDD), and that use is renamed. The implementation is tracked in #3432.

## References

- [ADR-050](adr-050-one-lane-contract-for-every-repo-agent-and-forge.md) — the lane contract every repo, agent and forge shares; owns the worktree reconciler (`dotf worktree sweep`).
- [ADR-057](adr-057-project-workspace-boundary-and-worktree-roots.md) — project workspace boundary and the roots a worktree may legally occupy.
- [ADR-032](adr-032-cross-harness-agent-orchestration.md) — cross-harness agent orchestration; the multi-agent topology this ADR's lanes run inside.
- [ADR-048](adr-048-one-permission-baseline-for-every-agent.md) — one permission baseline for every agent; the persona gate is a special case of it.

An earlier revision of this record declared `extends` on three ids that were never written:
`adr-032-worktree-first-isolation`, `adr-049-runtime-skill-enforcement-gate` and
`adr-052-deterministic-worktree-sweep`. Worktree isolation is ADR-057, the deterministic
sweep is ADR-050, and no ADR owns a "runtime skill enforcement gate" — that concept lives
in specs (`SDD-008-skill-pipeline`, `HARNESS-106-skill-capability`). The phantom ids were
removed rather than repointed, because citing the nearest real ADR would claim an authority
it does not hold.
