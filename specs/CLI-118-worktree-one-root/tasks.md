# Tasks — CLI-118-worktree-one-root

TDD order: each box is a failing test first, then the code that turns it green. One PR per group, each under the executable-LOC cap.

## PR 1 — root and path formula

- [x] `WORKTREE_ROOT` entry in `env-contract.json`. Test: `tests/herdr-config.bats` "WORKTREE_ROOT and herdr's worktrees.directory name the same root" (mutation-checked); full rendering is PR 7.
- [x] `ResolveWorktreePath(root, repo, branch)` slugify table test with the four measured herdr 0.9.3 rows (`feat/bar-baz`, `a/b/c`, `Feat/Up_Case.x`, `fix/x--y`).
- [x] `add` uses it; `ResolveSiblingPath` kept only as the legacy resolver for `done` and `migrate`.
- [x] `done <slug>` resolves under the root and the legacy sibling, refusing a slug found in both (moved here from PR 4: without it `done <slug>` broke as soon as `add` moved).
- [x] `list --all`: no change needed. It scans for main checkouts and lists each one's worktrees through `git worktree list`, wherever they live; `<root>` holds no `.git` directory, so it is skipped.
- [x] gate f / `pathWithin` re-tested with a symlinked root (macOS `/var` → `/private/var` class). Found and fixed: `gateF` compared the cwd by equality, missing a subdirectory and a symlinked path (`TestGateFRefusesACwdInsideTheWorktreeWithoutTheProcessScan`).

## PR 2 — branch validation

- [x] `ValidateBranch` table test: accepts `feat/foo-bar`, refuses `foo`, `Feat/x`, `feat/Foo`, `feat/phase-3-x`, `feat/milestone2`, naming the rule.
- [x] `add` default base from origin/HEAD → origin/main → origin/master (fetch first; refuse when none resolves).

## PR 3 — live-session refusal (#1653)

- [x] `RecentSession(projectsDir, wt, now, window)` with a fake `~/.claude/projects/<key>` dir: refuses when under 6h, allows when stale or absent.
- [x] Wire into `done` and `sweep`; `--ignore-recent-session` distinct from `--force`.
- [ ] File one ticket per non-Claude harness session store (agy, copilot, codex, opencode, pi). Proposed on #1653 during the unattended run; filing needs the owner's ID confirmation (new-ticket).

## PR 4 — adoption and done

- [x] `Classify` returns `unadopted` for a worktree under the root without metadata; `sweep` still refuses it.
- [x] `adopt` writes the lease; `sweep` then reaps a merged, clean, expired one.
- [x] (PR 4b, split for size) `done` deletes the local branch after a verified landing, prints the SHA; keeps it otherwise.

## PR 5 — migrate

- [x] `migrate --dry-run` plan output; `migrate` moves with `git worktree move`; skips refused or existing targets; second run is a no-op.
- [x] Skips name their reason: locked, the caller inside, a recent session or an unreadable session store (sweep's wording, no override), a target that exists.
- [x] Refuses when WORKTREE_ROOT does not resolve (`TestMigrateRefusesWithoutARoot`).
- [x] Cross-OS: the move test compares paths as locations (`samePath`), since Windows lists them in another spelling.

## PR 6 — `--agent` over herdr

- [x] herdr client over `tools.Runner` with a fake: `worktree open`, `agent start`, `agent prompt` request shape and order; no `--wait`.
- [x] States: herdr absent, server down (`herdr status server`), server up. Refusal before any git mutation.
- [x] Harness → kind: a dotfiles harness name is its herdr kind as is (`herdr.Kinds`, measured on 0.9.3); the integration target map does not apply (it maps agy to antigravity-cli, and the kind is agy).
- [x] Agent name derivation and collision refusal; `agent_not_ready` / `agent_blocked` legs exit non-zero.

## PR 7 — rendering, docs, skills, ADR

- [x] (PR 7a, reader) `expandPaths` renders TOML string values behind `paths:`, allowed on a TOML source from manifest version 6 only (a 0.71.0 reader refuses it per entry, so the entry must read as a newer schema). Five mutations killed.
- [ ] (PR 7b, flip, owner lands after every box runs the release carrying 7a) `ai/deploy.json` to version 6 in one move with CLI-113 phase 2 (owner, 2026-10-16: one fleet update, not two): herdr entry declares `paths` and `listKeys`; `ai/herdr/config.toml` uses `{WORKTREE_ROOT}`; `tests/herdr-config.bats` compares the token.
- [ ] Trust lists: Copilot `{WORKTREE_ROOT}/*/*`, agy `{WORKTREE_ROOT}` itself (agy does not read the glob, #2533); #2504's test and the exact-list assertions updated. JSON entries already render `paths`, and `WORKTREE_ROOT` resolves to its default when unset (measured: `dotf env path WORKTREE_ROOT` on 0.71.0), so this needs no fleet floor.
- [x] AGENTS.md §9; vault `00_meta/skills/using-git-worktrees` and `handoff` (knowledge `f9f68924`), records refreshed with `compile-harness.sh --refresh`.
- [x] ADR-032 dispatch clarification; lesson for the herdr path rule (lesson 436 already records it).
- [x] Doc drift found in passing: `done_proc_other.go:8`, `sweep_proc_other.go:26` and `done`'s Long text still say Linux-only after #2293.
