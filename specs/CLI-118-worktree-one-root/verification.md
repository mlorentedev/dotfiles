---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-118-worktree-one-root

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 root declared and rendered: declared in `env-contract.json` (PR 1), kept equal to herdr's directory by `tests/herdr-config.bats` "WORKTREE_ROOT and herdr's worktrees.directory name the same root" (mutation: changing herdr's directory fails it). Rendering into the three configs is PR 7.
- [x] AC2 path: `TestResolveWorktreePath` (four herdr 0.9.3 rows measured on #2503), `TestAddUnderWorktreeRoot`, `TestAddUnderWorktreeRootFollowsBranch` (PR 1).
- [x] AC3 branch validation (PR 2): `TestValidateBranch` (shape rows, planning-token rows, and near misses `phaseout-legacy`, `milestones-view`, `emphase` that must stay accepted), `TestAddRefusesAnInvalidBranchBeforeCreatingAnything` (given, defaulted and `--path` branches), `TestAcceptedBranchesSlugToThemselves`, `TestPlanningTokensMatchTheEnforcedRule` (mutation: adding `wave` to the list fails it). Default base: `TestAddCreatesFromTheRemoteDefaultBranch`, `TestAddKeepsAnExplicitBase`, `TestAddRefusesWhenNoRemoteDefaultResolves`, `TestRealDefaultBase` (local origin, with and without origin/HEAD).
- [ ] AC4 `--agent` (PR 6): `TestLaunchOpensStartsAndPromptsWithoutWaiting` (order, no `--wait`), `TestLaunchWithoutAPromptOnlyStarts`, `TestLaunchFailsOnANotReadyOrBlockedAgent`, `TestPreflightNamesTheStateItRefuses` (unknown kind, herdr absent, server down, `agent get` failing, name taken), `TestAgentName`, `TestEveryHarnessIsAKind` (model-map's harnesses against `herdr.Kinds`); CLI: `TestWorktreeAddStartsAnAgentInHerdr`, `TestWorktreeAddRefusesBeforeCreatingWhenHerdrIsDown`, `TestWorktreeAddRefusesAPromptWithoutAnAgent`. Preflight wiring and the `agent get` code check mutation-checked. Response shapes measured 2026-10-15 on herdr 0.9.3 in a throwaway session (`--session cli118probe`, deleted after): `status server --json` exits 0 with `running` false when down; `worktree open` answers `.result.root_pane.pane_id`; errors are `{"error":{"code","message"}}` with exit 1; `agent get` of a free name is `agent_not_found`. Open: a live run against a real server, which needs the owner present, since herdr has no session environment variable and the dev build would drive the operator's default server.
- [x] AC5 adoption (PR 4a): `TestClassify` (no lease is UNADOPTED, naming adopt), `TestAdoptWritesALeaseOnce` (second adopt keeps the lease), `TestAdoptRefusesWhatIsNotALinkedWorktree` (main checkout, plain dir, missing path), `TestSweepReapsAnAdoptedWorktreeOnly`, `TestWorktreeAdoptTakesAGitCreatedWorktree` (list shows UNADOPTED, adopt by slug, then not).
- [ ] AC6 done: slug resolution in PR 1 (`TestWorktreeDoneTakesTheSlugAddWasGiven`, `TestWorktreeDoneTakesTheSlugOfALegacySibling`, `TestWorktreeDoneRefusesASlugInBothLayouts`); branch deletion in PR 4b: `TestDoneDeletesALandedBranch` (ancestor of the base), `TestDoneDeletesASquashLandedBranch` (merged PR head), `TestDoneKeepsAnUnlandedBranchEvenWithForce`; both directions mutation-checked.
- [ ] AC7 live session: gate f cwd check fixed in PR 1 (`TestGateFRefusesACwdInsideTheWorktreeWithoutTheProcessScan`). Transcript check in PR 3: `TestRecentSession` (worktree, subdirectory, stale, other worktree, none), `TestDoneRefusesARecentSessionUnlessIgnored` (`--force` does not override), `TestSweepSkipsAWorktreeWithARecentSession`, `TestWorktreeDoneRefusesARecentSessionUntilIgnored` (CLI flag); both wirings mutation-checked. `migrate` gets it in PR 5.
- [ ] AC8 migrate (PR 5): `TestMigrateDryRunPlansWithoutMoving`, `TestMigrateMovesAndASecondRunHasNothingToDo`, `TestMigrateSkipsWhatItMustNotMove` (target exists, caller inside, recent session, locked), `TestMigrateRefusesWithoutARoot`, `TestWorktreeMigrateMovesALegacySibling` (CLI: dry run, move, no-op). Locked, dry-run and target-exists guards mutation-checked; the locked row runs as a dry run because `git worktree move` refuses a locked worktree on its own and hid the missing guard. Measured 2026-10-15 on the Mac: `migrate --dry-run --repo ~/Projects/dotfiles` printed `Nothing to migrate`, which matches `git worktree list` (every worktree already under the root).
- [ ] AC9, AC10 docs, skills, ADR: PR 7.
- [ ] AC11 cross-shell / cross-OS: per PR.

## Test status

- PR 1, macOS: `cd cli && go build ./... && go vet ./... && go test ./...` all ok; `GOOS=windows go vet ./internal/worktree/ ./internal/cmd/` clean; `golangci-lint run` (pinned 2.12.2) 0 issues; `bats tests/herdr-config.bats` 6 ok; env-contract consumers (`tests/env-contract.bats`, `guard-setup-preexports-present.bats`, `opencode.bats`, `setup-linux.bats`) 131 ok.
- No regressions in existing test suite: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `done <slug>` resolution moved from PR 4 into PR 1: once `add` writes under the root, a sibling-only `done <slug>` would have broken in between.
- `list --all` needed no change: it finds worktrees through `git worktree list`, not by directory layout.
- `add` refuses when WORKTREE_ROOT resolves empty and no `--path` is given (`TestAddRefusesWithoutARoot`), and `done` says the root was not searched (`TestWorktreeDoneSaysWhenTheRootWasNotSearched`). `env.ResolvePath` returns "" when env-contract.json is neither deployed nor reachable from the cwd; falling back to the sibling there would pick the layout by accident. Measured on the Mac the same day: `DOTFILES_REPO_DIR` names the main checkout, which reads its own env-contract.json first, and until PR 1 lands that file has no WORKTREE_ROOT. The dev build refused (`WORKTREE_ROOT could not be resolved`) where the previous code would have made `dotfiles-wt-<slug>` silently.
- dotf creates the worktree and herdr only opens it (owner, 2026-10-14), so the lease, branch check and isolation gate hold on every path, Windows included.

- `done` identifies a worktree under the root by its branch slug, the one name herdr's layout gives it, not by a second slug recorded in the lease: a herdr-made or adopted worktree would never carry that slug (#2527 review, PR-Agent finding 1).
- The planning tokens are exactly the words of `harness/enforced/no-phase-references.md` (phase, milestone), not the four the first draft named: that file is extracted from the vault's git-workflow pattern, and a validator that bans words the doctrine does not would be a second, unwritten rule. Widening it is a vault edit first.
- `add` with no `--base` used to branch from the caller's HEAD. It now fetches and uses origin's default branch, and refuses when none resolves; a failed fetch (offline) warns and uses the last-fetched refs. git 2.48+ restores origin/HEAD on fetch (`remote.<name>.followRemoteHEAD`), so the master fallback is reached only where that is off or the remote has no HEAD.

- The transcript key is computed from the physical path. Claude Code keys a project on getcwd: on this Mac the one scratchpad session under /tmp is `-private-tmp-…` and no project directory starts with `-tmp-` or `-var-`. done passes git's toplevel, which is physical too; the path as given is kept as a second key.
- A project directory whose name extends the worktree's key by `-` counts (a session started in `wt/cli`). It also matches a sibling `feat-x-y` for `feat-x`: a false refusal, the safe side, and the message names the transcript.

- UNADOPTED applies to every linked worktree without a lease, not only those under WORKTREE_ROOT as the proposal says: a git-created sibling is just as unknown to sweep, and `adopt` works on any linked worktree. Nothing outside the worktree package read the old ACTIVE "no dotf metadata" state (git grep).

- `migrate` moves a worktree out of the directory a Claude Code session is keyed on (`getcwd`), so it skips any worktree with a transcript in the last six hours rather than orphan the session; the help text says a session started in the old path does not follow it.

- AC7's `--ignore-recent-session` exists on `done` only. `sweep` and `migrate` act on many worktrees at once, often unattended, so a worktree with a recent session is skipped with the reason printed, never overridden; the operator who knows the session is dead runs `done <it> --ignore-recent-session`, one worktree at a time.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-118-worktree-one-root/` -> `specs/archive/CLI-118-worktree-one-root/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
