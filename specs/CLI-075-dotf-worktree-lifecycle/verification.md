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

- Unit test suite: `go -C cli test -v ./internal/worktree/...` -> 14/14 passing (exit 0)
- Regression test suite: `go -C cli test ./...` -> all packages passing (exit 0)
- Live smoke test:
  - `dotf worktree list` correctly discovered 7 worktrees in live repo, correctly identified `main repository` for active worktrees, flagged `DIRTY` on uncommitted branches, resolved GitHub PR status via `ParseGitHubSlug`, and handled `--json`.
  - `dotf worktree list --all` successfully scanned all sibling repositories in `~/Projects` in seconds.
  - `dotf worktree sweep --dry-run` reported `Found 0 reapable worktree(s), 6 skipped` (zero false positives).

## Decisions made during implementation

1. **Fail-Closed Default (F1):** The reaper never deletes based on heuristics alone. A worktree without explicit `.dotf-worktree.json` metadata (or with `reap_ok: false`) is strictly refused (`StateActive`).
2. **Fail-Closed Error Handling (F2):** Any error executing git status or loading metadata fails closed (`dirty = true`, `meta = nil`).
3. **Repository Slug Resolution (F3):** `ParseGitHubSlug` extracts `owner/repo` from remote origin URLs so `gh pr view --repo` executes reliably from any working directory.
4. **Gitignored Content Policy (F4):** Ephemeral worktree deletion cleans build caches (`node_modules/`, `target/`). Per ADR-028 doctrine, agents never store credentials in `.env` files; secrets are injected dynamically.
5. **Host Process CWD Guard (F5):** Scans `/proc/[0-9]*/cwd` on host systems to protect interactive terminal sessions from deletion.
6. **Spec Drift Resolution (F6-F8):** Added `--all` sibling repo scanning, Gate A auto-commit detection, and unpushed commit protection in `done`.
7. **Exact SHA Recovery (F9):** Logs full 40-character commit SHA to stderr before branch deletion for 100% undoability via `git branch <name> <sha>`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? Yes: lesson on fail-closed worktree garbage collection and container lease heartbeats.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? Covered by spec CLI-075.
- [ ] New pattern candidate for `00_meta/patterns/`? Consider promoting fail-closed workspace lease pattern to cross-agent doctrine.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-075-dotf-worktree-lifecycle/` -> `specs/archive/CLI-075-dotf-worktree-lifecycle/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
