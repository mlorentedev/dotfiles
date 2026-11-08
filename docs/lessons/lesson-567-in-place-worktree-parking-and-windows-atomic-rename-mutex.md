---
id: "lesson-567-in-place-worktree-parking-and-windows-atomic-rename-mutex"
type: lesson
status: active
title: "In-place worktree parking avoids Windows directory handle locks and concurrent os.Rename requires an in-process mutex"
created: "2026-11-08"
---

# In-place worktree parking avoids Windows directory handle locks and concurrent os.Rename requires an in-process mutex

## Context

In `HARNESS-186` (#3255), we introduced persistent named lanes (`feat-lane-primary`, `feat-lane-secondary`, `fast-track-lane`) and a sticky persona gate backed by JSON state files in `$STATE_DIR/gate/`.

## Problem

1. **Windows directory deletion locks on worktrees:** Ephemeral `git worktree remove` calls fail on Windows (`Permission denied`) whenever an interactive shell, IDE watcher, or backgrounded `post-checkout` hook (`board-pickup.sh`) holds an open handle to the worktree directory. Furthermore, `sweep`'s `GateF` (`sweep_proc_other.go`) fails closed on Windows (`Inside: true`), preventing automated worktree removal.
2. **Concurrent `os.Rename` collisions on Windows:** `atomicWriteFile` in `cli/internal/harness/dispatch.go` wrote a temp file and called `os.Rename(tmpPath, dst)` with no synchronization. Under a 16-goroutine concurrent stress test (`TestAtomicWriteFile_ConcurrentWritesOnWindowsDoNotFail`), `os.Rename` failed with `Access is denied` on Windows even with a short retry loop.

## Solution

1. **Park persistent lanes in-place (`Park()`):** Never delete persistent named lane directories (`reap_ok: false`). Instead, verify `branchLanding` tip SHA containment, run `git checkout --detach origin/main`, delete the merged local branch (`git branch -D`), remove `.dotf-lane-prompt.md` (and classify `.dotf-lane-prompt.md` as disposable in `clean.go`), and reset `.dotf-worktree.json` to `state: "PARKED"`. Also detach CWD (`cd "${HOME:-/}"`) in backgrounded `board-pickup.sh` before running network commands.
2. **Serialize `atomicWriteFile` with an in-process mutex + bounded backoff:** Guard `atomicWriteFile` with `atomicWriteMu sync.Mutex` alongside a 15-attempt `os.Rename` retry loop so concurrent goroutines and external file scanners on Windows never cause `Access is denied`.
