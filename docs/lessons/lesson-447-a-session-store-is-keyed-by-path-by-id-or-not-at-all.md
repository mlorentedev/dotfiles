---
id: "lesson-447-a-session-store-is-keyed-by-path-by-id-or-not-at-all"
type: lesson
status: active
title: "A harness session store is keyed by path, by id, or not at all; read each by its own key"
created: "2026-10-16"
owner: manu
tags: [worktree, harness, sessions, agnostic]
---

# A harness session store is keyed by path, by id, or not at all; read each by its own key

**Context:** `dotf worktree done`, `sweep` and `migrate` refuse to remove a worktree a stopped session would resume in (#1653). The first version read only Claude Code's store, `~/.claude/projects/<key>`, so a pi or Copilot session in the same worktree was stranded without a word (#2569).

**Problem:** "Add the other harnesses" sounds like repeating Claude's lookup with another directory. Measured on the Mac on 2026-10-16, the five stores are keyed three different ways. Claude Code and pi name a directory after the working directory: Claude replaces `/`, `\`, `:` and `.` with `-`; pi strips the leading `/`, replaces the remaining `/` with `-` (keeping `_` and `-`; a `.` is unmeasured) and wraps the result in `--…--`, so `-Users-manu-x` and `--Users-manu-x--` are the same cwd in two spellings. Copilot keys its store by session id (`~/.copilot/session-state/<uuid>/`) and writes the cwd as a field in `workspace.yaml`, so finding a worktree's sessions means reading every session. agy keeps protobuf blobs in SQLite with no readable cwd, and codex and opencode had no store on this Mac (2026-10-16). A loop over one key function would have matched nothing in Copilot's store and reported "no session" for it.

**Solution:** One row per harness, each with its own lookup, and four possible answers per row: consulted, absent (not installed), unreadable (refuse, naming the harness), or not keyed by directory (a stated gap, covered by the process scan while the session is alive). Copilot's age is its `events.jsonl`, not `workspace.yaml`, which is written when the session starts. pi's key is computed only for an absolute POSIX path; its spelling of a Windows path is unmeasured, so that row consults nothing there rather than guessing.

**Why:** A store's layout is the harness's private format, not a convention shared across tools. Before treating N tools as one shape, look at one real entry of each: the shape that differs is the one a generic loop reports as empty.
