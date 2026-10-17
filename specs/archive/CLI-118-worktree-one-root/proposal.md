---
id: "CLI-118-worktree-one-root"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-10-14"
issue: "mlorentedev/dotfiles#2503"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, worktree, herdr]
template_version: "1.0"
wip_override: "FIX-WIN-TUI-001 kept active pending ConPTY (#1650); CLI-118 prioritised by the owner (10 active, limit 10, 2026-10-14)"
---

# One worktree root, and `dotf worktree` as the orchestrator of parallel agent worktrees

## Why

Worktrees live in two layouts at once. `dotf worktree add` creates `<repo>-wt-<slug>` siblings (`ResolveSiblingPath`). herdr, the cockpit (ADR-044), creates `<root>/<repo>/<branch-slug>` and allows only the root to be configured. The owner fixed the root as `~/Projects/worktrees` (#2503, 2026-10-14). Every worktree herdr creates reads "no dotf metadata" in `list`, and `sweep` never reaps it. `done <slug>` cannot find it, and `done` leaves the local branch behind. An agent that finishes a task cannot move to the next one without removing its own worktree, which closes its pane and makes the session un-resumable (#1653).

## What

1. **One declared root.** A contract variable `WORKTREE_ROOT` (`env-contract.json`, default `$HOME/Projects/worktrees` on every OS) is the single declaration. `dotf deploy` renders it into `ai/herdr/config.toml` (`expandPaths` learns TOML string values), into agy `trustedWorkspaces` and into Copilot `trustedFolders`. The test from #2504 asserts the three agree.
2. **Path formula = herdr's.** `<WORKTREE_ROOT>/<repo>/<slugify(branch)>`, where slugify lowercases, replaces every run of non-alphanumerics with one `-` and trims. Measured against herdr 0.9.3 `worktree create --branch`, four inputs: `feat/bar-baz` gave `feat-bar-baz`, `a/b/c` gave `a-b-c`, `Feat/Up_Case.x` gave `feat-up-case-x`, `fix/x--y` gave `fix-x-y`. `ResolveSiblingPath` is replaced by `ResolveWorktreePath(root, repo, branch)`.
3. **`dotf worktree add <slug>`** creates from the remote default branch by default, using the existing resolution order (origin/HEAD, origin/main, origin/master), since `knowledge` is on master. It requires a `<type>/<slug>` branch (lowercase type, kebab slug, no phase/milestone token as a word, optionally followed by digits; exactly the words of `harness/enforced/no-phase-references.md`, which a test keeps in step) and writes the lease, as it does today. With `--agent <harness> --prompt-file <f>`:
   - herdr installed and its server reachable → dotf creates the worktree with git itself (owner, 2026-10-14; so lease, validation and isolation gate apply on every path). It then runs `herdr worktree open --cwd <repo> --path <wt>`, which was measured to return `root_pane` for a checkout herdr did not create, without `--trust-repository`. Next comes `herdr agent start --kind <k> --pane <root_pane>`, with the harness → kind mapping taken from `cli/internal/herdr/integrations.go`, then `herdr agent prompt` with the file's content, and `add` returns. The agent name is the path slug truncated to herdr's `[a-z][a-z0-9_-]{0,31}`. A live agent with that name refuses before any mutation (`herdr agent list`). If `agent start` returns `agent_not_ready`, no prompt is sent: dotf exits non-zero naming the pane, and the worktree and pane stay for the operator. If `agent prompt` returns `agent_blocked`, the same applies. No `--wait`; dotf reports that the prompt was submitted, never how the agent did.
   - herdr installed, server down → `--agent` refuses and names that state. Without `--agent`, herdr is never called.
   - herdr absent (Windows today) → `--agent` refuses, and plain `add` is git only.
4. **The jump between worktrees.** From a finished worktree, `add <next> --agent … --prompt-file …` opens the next one. The calling agent then runs `/handoff` and stops. No command removes the worktree it is called from.
5. **Adoption, fail-closed (owner, 2026-10-14).** `list` gives a worktree under the root with no dotf metadata its own status, `unadopted`, instead of "no dotf metadata". `dotf worktree adopt <path|slug>` writes the lease (`reap_ok: true`). `sweep` reaps only adopted or dotf-created worktrees; missing metadata stays a refusal (lesson 272, BUG-093). The recommended path is to always create through `dotf worktree add`.
6. **`done` finishes in one command.** It resolves a slug under the new root, and under the legacy sibling until migration ends. It refuses ambiguity. After removal it deletes the local branch only when it has verified the branch landed (merged PR or ancestor of base, the same condition that already waives `--force`), and prints the deleted SHA.
7. **No command removes a live session's worktree (#1653).** `done`, `sweep` and `migrate` all refuse when the caller's cwd or a process ancestor is inside the target (existing `checkCallerOutside`, which `--force` cannot override). That process check is the harness-agnostic floor. On top of it, a per-harness recent-session check: they refuse when the target's Claude project directory (`dotf mem project-key`) holds a transcript modified in the last 6 hours (#1653 AC1). The other five harnesses' session stores get a ticket each, so the rule is not Claude-only by omission. That second refusal has its own override flag `--ignore-recent-session` on `done`, distinct from `--force`; `sweep` and `migrate` act on many worktrees at once, often unattended, so they skip such a worktree and print why, with no override.
8. **Migration.** `dotf worktree migrate [--dry-run]` moves every `<repo>-wt-<slug>` of the repo to the new path with `git worktree move`. It refuses per worktree under rule 7, and refuses a target that already exists. It prints that the Claude auto-memory directory does not follow the move.
9. **Docs and skills read the root.** AGENTS.md §9 states `<WORKTREE_ROOT>/<repo>/<branch-slug>`. The `using-git-worktrees` and `handoff` skills are generated from the vault (`00_meta/skills/`), so the edit goes there and is re-deployed with `compile-harness.sh`; the handoff step stops removing the session's own worktree (#1653 AC3).
10. **ADR-032 clarification (owner, 2026-10-14).** "Dispatch" means the executor seam (`dotf agent run`: run role A on task T, return R). Launching an interactive agent into a pane an operator watches, returning after submission and reporting nothing about the outcome, is cockpit use and allowed.

## Out of scope

- An executor backend over herdr: ADR-032 amendment 2026-10-13, decision 5 is still the reopen trigger.
- Splitting dirty from ignored-content in `IsDirty` and `--discard-ignored` (#1607): separate bug, unchanged here.
- Windows ancestor-process walk (ToolHelp32) for `done`/`sweep`: the `other` leg stays fail-closed as today.
- Moving Claude auto-memory between project directories after a migration.

## Risks / open questions

- **Root on Windows.** `%USERPROFILE%\Projects\worktrees\<repo>\<branch-slug>` adds about 10 characters over the sibling layout. The corporate box has no admin, so long paths cannot be enabled. Measure the deepest path of the largest repo against 260 before setting the Windows default.
- **TOML rendering.** `expandPaths` walks JSON today. Teaching it TOML through the merge path in `deploy.go` must keep `~` literals untouched for keys that are not declared `paths:`.
- **herdr version drift.** The probe is against 0.9.3. A test with a fake `herdr` pins the request shape, and `dotf doctor` should report a herdr whose `api schema` lacks `WorktreeOpenParams.path`.
- **gate f and path canonicalisation.** The cwd comparison must canonicalise both sides (`/var/folders` vs `/private/var` on macOS). Re-test on Linux and macOS, and the Windows leg with fakes.
- **Slug resolution ambiguity.** `done foo` could match `feat-foo` and `fix-foo`: refuse and list both.

## Acceptance criteria

- [x] AC1: `WORKTREE_ROOT` is declared once in `env-contract.json`. `dotf deploy` renders it into herdr `config.toml`, agy `trustedWorkspaces` and Copilot `trustedFolders`, and a test fails if any of the three diverges.
- [x] AC2: `dotf worktree add foo` creates `<root>/<repo>/feat-foo` from the remote default branch on branch `feat/foo` and writes the lease. The slugify table test pins exactly the four measured herdr 0.9.3 rows.
- [x] AC3: `add` refuses a branch that is not `<type>/<kebab-slug>`, or that carries a phase/milestone token, naming the rule. Table-driven test.
- [x] AC4: `add --agent <h> --prompt-file <f>` with a fake `herdr` calls, in order, `worktree open --cwd --path`, `agent start --kind --pane <root_pane>` and `agent prompt <name> <text>` without `--wait`, then returns. With herdr absent or its server down, or an agent of the same name live, it refuses before creating anything, naming the state. `agent_not_ready` and `agent_blocked` exit non-zero without sending (or after failing to send) the prompt.
- [x] AC5: `list` shows `unadopted` for a herdr-created worktree under the root. `adopt` writes the lease. `sweep` reaps an adopted merged worktree and refuses an unadopted one.
- [x] AC6: `done <slug>` resolves under the new root and the legacy sibling, refuses ambiguity, and deletes the local branch only after a verified landing, printing the SHA.
- [x] AC7: `done`, `sweep` and `migrate` refuse a target containing the caller's cwd or ancestor regardless of `--force`. They also refuse one whose Claude project dir has a transcript under 6h old: `done` unless `--ignore-recent-session` is passed, `sweep` and `migrate` by skipping it with the reason printed, with no override. Tests cover both directions with a fake transcript directory (#1653 AC1, AC2, AC4).
- [x] AC8: `migrate --dry-run` lists every `<repo>-wt-*` with its target. `migrate` moves them with `git worktree move` and skips (naming why) any refused by AC7 or whose target exists. A second run reports nothing to do.
- [x] AC9: AGENTS.md §9 and the `using-git-worktrees` and `handoff` skills (vault source, re-deployed) state the new layout. The handoff skill no longer removes the session's own worktree.
- [x] AC10: ADR-032 carries the dispatch clarification.
- [x] AC11: the tests pass under bash and zsh for the shell layer and on Linux and macOS for Go. `GOOS=windows go vet ./...` is clean, and the Windows leg is covered with fakes.

## References

- Issue #2503 (decisions recorded there), #1653 (live-session removal), #1607 (out of scope), #2504 (config half).
- ADR-025 (env contract), ADR-032 amendment 2026-10-13 (herdr cockpit), ADR-044 (herdr replaces Orca).
- Lessons 272 (fail-closed GC and in-tree leases), 355 (squash-merge containment), 392, 431.
