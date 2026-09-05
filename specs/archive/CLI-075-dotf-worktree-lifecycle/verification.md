---
tags: [spec, verification, templates]
created: "2026-09-04"
---

# Verification - CLI-075-dotf-worktree-lifecycle

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof:

- [x] AC1 (`dotf worktree list` tabular/JSON output) -> `TestListWorktrees`, `TestListWithRunnerFromInsideWorktreeDoesNotMarkWorktreeAsMain`, `TestPRQueryCache` (PASS)
- [x] AC2 (submodule filtering via `modules/` check) -> `TestFilterSubmodules` (PASS)
- [x] AC3 (standardized creation with external sibling and lease) -> `TestAddWorktreeWithRunner` (PASS)
- [x] AC4 (fail-closed reaper with 6 gates and gitignored scratchpad protection) -> `TestSweepFailClosed`, `TestSweepTOCTOUDirtyErrorFailsClosed`, `TestSweepTOCTOUUnmergedFailsClosed`, `TestSweepPreservesGitignoredLocalFiles`, `TestIsDisposableIgnoredPath`, `TestHasNonDisposableIgnored` (PASS)
- [x] AC5 (commit SHA recorded before branch deletion) -> `TestSweepLogsSHA` (PASS)
- [x] AC6 (cross-platform file locking) -> `TestSweepFileLock` (PASS)
- [x] AC7 (safe self-service teardown) -> `TestDoneTeardown`, `TestDoneRefusesUnpushedCommitsWithNoUpstream`, `TestDoneSucceedsOnCleanBranchWithoutCommits`, `TestResolveWorktreeAndMainRepoRoot`, `TestDonePreservesGitignoredLocalFiles` (PASS)

## Test status

- Unit test suite: `go -C cli test -v ./internal/worktree/...` -> 26/26 top-level test functions passing (exit 0)
- Linter: `golangci-lint run` in `cli/` -> 0 issues (exit 0)
- Cyclomatic complexity: `gocyclo -over 9 cli/internal/worktree/*.go cli/internal/cmd/worktree.go` -> 0 functions > 9 across all production AND test functions (exit 0)
- Cross-compilation: Windows (`amd64`) and Darwin (`arm64`) build cleanly (exit 0)
- Regression test suite: `go -C cli test ./...` -> all packages passing (exit 0)
- Live smoke test:
  - `dotf worktree list` correctly discovered 7 worktrees in live repo, correctly identified `main repository` strictly on entry 0 (never on linked worktrees), evaluated current worktree as `DIRTY uncommitted changes`, resolved GitHub PR status via `ParseGitHubSlug`, and handled `--json`.
  - `dotf worktree list --all` successfully scanned all sibling repositories in `~/Projects` in seconds.
  - `dotf worktree sweep --dry-run` reported `Found 0 reapable worktree(s), 7 skipped` (zero false positives).

## Decisions made during implementation

1. **Fail-Closed Default (F1):** The reaper never deletes based on heuristics alone. A worktree without explicit `.dotf-worktree.json` metadata (or with `reap_ok: false`) is strictly refused (`StateActive`).
2. **Fail-Closed Error Handling (F2):** Any error executing git status or loading metadata fails closed (`dirty = true`, `meta = nil`).
3. **Repository Slug Resolution (F3):** `ParseGitHubSlug` extracts `owner/repo` from remote origin URLs so `gh pr view --repo` executes reliably from any working directory.
4. **Gitignored Content Policy & Scratchpad Protection (F4):** Distinguishes disposable build caches (`node_modules/`, `target/`, `.venv/`, `dist/`, etc. and `.dotf-worktree.json`) from untracked/gitignored scratchpad notes or local configs (`.env`, `*.tmp`, `notes*`). Both `sweep` and `done` inspect `git status --ignored --porcelain` via `HasNonDisposableIgnored`; any non-disposable ignored file classifies the worktree as DIRTY and refuses reap or teardown without `--force` to guarantee zero risk of data loss. Covered by named tests `TestSweepPreservesGitignoredLocalFiles` and `TestDonePreservesGitignoredLocalFiles`.
5. **Host Process CWD Guard (F5):** Scans `/proc/[0-9]*/cwd` on host systems to protect interactive terminal sessions from deletion. Evaluated during candidate discovery and re-checked under lock immediately prior to deletion.
6. **Spec Drift Resolution (F6-F8):** Added `--all` sibling repo scanning, Gate A auto-commit detection, and unpushed commit protection in `done`.
7. **Exact SHA Recovery (F9):** Logs full 40-character commit SHA to stderr before branch deletion for 100% undoability via `git branch <name> <sha>`.
8. **In-Worktree Teardown Resolution (F10):** `ResolveMainRepoRoot` uses `git rev-parse --git-common-dir` to locate the superproject root, and `ResolveWorktreeRoot` uses `git rev-parse --show-toplevel`, allowing `dotf worktree done` to tear down a worktree directly from inside it or its subdirectories.
9. **PR Query Caching (F11):** `RealGitRunner` caches PR resolution state (`prCache map[string]bool`) across calls and evaluates local `merge-base --is-ancestor` first with fallback to default branches via `resolveBaseRef`, preventing GitHub API rate limiting. Covered by `TestPRQueryCache`.
10. **Main Worktree Invariant (F12):** `isMainWorktree` checks if `.git` is a directory (physical Git architecture guarantee for main repository) with index fallback, eliminating ordering assumptions.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? Yes: lesson on fail-closed worktree garbage collection and container lease heartbeats.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? Covered by spec CLI-075.
- [ ] New pattern candidate for `00_meta/patterns/`? Consider promoting fail-closed workspace lease pattern to cross-agent doctrine.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-075-dotf-worktree-lifecycle/` -> `specs/archive/CLI-075-dotf-worktree-lifecycle/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
