---
id: "ADR-060-persistent-named-lanes-sticky-personas-and-meeseek-workers"
type: adr
status: accepted
owner: manu
date: "2026-11-07"
issue: "mlorentedev/dotfiles#3255"
extends: [adr-032-worktree-first-isolation, adr-049-runtime-skill-enforcement-gate, adr-050-one-lane-contract-for-every-repo-agent-and-forge, adr-052-deterministic-worktree-sweep, adr-057-project-workspace-boundary-and-worktree-roots]
tags: [architecture, decision, worktrees, lanes, personas, meeseek, harness, cross-platform]
created: "2026-11-07"
---

# ADR-060: Persistent named lanes with in-place park/recycle, sticky persona gate, and `meeseek-<role>-<fruit>` workers

## Status

Accepted 2026-11-07.

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
