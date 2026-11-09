---
id: "HARNESS-186-named-lanes-sticky-personas"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-07"
issue: "mlorentedev/dotfiles#3255"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, lanes, worktrees, personas, meeseek]
template_version: "1.0"
owner: manu
---

# HARNESS-186: Named persistent lanes, sticky persona gate, and meeseek fruit workers

> **Naming**: file lives at `<repo>/specs/HARNESS-186-named-lanes-sticky-personas/proposal.md`. `HARNESS-186-named-lanes-sticky-personas` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3255: HARNESS-186: Named persistent lanes, sticky persona gate, and meeseek fruit workers -->

Working across multiple agent harnesses (Herdr, Orca ADE, GitHub Copilot CLI, `tmux`, and bare terminals running `claude`, `agy`, `pi`, or `opencode`) and multiple operating systems (`EGW-LEN029` Windows, `macmini` macOS, `ace2` Linux) currently causes unbounded branch-per-directory worktree sprawl (35 worktrees on disk), Windows directory-deletion locks (`Permission denied` when a shell holds CWD), and a 95.1% `no-role` rate on main-thread sessions (`cli/internal/cmd/agent_auto.go:18`). Without a fixed-slot persistent lane topology (`feat-lane-primary`, `feat-lane-secondary`, `fast-track-lane`) with in-place park/recycle housekeeping, a sticky persona gate that does not thrash when updating `docs/adr/` or `specs/`, and deterministic `meeseek-<role>-<fruit>` worker IDs, parallel multi-agent execution drifts across machines and tools.

## What

1. **Layer 1 — 3 Persistent Named Lanes & In-Place Park/Recycle (`cli/internal/lane/`, `cli/internal/worktree/`):**
   - `dotf lane init` idempotently provisions `feat-lane-primary`, `feat-lane-secondary`, and `fast-track-lane` under `$WORKTREE_ROOT/<repo>/<lane>` detached at `origin/main` (`state: "PARKED"`, `reap_ok: false`).
   - `dotf lane switch <branch>` (and `handoff.go`) routes `feat|refactor|perf|build` branches to `feat-lane-primary` / `feat-lane-secondary` and `fix|hotfix|chore|docs|ci|test|revert` branches to `fast-track-lane` (with spillover to `secondary` if `fast-track` is `ACTIVE` and `secondary` is `PARKED`), checking out the branch in-place under `filelock.Lock(DefaultLockPath())`.
   - `dotf lane park` (and `dotf worktree done` / `dotf worktree sweep` when `meta.Lane != ""`) never deletes the lane directory. After verifying a clean tree and `branchLanding` (confirming `HEAD` SHA is merged in `origin/main` or `LandedHeads`), it detaches `HEAD` at `origin/main`, deletes the merged local branch (logging its 40-char SHA), removes `.dotf-lane-prompt.md`, releases `refs/dotf/claims/<issue>`, sets `.dotf-worktree.json` to `state: "PARKED"` (`issue: 0`, `persona: ""`), and moves `memory/threads/<branch>.md` to `memory/archive/` while pruning its active line from `MEMORY.md` under `LockHandoffMemory`.
2. **Layer 2 — Sticky Persona Gate & Windows Atomic State (`cli/internal/cmd/harness_gate.go`, `cli/internal/harness/`):**
   - Exempts read-only discovery tools (`Read`, `Grep`, `Glob`, `LS`, `read_file`, `grep_search`, `find_by_name`, `list_dir`, `view`) from `enforce: block`.
   - Binds and persists a sticky primary persona (`architect`, `planner`, `builder`, `reviewer`, `shipper`, `curator`) per `(session_id, branch)` in `$STATE_DIR/gate/<scope>.persona.json` so updating `docs/adr/`, `docs/lessons/`, or `specs/` to satisfy `## Knowledge` in the same session does not thrash the active persona.
   - Uses an in-process mutex (`atomicWriteMu`) and bounded retry on `os.Rename` (15 attempts, 5–75ms linear backoff) to eliminate Windows `PreToolUse` sharing violations.
3. **Layer 3 — Ephemeral Fruit Workers (`meeseek-<role>-<fruit>`) & `pi` Pool Fallback (`cli/internal/agent/`):**
   - Allocates deterministic worker IDs sequentially without replacement from a session-seeded shuffled permutation of the 64-fruit dictionary (`meeseek-reviewer-avocado`, `meeseek-builder-mango`, etc.), exported via `DOTF_WORKER_ID` and resolved to `<role>` in `effectiveRole()` and `loadGatePersona()`.
   - Updates `piBackend.BuildCommand` (`cli/internal/agent/backends.go`) to pass `--provider <pool>` (`--model <pool>:<model>`) and classifies missing pool authentication as `StatusPoolUnavailable` so `dotf agent auto` cascades cleanly to the next fallback candidate.
4. **Layer 4 — Cross-OS Forge Provenance & Scoped `pre-commit` Guard (`cli/internal/prland/`, `cli/internal/forgewrite/`, `git-hooks/`):**
   - Adds `Environment: <machine-id> (<goos>/<goarch>) · <lane>` to PR bodies and triage comments (`no-attribution` compliant; zero AI/agent/persona names in Git/GitHub/Gitea artifacts), while storing full agent/persona/worker provenance in `refs/dotf/claims/<issue>` and `memory/threads/<branch>.md`.
   - Enforces `lane-guard.sh` in `pre-commit` only for repositories containing `.dotf/lane.json` (exempting `.obsidian` vault and active `git rebase` / `cherry-pick` operations), and changes CWD to `$HOME` in backgrounded `board-pickup.sh` before network calls.

## Out of scope

- Deleting or force-cleaning the 5 existing `DIRTY` worktrees that currently have uncommitted changes or active sessions in other terminals.
- Modifying external third-party binaries (`herdr`, `orca`, `gh`, `claude`, `pi`).
- Changing the `no-attribution` policy (AI, persona, and `meeseek` identifiers remain strictly forbidden in git commits, PR bodies, and issue comments).

## Risks / open questions

- **Resolved (B0):** `pi --print` fails when `--provider <pool>` is omitted in `backends.go:23` and halts `agent auto` fallback; fixed by passing `--provider` and classifying missing provider auth as `StatusPoolUnavailable`.
- **Resolved (B1 & B5):** `Herdr.Find` and `dotf worktree done` must operate in-place on persistent named lanes (`meta.Lane != ""`) rather than creating a 4th worktree or running `git worktree remove`.
- **Resolved (B2 & B4):** Sticky persona per `(session_id, branch)` + read-only tool exemption + bounded Windows `os.Rename` retry prevents mid-session persona thrashing and file locking.
- **Resolved (B3 & B6):** `SessionEnd` is a no-op when `state == "PARKED"`; `pre-commit` lane guard only applies when `.dotf/lane.json` is present and `rebase-merge`/`rebase-apply`/`CHERRY_PICK_HEAD` are absent.

## Acceptance criteria

- [ ] **AC1 (Idempotent 3-Lane Provisioning & Routing):** `dotf lane init` creates `feat-lane-primary`, `feat-lane-secondary`, and `fast-track-lane` detached at `origin/main` (`state: "PARKED"`, `reap_ok: false`) with `changed=0` on re-run, and `dotf lane switch` / `handoff` routes all valid branch prefixes (`feat|refactor|perf|build` vs `fix|hotfix|chore|docs|ci|test|revert`) in-place while enforcing the 2-feature WIP cap under `filelock`.
- [ ] **AC2 (Fail-Closed In-Place Park, Recycle & Memory Archival):** `dotf lane park` (and `dotf worktree done` / `sweep` on worktrees with `meta.Lane != ""`) verifies a clean working tree and `branchLanding` tip SHA containment before detaching at `origin/main`, deleting the merged local branch (logging its 40-char SHA), removing `.dotf-lane-prompt.md`, releasing `refs/dotf/claims/<issue>`, archiving `memory/threads/<branch>.md` to `memory/archive/` while pruning `MEMORY.md` under `LockHandoffMemory`, and leaving the lane directory intact (`SessionEnd` is a no-op on `PARKED` lanes).
- [ ] **AC3 (Sticky Persona Gate & Windows Atomic State):** `dotf harness gate` exempts read-only tools (`Read`, `Grep`, `Glob`, `LS`), binds and persists a sticky primary persona per `(session_id, branch)` in `$STATE_DIR/gate/<scope>.persona.json` without thrashing when `docs/adr/` or `specs/` are edited in the same session, and uses atomic replace with bounded Windows retry.
- [ ] **AC4 (Ephemeral `meeseek-<role>-<fruit>` Workers & `pi` Fallback):** `dotf agent auto/run` allocates `meeseek-<role>-<fruit>` sequentially without replacement from a 64-fruit dictionary (0% collision across 64 dispatches), exports `DOTF_WORKER_ID`, resolves `<role>` in `effectiveRole()` and `loadGatePersona()`, passes `--provider <pool>` in `piBackend`, and treats unauthenticated pool errors as `StatusPoolUnavailable`.
- [ ] **AC5 (Scoped `pre-commit` Lane Guard & Background Hook CWD Safety):** `git-hooks/lib/lane-guard.sh` is POSIX `sh` (shellcheck-clean with `-s sh`, no bashisms) and activates **only** in repos that carry the marker `dotf lane init` writes (`.dotf/lanes.json`, `version`/`canonical_lanes`/`third_party_roots`); the unrelated issue-queue file `.dotf/lane.json` from #2532 must **not** activate it. When active it blocks commits on `main`, in `PARKED` lanes, and inside every root declared in `third_party_roots` — with no host name, agent name or absolute path hardcoded in the script — while allowing `.obsidian` vault commits and active `git rebase` / `cherry-pick` operations, and `board-pickup.sh` changes directory to `$HOME` before network calls.
- [ ] **AC6 (Cross-OS Forge Provenance & Zero AI Attribution):** PR bodies and review triage comments include `Environment: <machine-id> (<goos>/<goarch>) · <lane>` passing all `no-attribution` guards, while full agent/persona/meeseek metadata is persisted in `refs/dotf/claims/<issue>` and `memory/threads/<branch>.md`.

## References

- Bitácora board: `mlorentedev/dotfiles#3255`
- Vault Prestudy: `10_projects/dotfiles/prestudy/2026-11-07-named-lanes-sticky-personas-meeseek-spec.md`
- Related ADRs: `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md`, `docs/adr/adr-057-project-workspace-boundary-and-worktree-roots.md`, `docs/adr/adr-032-cross-harness-agent-orchestration.md`, `docs/adr/adr-048-one-permission-baseline-for-every-agent.md`
- Governing ADR for this spec: `docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md`
- Predecessor lane subsystem (issue-queue `lane.json`): `mlorentedev/dotfiles#2532`
