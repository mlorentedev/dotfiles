---
id: "CLI-075-dotf-worktree-lifecycle"
type: spec
status: draft
created: "2026-09-04"
issue: "mlorentedev/dotfiles#1500"
tags: [spec, proposal, worktree, gc, lifecycle]
template_version: "1.0"
---

# CLI-075-dotf-worktree-lifecycle

> **Naming**: file lives at `<repo>/specs/CLI-075-dotf-worktree-lifecycle/proposal.md`.

## Why

Parallel AI coding agent sessions (Claude Code, Antigravity, OpenCode, Pi) frequently create Git worktrees to isolate tasks. Over time, dozens of worktrees accumulate across `~/Projects/` because operators fear data loss (uncommitted WIP, untracked scratchpad notes, active agent processes) and Git provides no native lifecycle or garbage collection layer. A naive auto-cleanup introduces catastrophic failure modes: false-positive deletion of freshly created branches, destroying workspaces of containerized agents, and leaking secrets to disk. A fail-closed, lease-based worktree management subsystem (`dotf worktree`) restores workstation hygiene with zero risk of data loss.

## What

Introduces the `dotf worktree` command suite in Go under `cli/internal/worktree/`:

1. **Reconciliation-First Discovery (`dotf worktree list`):**
   - Discovers linked worktrees deterministically via `git worktree list --porcelain`.
   - Rejects submodules by verifying the worktree `gitdir` does not point into a `modules/` directory.
   - Reads per-worktree metadata from `.dotf-worktree.json` located inside the worktree root.
   - Inspects git status (clean vs dirty), remote tracking (`synced`, `ahead`, `behind`, `gone`), and PR status via `gh pr view` / cached metadata.
   - Classifies each worktree into an actionable state: `ACTIVE`, `REAPABLE`, `DIRTY`, `UNMERGED`, or `ORPHAN` (where `ORPHAN` indicates upstream branch is gone and PR is not merged).
   - Supports `--json` for machine consumption and `--all` to scan all sibling repositories in `~/Projects/`.

2. **Standardized Creation (`dotf worktree add <slug> [--issue <N>] [--ttl <duration>]`):**
   - Automatically computes and validates an external sibling directory (`<repo>-wt-<slug>`).
   - Enforces physical isolation: validates that the resolved path is strictly outside the parent repository's working tree.
   - Evaluates Gate A (warns if parent repository has active auto-commit hooks/plugins).
   - Initializes the new branch and links the worktree.
   - Writes `.dotf-worktree.json` containing creator (`$AGENT_NAME` / `$USER`), issue number, creation timestamp, and initial lease expiration. Adds `.dotf-worktree.json` to `.git/info/exclude`.
   - Detects and runs project dependency bootstrap (`go mod download`, `npm install`, etc.).

3. **Fail-Closed Reaper (`dotf worktree sweep [--dry-run]`):**
   - Acquires an exclusive cross-platform file lock on `<temp-dir>/dotf-worktree.lock` (`syscall.Flock` on Unix, exclusive `CreateFile` with `dwShareMode=0` on Windows, matching `cli/internal/agent/lock_*.go`) to prevent concurrent sweep races.
   - Reaps **ONLY** worktrees that satisfy **ALL** of the following positive fail-closed gates:
     a. Explicit metadata exists in `.dotf-worktree.json` with `reap_ok: true` (defaults to true on creation; operator/agent can set to false as a hold).
     b. Authoritative lease expired: current time > `lease_expires_at`. This is the primary liveness signal, protecting containerized and background agents.
     c. Working tree is 100% clean (`git status --porcelain` is empty AND no non-disposable gitignored local content exists, e.g. `.env` or scratchpad notes; verified initially and re-checked under lock immediately prior to deletion; standard disposable build caches like `node_modules/`, `target/`, `.venv/` and `.dotf-worktree.json` are permitted).
     d. Positive merge confirmation: branch is associated with a GitHub PR whose state is explicitly `MERGED`, OR its commit tip is an ancestor of the repository's default branch (`git merge-base --is-ancestor <branch> <base>`). PR queries are cached within runner invocations. If upstream is `gone` but no merged PR exists and it is not an ancestor, it is classified as `ORPHAN` and refused.
     e. Minimum age guard: creation time is > 15 minutes ago (eliminates the zero-diff fresh branch false positive).
     f. Host-side defensive check: neither the sweep command's own cwd nor any host terminal process has cwd inside the target path (host process scanning supported on Linux `/proc`; sweep command cwd check enforced cross-platform; evaluated initially and re-verified under lock).
   - Reaping action: executes `git worktree remove <path>`, logs the previous commit SHA to stderr before `git branch -D <branch>` (ensuring instant recovery of committed state via `git branch <name> <sha>`), and executes `git worktree prune`.

4. **Self-Service Teardown (`dotf worktree done`):**
   - Single-command clean exit for human operators or agents finishing a task.
   - Verifies changes are committed and pushed; removes worktree and prunes metadata cleanly.

## Out of scope

- Destructive purging of dirty uncommitted worktrees without explicit operator confirmation (`--force`).
- Full-directory tarball archives that duplicate `node_modules/`, `target/`, `.venv/` or risk leaking unencrypted `.env` secrets into trash directories (ADR-028 compliance).
- Unattended background daemon/cron polling (sweeping is event-driven: triggered on-demand, in `/handoff`, and as pre-flight in `dotf agent run`).
- Remote branch deletion on GitHub (governed separately by repo policy `delete_branch_on_merge`).

## Risks / open questions

- **GitHub API Rate Limits**: Probing GitHub for dozens of worktrees could exhaust API quotas.
  - *Mitigation*: Evaluate local git state first (`git branch -vv` for `gone`). Only query GitHub API for branches marked `gone` or candidate for reaping, and cache PR query results.
- **Container / Sandbox Isolation**: Processes in Docker or PID namespaces are invisible to host `/proc`.
  - *Mitigation*: The lease file (`.dotf-worktree.json`) lives inside the worktree filesystem and travels with the mounted volume. The lease is the primary authoritative liveness signal; host `/proc` is a secondary floor for host shells.
- **Concurrent Sweeps**: Multiple agents running `/handoff` or `sweep` simultaneously could race on Git metadata.
  - *Mitigation*: Cross-process single-writer lock via cross-platform OS locking (`cli/internal/agent/lock_*.go`).

## Acceptance criteria

- [ ] **[AC1]** `dotf worktree list` prints a table or JSON object reporting path, branch, git status (clean/dirty), lease status, PR status, and lifecycle classification across all linked worktrees.
- [ ] **[AC2]** `dotf worktree list` ignores submodule checkouts whose `gitdir` points into a `modules/` directory.
- [ ] **[AC3]** `dotf worktree add <slug>` creates an external sibling worktree, writes `.dotf-worktree.json` with creator/issue/lease, and excludes `.dotf-worktree.json` via `.git/info/exclude`.
- [ ] **[AC4]** `dotf worktree sweep` reaps ONLY worktrees satisfying all 6 fail-closed conditions, refusing to touch dirty, unmerged, unexpired, or newly-created branches.
- [ ] **[AC5]** `dotf worktree sweep` logs the exact commit SHA before deleting any branch, guaranteeing instant recovery of committed state.
- [ ] **[AC6]** `dotf worktree sweep` acquires an exclusive cross-platform file lock, preventing concurrent races between parallel sessions.
- [ ] **[AC7]** `dotf worktree done` validates clean push status and tears down the current worktree safely.

## References

- Bitácora board: [mlorentedev/dotfiles#1500](https://github.com/mlorentedev/dotfiles/issues/1500)
- Related patterns: `00_meta/patterns/pattern-github-branch-hygiene.md`, `00_meta/patterns/pattern-git-workflow.md`
- Related runbooks: `00_meta/runbooks/runbook-worktree-safety.md`
- Skill: `harness/skills/using-git-worktrees/SKILL.md`
