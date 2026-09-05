---
tags: [spec, verification, templates]
created: "2026-09-04"
---

# Verification - CLI-075-dotf-worktree-lifecycle

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof:

- [x] AC1 (`dotf worktree list` tabular/JSON output) -> `TestListWorktrees` (PASS)
- [x] AC2 (submodule filtering via `modules/` check) -> `TestFilterSubmodules` (PASS)
- [x] AC3 (standardized creation with external sibling and lease) -> `TestAddWorktreeWithRunner` (PASS)
- [x] AC4 (fail-closed reaper with 6 gates) -> `TestSweepFailClosed` (PASS)
- [x] AC5 (commit SHA recorded before branch deletion) -> `TestSweepLogsSHA` (PASS)
- [x] AC6 (cross-platform file locking) -> `TestSweepFileLock` (PASS)
- [x] AC7 (safe self-service teardown) -> `TestDoneTeardown` (PASS)

## Test status

- Unit test suite: `go -C cli test -v ./internal/worktree/...` -> 7/7 passing (exit 0)
- Regression test suite: `go -C cli test ./...` -> all packages passing (exit 0)
- Live smoke test:
  - `dotf worktree list` correctly discovered 7 worktrees in live repo, correctly identified `main repository` for active worktrees, flagged `DIRTY` on uncommitted branches, and handled `--json`.
  - `dotf worktree sweep --dry-run` reported `Found 0 reapable worktree(s), 6 skipped` (zero false positives).

## Decisions made during implementation

1. **Fail-Closed Default:** The reaper never deletes based on heuristics alone. It requires explicit metadata (`reap_ok`), expired lease, clean git status, and confirmed merged PR.
2. **In-Tree Lease vs Host `/proc`:** Primary liveness signal is `.dotf-worktree.json` inside the worktree filesystem, providing container and PID-namespace agnosticism. Host `/proc` is a secondary floor for interactive shells.
3. **Cross-Platform Lock:** Uses `syscall.Flock` on Unix and `CreateFile` with `dwShareMode=0` on Windows, matching `cli/internal/agent/lock_*.go`.
4. **Git-Aware Recovery:** Logs exact commit SHA to stderr before deleting branches so `git branch <name> <sha>` offers 100% instant recovery of committed state.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? Yes: lesson on fail-closed worktree garbage collection and container lease heartbeats.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? Covered by spec CLI-075.
- [ ] New pattern candidate for `00_meta/patterns/`? Consider promoting fail-closed workspace lease pattern to cross-agent doctrine.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-075-dotf-worktree-lifecycle/` -> `specs/archive/CLI-075-dotf-worktree-lifecycle/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
