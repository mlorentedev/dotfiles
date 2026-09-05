---
id: lesson-272
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, git, worktrees, lifecycle, garbage-collection, fail-closed, agents]
---

# Fail-closed worktree garbage collection requires in-tree leases and positive confirmation

## What happened

When designing automated garbage collection for parallel git worktrees across autonomous coding agents (Claude Code, Antigravity, OpenCode, Pi), the initial heuristic approach classified worktrees as reapable based solely on:
1. `git status --porcelain` being clean
2. The branch appearing merged or having an unmerged commit that was an ancestor of base

During the adversarial review on spec `CLI-075-dotf-worktree-lifecycle` (conducted by `nan/deepseek-v4-flash`), the reviewer demonstrated critical failure modes:
1. **The 0-diff fresh branch trap**: A newly created worktree before an agent writes its first file is 100% clean and has 0 diff against base. A naive reaper immediately deletes the agent's worktree under its feet.
2. **Container PID blindspot**: Containerized or PID-namespaced agents (Docker, dev-containers) are invisible to host-level `/proc/*/cwd` inspections. Inferring agent liveness from host process tables is a blind spot.
3. **Metadata-less fail-open**: If classification only checks `info.Metadata != nil` for holds, any worktree created outside `dotf worktree add` (manual human worktrees, submodule clones) lacks metadata, bypasses the lease and age guard, and gets reaped if clean.
4. **Git status error swallowing**: If a `git status` check fails due to transient I/O or permissions, defaulting to `dirty = false` treats the directory as clean and deletes it.

## The doctrine: Fail-Closed Garbage Collection

An automated reaper must **never** delete based on absence of dirty state alone. It must require explicit, conjunctive positive confirmation across all dimensions:

1. **In-tree Lease as Primary Liveness**: The liveness signal (`.dotf-worktree.json`) must live inside the worktree filesystem itself. Because it is mounted into the container/sandbox, it travels with the execution context regardless of PID namespaces. A worktree is only reapable if `now > lease_expires_at`.
2. **Explicit Opt-in (`reap_ok: true`)**: A worktree without metadata is unknown and must be treated as `StateActive` ("no dotf metadata — reap refused"). Reaping requires positive opt-in.
3. **Minimum Age Guard (> 15 minutes)**: Eliminates the fresh-branch race condition.
4. **Authoritative Merge Confirmation**: Merge status must query the forge (`gh pr view --repo owner/repo --json state`) or default ancestor, never local heuristics alone.
5. **Errors Fail Closed**: Any failure to inspect git status, read metadata, or check locks must immediately mark the worktree ineligible for deletion (`dirty = true`, `meta = nil`).
6. **Double-Check Under Lock & Commit Logging**: Re-verify status immediately before deletion under an OS file lock, and log the exact 40-character commit SHA to `stderr` before branch deletion (`git branch -D`) so any deleted commit is 100% recoverable via `git branch <name> <sha>`.
